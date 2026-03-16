package domain

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (m *FileSystemManager) buildCatalogItems(skills []Skill) ([]CatalogItem, error) {
	sortedSkills := append([]Skill(nil), skills...)
	sort.Slice(sortedSkills, func(i, j int) bool {
		leftID := strings.TrimSpace(sortedSkills[i].ID)
		rightID := strings.TrimSpace(sortedSkills[j].ID)
		leftKey := CanonicalSkillCatalogKey(leftID)
		rightKey := CanonicalSkillCatalogKey(rightID)

		if leftKey != rightKey {
			return leftKey < rightKey
		}
		return leftID < rightID
	})

	items := make([]CatalogItem, 0, len(sortedSkills))
	seenPromptKeys := make(map[string]struct{})
	seenRuleKeys := make(map[string]struct{})
	promptDirAllowlist := m.PromptCatalogDirectoryAllowlist()
	ruleDirAllowlist := m.RuleCatalogDirectoryAllowlist()
	ruleFilenameAllowlist := m.RuleCatalogFilenameAllowlist()

	for _, skill := range sortedSkills {
		skillID := strings.TrimSpace(skill.ID)
		if skillID == "" {
			skillID = strings.TrimSpace(skill.Name)
		}

		canonicalSkillKey := CanonicalSkillCatalogKey(skillID)
		if canonicalSkillKey == "" {
			return nil, fmt.Errorf("catalog skill item has an empty canonical skill key")
		}

		skillDescription := ""
		if skill.Metadata != nil {
			skillDescription = skill.Metadata.Description
		}

		skillName := strings.TrimSpace(skill.Name)
		if skillName == "" {
			skillName = skillID
		}

		items = append(items, CatalogItem{
			ID:               BuildSkillCatalogItemID(skillID),
			Classifier:       CatalogClassifierSkill,
			Name:             skillName,
			Description:      skillDescription,
			Content:          skill.Content,
			ContentWritable:  !skill.ReadOnly,
			MetadataWritable: true,
			ReadOnly:         skill.ReadOnly,
		})

		if !m.enablePromptCatalog && !m.enableRuleCatalog {
			continue
		}

		resources, err := m.listSkillResources(skillID, skillResourceListOptions{
			includeExplicitImports:         m.enableImportDiscovery,
			includeImplicitGitPromptShares: m.enablePromptCatalog,
			includeImplicitGitRuleShares:   m.enableRuleCatalog,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list resources for skill %q: %w", skillID, err)
		}

		if m.enablePromptCatalog {
			for _, resource := range resources {
				resourcePath := CanonicalPromptCatalogResourcePath(resource.Path)
				if !IsPromptCatalogCandidate(resourcePath, promptDirAllowlist) {
					continue
				}

				promptContent, err := m.readCatalogResourceContent(skillID, resourcePath)
				if err != nil {
					return nil, fmt.Errorf("failed to read prompt resource %q for skill %q: %w", resourcePath, skillID, err)
				}

				promptName := filepath.Base(filepath.FromSlash(resourcePath))
				if promptName == "" || promptName == "." {
					promptName = resource.Name
				}
				promptName, promptDescription := derivePromptCatalogPresentation(promptName, promptContent)
				promptKey := buildPromptCatalogDedupeKey(promptName, promptDescription, resourcePath)
				if promptKey == "" {
					continue
				}
				if _, exists := seenPromptKeys[promptKey]; exists {
					continue
				}

				items = append(items, CatalogItem{
					ID:               BuildPromptCatalogItemID(skillID, resourcePath),
					Classifier:       CatalogClassifierPrompt,
					Name:             promptName,
					Description:      promptDescription,
					Content:          promptContent,
					ParentSkillID:    skillID,
					ResourcePath:     resourcePath,
					ContentWritable:  !(skill.ReadOnly || !resource.Writable),
					MetadataWritable: true,
					ReadOnly:         skill.ReadOnly || !resource.Writable,
				})
				seenPromptKeys[promptKey] = struct{}{}
			}
		}

		if !m.enableRuleCatalog {
			continue
		}

		ruleCandidates, err := m.listRuleCatalogCandidates(skillID, resources, ruleDirAllowlist, ruleFilenameAllowlist)
		if err != nil {
			return nil, fmt.Errorf("failed to discover rule resources for skill %q: %w", skillID, err)
		}
		for _, ruleCandidate := range ruleCandidates {
			ruleName := filepath.Base(filepath.FromSlash(ruleCandidate.resourcePath))
			if ruleName == "" || ruleName == "." {
				ruleName = ruleCandidate.name
			}
			ruleName, ruleDescription := deriveRuleCatalogPresentation(ruleName, ruleCandidate.content)
			ruleKey := buildRuleCatalogDedupeKey(skillID, ruleCandidate.resourcePath)
			if ruleKey == "" {
				continue
			}
			if _, exists := seenRuleKeys[ruleKey]; exists {
				continue
			}
			readOnly := skill.ReadOnly || !ruleCandidate.writable

			items = append(items, CatalogItem{
				ID:               BuildRuleCatalogItemID(skillID, ruleCandidate.resourcePath),
				Classifier:       CatalogClassifierRule,
				Name:             ruleName,
				Description:      ruleDescription,
				Content:          ruleCandidate.content,
				ParentSkillID:    skillID,
				ResourcePath:     ruleCandidate.resourcePath,
				ContentWritable:  !readOnly,
				MetadataWritable: true,
				ReadOnly:         readOnly,
			})
			seenRuleKeys[ruleKey] = struct{}{}
		}
	}

	sortCatalogItems(items)
	return items, nil
}

func (m *FileSystemManager) readCatalogResourceContent(skillID, resourcePath string) (string, error) {
	resourceContent, err := m.ReadSkillResource(skillID, resourcePath)
	if err != nil {
		return "", err
	}
	if resourceContent == nil {
		return "", nil
	}
	if strings.EqualFold(resourceContent.Encoding, "utf-8") {
		return resourceContent.Content, nil
	}
	return "", nil
}

type ruleCatalogCandidate struct {
	resourcePath string
	name         string
	content      string
	writable     bool
}

func (m *FileSystemManager) listRuleCatalogCandidates(
	skillID string,
	resources []SkillResource,
	ruleDirAllowlist []string,
	ruleFilenameAllowlist []string,
) ([]ruleCatalogCandidate, error) {
	skillPath, err := m.getSkillPath(skillID)
	if err != nil {
		return nil, err
	}
	allowedRoot := m.getSkillAllowedRoot(skillPath)

	candidateByTarget := map[string]ruleCatalogCandidate{}
	addCandidate := func(candidate ruleCatalogCandidate) {
		targetKey := candidate.resourcePath
		if resolvedPath, _, resolveErr := resolveSkillResourcePath(skillPath, allowedRoot, candidate.resourcePath); resolveErr == nil {
			if canonicalPath, canonicalErr := canonicalizeExistingPath(resolvedPath); canonicalErr == nil {
				targetKey = canonicalPath
			}
		}

		existing, exists := candidateByTarget[targetKey]
		if !exists {
			candidateByTarget[targetKey] = candidate
			return
		}

		existingIsDirect := !IsImportedResourcePath(existing.resourcePath)
		candidateIsDirect := !IsImportedResourcePath(candidate.resourcePath)
		if existingIsDirect != candidateIsDirect {
			if candidateIsDirect {
				candidateByTarget[targetKey] = candidate
			}
			return
		}
		if candidate.resourcePath < existing.resourcePath {
			candidateByTarget[targetKey] = candidate
		}
	}

	for _, resource := range resources {
		resourcePath := CanonicalRuleCatalogResourcePath(resource.Path)
		if !IsRuleCatalogCandidate(resourcePath, ruleDirAllowlist, ruleFilenameAllowlist) {
			continue
		}

		content, err := m.readCatalogResourceContent(skillID, resourcePath)
		if err != nil {
			return nil, fmt.Errorf("read rule resource %q: %w", resourcePath, err)
		}
		addCandidate(ruleCatalogCandidate{
			resourcePath: resourcePath,
			name:         resource.Name,
			content:      content,
			writable:     resource.Writable,
		})
	}

	directCandidates, err := m.listDirectRuleCatalogCandidates(skillPath, ruleDirAllowlist, ruleFilenameAllowlist)
	if err != nil {
		return nil, err
	}
	for _, candidate := range directCandidates {
		addCandidate(candidate)
	}

	candidates := make([]ruleCatalogCandidate, 0, len(candidateByTarget))
	for _, candidate := range candidateByTarget {
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].resourcePath < candidates[j].resourcePath })

	return candidates, nil
}

