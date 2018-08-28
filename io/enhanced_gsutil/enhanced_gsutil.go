package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: /path/to/gsutil <arguments>")
		os.Exit(1)
	}

	gsutilPath := os.Args[1]
	gsutilArgs := os.Args[2:]

	var argsNoRequesterPaysFlag []string
	hasRequesterPaysFlag := false

	for i, v := range gsutilArgs {
		if v == "-u" {
			hasRequesterPaysFlag = true
			argsNoRequesterPaysFlag = append(argsNoRequesterPaysFlag, gsutilArgs[i+2:]...)
			break
		}
		argsNoRequesterPaysFlag = append(argsNoRequesterPaysFlag, v)
	}

	var stderrBuffer bytes.Buffer
	commandWithoutRequesterPays := makeCommand(gsutilPath, argsNoRequesterPaysFlag, &stderrBuffer)

	if err := commandWithoutRequesterPays.Run(); err != nil {
		if isRequesterPaysFailure(&stderrBuffer) && hasRequesterPaysFlag {
			log.Printf("Retrying command with requester pays flag")
			commandWithRequesterPays := makeCommand(gsutilPath, gsutilArgs, &stderrBuffer)
			if err := commandWithRequesterPays.Run(); err != nil {
				failCommand(err, &stderrBuffer)
			}
		} else {
			failCommand(err, &stderrBuffer)
		}
	}

	fmt.Fprintf(os.Stderr, stderrBuffer.String())
}

func failCommand(err error, errBuffer *bytes.Buffer) {
	fmt.Fprintf(os.Stderr, "gsutil failed: %v\n", err)
	fmt.Fprintf(os.Stderr, errBuffer.String())

	if err, ok := err.(*exec.ExitError); ok {
		os.Exit(err.Sys().(syscall.WaitStatus).ExitStatus())
	}
	os.Exit(1)
}

func makeCommand(gsutilPath string, args []string, errbuf *bytes.Buffer) (command *exec.Cmd) {
	command = exec.Command(gsutilPath, args...)
	command.Stderr = errbuf
	command.Stdout = os.Stdout
	return
}

func isRequesterPaysFailure(errBuffer *bytes.Buffer) bool {
	return strings.Contains(errBuffer.String(), "Bucket is requester pays bucket but no user project provided.")
}
