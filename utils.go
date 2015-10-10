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
func execShell(shellCmd, input string, varsNames []string) (result string) {
	shell, params := "sh", []string{"-c", shellCmd}
	osExecCommand := exec.Command(shell, params...)
	osExecCommand.Stderr = os.Stderr

	if input != "" {
		if len(varsNames) > 0 {
			// set user input to shell vars
			arguments := regexp.MustCompile(`\s+`).Split(input, len(varsNames))
			for i, arg := range arguments {
				osExecCommand.Env = append(osExecCommand.Env, fmt.Sprintf("%s=%s", varsNames[i], arg))
			}
		} else {
			// write user input to STDIN
			stdin, err := osExecCommand.StdinPipe()
			if err == nil {
				io.WriteString(stdin, input)
				stdin.Close()
			} else {
				log.Print("get STDIN error: ", err)
			}
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

// parseBotCommand - parse command-line arguments for one bot command
func parseBotCommand(pathRaw, shellCmd string) (path string, command Command, err error) {
	if len(pathRaw) == 0 || pathRaw[0] != '/' {
		return "", command, fmt.Errorf("error: path %s dont starts with /", pathRaw)
	}
	if stringIsEmpty(shellCmd) {
		return "", command, fmt.Errorf("error: shell command cannot be empty")
	}

	_parseVars := func(varsParts []string) (desc string, vars []string, err error) {
		for _, oneVar := range varsParts {
			oneVarParts := regexp.MustCompile("=").Split(oneVar, 2)
			if len(oneVarParts) != 2 {
				err = fmt.Errorf("error: parse command modificators: %s", oneVar)
				return
			} else if oneVarParts[0] == "desc" {
				desc = oneVarParts[1]
				if desc == "" {
					err = fmt.Errorf("error: command description cannot be empty")
					return
				}
			} else if oneVarParts[0] == "vars" {
				vars = regexp.MustCompile(",").Split(oneVarParts[1], -1)
				for _, oneVarName := range vars {
					if oneVarName == "" {
						err = fmt.Errorf("error: var name cannot be empty")
						return
					}
				}
			} else if oneVarParts[0] == "image_out" {
				log.Print("Not implemented")
			} else {
				err = fmt.Errorf("error: parse command modificators, not found %s", oneVarParts[0])
				return
			}
		}

		return desc, vars, nil
	}

	pathParts := regexp.MustCompile(":").Split(pathRaw, -1)
	desc, vars := "", []string{}
	switch {
	case len(pathParts) == 1:
		// /, /cmd
		path = pathParts[0]
	case pathParts[0] == "/" && regexp.MustCompile("^(plain_text|image)$").MatchString(pathParts[1]):
		// /:plain_text, /:image, /:plain_text:desc=name
		path = "/:" + pathParts[1]
		if pathParts[1] == "image" {
			log.Print("/:image not implemented")
		}
		if len(pathParts) > 2 {
			desc, vars, err = _parseVars(pathParts[2:])
		}
	case len(pathParts) > 1:
		// commands with modificators :desc, :vars
		path = pathParts[0]
		desc, vars, err = _parseVars(pathParts[1:])
	}
	if err != nil {
		return "", command, err
	}

	command = Command{
		shell:       shellCmd,
		description: desc,
		vars:        vars,
	}

	// pp.Println(path, command)
	return path, command, nil
}

// stringIsEmpty - check string is empty
func stringIsEmpty(str string) bool {
	isEmpty, _ := regexp.MatchString(`^\s*$`, str)
	return isEmpty
}
