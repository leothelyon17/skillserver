package domain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogMaterializationService_DryRunPlansTargetsWithoutFilesystemSideEffects(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	writeSkillFixture(t, skillsDir, "planner", map[string]string{
		"prompts/system.md": "# System\nUse deterministic output.\n",
	})

	manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
	item := mustFindCatalogItem(t, manager, CatalogClassifierPrompt, "prompts/system.md")

	allowedRoot := t.TempDir()
	destinationDir := filepath.Join(allowedRoot, "project")
	service := newCatalogMaterializationServiceForTest(t, manager, []string{allowedRoot})

	result, err := service.Materialize(context.Background(), CatalogMaterializationRequest{
		ItemIDs:        []string{item.ID},
		DestinationDir: destinationDir,
		DryRun:         true,
	})
	if err != nil {
		t.Fatalf("expected dry-run materialization to succeed, got %v", err)
	}

	if !result.DryRun {
		t.Fatalf("expected dry-run result")
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected one result item, got %d", len(result.Items))
	}

	itemResult := result.Items[0]
	if itemResult.Status != CatalogMaterializationItemStatusPlanned {
		t.Fatalf("expected item status %q, got %q", CatalogMaterializationItemStatusPlanned, itemResult.Status)
	}
	if len(itemResult.Files) != 1 {
		t.Fatalf("expected one file result, got %d", len(itemResult.Files))
	}
	fileResult := itemResult.Files[0]
	if fileResult.Action != CatalogMaterializationActionCreate {
		t.Fatalf("expected file action %q, got %q", CatalogMaterializationActionCreate, fileResult.Action)
	}
	if fileResult.Written {
		t.Fatalf("expected dry-run file result to have written=false")
	}

	targetPath := filepath.Join(destinationDir, filepath.FromSlash(fileResult.TargetPath))
	if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected no target file writes during dry-run, stat err=%v", statErr)
	}
}

func TestCatalogMaterializationService_DryRunBareAndCanonicalSkillIDsResolveToSameCanonicalItemID(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	writeSkillFixture(t, skillsDir, "planner", map[string]string{
		"prompts/system.md": "# System\nUse deterministic output.\n",
	})

	manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
	allowedRoot := t.TempDir()
	destinationDir := filepath.Join(allowedRoot, "project")
	service := newCatalogMaterializationServiceForTest(t, manager, []string{allowedRoot})

	bareResult, err := service.Materialize(context.Background(), CatalogMaterializationRequest{
		ItemIDs:        []string{"planner"},
		DestinationDir: destinationDir,
		DryRun:         true,
	})
	if err != nil {
		t.Fatalf("expected bare skill materialization dry-run to succeed, got %v", err)
	}

	canonicalResult, err := service.Materialize(context.Background(), CatalogMaterializationRequest{
		ItemIDs:        []string{BuildSkillCatalogItemID("planner")},
		DestinationDir: destinationDir,
		DryRun:         true,
	})
	if err != nil {
		t.Fatalf("expected canonical skill materialization dry-run to succeed, got %v", err)
	}

	if len(bareResult.Items) != 1 || len(canonicalResult.Items) != 1 {
		t.Fatalf("expected one item result per materialization plan")
	}

	bareItem := bareResult.Items[0]
	canonicalItem := canonicalResult.Items[0]
	if bareItem.ItemID != BuildSkillCatalogItemID("planner") {
		t.Fatalf("expected canonical bare item id %q, got %q", BuildSkillCatalogItemID("planner"), bareItem.ItemID)
	}
	if bareItem.ItemID != canonicalItem.ItemID || bareItem.TargetPath != canonicalItem.TargetPath {
		t.Fatalf("expected bare/canonical plans to match, bare=%+v canonical=%+v", bareItem, canonicalItem)
	}
}

