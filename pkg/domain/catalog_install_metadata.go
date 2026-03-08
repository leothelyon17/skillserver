package domain

import (
	"fmt"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// CatalogMaterializeConflictPolicy defines install-time file conflict handling.
type CatalogMaterializeConflictPolicy string

const (
	CatalogMaterializeConflictPolicyError     CatalogMaterializeConflictPolicy = "error"
	CatalogMaterializeConflictPolicyOverwrite CatalogMaterializeConflictPolicy = "overwrite"
	CatalogMaterializeConflictPolicySkip      CatalogMaterializeConflictPolicy = "skip"
)

// CatalogMaterializeMetadata captures additive install hints from frontmatter.
type CatalogMaterializeMetadata struct {
	TargetPath     string                           `json:"target_path,omitempty"`
	ConflictPolicy CatalogMaterializeConflictPolicy `json:"conflict_policy,omitempty"`
}

// CatalogInstallMetadata captures optional install metadata for file-backed catalog items.
type CatalogInstallMetadata struct {
	Materialize CatalogMaterializeMetadata `json:"materialize"`
}

// IsValid reports whether the policy is one of the supported values.
func (p CatalogMaterializeConflictPolicy) IsValid() bool {
	switch p {
	case CatalogMaterializeConflictPolicyError, CatalogMaterializeConflictPolicyOverwrite, CatalogMaterializeConflictPolicySkip:
		return true
	default:
		return false
	}
}

// ParseCatalogMaterializeConflictPolicy parses and validates conflict policy input.
func ParseCatalogMaterializeConflictPolicy(raw string) (CatalogMaterializeConflictPolicy, error) {
	policy := CatalogMaterializeConflictPolicy(strings.ToLower(strings.TrimSpace(raw)))
	if !policy.IsValid() {
		return "", fmt.Errorf("materialize conflict policy %q is invalid", raw)
	}
	return policy, nil
}

// ParseCatalogInstallMetadata parses optional install metadata from markdown frontmatter.
//
// The parser is intentionally additive:
//   - Missing or malformed frontmatter does not raise an error.
//   - Validation errors are returned only when a materialize block is present.
func ParseCatalogInstallMetadata(content string) (*CatalogInstallMetadata, error) {
	metadata, _, ok := ParseCatalogFrontmatter(content)
	if !ok {
		return nil, nil
	}

	rawMaterialize, exists := metadata["materialize"]
	if !exists || rawMaterialize == nil {
		return nil, nil
	}

	materializeMap, ok := rawMaterialize.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("frontmatter materialize must be a map")
	}

	parsed := &CatalogInstallMetadata{}
	if rawTargetPath, exists := materializeMap["target_path"]; exists {
		targetPath, ok := rawTargetPath.(string)
		if !ok {
			return nil, fmt.Errorf("frontmatter materialize.target_path must be a string")
		}
		normalizedTargetPath, err := ValidateCatalogMaterializeTargetPath(targetPath)
		if err != nil {
			return nil, fmt.Errorf("frontmatter materialize.target_path is invalid: %w", err)
		}
		parsed.Materialize.TargetPath = normalizedTargetPath
	}

	if rawConflictPolicy, exists := materializeMap["conflict_policy"]; exists {
		conflictPolicy, ok := rawConflictPolicy.(string)
		if !ok {
			return nil, fmt.Errorf("frontmatter materialize.conflict_policy must be a string")
		}
		parsedConflictPolicy, err := ParseCatalogMaterializeConflictPolicy(conflictPolicy)
		if err != nil {
			return nil, fmt.Errorf("frontmatter materialize.conflict_policy is invalid: %w", err)
		}
		parsed.Materialize.ConflictPolicy = parsedConflictPolicy
	}

	return parsed, nil
}

// ParseCatalogFrontmatter extracts YAML frontmatter from markdown content.
func ParseCatalogFrontmatter(content string) (map[string]any, string, bool) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---") {
		return nil, trimmed, false
	}

	endIdx := strings.Index(trimmed[3:], "---")
	if endIdx == -1 {
		return nil, trimmed, false
	}

	frontmatter := trimmed[3 : endIdx+3]
	body := strings.TrimSpace(trimmed[endIdx+6:])

	metadata := map[string]any{}
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil, trimmed, false
	}

	return metadata, body, true
}

// ValidateCatalogMaterializeTargetPath validates and normalizes install target paths.
func ValidateCatalogMaterializeTargetPath(raw string) (string, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return "", fmt.Errorf("target path cannot be empty")
	}

	normalized = strings.ReplaceAll(normalized, "\\", "/")
	if strings.HasPrefix(normalized, "/") {
		return "", fmt.Errorf("target path must be relative")
	}
	if len(normalized) >= 2 && normalized[1] == ':' {
		return "", fmt.Errorf("target path must be relative")
	}

	cleaned := path.Clean(normalized)
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("target path cannot be empty")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("target path cannot escape destination root")
	}
	if strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("target path must be relative")
	}

	for strings.HasPrefix(cleaned, "./") {
		cleaned = strings.TrimPrefix(cleaned, "./")
	}
	if cleaned == "" {
		return "", fmt.Errorf("target path cannot be empty")
	}

	return cleaned, nil
}
