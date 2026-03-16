package domain_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mudler/skillserver/pkg/domain"
)

func TestFileSystemManager_KeepsSkillsWithMismatchedFrontmatterNames(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	repoName := "astra-agents"

	skillNames := map[string]string{
		"planning-brainstorm-session":                  "planning-brainstorm-session",
		"planning-complete-implementation-plan":        "planning-complete-implementation-plan",
		"planning-create-architecture-decision-record": "planning-planning-create-architecture-decision-record",
		"planning-create-implementation-plan":          "planning-planning-create-implementation-plan",
		"planning-execute-work-package":                "planning-execute-work-package",
	}

	for dirName, frontmatterName := range skillNames {
		skillDir := filepath.Join(skillsDir, repoName, "skills", dirName)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create skill dir %q: %v", dirName, err)
		}

		skillMarkdown := fmt.Sprintf(`---
name: %s
description: %s fixture
---
# %s
Planning workflow fixture.
`, frontmatterName, dirName, dirName)
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMarkdown), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md for %q: %v", dirName, err)
		}
	}

	manager, err := domain.NewFileSystemManager(skillsDir, []string{repoName})
	if err != nil {
		t.Fatalf("failed to create file system manager: %v", err)
	}
	t.Cleanup(func() {
		_ = manager.Close()
	})

	skills, err := manager.ListSkills()
	if err != nil {
		t.Fatalf("expected ListSkills to succeed, got %v", err)
	}
	if len(skills) != 5 {
		t.Fatalf("expected 5 planning skills, got %d (%+v)", len(skills), skills)
	}

	skillByID := make(map[string]domain.Skill, len(skills))
	for _, skill := range skills {
		skillByID[skill.ID] = skill
	}

	for _, expectedID := range []string{
		"astra-agents/planning-brainstorm-session",
		"astra-agents/planning-complete-implementation-plan",
		"astra-agents/planning-create-architecture-decision-record",
		"astra-agents/planning-create-implementation-plan",
		"astra-agents/planning-execute-work-package",
	} {
		if _, ok := skillByID[expectedID]; !ok {
			t.Fatalf("expected skill %q in catalog snapshot, got %+v", expectedID, skillByID)
		}
	}

	adrSkill, err := manager.ReadSkill("astra-agents/planning-create-architecture-decision-record")
	if err != nil {
		t.Fatalf("expected ADR skill to be readable, got %v", err)
	}
	if got := adrSkill.Metadata.Name; got != "planning-create-architecture-decision-record" {
		t.Fatalf("expected ADR metadata name to be canonicalized to directory name, got %q", got)
	}

	planSkill, err := manager.ReadSkill("astra-agents/planning-create-implementation-plan")
	if err != nil {
		t.Fatalf("expected implementation-plan skill to be readable, got %v", err)
	}
	if got := planSkill.Metadata.Name; got != "planning-create-implementation-plan" {
		t.Fatalf("expected implementation-plan metadata name to be canonicalized to directory name, got %q", got)
	}

	if err := manager.RebuildIndex(); err != nil {
		t.Fatalf("expected RebuildIndex to succeed, got %v", err)
	}

	classifier := domain.CatalogClassifierSkill
	results, err := manager.SearchCatalogItems("planning", &classifier)
	if err != nil {
		t.Fatalf("expected SearchCatalogItems to succeed, got %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 planning search results, got %d (%+v)", len(results), results)
	}
}

func TestImportSkill_AcceptsArchiveWithMismatchedFrontmatterName(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	skillName := "planning-create-implementation-plan"
	skillDir := filepath.Join(tempDir, skillName)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}

	skillMarkdown := `---
name: planning-planning-create-implementation-plan
description: Planning implementation plan fixture
---
# Create Implementation Plan
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMarkdown), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	archiveData, err := domain.ExportSkill(skillName, tempDir)
	if err != nil {
		t.Fatalf("expected ExportSkill to succeed, got %v", err)
	}

	importDir := filepath.Join(tempDir, "imported")
	if err := os.MkdirAll(importDir, 0o755); err != nil {
		t.Fatalf("failed to create import dir: %v", err)
	}

	importedSkillName, err := domain.ImportSkill(archiveData, importDir)
	if err != nil {
		t.Fatalf("expected ImportSkill to succeed, got %v", err)
	}
	if importedSkillName != skillName {
		t.Fatalf("expected imported skill name %q, got %q", skillName, importedSkillName)
	}

	manager, err := domain.NewFileSystemManager(importDir, nil)
	if err != nil {
		t.Fatalf("failed to create file system manager for imported archive: %v", err)
	}
	t.Cleanup(func() {
		_ = manager.Close()
	})

	skill, err := manager.ReadSkill(skillName)
	if err != nil {
		t.Fatalf("expected imported skill to be readable, got %v", err)
	}
	if got := skill.Metadata.Name; got != skillName {
		t.Fatalf("expected imported skill metadata name to be canonicalized, got %q", got)
	}
}
