package domain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogExportService_ExportLocalSkill_ReturnsDeterministicArchiveCompatibleWithImport(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	writeSkillFixture(t, skillsDir, "test-skill", map[string]string{
		"scripts/run.sh":   "#!/bin/bash\necho hello\n",
		"prompts/dev.md":   "Use deterministic output.\n",
		"references/a.txt": "reference\n",
	})

	manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
	service := newCatalogExportServiceForTest(t, manager)

	result, err := service.Export(context.Background(), CatalogExportRequest{
		ItemIDs: []string{BuildSkillCatalogItemID("test-skill")},
	})
	if err != nil {
		t.Fatalf("expected local skill export to succeed, got %v", err)
	}

	if result.DryRun {
		t.Fatalf("expected non-dry-run export result")
	}
	if result.FileName != "test-skill.tar.gz" {
		t.Fatalf("expected deterministic file name %q, got %q", "test-skill.tar.gz", result.FileName)
	}
	if result.ContentType != "application/gzip" {
		t.Fatalf("expected content type application/gzip, got %q", result.ContentType)
	}
	if len(result.ArchiveData) == 0 {
		t.Fatalf("expected archive payload bytes")
	}
	if len(result.Manifest.Items) != 1 {
		t.Fatalf("expected one manifest item, got %d", len(result.Manifest.Items))
	}

	item := result.Manifest.Items[0]
	if item.ItemID != BuildSkillCatalogItemID("test-skill") {
		t.Fatalf("expected manifest item id %q, got %q", BuildSkillCatalogItemID("test-skill"), item.ItemID)
	}
	if item.ArchiveRoot != "test-skill" {
		t.Fatalf("expected archive root %q, got %q", "test-skill", item.ArchiveRoot)
	}
	if item.ArchiveFileName != "test-skill.tar.gz" {
		t.Fatalf("expected archive file name %q, got %q", "test-skill.tar.gz", item.ArchiveFileName)
	}

	importDir := t.TempDir()
	importedSkillName, err := ImportSkill(result.ArchiveData, importDir)
	if err != nil {
		t.Fatalf("expected ImportSkill to accept exported archive, got %v", err)
	}
	if importedSkillName != "test-skill" {
		t.Fatalf("expected imported skill name %q, got %q", "test-skill", importedSkillName)
	}

	importedSkillPath := filepath.Join(importDir, importedSkillName)
	assertFileExists(t, filepath.Join(importedSkillPath, "SKILL.md"))
	assertFileExists(t, filepath.Join(importedSkillPath, "scripts", "run.sh"))
	assertFileExists(t, filepath.Join(importedSkillPath, "prompts", "dev.md"))
}

func TestCatalogExportService_ExportGitSkill_ReturnsDeterministicArchiveCompatibleWithImport(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	writeSkillFixture(t, filepath.Join(skillsDir, "fixture-git"), "git-skill", map[string]string{
		"scripts/deploy.sh": "echo deploy\n",
	})

	manager := newCatalogExportServiceTestManager(t, skillsDir, []string{"fixture-git"})
	service := newCatalogExportServiceForTest(t, manager)

	result, err := service.Export(context.Background(), CatalogExportRequest{
		ItemIDs: []string{BuildSkillCatalogItemID("fixture-git/git-skill")},
	})
	if err != nil {
		t.Fatalf("expected git skill export to succeed, got %v", err)
	}

	if result.FileName != "fixture-git-git-skill.tar.gz" {
		t.Fatalf(
			"expected deterministic file name %q, got %q",
			"fixture-git-git-skill.tar.gz",
			result.FileName,
		)
	}
	if len(result.Manifest.Items) != 1 {
		t.Fatalf("expected one manifest item, got %d", len(result.Manifest.Items))
	}
	if result.Manifest.Items[0].ArchiveRoot != "git-skill" {
		t.Fatalf("expected archive root %q, got %q", "git-skill", result.Manifest.Items[0].ArchiveRoot)
	}
	if len(result.ArchiveData) == 0 {
		t.Fatalf("expected archive payload bytes")
	}

	importDir := t.TempDir()
	importedSkillName, err := ImportSkill(result.ArchiveData, importDir)
	if err != nil {
		t.Fatalf("expected ImportSkill to accept git-skill archive, got %v", err)
	}
	if importedSkillName != "git-skill" {
		t.Fatalf("expected imported git skill name %q, got %q", "git-skill", importedSkillName)
	}
	assertFileExists(t, filepath.Join(importDir, importedSkillName, "scripts", "deploy.sh"))
}

