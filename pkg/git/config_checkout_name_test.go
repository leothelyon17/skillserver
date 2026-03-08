package git_test

import (
	"testing"

	"github.com/mudler/skillserver/pkg/git"
)

func TestResolveRepoCheckoutNamePrefersSanitizedConfigName(t *testing.T) {
	repo := git.GitRepoConfig{
		ID:   "gitrepo_1234abcd",
		URL:  "https://github.com/acme/private-repo.git",
		Name: " ../private repo ",
	}

	got := git.ResolveRepoCheckoutName(repo)
	if got != "private-repo" {
		t.Fatalf("expected sanitized checkout name private-repo, got %q", got)
	}
}

func TestResolveRepoCheckoutNameFallsBackToStableID(t *testing.T) {
	repo := git.GitRepoConfig{
		ID:   "gitrepo_deadbeef",
		URL:  "://invalid",
		Name: " .. ",
	}

	got := git.ResolveRepoCheckoutName(repo)
	want := git.GenerateID(repo.URL)
	if got != want {
		t.Fatalf("expected checkout fallback to generated id %q, got %q", want, got)
	}
}
