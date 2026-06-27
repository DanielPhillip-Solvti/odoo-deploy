package helpers

import (
	"fmt"
	"os/exec"
	"strings"
)

func RunCmd(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s failed: %w\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out), nil
}

func RunCmdInDir(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s failed inside %s: %w\n%s", name, strings.Join(args, " "), dir, err, out)
	}
	return string(out), nil
}

func RunCmdWithEnv(env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s failed: %w\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out), nil
}
