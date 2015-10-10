package main

import "testing"

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
