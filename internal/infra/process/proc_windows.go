//go:build windows

package process

import "os/exec"

func setSysProcAttr(_ *exec.Cmd) {
	// Pdeathsig is not supported on Windows
}

func setCmdCancel(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
}
