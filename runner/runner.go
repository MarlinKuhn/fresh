package runner

import (
	"io"
	"os/exec"
	"strconv"
	"strings"
)

func getDebugCommand(pid int) (string, []string) {
	params := []string{"attach",
		strconv.Itoa(pid),
		"--listen=:40000",
		"--headless",
		"--api-version=2",
		"--accept-multiclient"}
	if len(delveArgs()) > 0 {
		params = append(params, strings.Fields(delveArgs())...)
	}

	return "dlv", params
}

func run() {
	cmd := Cmd(buildPath(), runArgs())
	runnerLog("Starting %v", CmdStr(cmd))

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

	if isDebug() {
		runnerLog("PID %d", cmd.Process.Pid)
	}

	go io.Copy(appLogWriter{}, stderr)
	go io.Copy(appLogWriter{}, stdout)

	var debugCmd *exec.Cmd
	if mustUseDelve() {
		command, strings := getDebugCommand(cmd.Process.Pid)
		debugCmd = exec.Command(command, strings...)
		runnerLog("Starting debugger %v", CmdStr(debugCmd))

		stderr, err := debugCmd.StderrPipe()
		if err != nil {
			fatal(err)
		}

		stdout, err := debugCmd.StdoutPipe()
		if err != nil {
			fatal(err)
		}

		err = debugCmd.Start()
		if err != nil {
			fatal(err)
		}

		go io.Copy(debuggerLogWriter{}, stderr)
		go io.Copy(debuggerLogWriter{}, stdout)
	}

	go func() {
		defer func() {
			killDoneChannel <- struct{}{}
		}()
		<-killChannel

		pid := cmd.Process.Pid
		runnerLog("Killing PID %d", pid)
		if err := cmd.Process.Kill(); err != nil {
			if isDebug() {
				runnerLog("Killing PID %d error: %v", pid, err)
			}
		}

		if debugCmd != nil {
			runnerLog("Killing debugger")
			if err := debugCmd.Process.Kill(); err != nil {
				if isDebug() {
					runnerLog("Killing debugger %v error: %v", CmdStr(debugCmd), err)
				}
			}
		}

		if exiting {
			resetTermColors()
			doneChannel <- struct{}{}
		}

		_, err := cmd.Process.Wait()
		if isDebug() {
			if err != nil {
				runnerLog("PID %d exit error: %v", pid, err)
			}
		}

		if debugCmd != nil {
			_, err := debugCmd.Process.Wait()
			if isDebug() {
				if err != nil {
					runnerLog("Debugger exit error: %v", err)
				}
			}
		}
	}()
}
