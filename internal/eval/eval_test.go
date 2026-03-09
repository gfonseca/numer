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

func TestSumTotal(t *testing.T) {
	t.Run("sums expressions", func(t *testing.T) {
		e := New()
		e.EvalLine("100")
		e.EvalLine("200")
		e.EvalLine("300")
		res, errMsg := e.EvalLine("sum-total")
		if errMsg != "" {
			t.Fatalf("unexpected error: %s", errMsg)
		}
		if res != "600" {
			t.Errorf("got %q, want %q", res, "600")
		}
	})

	t.Run("sums assignments", func(t *testing.T) {
		e := New()
		e.EvalLine("rent = 1200")
		e.EvalLine("groceries = 300")
		e.EvalLine("utilities = 150")
		res, _ := e.EvalLine("sum-total")
		if res != "1650" {
			t.Errorf("got %q, want %q", res, "1650")
		}
	})

	t.Run("resets after sum-total so subsequent calls don't double-count", func(t *testing.T) {
		e := New()
		e.EvalLine("100")
		e.EvalLine("200")
		e.EvalLine("sum-total") // = 300, resets accumulator
		e.EvalLine("50")
		res, _ := e.EvalLine("sum-total")
		if res != "50" {
			t.Errorf("got %q, want %q", res, "50")
		}
	})

	t.Run("case-insensitive", func(t *testing.T) {
		e := New()
		e.EvalLine("10")
		res, errMsg := e.EvalLine("SUM-TOTAL")
		if errMsg != "" {
			t.Fatalf("unexpected error: %s", errMsg)
		}
		if res != "10" {
			t.Errorf("got %q, want %q", res, "10")
		}
	})

	t.Run("updates last", func(t *testing.T) {
		e := New()
		e.EvalLine("40")
		e.EvalLine("60")
		e.EvalLine("sum-total") // total = 100, sets last = 100
		res, _ := e.EvalLine("last * 2")
		if res != "200" {
			t.Errorf("got %q, want %q", res, "200")
		}
	})

	t.Run("blank lines and comments don't contribute", func(t *testing.T) {
		e := New()
		e.EvalLine("100")
		e.EvalLine("")
		e.EvalLine("# ignored")
		e.EvalLine("// also ignored")
		res, _ := e.EvalLine("sum-total")
		if res != "100" {
			t.Errorf("got %q, want %q", res, "100")
		}
	})
}
