package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
)

// exec shell commands with text to STDIN
func execShell(shellCmd, input string) (result string) {
	shell, params := "sh", []string{"-c", shellCmd}
	osExecCommand := exec.Command(shell, params...)
	osExecCommand.Stderr = os.Stderr

	// write user input to STDIN
	if input != "" {
		stdin, err := osExecCommand.StdinPipe()
		if err == nil {
			io.WriteString(stdin, input)
			stdin.Close()
		} else {
			log.Print("get STDIN error: ", err)
		}
	}

	shellOut, err := osExecCommand.Output()
	if err != nil {
		log.Print("exec error: ", err)
		result = fmt.Sprintf("exec error: %s", err)
	} else {
		result = string(shellOut)
	}

	return result
}

// return 2 strings, second="" if string dont contain space
func splitStringHalfBySpace(str string) (one, two string) {
	array := regexp.MustCompile(`\s+`).Split(str, 2)
	one, two = array[0], ""
	if len(array) > 1 {
		two = array[1]
	}

	return one, two
}

// cleanUserName - remove @ from telegram username
func cleanUserName(in string) string {
	return regexp.MustCompile("@").ReplaceAllLiteralString(in, "")
}

// getRandomCode - generate random code for authorize user
func getRandomCode() string {
	buffer := make([]byte, CODE_BYTES_LENGTH)
	_, err := rand.Read(buffer)
	if err != nil {
		log.Print("Get code error: ", err)
		return ""
	}

	return base64.URLEncoding.EncodeToString(buffer)
}
