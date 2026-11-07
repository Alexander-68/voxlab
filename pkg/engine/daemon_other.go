//go:build !windows

package engine

import "os/exec"

// killWithParent is a no-op here: daemons can outlive a crash of this process.
func killWithParent(*exec.Cmd) error { return nil }