func TestCatalogMaterializationService_ConflictPolicies(t *testing.T) {
	t.Parallel()

	const promptContent = "# System\nUse deterministic output.\n"
	testCases := []struct {
		name            string
		policy          CatalogMaterializeConflictPolicy
		expectedAction  CatalogMaterializationAction
		expectedStatus  CatalogMaterializationItemStatus
		expectedContent string
		expectedWritten bool
	}{
		{
			name:            "error policy marks conflicts without writing",
			policy:          CatalogMaterializeConflictPolicyError,
			expectedAction:  CatalogMaterializationActionConflict,
			expectedStatus:  CatalogMaterializationItemStatusConflict,
			expectedContent: "existing",
			expectedWritten: false,
		},
		{
			name:            "skip policy keeps existing file",
			policy:          CatalogMaterializeConflictPolicySkip,
			expectedAction:  CatalogMaterializationActionSkip,
			expectedStatus:  CatalogMaterializationItemStatusSkipped,
			expectedContent: "existing",
			expectedWritten: false,
		},
		{
			name:            "overwrite policy replaces existing file",
			policy:          CatalogMaterializeConflictPolicyOverwrite,
			expectedAction:  CatalogMaterializationActionOverwrite,
			expectedStatus:  CatalogMaterializationItemStatusWritten,
			expectedContent: promptContent,
			expectedWritten: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			skillsDir := t.TempDir()
			writeSkillFixture(t, skillsDir, "planner", map[string]string{
				"prompts/system.md": promptContent,
			})
			manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
			item := mustFindCatalogItem(t, manager, CatalogClassifierPrompt, "prompts/system.md")

			allowedRoot := t.TempDir()
			destinationDir := filepath.Join(allowedRoot, "project")
			conflictingTarget := filepath.Join(destinationDir, "prompts", "system.md")
			if err := os.MkdirAll(filepath.Dir(conflictingTarget), 0755); err != nil {
				t.Fatalf("expected target parent directory creation to succeed, got %v", err)
			}
			if err := os.WriteFile(conflictingTarget, []byte("existing"), 0644); err != nil {
				t.Fatalf("expected conflicting target file setup to succeed, got %v", err)
			}

			service := newCatalogMaterializationServiceForTest(t, manager, []string{allowedRoot})
			result, err := service.Materialize(context.Background(), CatalogMaterializationRequest{
				ItemIDs:        []string{item.ID},
				DestinationDir: destinationDir,
				ConflictPolicy: testCase.policy,
			})
			if err != nil {
				t.Fatalf("expected materialization to succeed, got %v", err)
			}

			itemResult := result.Items[0]
			if itemResult.Status != testCase.expectedStatus {
				t.Fatalf(
					"expected item status %q, got %q",
					testCase.expectedStatus,
					itemResult.Status,
				)
			}
			if len(itemResult.Files) != 1 {
				t.Fatalf("expected one file result, got %d", len(itemResult.Files))
			}

			fileResult := itemResult.Files[0]
			if fileResult.Action != testCase.expectedAction {
				t.Fatalf("expected file action %q, got %q", testCase.expectedAction, fileResult.Action)
			}
			if fileResult.Written != testCase.expectedWritten {
				t.Fatalf("expected written=%t, got %t", testCase.expectedWritten, fileResult.Written)
			}

			content, readErr := os.ReadFile(conflictingTarget)
			if readErr != nil {
				t.Fatalf("expected target file read to succeed, got %v", readErr)
			}
			if string(content) != testCase.expectedContent {
				t.Fatalf("expected target content %q, got %q", testCase.expectedContent, string(content))
			}
		})
	}
}

