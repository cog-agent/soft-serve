package utils

import (
	"strings"
	"testing"
)

func TestValidateRepo(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		for _, repo := range []string{
			"lower",
			"Upper",
			"with-dash",
			"with/slash",
			"withnumb3r5",
			"with.dot",
			"with_underline",
			"a/b/c",
			"foo.github/bar",
			"gitfoo/bar.gitx",
		} {
			t.Run(repo, func(t *testing.T) {
				if err := ValidateRepo(repo); err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			})
		}
	})
	t.Run("invalid", func(t *testing.T) {
		for _, repo := range []string{
			"with$",
			"with@",
			"with!",
			"",
			".",
			"..",
			"nested.git/refs/heads/x",
			"A.GIT/b",
			"victim.git/hooks",
			"foo/.git",
			"foo/../bar",
			"foo//bar",
			"/leading",
			"trailing/",
			"suffix.git",
			"double.git.git",
		} {
			t.Run(repo, func(t *testing.T) {
				if err := ValidateRepo(repo); err == nil {
					t.Error("expected an error, got nil")
				}
			})
		}
	})
}

func TestSanitizeRepo(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"lower", "lower"},
		{"Upper", "Upper"},
		{"with/slash", "with/slash"},
		{"with.dot", "with.dot"},
		{"/with_forward_slash", "with_forward_slash"},
		{"withgitsuffix.git", "withgitsuffix"},
		{"double.git.git", "double"},
		{"triple.git.git.git", "triple"},
		{"..git", ""},
		{".git", ""},
		{"foo/.git", "foo"},
		{"foo/../bar", "bar"},
		{"../../etc", "etc"},
		{"trailing/", "trailing"},
		{"nested.git/refs/heads/x", "nested.git/refs/heads/x"},
		{"a/..git", "a"},
		{"a/b/...git", "a"},
		{"a/b/..git", "a/b"},
		{"foo.git/.git", "foo"},
		{strings.Repeat(".git", 20000), ""},
		{strings.Repeat("/.git", 20000), ""},
		{"x" + strings.Repeat(".git", 20000), "x"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := SanitizeRepo(c.in)
			if got != c.out {
				t.Errorf("expected %q, got %q", c.out, got)
			}
			if again := SanitizeRepo(got); again != got {
				t.Errorf("not a fixed point: %q -> %q", got, again)
			}
		})
	}
}

func TestSanitizeStripsTerminalControls(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"plain":           {"hello world", "hello world"},
		"keeps layout":    {"a\tb\nc", "a\tb\nc"},
		"csi":             {"a\x1b[31mred\x1b[0mb", "aredb"},
		"osc bel":         {"a\x1b]0;title\x07b", "ab"},
		"osc st":          {"a\x1b]0;title\x1b\\b", "ab"},
		"osc52 clipboard": {"a\x1b]52;c;aGk=\x07b", "ab"},
		"dcs":             {"a\x1bPq\x1b\\b", "ab"},
		"clear screen":    {"a\x1b[2J\x1b[Hb", "ab"},
		"truncated esc":   {"a\x1b[31", "a"},
		"raw c1 csi":      {"a\x9b31mb", "ab"},
		"utf8 c1 csi":     {"a\u009b31mb", "ab"},
		"utf8 c1 osc":     {"a\u009d0;t\u009cb", "ab"},
		"carriage return": {"Author: ok\rAuthor: forged", "Author: okAuthor: forged"},
		"crlf":            {"one\r\ntwo", "one\ntwo"},
		"backspace":       {"abc\b\b\bxyz", "abcxyz"},
		"bell and nul":    {"a\x07\x00b", "ab"},
		"del":             {"a\x7fb", "ab"},
		"unicode kept":    {"héllo 世界", "héllo 世界"},
	} {
		t.Run(name, func(t *testing.T) {
			got := Sanitize(tc.in)
			if got != tc.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if again := Sanitize(got); again != got {
				t.Errorf("Sanitize is not idempotent: %q -> %q", got, again)
			}
		})
	}
}
