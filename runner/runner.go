package runner

import (
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func run() {
	var cmd *exec.Cmd
	if mustUseDelve() {
		cmd = Cmd("dlv", delveArgs())
	} else {
		cmd = Cmd(buildPath(), runArgs())
	}
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

	go func() {
		defer func() {
			killDoneChannel <- struct{}{}
		}()
		<-killChannel

		pid := cmd.Process.Pid
		runnerLog("Killing PID %d", pid)

		if runtime.GOOS == "windows" || !isGracefulStop() {
			if err := cmd.Process.Kill(); err != nil {
				if isDebug() {
					runnerLog("Killing PID %d error: %v", pid, err)
				}
			}
		} else {
			go func() {
				time.Sleep(10 * time.Second)
				_ = cmd.Process.Kill()
				if isDebug() {
					runnerLog("Force killing PID %d error: %v", pid, err)
				}
			}()
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				if isDebug() {
					runnerLog("Interrupting PID %d error: %v", pid, err)
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
	}()
}
