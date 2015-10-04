package main

import "testing"

func Test_splitStringHalfBySpace(t *testing.T) {
	data := []struct {
		in               string
		out_one, out_two string
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
		if !(one == item.out_one && two == item.out_two) {
			t.Errorf("Failing for \"%s\"\nexpected: (%#v, %#v)\nreal: (%#v, %#v)\n", item.in, item.out_one, item.out_two, one, two)
		}
	}
}