func (m *FileSystemManager) listDirectRuleCatalogCandidates(
	skillPath string,
	ruleDirAllowlist []string,
	ruleFilenameAllowlist []string,
) ([]ruleCatalogCandidate, error) {
	canonicalSkillPath, err := canonicalizeExistingPath(skillPath)
	if err != nil {
		return nil, fmt.Errorf("resolve skill path for rule discovery: %w", err)
	}

	writable := !m.isGitRepoPath(skillPath)
	candidateByPath := map[string]ruleCatalogCandidate{}

	addCandidate := func(fullPath string) {
		canonicalTargetPath, err := canonicalizeExistingPath(fullPath)
		if err != nil {
			return
		}
		if !isWithinRoot(canonicalTargetPath, canonicalSkillPath) {
			return
		}

		relativePath, err := filepath.Rel(canonicalSkillPath, canonicalTargetPath)
		if err != nil {
			return
		}
		resourcePath := CanonicalRuleCatalogResourcePath(relativePath)
		if !IsRuleCatalogCandidate(resourcePath, ruleDirAllowlist, ruleFilenameAllowlist) {
			return
		}

		content, err := os.ReadFile(canonicalTargetPath)
		if err != nil {
			return
		}

		resourceName := filepath.Base(filepath.FromSlash(resourcePath))
		candidateByPath[resourcePath] = ruleCatalogCandidate{
			resourcePath: resourcePath,
			name:         resourceName,
			content:      string(content),
			writable:     writable,
		}
	}

	for _, filename := range ruleFilenameAllowlist {
		normalizedFilename := strings.TrimSpace(filename)
		if normalizedFilename == "" {
			continue
		}
		fullPath := filepath.Join(canonicalSkillPath, filepath.FromSlash(normalizedFilename))
		info, statErr := os.Stat(fullPath)
		if statErr != nil || info.IsDir() {
			continue
		}
		addCandidate(fullPath)
	}

	for _, directory := range ruleDirAllowlist {
		normalizedDirectory := strings.TrimSpace(directory)
		if normalizedDirectory == "" {
			continue
		}

		ruleDirPath := filepath.Join(canonicalSkillPath, filepath.FromSlash(normalizedDirectory))
		if _, statErr := os.Stat(ruleDirPath); statErr != nil {
			continue
		}

		_ = filepath.WalkDir(ruleDirPath, func(currentPath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			addCandidate(currentPath)
			return nil
		})
	}

	sortedPaths := make([]string, 0, len(candidateByPath))
	for resourcePath := range candidateByPath {
		sortedPaths = append(sortedPaths, resourcePath)
	}
	sort.Strings(sortedPaths)

	candidates := make([]ruleCatalogCandidate, 0, len(sortedPaths))
	for _, resourcePath := range sortedPaths {
		candidates = append(candidates, candidateByPath[resourcePath])
	}
	return candidates, nil
}

