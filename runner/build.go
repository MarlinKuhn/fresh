package runner

import (
	"errors"
	"io"
	"os"
	"strings"
)

func build() error {
	parts := []string{"build"}
	if buildPath() != "" {
		parts = append(parts, "-o")
		parts = append(parts, buildPath())
	}
	if mustUseDelve() {
		parts = append(parts, "-gcflags", "all=-N -l")
	}
	if buildArgs() != "" {
		parts = append(parts, buildArgs())
	}
	if mainPath() != "" {
		parts = append(parts, mainPath())
	}
	cmd := Cmd("go", strings.Join(parts, " "))
	buildLog("Building %v", CmdStr(cmd))

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fatal(err)
	}

	err = cmd.Start()
	if err != nil {
		fatal(err)
	}

	io.Copy(os.Stdout, stdout)
	errBuf, _ := io.ReadAll(stderr)

	err = cmd.Wait()
	if err != nil {
		return errors.New(string(errBuf))
	}

	return nil
}
