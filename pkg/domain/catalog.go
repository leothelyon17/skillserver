package domain

import (
	"fmt"
	"path"
	"strings"
)

// CatalogClassifier identifies the top-level catalog item type.
type CatalogClassifier string

const (
	CatalogClassifierSkill  CatalogClassifier = "skill"
	CatalogClassifierPrompt CatalogClassifier = "prompt"
)

const (
	skillCatalogIDPrefix  = "skill:"
	promptCatalogIDPrefix = "prompt:"
)

var defaultPromptDirectoryAllowlist = []string{"agent", "agents", "prompt", "prompts"}

// CatalogItem represents a first-class searchable catalog object.
type CatalogItem struct {
	ID            string            `json:"id"`
	Classifier    CatalogClassifier `json:"classifier"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	Content       string            `json:"content,omitempty"`
	ParentSkillID string            `json:"parent_skill_id,omitempty"`
	ResourcePath  string            `json:"resource_path,omitempty"`
	ReadOnly      bool              `json:"read_only"`
}

// IsValid reports whether the classifier is supported.
func (c CatalogClassifier) IsValid() bool {
	switch c {
	case CatalogClassifierSkill, CatalogClassifierPrompt:
		return true
	default:
		return false
	}
}

// ParseCatalogClassifier parses and validates classifier input.
func ParseCatalogClassifier(raw string) (CatalogClassifier, error) {
	classifier := CatalogClassifier(strings.ToLower(strings.TrimSpace(raw)))
	if !classifier.IsValid() {
		return "", fmt.Errorf("invalid catalog classifier %q", raw)
	}
	return classifier, nil
}

// DefaultPromptDirectoryAllowlist returns a defensive copy of default prompt directory names.
func DefaultPromptDirectoryAllowlist() []string {
	copied := make([]string, len(defaultPromptDirectoryAllowlist))
	copy(copied, defaultPromptDirectoryAllowlist)
	return copied
}

// NormalizePromptDirectoryAllowlist normalizes and de-duplicates prompt directory names.
func NormalizePromptDirectoryAllowlist(promptDirs []string) []string {
	normalized := make([]string, 0, len(promptDirs))
	seen := make(map[string]struct{}, len(promptDirs))

	for _, entry := range promptDirs {
		value := strings.ToLower(strings.TrimSpace(entry))
		if value == "" {
			continue
		}

		value = strings.Trim(value, "/")
		if value == "" {
			continue
		}
		if strings.Contains(value, "/") {
			continue
		}

		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	return normalized
}

// ClassifyCatalogPath classifies a path into a catalog item type when it can be inferred.
func ClassifyCatalogPath(resourcePath string, promptDirAllowlist []string) (CatalogClassifier, bool) {
	normalizedPath := normalizeCatalogPath(resourcePath)
	if normalizedPath == "" {
		return "", false
	}

	if isSkillDefinitionPath(normalizedPath) {
		return CatalogClassifierSkill, true
	}

	if IsPromptCatalogCandidate(normalizedPath, promptDirAllowlist) {
		return CatalogClassifierPrompt, true
	}

	return "", false
}

// IsPromptCatalogCandidate reports whether a resource path should be classified as a prompt catalog item.
func IsPromptCatalogCandidate(resourcePath string, promptDirAllowlist []string) bool {
	normalizedPath := normalizeCatalogPath(resourcePath)
	if normalizedPath == "" {
		return false
	}

	if isSkillDefinitionPath(normalizedPath) {
		return false
	}

	if !isMarkdownPath(normalizedPath) {
		return false
	}

	allowlist := NormalizePromptDirectoryAllowlist(promptDirAllowlist)
	if len(allowlist) == 0 {
		allowlist = DefaultPromptDirectoryAllowlist()
	}

	allowed := make(map[string]struct{}, len(allowlist))
	for _, entry := range allowlist {
		allowed[entry] = struct{}{}
	}

	segments := strings.Split(strings.ToLower(normalizedPath), "/")
	for _, segment := range segments[:len(segments)-1] {
		if _, ok := allowed[segment]; ok {
			return true
		}
	}

	return false
}

// CanonicalSkillCatalogKey normalizes skill IDs for deterministic catalog key generation.
func CanonicalSkillCatalogKey(skillID string) string {
	return normalizeCatalogPath(skillID)
}

// CanonicalPromptCatalogResourcePath normalizes prompt resource paths for deterministic keys/IDs.
func CanonicalPromptCatalogResourcePath(resourcePath string) string {
	return normalizeCatalogPath(resourcePath)
}

// CanonicalPromptCatalogKey returns a deterministic prompt dedupe key.
func CanonicalPromptCatalogKey(skillID, resourcePath string) string {
	skillKey := CanonicalSkillCatalogKey(skillID)
	resourceKey := CanonicalPromptCatalogResourcePath(resourcePath)

	if skillKey == "" {
		return resourceKey
	}
	if resourceKey == "" {
		return skillKey
	}

	return skillKey + ":" + resourceKey
}

// BuildSkillCatalogItemID returns a deterministic ID for skill catalog items.
func BuildSkillCatalogItemID(skillID string) string {
	return skillCatalogIDPrefix + CanonicalSkillCatalogKey(skillID)
}

// BuildPromptCatalogItemID returns a deterministic ID for prompt catalog items.
func BuildPromptCatalogItemID(skillID, resourcePath string) string {
	return promptCatalogIDPrefix + CanonicalPromptCatalogKey(skillID, resourcePath)
}

func isSkillDefinitionPath(resourcePath string) bool {
	return strings.EqualFold(path.Base(resourcePath), "SKILL.md")
}

func isMarkdownPath(resourcePath string) bool {
	ext := strings.ToLower(path.Ext(resourcePath))
	return ext == ".md" || ext == ".markdown"
}

func normalizeCatalogPath(raw string) string {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return ""
	}

	cleaned = strings.ReplaceAll(cleaned, "\\", "/")
	cleaned = path.Clean(cleaned)

	if cleaned == "." {
		return ""
	}

	for strings.HasPrefix(cleaned, "./") {
		cleaned = strings.TrimPrefix(cleaned, "./")
	}

	cleaned = strings.TrimPrefix(cleaned, "/")
	return cleaned
}
