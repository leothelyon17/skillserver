package domain

import (
	"fmt"
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
	promptDirAllowlist := m.PromptCatalogDirectoryAllowlist()

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
			ID:          BuildSkillCatalogItemID(skillID),
			Classifier:  CatalogClassifierSkill,
			Name:        skillName,
			Description: skillDescription,
			Content:     skill.Content,
			ReadOnly:    skill.ReadOnly,
		})

		if !m.enablePromptCatalog {
			continue
		}

		resources, err := m.ListSkillResources(skillID)
		if err != nil {
			return nil, fmt.Errorf("failed to list resources for skill %q: %w", skillID, err)
		}

		for _, resource := range resources {
			resourcePath := CanonicalPromptCatalogResourcePath(resource.Path)
			if !IsPromptCatalogCandidate(resourcePath, promptDirAllowlist) {
				continue
			}

			promptKey := CanonicalPromptCatalogKey(skillID, resourcePath)
			if promptKey == "" {
				continue
			}
			if _, exists := seenPromptKeys[promptKey]; exists {
				continue
			}

			promptContent, err := m.readPromptCatalogContent(skillID, resourcePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read prompt resource %q for skill %q: %w", resourcePath, skillID, err)
			}

			promptName := filepath.Base(filepath.FromSlash(resourcePath))
			if promptName == "" || promptName == "." {
				promptName = resource.Name
			}

			items = append(items, CatalogItem{
				ID:            BuildPromptCatalogItemID(skillID, resourcePath),
				Classifier:    CatalogClassifierPrompt,
				Name:          promptName,
				Description:   skillDescription,
				Content:       promptContent,
				ParentSkillID: skillID,
				ResourcePath:  resourcePath,
				ReadOnly:      skill.ReadOnly || !resource.Writable,
			})
			seenPromptKeys[promptKey] = struct{}{}
		}
	}

	sortCatalogItems(items)
	return items, nil
}

func (m *FileSystemManager) readPromptCatalogContent(skillID, resourcePath string) (string, error) {
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
