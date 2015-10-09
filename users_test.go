package main

import "testing"

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