func sortCatalogItems(items []CatalogItem) {
	sort.Slice(items, func(i, j int) bool {
		leftItem := items[i]
		rightItem := items[j]

		if leftItem.Classifier != rightItem.Classifier {
			return leftItem.Classifier < rightItem.Classifier
		}
		if leftItem.ID != rightItem.ID {
			return leftItem.ID < rightItem.ID
		}
		if leftItem.ParentSkillID != rightItem.ParentSkillID {
			return leftItem.ParentSkillID < rightItem.ParentSkillID
		}
		if leftItem.ResourcePath != rightItem.ResourcePath {
			return leftItem.ResourcePath < rightItem.ResourcePath
		}
		return leftItem.Name < rightItem.Name
	})
}

func derivePromptCatalogPresentation(fallbackName, promptContent string) (string, string) {
	return deriveCatalogPresentation(fallbackName, promptContent)
}

func deriveRuleCatalogPresentation(fallbackName, ruleContent string) (string, string) {
	return deriveCatalogPresentation(fallbackName, ruleContent)
}

func deriveCatalogPresentation(fallbackName, content string) (string, string) {
	name := strings.TrimSpace(fallbackName)
	description := ""
	contentBody := strings.TrimSpace(content)

	if metadata, body, ok := parsePromptCatalogFrontmatter(content); ok {
		if metadataName, ok := metadata["name"].(string); ok && strings.TrimSpace(metadataName) != "" {
			name = strings.TrimSpace(metadataName)
		}
		if metadataDescription, ok := metadata["description"].(string); ok && strings.TrimSpace(metadataDescription) != "" {
			description = strings.TrimSpace(metadataDescription)
		}
		if strings.TrimSpace(body) != "" {
			contentBody = strings.TrimSpace(body)
		}
	}

	if description == "" {
		description = extractFirstParagraph(contentBody)
	}
	if description == "" {
		description = strings.TrimSpace(contentBody)
	}

	return name, description
}

func parsePromptCatalogFrontmatter(content string) (map[string]any, string, bool) {
	return ParseCatalogFrontmatter(content)
}

func extractFirstParagraph(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	paragraph := make([]string, 0, 4)
	inCodeBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}
		if trimmed == "" {
			if len(paragraph) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		paragraph = append(paragraph, trimmed)
	}

	if len(paragraph) == 0 {
		return ""
	}
	return strings.Join(paragraph, " ")
}

func buildPromptCatalogDedupeKey(promptName, promptDescription, resourcePath string) string {
	nameKey := strings.ToLower(strings.TrimSpace(promptName))
	descriptionKey := strings.ToLower(strings.Join(strings.Fields(promptDescription), " "))
	switch {
	case nameKey != "" && descriptionKey != "":
		return nameKey + ":" + descriptionKey
	case nameKey != "":
		return nameKey
	default:
		return CanonicalPromptCatalogResourcePath(resourcePath)
	}
}

func buildRuleCatalogDedupeKey(skillID, resourcePath string) string {
	normalizedResourcePath := CanonicalRuleCatalogResourcePath(resourcePath)
	if normalizedResourcePath == "" {
		return ""
	}

	if IsImportedResourcePath(normalizedResourcePath) {
		return normalizedResourcePath
	}

	return CanonicalRuleCatalogKey(skillID, normalizedResourcePath)
}
