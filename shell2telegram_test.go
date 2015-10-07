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
