package servicelib

import (
	"os/exec"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zviadm/zlog"
)

// Clear all rules. Tests that mess with Iptables, should run
// `defer servicelib.IptablesClearAll()` call at the beginning of the test function.
func IptablesClearAll(t *testing.T) {
	out, err := exec.Command("iptables", "-F").CombinedOutput()
	require.NoError(t, err, string(out))
}

// Blocks any incoming traffic to a given port on loopback interface.
func IptablesBlockPort(t *testing.T, port int) {
	zlog.Info("TEST: iptables: blocking port ", port)
	out, err := exec.Command(
		"iptables", "-I", "INPUT",
		"-p", "tcp", "--dport", strconv.Itoa(port),
		"-i", "lo", "-j", "DROP").CombinedOutput()
	require.NoError(t, err, string(out))
}

// TODO(zviad): actually implement unblock.
// func IptablesUnblockPort(t *testing.T, port int) {
// 	zlog.Info("TEST: iptables: unblocking port ", port)
// 	out, err := exec.Command("iptables", "-D", "INPUT", "1").CombinedOutput()
// 	require.NoError(t, err, string(out))
// }
