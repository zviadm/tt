package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/zviadm/tt"
	"golang.org/x/mod/modfile"
)

var (
	verbose   = flag.Bool("v", false, "verbose output")
	ttMemory  = flag.String("tt.memory", "1gb", "memory limit passed to docker run command")
	ttRebuild = flag.Bool("tt.rebuild", false, "if true, will rebuild docker image")

	// go test flags
	runFlag   = flag.String("run", "", "see: go help testflag")
	benchFlag = flag.String("bench", "", "see: go help testflag")
	countFlag = flag.Int("count", 0, "see: go help testflag")
	raceFlag  = flag.Bool("race", false, "see: go help testflag")
	shortFlag = flag.Bool("short", false, "see: go help testflag")
)

type pkgConfig struct {
	MountDir     string
	GoModDir     string // Parent path that contains go.mod file
	TTDockerfile string
	RelativePath string // Relative path from GoModDir

	ModPath   string // Go module path
	GoVersion string // Go version defined for module
}

func pkgConfigFor(pkg string) (*pkgConfig, error) {
	pkgD, err := filepath.Abs(pkg)
	if err != nil {
		return nil, err
	}
	d := pkgD
	cfg := &pkgConfig{}
	for {
		if cfg.GoModDir == "" {
			if _, err := os.Stat(path.Join(d, "go.mod")); !os.IsNotExist(err) {
				cfg.GoModDir = d
				cfg.RelativePath = "." + pkgD[len(d):]
			}
		}
		if cfg.TTDockerfile == "" {
			dockerfile := path.Join(d, "tt.Dockerfile")
			if _, err := os.Stat(dockerfile); !os.IsNotExist(err) {
				cfg.TTDockerfile = dockerfile
			}
		}
		if _, err := os.Stat(path.Join(d, ".git")); !os.IsNotExist(err) {
			cfg.MountDir = d
			break
		}
		if d == "/" {
			break
		}
		d = path.Dir(d)
	}
	if cfg.GoModDir == "" {
		return nil, errors.New(fmt.Sprintf("go.mod not found for: %s", pkgD))
	}
	if cfg.MountDir == "" {
		cfg.MountDir = cfg.GoModDir
	}
	goModFile := path.Join(cfg.GoModDir, "go.mod")
	goModData, err := ioutil.ReadFile(goModFile)
	if err != nil {
		return nil, err
	}
	f, err := modfile.ParseLax(goModFile, goModData, nil)
	if err != nil {
		return nil, err
	}
	if f.Module.Mod.Path == "" {
		return nil, errors.New(fmt.Sprintf("module path not found in %s", goModFile))
	}
	if f.Go.Version == "" {
		return nil, errors.New(fmt.Sprintf("go version not found in %s", goModFile))
	}
	cfg.ModPath = f.Module.Mod.Path
	cfg.GoVersion = "go" + f.Go.Version
	return cfg, nil
}

func findPkgGroups(pkgs []string) ([][]*pkgConfig, error) {
	pkgGroups := make(map[string][]*pkgConfig)
	for _, pkg := range pkgs {
		if strings.HasSuffix(pkg, "/...") || pkg == "..." {
			pathToExpand, err := filepath.Abs(pkg[:len(pkg)-3])
			if err != nil {
				return nil, err
			}
			goModPaths := []string{pathToExpand}
			err = filepath.Walk(
				pathToExpand,
				func(p string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if info.IsDir() || info.Name() != "go.mod" || path.Dir(p) == pathToExpand {
						return nil
					}
					goModPaths = append(goModPaths, path.Dir(p))
					return nil
				})
			if err != nil {
				return nil, err
			}
			for _, goModPath := range goModPaths {
				cfg, err := pkgConfigFor(path.Join(goModPath, "..."))
				if err != nil {
					return nil, err
				}
				pkgGroups[cfg.ModPath] = append(pkgGroups[cfg.ModPath], cfg)
			}
		} else {
			cfg, err := pkgConfigFor(pkg)
			if err != nil {
				return nil, err
			}
			pkgGroups[cfg.ModPath] = append(pkgGroups[cfg.ModPath], cfg)
		}
	}
	modules := make(sort.StringSlice, 0, len(pkgGroups))
	for modPath := range pkgGroups {
		modules = append(modules, modPath)
	}
	modules.Sort()
	r := make([][]*pkgConfig, len(modules))
	for idx, modPath := range modules {
		r[idx] = pkgGroups[modPath]
	}
	return r, nil
}

func runTests(cacheDir string, pkgs []*pkgConfig) error {
	imgName := "tt-" + path.Base(pkgs[0].ModPath)
	if !*ttRebuild {
		out, err := exec.Command("docker", "images", "-q", imgName).CombinedOutput()
		if err != nil {
			return err
		}
		if len(out) == 0 {
			*ttRebuild = true
		}
	}
	if *ttRebuild {
		cmd := exec.Command("docker", "build", "-t", imgName, "-f", pkgs[0].TTDockerfile, ".")
		cmd.Stdin = os.Stdin
		if *verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			fmt.Println("$:", cmd)
		} else {
			fmt.Println("building:", imgName, "from", pkgs[0].TTDockerfile, "...")
		}
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	workDir := path.Join(tt.SourceDir, pkgs[0].GoModDir[len(pkgs[0].MountDir):])
	_ = exec.Command("docker", "rm", "--force", imgName).Run()
	args := []string{
		"run", "-i", "-t",
		"--name", imgName,
		"-v", pkgs[0].MountDir + ":" + tt.SourceDir + ":cached",
		"-v", cacheDir + ":" + tt.CacheDir + ":delegated",
		"-w", workDir,
		"--memory", *ttMemory,
		"--memory-swap", *ttMemory,
		"--cap-add", "NET_ADMIN",
		imgName + ":latest",
		tt.GoBin(pkgs[0].GoVersion),
		"test", "-p", "1", "-failfast",
	}
	if *verbose {
		args = append(args, "-v")
	}
	if *raceFlag {
		args = append(args, "-race")
	}
	if *shortFlag {
		args = append(args, "-short")
	}
	if *runFlag != "" {
		args = append(args, "-run", *runFlag)
	}
	if *benchFlag != "" {
		args = append(args, "-bench", *benchFlag)
	}
	if *countFlag > 0 {
		args = append(args, "-count", strconv.Itoa(*countFlag))
	}
	for _, pkg := range pkgs {
		args = append(args, pkg.RelativePath)
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if *verbose {
		fmt.Println("$:", cmd)
	}
	return cmd.Run()
}

func main() {
	flag.Parse()
	pkgs := flag.Args()
	if len(pkgs) < 1 {
		fmt.Println("Must provide at least one package to test.")
		os.Exit(1)
	}
	pkgGroups, err := findPkgGroups(pkgs)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cacheDir := path.Join(homeDir, ".tt_cache") // TODO(zviad): make this configurable?
	exitCode := 0
	for _, pkgs := range pkgGroups {
		if len(pkgGroups) > 0 {
			fmt.Println("Module:", pkgs[0].ModPath)
		}
		err := runTests(cacheDir, pkgs)
		if err != nil {
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}
