package domain

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	markdownLinkImportPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)\n]+)\)`)
	includeTokenImportPattern = regexp.MustCompile(`(?:^|[\s(\[{>])@([^\s)\]}>]+)`)
)

// ResolvedImport represents a validated import target and its stable virtual path.
type ResolvedImport struct {
	Candidate   string
	SourcePath  string
	TargetPath  string
	VirtualPath string
}

// ParseImportCandidates extracts supported import candidates from markdown content.
// It supports markdown links and @include-style tokens and returns deterministic output.
func ParseImportCandidates(markdown string) []string {
	if strings.TrimSpace(markdown) == "" {
		return nil
	}

	seen := make(map[string]struct{})
	candidates := make([]string, 0)

	addCandidate := func(raw string, requireSlash bool) {
		candidate, ok := normalizeImportCandidate(raw)
		if !ok {
			return
		}
		if requireSlash && !strings.Contains(candidate, "/") {
			return
		}
		if _, exists := seen[candidate]; exists {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}

	for _, match := range markdownLinkImportPattern.FindAllStringSubmatch(markdown, -1) {
		if len(match) < 2 {
			continue
		}
		addCandidate(match[1], false)
	}

	for _, match := range includeTokenImportPattern.FindAllStringSubmatch(markdown, -1) {
		if len(match) < 2 {
			continue
		}
		addCandidate(match[1], true)
	}

	sort.Strings(candidates)
	return candidates
}

// ResolveImportTarget resolves an import candidate from a source markdown file.
// The resolved target must be a file within the allowed root after symlink evaluation.
func ResolveImportTarget(sourcePath, candidate, allowedRoot string) (*ResolvedImport, error) {
	normalizedCandidate, ok := normalizeImportCandidate(candidate)
	if !ok {
		return nil, fmt.Errorf("invalid import candidate: %q", candidate)
	}

	canonicalRoot, err := canonicalizeExistingPath(allowedRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve allowed root: %w", err)
	}

	canonicalSource, err := canonicalizeExistingPath(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source path: %w", err)
	}
	if !isWithinRoot(canonicalSource, canonicalRoot) {
		return nil, fmt.Errorf("source path is outside allowed root")
	}

	unresolvedTarget := resolveImportCandidatePath(canonicalSource, normalizedCandidate, canonicalRoot)
	canonicalTarget, err := canonicalizeExistingPath(unresolvedTarget)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve import target: %w", err)
	}
	if !isWithinRoot(canonicalTarget, canonicalRoot) {
		return nil, fmt.Errorf("import target escapes allowed root")
	}

	targetInfo, err := os.Stat(canonicalTarget)
	if err != nil {
		return nil, fmt.Errorf("failed to stat import target: %w", err)
	}
	if targetInfo.IsDir() {
		return nil, fmt.Errorf("import target is not a file")
	}

	virtualPath, err := BuildImportedVirtualPath(canonicalRoot, canonicalTarget)
	if err != nil {
		return nil, err
	}

	return &ResolvedImport{
		Candidate:   normalizedCandidate,
		SourcePath:  canonicalSource,
		TargetPath:  canonicalTarget,
		VirtualPath: virtualPath,
	}, nil
}

// BuildImportedVirtualPath maps a canonical target path to a deterministic virtual path under imports/.
func BuildImportedVirtualPath(allowedRoot, targetPath string) (string, error) {
	canonicalRoot, err := canonicalizeExistingPath(allowedRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve allowed root: %w", err)
	}
	canonicalTarget, err := canonicalizeExistingPath(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve target path: %w", err)
	}
	if !isWithinRoot(canonicalTarget, canonicalRoot) {
		return "", fmt.Errorf("target path is outside allowed root")
	}

	relativeTarget, err := filepath.Rel(canonicalRoot, canonicalTarget)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative import path: %w", err)
	}

	normalizedRelative := filepath.ToSlash(filepath.Clean(relativeTarget))
	if normalizedRelative == "." || normalizedRelative == ".." || strings.HasPrefix(normalizedRelative, "../") {
		return "", fmt.Errorf("target path cannot be mapped under imports/")
	}

	return path.Join(strings.TrimSuffix(resourceDirImports, "/"), normalizedRelative), nil
}

func normalizeImportCandidate(raw string) (string, bool) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return "", false
	}

	// Markdown links can include optional titles: (path "title").
	if strings.ContainsAny(candidate, " \t\r\n") {
		fields := strings.Fields(candidate)
		if len(fields) == 0 {
			return "", false
		}
		candidate = fields[0]
	}

	candidate = strings.TrimSpace(candidate)
	candidate = strings.Trim(candidate, "<>")
	candidate = strings.Trim(candidate, `"'`)
	candidate = strings.TrimRight(candidate, ".,;:")
	candidate = strings.ReplaceAll(candidate, `\`, "/")

	if idx := strings.IndexAny(candidate, "#?"); idx >= 0 {
		candidate = candidate[:idx]
	}
	candidate = strings.TrimSpace(candidate)

	if candidate == "" || candidate == "." || candidate == ".." || candidate == "/" {
		return "", false
	}
	if strings.HasPrefix(candidate, "#") {
		return "", false
	}
	if strings.Contains(candidate, "://") {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(candidate), "mailto:") {
		return "", false
	}
	// Keep parsing bounded to file-like path candidates.
	if strings.Contains(candidate, ":") {
		return "", false
	}

	candidate = path.Clean(candidate)
	if candidate == "." || candidate == ".." || candidate == "/" {
		return "", false
	}

	return candidate, true
}

func resolveImportCandidatePath(sourcePath, candidate, allowedRoot string) string {
	if strings.HasPrefix(candidate, "/") {
		rootRelative := strings.TrimPrefix(candidate, "/")
		return filepath.Clean(filepath.Join(allowedRoot, filepath.FromSlash(rootRelative)))
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourcePath), filepath.FromSlash(candidate)))
}

func canonicalizeExistingPath(targetPath string) (string, error) {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return "", err
	}

	resolvedPath, err := filepath.EvalSymlinks(filepath.Clean(absPath))
	if err != nil {
		return "", err
	}

	resolvedAbsPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolvedAbsPath), nil
}

func isWithinRoot(targetPath, allowedRoot string) bool {
	relativePath, err := filepath.Rel(allowedRoot, targetPath)
	if err != nil {
		return false
	}

	relativePath = filepath.Clean(relativePath)
	if relativePath == ".." {
		return false
	}
	return !strings.HasPrefix(relativePath, ".."+string(filepath.Separator))
}
