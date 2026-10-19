package claims

import (
	"reflect"
	"testing"
)

func TestMatch(t *testing.T) {
	cases := []struct {
		pattern, name string
		want          bool
	}{
		{"invoices/**", "invoices/march.csv", true},
		{"invoices/**", "invoices/2026/q1.csv", true},
		{"invoices/**", "invoices", true},
		{"invoices/**", "invoicesx/a.csv", false},
		{"invoices/**", "skills/invoices/a.csv", false},
		{"*.csv", "a.csv", true},
		{"*.csv", "d/a.csv", false},
		{"**/*.csv", "a.csv", true},
		{"**/*.csv", "d/e/a.csv", true},
		{"**/*.csv", "d/e/a.txt", false},
		{"README.md", "README.md", true},
		{"README.md", "docs/README.md", false},
		{"~/.ssh/**", "~/.ssh/id_rsa", true},
		{"**", "anything/at/all", true},
		{"a/**/z", "a/z", true},
		{"a/**/z", "a/b/c/z", true},
		{"a/**/z", "a/b/c/y", false},
		{"[", "[", false},
	}
	for _, c := range cases {
		if got := Match(c.pattern, c.name); got != c.want {
			t.Errorf("Match(%q, %q) = %v, want %v", c.pattern, c.name, got, c.want)
		}
	}
}

func TestGeneralize(t *testing.T) {
	got := Generalize([]string{
		"invoices/march.csv",
		"invoices/april.csv",
		"skills/invoice/reconcile.mjs",
		"skills/invoice/lib/sum.mjs",
		"package.json",
	})
	want := []string{"invoices/**", "package.json", "skills/invoice/**"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Generalize = %v, want %v", got, want)
	}
}

func TestGeneralizeEmpty(t *testing.T) {
	if got := Generalize(nil); got == nil || len(got) != 0 {
		t.Errorf("Generalize(nil) = %#v, want an empty, non-nil slice", got)
	}
}
