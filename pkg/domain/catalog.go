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
	CatalogClassifierRule   CatalogClassifier = "rule"
)

const (
	skillCatalogIDPrefix  = "skill:"
	promptCatalogIDPrefix = "prompt:"
	ruleCatalogIDPrefix   = "rule:"
)

var defaultPromptDirectoryAllowlist = []string{"agent", "agents", "prompt", "prompts"}
var defaultRuleDirectoryAllowlist = []string{"rule", "rules"}
var defaultRuleFilenameAllowlist = []string{"agents.md", "rules.md", "claude.md", "gemini.md"}

// CatalogItem represents a first-class searchable catalog object.
type CatalogItem struct {
	ID                 string                     `json:"id"`
	Classifier         CatalogClassifier          `json:"classifier"`
	Name               string                     `json:"name"`
	Description        string                     `json:"description,omitempty"`
	Content            string                     `json:"content,omitempty"`
	ParentSkillID      string                     `json:"parent_skill_id,omitempty"`
	ResourcePath       string                     `json:"resource_path,omitempty"`
	PrimaryDomain      *CatalogTaxonomyReference  `json:"primary_domain,omitempty"`
	PrimarySubdomain   *CatalogTaxonomyReference  `json:"primary_subdomain,omitempty"`
	SecondaryDomain    *CatalogTaxonomyReference  `json:"secondary_domain,omitempty"`
	SecondarySubdomain *CatalogTaxonomyReference  `json:"secondary_subdomain,omitempty"`
	Tags               []CatalogTaxonomyReference `json:"tags,omitempty"`
	ContentWritable    bool                       `json:"content_writable"`
	MetadataWritable   bool                       `json:"metadata_writable"`
	CustomMetadata     map[string]any             `json:"custom_metadata,omitempty"`
	Labels             []string                   `json:"labels,omitempty"`
	ReadOnly           bool                       `json:"read_only"`
}

// CatalogTaxonomyReference is a lightweight domain/subdomain/tag reference on effective catalog items.
type CatalogTaxonomyReference struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

func normalizeCatalogItemMutability(item CatalogItem) (contentWritable bool, metadataWritable bool, readOnly bool) {
	contentWritable = item.ContentWritable
	if item.ReadOnly {
		contentWritable = false
	} else if !contentWritable {
		// Preserve legacy behavior for callers that only set read_only.
		contentWritable = true
	}

	metadataWritable = item.MetadataWritable
	if !metadataWritable {
		// Preserve legacy behavior for callers that predate metadata_writable.
		metadataWritable = true
	}

	readOnly = !contentWritable
	return
}

