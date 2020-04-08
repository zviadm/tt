package servicelib

import (
	"os/exec"
	"strconv"

	"github.com/pkg/errors"
	"github.com/zviadm/zlog"
)

// Clear all rules. Tests that mess with Iptables, should run
// `defer servicelib.IptablesClearAll()` call at the beginning of the test function.
func IptablesClearAll() error {
	out, err := exec.Command("iptables", "-F").CombinedOutput()
	if err != nil {
		return errors.Errorf("%s\n%s", err, out)
	}
	return nil
}

// Blocks any incoming traffic to a given port on loopback interface.
func IptablesBlockPort(port int) error {
	zlog.Info("TEST: iptables: blocking port ", port)
	out, err := exec.Command(
		"iptables", "-I", "INPUT",
		"-p", "tcp", "--dport", strconv.Itoa(port),
		"-i", "lo", "-j", "DROP").CombinedOutput()
	if err != nil {
		return errors.Errorf("%s\n%s", err, out)
	}
	return nil
}

// TODO(zviad): actually implement unblock.
// func IptablesUnblockPort(port int) error {
// 	zlog.Info("TEST: iptables: unblocking port ", port)
// 	out, err := exec.Command("iptables", "-D", "INPUT", "1").CombinedOutput()
// }
