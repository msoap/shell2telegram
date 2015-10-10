package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func Test_splitStringHalfBySpace(t *testing.T) {
	data := []struct {
		in             string
		outOne, outTwo string
	}{
		{
			"/cmd args",
			"/cmd", "args",
		}, {
			"/cmd   args",
			"/cmd", "args",
		}, {
			"/cmd",
			"/cmd", "",
		}, {
			"plain text",
			"plain", "text",
		}, {
			"plain     text",
			"plain", "text",
		}, {
			"",
			"", "",
		},
	}

	for _, item := range data {
		one, two := splitStringHalfBySpace(item.in)
		if !(one == item.outOne && two == item.outTwo) {
			t.Errorf("Failing for \"%s\"\nexpected: (%#v, %#v)\nreal: (%#v, %#v)\n", item.in, item.outOne, item.outTwo, one, two)
		}
	}
}

func Test_cleanUserName(t *testing.T) {
	data := []struct {
		in  string
		out string
	}{
		{
			"1234",
			"1234",
		}, {
			"name",
			"name",
		}, {
			"@name",
			"name",
		}, {
			" name@str ",
			" namestr ",
		}, {
			"",
			"",
		},
	}

	for _, item := range data {
		out := cleanUserName(item.in)
		if out != item.out {
			t.Errorf("Failing for \"%s\"\nexpected: %s, real: %s\n", item.in, item.out, out)
		}
	}
}

func Test_parseBotCommand(t *testing.T) {
	data := []struct {
		// in
		pathRaw, shellCmd string
		// out
		path    string
		command Command
		errFunc error
	}{
		{
			pathRaw:  "/cmd",
			shellCmd: "ls",
			// out
			path: "/cmd",
			command: Command{
				shell:       "ls",
				description: "",
				vars:        []string{},
			},
			errFunc: nil,
		},
		{
			pathRaw:  "/",
			shellCmd: "ls",
			// out
			path: "/",
			command: Command{
				shell:       "ls",
				description: "",
				vars:        []string{},
			},
			errFunc: nil,
		},
		// empty shell command
		{
			pathRaw:  "/cmd",
			shellCmd: "",
			// out
			path: "",
			command: Command{
				shell:       "",
				description: "",
				vars:        []string{},
			},
			errFunc: fmt.Errorf("error"),
		},
		{
			pathRaw:  "/cmd:vars=VAR1,VAR2:desc=Command name",
			shellCmd: "ls",
			// out
			path: "/cmd",
			command: Command{
				shell:       "ls",
				description: "Command name",
				vars:        []string{"VAR1", "VAR23"},
			},
			errFunc: nil,
		},
	}

	for _, item := range data {
		path, command, errFunc := parseBotCommand(item.pathRaw, item.shellCmd)
		jsonMust, _ := json.Marshal(item.command)
		jsonIn, _ := json.Marshal(command)
		if path != item.path || ((errFunc == nil) != (item.errFunc == nil) || string(jsonIn) != string(jsonMust)) {
			t.Errorf("Failing for %v\nGot: path: %s, %#v\n", item, path, command)
		}
	}

	invalidPaths := []string{
		"",
		" ",
		"NotValidPath",
		" /cmd",
		"/:aaa",
		"/cmd:aaa=23",
		"/cmd:aaa",
		"/cmd:desc",
		"/cmd:desc=",
		"/cmd:vars=,,,,",
	}
	for _, path := range invalidPaths {
		_, _, errFunc := parseBotCommand(path, "ls")
		if errFunc == nil {
			t.Errorf("Failing check invalid path for: %s", path)
		}
	}
}