func TestCatalogMaterializationService_RejectsOutsideAllowedRoots(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	writeSkillFixture(t, skillsDir, "planner", map[string]string{
		"prompts/system.md": "# System\nUse deterministic output.\n",
	})
	manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
	item := mustFindCatalogItem(t, manager, CatalogClassifierPrompt, "prompts/system.md")

	allowedRoot := t.TempDir()
	destinationDir := t.TempDir()
	service := newCatalogMaterializationServiceForTest(t, manager, []string{allowedRoot})

	_, err := service.Materialize(context.Background(), CatalogMaterializationRequest{
		ItemIDs:        []string{item.ID},
		DestinationDir: destinationDir,
	})
	if err == nil {
		t.Fatalf("expected materialization outside allowed roots to fail")
	}
	if !errors.Is(err, ErrCatalogMaterializationDestinationOutsideAllowedRoots) {
		t.Fatalf(
			"expected ErrCatalogMaterializationDestinationOutsideAllowedRoots, got %v",
			err,
		)
	}
}

func TestCatalogMaterializationService_RejectsInvalidMaterializeTargetPaths(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		promptContent string
	}{
		{
			name: "absolute target path is rejected",
			promptContent: `---
materialize:
  target_path: /tmp/AGENTS.md
---
# Prompt
`,
		},
		{
			name: "traversal target path is rejected",
			promptContent: `---
materialize:
  target_path: ../AGENTS.md
---
# Prompt
`,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			skillsDir := t.TempDir()
			writeSkillFixture(t, skillsDir, "planner", map[string]string{
				"prompts/system.md": testCase.promptContent,
			})
			manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
			item := mustFindCatalogItem(t, manager, CatalogClassifierPrompt, "prompts/system.md")

			allowedRoot := t.TempDir()
			destinationDir := filepath.Join(allowedRoot, "project")
			service := newCatalogMaterializationServiceForTest(t, manager, []string{allowedRoot})

			_, err := service.Materialize(context.Background(), CatalogMaterializationRequest{
				ItemIDs:        []string{item.ID},
				DestinationDir: destinationDir,
			})
			if err == nil {
				t.Fatalf("expected invalid target path to fail")
			}
			if !errors.Is(err, ErrCatalogMaterializationInvalidRequest) {
				t.Fatalf("expected ErrCatalogMaterializationInvalidRequest, got %v", err)
			}
		})
	}
}

func TestCatalogMaterializationService_RuleAllowlistedBasenameMaterializesAtProjectRoot(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	writeSkillFixture(t, skillsDir, "planner", map[string]string{
		"AGENTS.md": "# AGENTS\nRepository contributor rules.\n",
	})
	skillPath := filepath.Join(skillsDir, "planner")
	skillMarkdown := `---
name: planner
description: Planner skill
---
# Planner
[Repo Root Rule](/AGENTS.md)
`
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillMarkdown), 0644); err != nil {
		t.Fatalf("expected SKILL.md rewrite to succeed, got %v", err)
	}

	manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
	item := mustFindCatalogItem(t, manager, CatalogClassifierRule, "imports/AGENTS.md")

	allowedRoot := t.TempDir()
	destinationDir := filepath.Join(allowedRoot, "project")
	service := newCatalogMaterializationServiceForTest(t, manager, []string{allowedRoot})

	result, err := service.Materialize(context.Background(), CatalogMaterializationRequest{
		ItemIDs:        []string{item.ID},
		DestinationDir: destinationDir,
		ConflictPolicy: CatalogMaterializeConflictPolicyOverwrite,
	})
	if err != nil {
		t.Fatalf("expected rule materialization to succeed, got %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("expected one item result, got %d", len(result.Items))
	}
	if result.Items[0].TargetPath != "AGENTS.md" {
		t.Fatalf("expected rule target path AGENTS.md, got %q", result.Items[0].TargetPath)
	}

	rootTarget := filepath.Join(destinationDir, "AGENTS.md")
	rulesTarget := filepath.Join(destinationDir, "rules", "AGENTS.md")
	if _, statErr := os.Stat(rootTarget); statErr != nil {
		t.Fatalf("expected AGENTS.md at project root, got %v", statErr)
	}
	if _, statErr := os.Stat(rulesTarget); !os.IsNotExist(statErr) {
		t.Fatalf("expected no rules/AGENTS.md target, stat err=%v", statErr)
	}
}