// IsValid reports whether the classifier is supported.
func (c CatalogClassifier) IsValid() bool {
	switch c {
	case CatalogClassifierSkill, CatalogClassifierPrompt, CatalogClassifierRule:
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

// DefaultRuleDirectoryAllowlist returns a defensive copy of default rule directory names.
func DefaultRuleDirectoryAllowlist() []string {
	copied := make([]string, len(defaultRuleDirectoryAllowlist))
	copy(copied, defaultRuleDirectoryAllowlist)
	return copied
}

// DefaultRuleFilenameAllowlist returns a defensive copy of default project-rule filenames.
func DefaultRuleFilenameAllowlist() []string {
	copied := make([]string, len(defaultRuleFilenameAllowlist))
	copy(copied, defaultRuleFilenameAllowlist)
	return copied
}

// NormalizePromptDirectoryAllowlist normalizes and de-duplicates prompt directory names.
func NormalizePromptDirectoryAllowlist(promptDirs []string) []string {
	return normalizeCatalogDirectoryAllowlist(promptDirs)
}

// NormalizeRuleDirectoryAllowlist normalizes and de-duplicates rule directory names.
func NormalizeRuleDirectoryAllowlist(ruleDirs []string) []string {
	return normalizeCatalogDirectoryAllowlist(ruleDirs)
}

// NormalizeRuleFilenameAllowlist normalizes and de-duplicates rule filenames.
func NormalizeRuleFilenameAllowlist(filenames []string) []string {
	normalized := make([]string, 0, len(filenames))
	seen := make(map[string]struct{}, len(filenames))

	for _, entry := range filenames {
		value := strings.ToLower(strings.TrimSpace(entry))
		if value == "" {
			continue
		}

		value = strings.ReplaceAll(value, "\\", "/")
		value = strings.Trim(value, "/")
		if value == "" || strings.Contains(value, "/") {
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
	return ClassifyCatalogPathWithAllowlists(resourcePath, promptDirAllowlist, nil, nil)
}

// ClassifyCatalogPathWithAllowlists classifies a path using prompt and rule allowlist controls.
func ClassifyCatalogPathWithAllowlists(
	resourcePath string,
	promptDirAllowlist []string,
	ruleDirAllowlist []string,
	ruleFilenameAllowlist []string,
) (CatalogClassifier, bool) {
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

	if IsRuleCatalogCandidate(normalizedPath, ruleDirAllowlist, ruleFilenameAllowlist) {
		return CatalogClassifierRule, true
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

	return isCatalogPathInAllowedDirectory(normalizedPath, allowlist)
}

// IsRuleCatalogCandidate reports whether a resource path should be classified as a rule catalog item.
func IsRuleCatalogCandidate(resourcePath string, ruleDirAllowlist []string, ruleFilenameAllowlist []string) bool {
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

	filenameAllowlist := NormalizeRuleFilenameAllowlist(ruleFilenameAllowlist)
	if len(filenameAllowlist) == 0 {
		filenameAllowlist = DefaultRuleFilenameAllowlist()
	}
	if isRuleAllowlistedFilename(normalizedPath, filenameAllowlist) {
		return true
	}

	dirAllowlist := NormalizeRuleDirectoryAllowlist(ruleDirAllowlist)
	if len(dirAllowlist) == 0 {
		dirAllowlist = DefaultRuleDirectoryAllowlist()
	}

	return isCatalogPathInAllowedDirectory(normalizedPath, dirAllowlist)
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

// CanonicalRuleCatalogResourcePath normalizes rule resource paths for deterministic keys/IDs.
func CanonicalRuleCatalogResourcePath(resourcePath string) string {
	return normalizeCatalogPath(resourcePath)
}

// CanonicalRuleCatalogKey returns a deterministic rule dedupe key.
func CanonicalRuleCatalogKey(skillID, resourcePath string) string {
	skillKey := CanonicalSkillCatalogKey(skillID)
	resourceKey := CanonicalRuleCatalogResourcePath(resourcePath)

	if skillKey == "" {
		return resourceKey
	}
	if resourceKey == "" {
		return skillKey
	}

	return skillKey + ":" + resourceKey
}

// BuildRuleCatalogItemID returns a deterministic ID for rule catalog items.
func BuildRuleCatalogItemID(skillID, resourcePath string) string {
	return ruleCatalogIDPrefix + CanonicalRuleCatalogKey(skillID, resourcePath)
}

func normalizeCatalogDirectoryAllowlist(entries []string) []string {
	normalized := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	for _, entry := range entries {
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

func isCatalogPathInAllowedDirectory(resourcePath string, allowlist []string) bool {
	if len(allowlist) == 0 {
		return false
	}

	allowed := make(map[string]struct{}, len(allowlist))
	for _, entry := range allowlist {
		allowed[entry] = struct{}{}
	}

	segments := strings.Split(strings.ToLower(resourcePath), "/")
	for _, segment := range segments[:len(segments)-1] {
		if _, ok := allowed[segment]; ok {
			return true
		}
	}

	return false
}

func isRuleAllowlistedFilename(resourcePath string, filenameAllowlist []string) bool {
	if len(filenameAllowlist) == 0 {
		return false
	}

	allowed := make(map[string]struct{}, len(filenameAllowlist))
	for _, entry := range filenameAllowlist {
		allowed[entry] = struct{}{}
	}

	filename := strings.ToLower(path.Base(resourcePath))
	_, ok := allowed[filename]
	return ok
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
