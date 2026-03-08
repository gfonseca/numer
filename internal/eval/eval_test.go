package eval

import (
	"testing"
)

func TestBasicArith(t *testing.T) {
	e := New()
	cases := []struct {
		line string
		want string
	}{
		{"2 + 2", "4"},
		{"10 - 3", "7"},
		{"3 * 4", "12"},
		{"10 / 4", "2.5"},
		{"10 % 3", "1"},
		{"2 ^ 10", "1024"},
		{"(1 + 2) * 3", "9"},
		{"-5 + 3", "-2"},
		{"sqrt(9)", "3"},
		{"abs(-7)", "7"},
		{"round(3.7)", "4"},
	}
	for _, c := range cases {
		res, errMsg := e.EvalLine(c.line)
		if errMsg != "" {
			t.Errorf("%q => error: %s", c.line, errMsg)
			continue
		}
		if res != c.want {
			t.Errorf("%q => %q, want %q", c.line, res, c.want)
		}
	}
}

func TestVariables(t *testing.T) {
	e := New()
	res, _ := e.EvalLine("x = 10")
	if res != "x = 10" {
		t.Errorf("assignment result = %q", res)
	}
	res, _ = e.EvalLine("x * 2")
	if res != "20" {
		t.Errorf("var use = %q", res)
	}
	res, _ = e.EvalLine("last + 5")
	if res != "25" {
		t.Errorf("last = %q", res)
	}
}

func TestConstants(t *testing.T) {
	e := New()
	res, errMsg := e.EvalLine("pi")
	if errMsg != "" {
		t.Fatal(errMsg)
	}
	if res == "" {
		t.Error("pi returned empty")
	}
}

func TestComments(t *testing.T) {
	e := New()
	res, errMsg := e.EvalLine("# comment")
	if res != "" || errMsg != "" {
		t.Errorf("comment should produce no output, got %q %q", res, errMsg)
	}
	res, errMsg = e.EvalLine("// comment")
	if res != "" || errMsg != "" {
		t.Errorf("// comment should produce no output, got %q %q", res, errMsg)
	}
}