func TestCatalogMaterializationService_MixedBatchSupportsDirectAndImportedItems(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	repoName := "demo-repo"
	skillPath := filepath.Join(skillsDir, repoName, "plugins", "rule-teams", "skills", "planner")
	repoRootRulePath := filepath.Join(skillsDir, repoName, "AGENTS.md")

	if err := os.MkdirAll(filepath.Join(skillPath, "prompts"), 0755); err != nil {
		t.Fatalf("expected planner prompt directory creation to succeed, got %v", err)
	}
	skillMarkdown := `---
name: planner
description: Planner skill
---
# Planner
[Repo Root Rule](/AGENTS.md)
`
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillMarkdown), 0644); err != nil {
		t.Fatalf("expected SKILL.md write to succeed, got %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "prompts", "system.md"), []byte("# Prompt\nDirect prompt item.\n"), 0644); err != nil {
		t.Fatalf("expected prompt write to succeed, got %v", err)
	}
	if err := os.WriteFile(repoRootRulePath, []byte("# AGENTS\nImported repo-root rule.\n"), 0644); err != nil {
		t.Fatalf("expected repo-root rule write to succeed, got %v", err)
	}

	manager := newCatalogExportServiceTestManager(t, skillsDir, []string{repoName})
	skillID := BuildSkillCatalogItemID("demo-repo/planner")
	promptID := BuildPromptCatalogItemID("demo-repo/planner", "prompts/system.md")
	ruleID := BuildRuleCatalogItemID("demo-repo/planner", "imports/AGENTS.md")
	mustFindCatalogItemByID(t, manager, skillID)
	mustFindCatalogItemByID(t, manager, promptID)
	mustFindCatalogItemByID(t, manager, ruleID)

	allowedRoot := t.TempDir()
	destinationDir := filepath.Join(allowedRoot, "project")
	conflictingPromptTarget := filepath.Join(destinationDir, "prompts", "system.md")
	if err := os.MkdirAll(filepath.Dir(conflictingPromptTarget), 0755); err != nil {
		t.Fatalf("expected conflict target parent creation to succeed, got %v", err)
	}
	if err := os.WriteFile(conflictingPromptTarget, []byte("existing prompt content"), 0644); err != nil {
		t.Fatalf("expected conflict target setup to succeed, got %v", err)
	}

	service := newCatalogMaterializationServiceForTest(t, manager, []string{allowedRoot})
	result, err := service.Materialize(context.Background(), CatalogMaterializationRequest{
		ItemIDs: []string{
			skillID,
			promptID,
			ruleID,
		},
		DestinationDir: destinationDir,
		ConflictPolicy: CatalogMaterializeConflictPolicySkip,
	})
	if err != nil {
		t.Fatalf("expected mixed-batch materialization to succeed, got %v", err)
	}

	if len(result.Items) != 3 {
		t.Fatalf("expected three item results, got %d", len(result.Items))
	}
	if result.Items[0].ItemID != skillID || result.Items[1].ItemID != promptID || result.Items[2].ItemID != ruleID {
		t.Fatalf(
			"expected deterministic item ordering [%q %q %q], got [%q %q %q]",
			skillID,
			promptID,
			ruleID,
			result.Items[0].ItemID,
			result.Items[1].ItemID,
			result.Items[2].ItemID,
		)
	}

	// Skill files are materialized under skills/<skill-dir>/...
	skillTargetPath := filepath.Join(destinationDir, "skills", "planner", "SKILL.md")
	if _, statErr := os.Stat(skillTargetPath); statErr != nil {
		t.Fatalf("expected skill file target %q to exist, got %v", skillTargetPath, statErr)
	}

	// Prompt conflict under skip policy should preserve existing content.
	conflictingPromptBytes, readErr := os.ReadFile(conflictingPromptTarget)
	if readErr != nil {
		t.Fatalf("expected conflicting prompt target read to succeed, got %v", readErr)
	}
	if string(conflictingPromptBytes) != "existing prompt content" {
		t.Fatalf("expected skip policy to preserve conflicting prompt content")
	}

	// Imported AGENTS.md rule should preserve the known project-root basename.
	ruleTargetPath := filepath.Join(destinationDir, "AGENTS.md")
	if _, statErr := os.Stat(ruleTargetPath); statErr != nil {
		t.Fatalf("expected imported AGENTS.md rule target %q to exist, got %v", ruleTargetPath, statErr)
	}
}

