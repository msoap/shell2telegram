package main

import (
	"fmt"
	"os"
	"reflect"
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
				shellCmd:    "ls",
				description: "",
				vars:        nil,
				isMarkdown:  false,
			},
			errFunc: nil,
		},
		{
			pathRaw:  "/",
			shellCmd: "ls",
			// out
			path: "/",
			command: Command{
				shellCmd:    "ls",
				description: "",
				vars:        nil,
				isMarkdown:  false,
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
				shellCmd:    "",
				description: "",
				vars:        nil,
				isMarkdown:  false,
			},
			errFunc: fmt.Errorf("error"),
		},
		{
			pathRaw:  "/cmd:vars=VAR1,VAR2:desc=Command name",
			shellCmd: "ls",
			// out
			path: "/cmd",
			command: Command{
				shellCmd:    "ls",
				description: "Command name",
				vars:        []string{"VAR1", "VAR2"},
				isMarkdown:  false,
			},
			errFunc: nil,
		},
		{
			// markdown test
			pathRaw:  "/cmd:vars=VAR1,VAR2:desc=Command name:md",
			shellCmd: "ls",
			// out
			path: "/cmd",
			command: Command{
				shellCmd:    "ls",
				description: "Command name",
				vars:        []string{"VAR1", "VAR2"},
				isMarkdown:  true,
			},
			errFunc: nil,
		},
		{
			pathRaw:  "/:plain_text",
			shellCmd: "ls",
			// out
			path: "/:plain_text",
			command: Command{
				shellCmd:    "ls",
				description: "",
				vars:        nil,
				isMarkdown:  false,
			},
			errFunc: nil,
		},
		{
			pathRaw:  "/:image",
			shellCmd: "ls",
			// out
			path: "",
			command: Command{
				shellCmd:    "",
				description: "",
				vars:        nil,
				isMarkdown:  false,
			},
			errFunc: fmt.Errorf("/:image not implemented"),
		},
		{
			pathRaw:  "/:plain_text:desc=Name",
			shellCmd: "ls",
			// out
			path: "/:plain_text",
			command: Command{
				shellCmd:    "ls",
				description: "Name",
				vars:        nil,
				isMarkdown:  false,
			},
			errFunc: nil,
		},
	}

	for _, item := range data {
		path, command, errFunc := parseBotCommand(item.pathRaw, item.shellCmd)
		commandMust := fmt.Sprintf("%#v", item.command)
		commandGet := fmt.Sprintf("%#v", command)

		if path != item.path || ((errFunc == nil) != (item.errFunc == nil) || commandGet != commandMust) {
			t.Errorf("Failing for %v (path: %s)\nMust: %s\nGot:  %#v\n", item, path, commandMust, command)
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

func Test_stringIsEmpty(t *testing.T) {
	data := []struct {
		in  string
		out bool
	}{
		{
			"1234",
			false,
		}, {
			" str ",
			false,
		}, {
			"",
			true,
		}, {
			"  ",
			true,
		}, {
			"\n",
			true,
		}, {
			"  \ndew",
			false,
		},
	}

	for _, item := range data {
		out := stringIsEmpty(item.in)
		if out != item.out {
			t.Errorf("Failing for %#v\nexpected: %v, real: %v\n", item.in, item.out, out)
		}
	}
}

func Test_splitStringLinesBySize(t *testing.T) {
	data := []struct {
		in      string
		maxSize int
		out     []string
	}{
		{
			"12345",
			6,
			[]string{"12345"},
		}, {
			"12345\n67890",
			11,
			[]string{"12345\n67890"},
		}, {
			"1234567890\n1234567890",
			3,
			[]string{"1234567890", "1234567890"},
		}, {
			"12\n34\n56\n78\n90",
			6,
			[]string{"12\n34", "56\n78", "90"},
		}, {
			"12\n34aaaaaaaaaaaaa\n56\n78\n90",
			6,
			[]string{"12", "34aaaaaaaaaaaaa", "56\n78", "90"},
		},
	}

	for _, item := range data {
		out := splitStringLinesBySize(item.in, item.maxSize)
		mustOut := fmt.Sprintf("%#v", item.out)
		getOut := fmt.Sprintf("%#v", out)
		if mustOut != getOut {
			t.Errorf("Failing for %#v (by %d)\nexpected: %s, real: %s\n", item.in, item.maxSize, mustOut, getOut)
		}
	}
}

func Test_getRandomCode(t *testing.T) {
	rnd := getRandomCode()
	if len(rnd) == 0 {
		t.Errorf("getRandomCode() failed")
	}
}

func Test_getOsUserHomeDir(t *testing.T) {
	userDir := getOsUserHomeDir()
	if len(userDir) == 0 {
		t.Errorf("1. getOsUserHomeDir() failed")
	}
	_, err := os.Stat(userDir)
	if err != nil {
		t.Errorf("2. getOsUserHomeDir() failed")
	}
}

func Test_errChain(t *testing.T) {
	err := errChain()
	if err != nil {
		t.Errorf("1. errChain() empty failed")
	}

	err = errChain(func() error { return nil })
	if err != nil {
		t.Errorf("2. errChain() failed")
	}

	err = errChain(func() error { return nil }, func() error { return nil })
	if err != nil {
		t.Errorf("3. errChain() failed")
	}

	err = errChain(func() error { return fmt.Errorf("error") })
	if err == nil {
		t.Errorf("4. errChain() failed")
	}

	err = errChain(func() error { return nil }, func() error { return fmt.Errorf("error") })
	if err == nil {
		t.Errorf("5. errChain() failed")
	}

	var1 := false
	err = errChain(func() error { return fmt.Errorf("error") }, func() error { var1 = true; return nil })
	if err == nil || var1 {
		t.Errorf("6. errChain() failed")
	}
}

func Test_getShellAndParams(t *testing.T) {
	shell, params, err := getShellAndParams("ls", "sh", false)
	if shell != "sh" || !reflect.DeepEqual(params, []string{"-c", "ls"}) || err != nil {
		t.Errorf("1. getShellAndParams() failed")
	}

	shell, params, err = getShellAndParams("ls", "bash", false)
	if shell != "bash" || !reflect.DeepEqual(params, []string{"-c", "ls"}) || err != nil {
		t.Errorf("3. getShellAndParams() failed")
	}

	shell, params, err = getShellAndParams("ls -l -a", "", false)
	if shell != "ls" || !reflect.DeepEqual(params, []string{"-l", "-a"}) || err != nil {
		t.Errorf("4. getShellAndParams() failed")
	}

	shell, params, err = getShellAndParams("ls -l 'a b'", "", false)
	if shell != "ls" || !reflect.DeepEqual(params, []string{"-l", "a b"}) || err != nil {
		t.Errorf("5. getShellAndParams() failed")
	}

	_, _, err = getShellAndParams("ls '-l", "", false)
	if err == nil {
		t.Errorf("6. getShellAndParams() failed")
	}
}
