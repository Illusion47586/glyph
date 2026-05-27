package store

import "testing"

func TestRemoteURL(t *testing.T) {
	tests := map[string]string{
		"github:owner/repo":             "git@github.com:owner/repo.git",
		"https://github.com/owner/repo": "https://github.com/owner/repo",
		"git@github.com:owner/repo.git": "git@github.com:owner/repo.git",
		"not-a-remote":                  "",
	}
	for spec, want := range tests {
		if got := remoteURL(spec); got != want {
			t.Fatalf("remoteURL(%q) = %q, want %q", spec, got, want)
		}
	}
}
