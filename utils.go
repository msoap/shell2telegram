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
	"runtime"
	"strconv"
	"strings"
)

// CODE_BYTES_LENGTH - length of random code in bytes
const CODE_BYTES_LENGTH = 15

// exec shell commands with text to STDIN
func execShell(shellCmd, input string, varsNames []string, userID, chatID int, userName, userDisplayName string) (result []byte) {
	shell, params := "sh", []string{"-c", shellCmd}
	if runtime.GOOS == "windows" {
		shell, params = "cmd", []string{"/C", shellCmd}
	}
	osExecCommand := exec.Command(shell, params...)
	osExecCommand.Stderr = os.Stderr

	// copy variables from parent process
	for _, envRaw := range os.Environ() {
		osExecCommand.Env = append(osExecCommand.Env, envRaw)
	}

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

	// set S2T_* env vars
	s2tVariables := [...]struct{ name, value string }{
		{"S2T_LOGIN", userName},
		{"S2T_USERID", strconv.Itoa(userID)},
		{"S2T_USERNAME", userDisplayName},
		{"S2T_CHATID", strconv.Itoa(userID)},
	}
	for _, row := range s2tVariables {
		osExecCommand.Env = append(osExecCommand.Env, fmt.Sprintf("%s=%s", row.name, row.value))
	}

	shellOut, err := osExecCommand.Output()
	if err != nil {
		log.Print("exec error: ", err)
		result = []byte(fmt.Sprintf("exec error: %s", err))
	} else {
		result = shellOut
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
		shellCmd:    shellCmd,
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

// split string by chunks less maxSize size (whole rows)
func splitStringLinesBySize(input string, maxSize int) []string {
	result := []string{}
	parts := regexp.MustCompile("\n").Split(input, -1)
	chunks := []string{parts[0]}
	chunkSize := len(parts[0])

	for _, part := range parts[1:] {
		// current + "\n" + next > maxSize
		if chunkSize+1+len(part) > maxSize {
			result = append(result, strings.Join(chunks, "\n"))
			chunks = []string{part}
			chunkSize = len(part)
		} else {
			chunks = append(chunks, part)
			chunkSize += 1 + len(part)
		}
	}
	if len(chunks) > 0 {
		result = append(result, strings.Join(chunks, "\n"))
	}

	return result
}

// create dir if it is not exists
func createDirIfNeed(dir string) {
	if _, err := os.Stat(dir); err != nil {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatal("create dir error:", dir)
		}
	}
}

// get home dir
func getOsUserHomeDir() string {
	homeDir := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		homeDir = os.Getenv("APPDATA")
	}
	return homeDir
}

// read default or user db file name
func getDBFilePath(usersDBFile string, needCreateDir bool) string {
	fileName := ""
	if usersDBFile == "" {
		dirName := getOsUserHomeDir() + string(os.PathSeparator) + ".config"
		if needCreateDir {
			createDirIfNeed(dirName)
		}
		fileName = dirName + string(os.PathSeparator) + DB_FILE_NAME
	} else {
		fileName = usersDBFile
	}

	return fileName
}
