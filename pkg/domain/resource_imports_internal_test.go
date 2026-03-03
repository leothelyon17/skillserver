package domain

import (
	"path/filepath"
	"testing"
)

func TestIsWithinRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "skillserver-root")
	inside := filepath.Join(root, "references", "guide.md")
	outside := filepath.Join(string(filepath.Separator), "tmp", "outside.md")

	if !isWithinRoot(inside, root) {
		t.Fatalf("expected path %q to be inside root %q", inside, root)
	}
	if isWithinRoot(outside, root) {
		t.Fatalf("expected path %q to be outside root %q", outside, root)
	}
	// Mixed absolute/relative paths force filepath.Rel to error.
	if isWithinRoot("relative/path.md", root) {
		t.Fatalf("expected mixed absolute/relative paths to be rejected")
	}
}