func TestCatalogMaterializationService_NoPartialWritesOnPlanningValidationFailure(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	writeSkillFixture(t, skillsDir, "planner", map[string]string{
		"prompts/ok.md": "# OK\nSafe target path.\n",
		"prompts/bad.md": `---
materialize:
  target_path: ../escape.md
---
# Bad
Unsafe target path.
`,
	})

	manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
	okPrompt := mustFindCatalogItem(t, manager, CatalogClassifierPrompt, "prompts/ok.md")
	badPrompt := mustFindCatalogItem(t, manager, CatalogClassifierPrompt, "prompts/bad.md")

	allowedRoot := t.TempDir()
	destinationDir := filepath.Join(allowedRoot, "project")
	service := newCatalogMaterializationServiceForTest(t, manager, []string{allowedRoot})

	_, err := service.Materialize(context.Background(), CatalogMaterializationRequest{
		ItemIDs: []string{
			okPrompt.ID,
			badPrompt.ID,
		},
		DestinationDir: destinationDir,
		ConflictPolicy: CatalogMaterializeConflictPolicyOverwrite,
	})
	if err == nil {
		t.Fatalf("expected planning validation failure for unsafe target path")
	}
	if !errors.Is(err, ErrCatalogMaterializationInvalidRequest) {
		t.Fatalf("expected ErrCatalogMaterializationInvalidRequest, got %v", err)
	}

	okPromptTarget := filepath.Join(destinationDir, "prompts", "ok.md")
	if _, statErr := os.Stat(okPromptTarget); !os.IsNotExist(statErr) {
		t.Fatalf("expected no partial writes after planning failure, stat err=%v", statErr)
	}
}

func newCatalogMaterializationServiceForTest(
	t *testing.T,
	catalogReader catalogMaterializationCatalogReader,
	allowedRoots []string,
) *CatalogMaterializationService {
	t.Helper()

	service, err := NewCatalogMaterializationService(catalogReader, allowedRoots)
	if err != nil {
		t.Fatalf("expected NewCatalogMaterializationService to succeed, got %v", err)
	}
	return service
}

func mustFindCatalogItem(
	t *testing.T,
	manager *FileSystemManager,
	classifier CatalogClassifier,
	resourcePath string,
) CatalogItem {
	t.Helper()

	items, err := manager.ListCatalogItems()
	if err != nil {
		t.Fatalf("expected ListCatalogItems to succeed, got %v", err)
	}

	for _, item := range items {
		if item.Classifier != classifier {
			continue
		}
		if resourcePath == "" || item.ResourcePath == resourcePath {
			return item
		}
	}
	t.Fatalf(
		"expected catalog item classifier=%q resource_path=%q to exist",
		classifier,
		resourcePath,
	)
	return CatalogItem{}
}

func mustFindCatalogItemByID(t *testing.T, manager *FileSystemManager, itemID string) CatalogItem {
	t.Helper()

	items, err := manager.ListCatalogItems()
	if err != nil {
		t.Fatalf("expected ListCatalogItems to succeed, got %v", err)
	}

	for _, item := range items {
		if item.ID == itemID {
			return item
		}
	}
	t.Fatalf("expected catalog item id %q to exist", itemID)
	return CatalogItem{}
}