func TestCatalogExportService_ExportMissingSkill_ReturnsExplicitNotFound(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
	service := newCatalogExportServiceForTest(t, manager)

	_, err := service.Export(context.Background(), CatalogExportRequest{
		ItemIDs: []string{BuildSkillCatalogItemID("missing-skill")},
	})
	if err == nil {
		t.Fatalf("expected missing skill export to fail")
	}
	if !errors.Is(err, ErrCatalogExportItemNotFound) {
		t.Fatalf("expected ErrCatalogExportItemNotFound, got %v", err)
	}
}

func TestCatalogExportService_ExportUnsupportedClassifier_ReturnsExplicitError(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
	service := newCatalogExportServiceForTest(t, manager)

	_, err := service.Export(context.Background(), CatalogExportRequest{
		ItemIDs: []string{BuildPromptCatalogItemID("demo-skill", "prompts/system.md")},
	})
	if err == nil {
		t.Fatalf("expected prompt item export to fail during WP-001 scope")
	}
	if !errors.Is(err, ErrCatalogExportUnsupportedClassifier) {
		t.Fatalf("expected ErrCatalogExportUnsupportedClassifier, got %v", err)
	}
}

func TestCatalogExportService_ExportDryRun_ReturnsManifestWithoutPayload(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	writeSkillFixture(t, skillsDir, "dry-run-skill", map[string]string{
		"scripts/check.sh": "echo check\n",
	})

	manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
	service := newCatalogExportServiceForTest(t, manager)

	result, err := service.Export(context.Background(), CatalogExportRequest{
		ItemIDs: []string{"dry-run-skill"},
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("expected dry-run export to succeed, got %v", err)
	}

	if !result.DryRun {
		t.Fatalf("expected dry_run=true result")
	}
	if len(result.ArchiveData) != 0 {
		t.Fatalf("expected no archive payload bytes for dry-run export")
	}
	if result.FileName != "" || result.ContentType != "" {
		t.Fatalf("expected no output file metadata for dry-run export, got file=%q content_type=%q", result.FileName, result.ContentType)
	}
	if len(result.Manifest.Items) != 1 {
		t.Fatalf("expected one manifest item, got %d", len(result.Manifest.Items))
	}
	if result.Manifest.Items[0].ArchiveFileName != "dry-run-skill.tar.gz" {
		t.Fatalf(
			"expected deterministic archive filename %q, got %q",
			"dry-run-skill.tar.gz",
			result.Manifest.Items[0].ArchiveFileName,
		)
	}
}

func TestCatalogExportService_ExportBareAndCanonicalSkillIDsResolveToSameCanonicalManifestItem(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	writeSkillFixture(t, skillsDir, "compat-skill", map[string]string{
		"scripts/check.sh": "echo check\n",
	})

	manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
	service := newCatalogExportServiceForTest(t, manager)

	bareResult, err := service.Export(context.Background(), CatalogExportRequest{
		ItemIDs: []string{"compat-skill"},
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("expected bare skill export dry-run to succeed, got %v", err)
	}

	canonicalResult, err := service.Export(context.Background(), CatalogExportRequest{
		ItemIDs: []string{BuildSkillCatalogItemID("compat-skill")},
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("expected canonical skill export dry-run to succeed, got %v", err)
	}

	if len(bareResult.Manifest.Items) != 1 || len(canonicalResult.Manifest.Items) != 1 {
		t.Fatalf("expected one manifest item per export result")
	}

	bareItem := bareResult.Manifest.Items[0]
	canonicalItem := canonicalResult.Manifest.Items[0]
	if bareItem != canonicalItem {
		t.Fatalf("expected bare/canonical manifests to match, bare=%+v canonical=%+v", bareItem, canonicalItem)
	}
	if bareItem.ItemID != BuildSkillCatalogItemID("compat-skill") {
		t.Fatalf("expected canonical manifest item id %q, got %q", BuildSkillCatalogItemID("compat-skill"), bareItem.ItemID)
	}
}

func TestCatalogExportService_ExportMatchesLegacyArchiveFileSet(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	writeSkillFixture(t, skillsDir, "compat-skill", map[string]string{
		"scripts/compat.sh":   "echo compat\n",
		"references/notes.md": "legacy compatibility\n",
	})

	manager := newCatalogExportServiceTestManager(t, skillsDir, nil)
	service := newCatalogExportServiceForTest(t, manager)

	serviceResult, err := service.Export(context.Background(), CatalogExportRequest{
		ItemIDs: []string{"compat-skill"},
	})
	if err != nil {
		t.Fatalf("expected service export to succeed, got %v", err)
	}

	legacyArchiveData, err := ExportSkill("compat-skill", skillsDir)
	if err != nil {
		t.Fatalf("expected legacy ExportSkill to succeed, got %v", err)
	}

	serviceImportDir := t.TempDir()
	legacyImportDir := t.TempDir()

	serviceSkillName, err := ImportSkill(serviceResult.ArchiveData, serviceImportDir)
	if err != nil {
		t.Fatalf("expected service archive import to succeed, got %v", err)
	}
	legacySkillName, err := ImportSkill(legacyArchiveData, legacyImportDir)
	if err != nil {
		t.Fatalf("expected legacy archive import to succeed, got %v", err)
	}

	serviceFiles := readFileTree(t, filepath.Join(serviceImportDir, serviceSkillName))
	legacyFiles := readFileTree(t, filepath.Join(legacyImportDir, legacySkillName))
	if len(serviceFiles) != len(legacyFiles) {
		t.Fatalf(
			"expected identical imported file counts, service=%d legacy=%d",
			len(serviceFiles),
			len(legacyFiles),
		)
	}

	for relPath, legacyContent := range legacyFiles {
		serviceContent, ok := serviceFiles[relPath]
		if !ok {
			t.Fatalf("expected service import to contain file %q", relPath)
		}
		if serviceContent != legacyContent {
			t.Fatalf("expected file %q to match legacy import content", relPath)
		}
	}
}

func newCatalogExportServiceForTest(t *testing.T, skillReader catalogExportSkillReader) *CatalogExportService {
	t.Helper()

	service, err := NewCatalogExportService(skillReader)
	if err != nil {
		t.Fatalf("expected NewCatalogExportService to succeed, got %v", err)
	}
	return service
}

func newCatalogExportServiceTestManager(t *testing.T, skillsDir string, gitRepos []string) *FileSystemManager {
	t.Helper()

	manager, err := NewFileSystemManager(skillsDir, gitRepos)
	if err != nil {
		t.Fatalf("expected NewFileSystemManager to succeed, got %v", err)
	}

	t.Cleanup(func() {
		_ = manager.Close()
	})
	return manager
}

func writeSkillFixture(t *testing.T, rootDir, skillName string, files map[string]string) {
	t.Helper()

	skillDir := filepath.Join(rootDir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("expected skill directory creation to succeed, got %v", err)
	}

	skillMarkdown := `---
name: ` + skillName + `
description: Test fixture skill
---
# ` + skillName + `
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMarkdown), 0644); err != nil {
		t.Fatalf("expected SKILL.md write to succeed, got %v", err)
	}

	for relPath, content := range files {
		targetPath := filepath.Join(skillDir, relPath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			t.Fatalf("expected fixture subdirectory creation to succeed, got %v", err)
		}
		if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
			t.Fatalf("expected fixture file write to succeed, got %v", err)
		}
	}
}

func assertFileExists(t *testing.T, filePath string) {
	t.Helper()

	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected %q to exist, got %v", filePath, err)
	}
}

func readFileTree(t *testing.T, rootDir string) map[string]string {
	t.Helper()

	contentsByPath := make(map[string]string)
	if err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		contentsByPath[filepath.ToSlash(relPath)] = string(content)
		return nil
	}); err != nil {
		t.Fatalf("expected file tree walk to succeed, got %v", err)
	}

	return contentsByPath
}
