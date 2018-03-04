package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func setupIptables() (func(), error) {
	if err := iptablesCleanup(); err != nil {
		return func() {}, err
	}

	cu := func() {
		_ = iptablesCleanup
	}

	if err := iptablesAddTap("80"); err != nil {
		return cu, err
	}

	if err := iptablesAddTap("443"); err != nil {
		return cu, err
	}

	return cu, nil
}

func iptables(args ...string) error {
	args = append([]string{"-t", "nat"}, args...)

	cmd := exec.Command("iptables", args...)

	_, err := cmd.Output()
	if err != nil {
		if eerr, ok := err.(*exec.ExitError); ok {
			errStr := strings.TrimSpace(string(eerr.Stderr))
			errStr = strings.Replace(errStr, "\n", " ", -1)

			return fmt.Errorf(errStr)
		}

		return err
	}

	return nil
}

func iptablesCleanup() error {
	return iptables("-F")
}

func iptablesAddTap(port string) error {
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("provided port %q is not a number", port)
	}

	return iptables(
		"-A", "PREROUTING",
		"-i", "wlan0",
		"-p", "tcp",
		"--dport", port,
		"-m", "tcp",
		"-j", "REDIRECT",
		"--to-ports", port,
	)
}
