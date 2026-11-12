package version

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.27.4", "1.27.4", 0},
		{"1.27.3", "1.27.4", -1},
		{"1.27.10", "1.27.9", 1},
		{"1.28", "1.27.9", 1},
		{"1.28", "1.28.0", 0},
		{"0", "1.27.4", -1},
	}
	for _, c := range cases {
		a, err := Parse(c.a)
		if err != nil {
			t.Fatal(err)
		}
		b, err := Parse(c.b)
		if err != nil {
			t.Fatal(err)
		}
		if got := a.Compare(b); got != c.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	for _, s := range []string{"", "1.28rc1", "1..2", "v1.2", "-1"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", s)
		}
	}
}
