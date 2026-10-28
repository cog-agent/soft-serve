package jobs

import "testing"

func TestDisabled(t *testing.T) {
	for spec, want := range map[string]bool{
		"off":         true,
		"OFF":         true,
		" disabled ":  true,
		"-":           true,
		"":            false,
		"@every 10m":  false,
		"*/5 * * * *": false,
		"offline":     false,
	} {
		if got := Disabled(spec); got != want {
			t.Errorf("Disabled(%q) = %v, want %v", spec, got, want)
		}
	}
}
