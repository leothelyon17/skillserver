package domain_test

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mudler/skillserver/pkg/domain"
)

func BenchmarkFileSystemManager_RebuildIndex_PromptHeavyRepository(b *testing.B) {
	// Bleve emits non-fatal segment cleanup warnings via the standard logger during index rebuilds.
	// Suppress benchmark noise to keep measurement output readable.
	previousOutput := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() {
		log.SetOutput(previousOutput)
	})

	tempDir := b.TempDir()

	const (
		repoName                = "prompt-heavy-repo"
		skillCount              = 20
		directPromptsPerSkill   = 6
		importedPromptsPerSkill = 4
	)

	repoRoot := filepath.Join(tempDir, repoName)
	sharedPromptsRoot := filepath.Join(repoRoot, "prompts")
	if err := os.MkdirAll(sharedPromptsRoot, 0o755); err != nil {
		b.Fatalf("failed to create shared prompts root: %v", err)
	}

	for skillIndex := 0; skillIndex < skillCount; skillIndex++ {
		skillName := fmt.Sprintf("skill-%03d", skillIndex)
		skillRoot := filepath.Join(repoRoot, "skills", skillName)
		if err := os.MkdirAll(filepath.Join(skillRoot, "prompts"), 0o755); err != nil {
			b.Fatalf("failed to create prompts directory for %s: %v", skillName, err)
		}

		skillLines := []string{
			"---",
			fmt.Sprintf("name: %s", skillName),
			"description: Prompt-heavy benchmark fixture",
			"---",
			"",
			fmt.Sprintf("# %s", skillName),
		}

		for promptIndex := 0; promptIndex < directPromptsPerSkill; promptIndex++ {
			promptPath := filepath.Join(skillRoot, "prompts", fmt.Sprintf("direct-%02d.md", promptIndex))
			promptContent := fmt.Sprintf("# Direct Prompt %d\n\nSkill %s direct prompt.", promptIndex, skillName)
			if err := os.WriteFile(promptPath, []byte(promptContent), 0o644); err != nil {
				b.Fatalf("failed to create direct prompt fixture %s: %v", promptPath, err)
			}
		}

		for promptIndex := 0; promptIndex < importedPromptsPerSkill; promptIndex++ {
			sharedName := fmt.Sprintf("shared-%03d-%02d.md", skillIndex, promptIndex)
			sharedPath := filepath.Join(sharedPromptsRoot, sharedName)
			sharedContent := fmt.Sprintf("# Shared Prompt %d\n\nImported prompt for %s.", promptIndex, skillName)
			if err := os.WriteFile(sharedPath, []byte(sharedContent), 0o644); err != nil {
				b.Fatalf("failed to create shared prompt fixture %s: %v", sharedPath, err)
			}

			sharedRelativePath := filepath.ToSlash(filepath.Join("..", "..", "prompts", sharedName))
			skillLines = append(skillLines,
				fmt.Sprintf("[Shared Prompt %d](%s)", promptIndex, sharedRelativePath),
				fmt.Sprintf("@/%s", sharedRelativePath),
			)
		}

		skillMarkdown := strings.Join(skillLines, "\n") + "\n"
		if err := os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte(skillMarkdown), 0o644); err != nil {
			b.Fatalf("failed to write SKILL.md for %s: %v", skillName, err)
		}
	}

	manager, err := domain.NewFileSystemManager(tempDir, []string{repoName})
	if err != nil {
		b.Fatalf("failed to create file system manager: %v", err)
	}

	items, err := manager.ListCatalogItems()
	if err != nil {
		b.Fatalf("failed to build benchmark catalog fixture: %v", err)
	}
	expectedItems := skillCount * (1 + directPromptsPerSkill + importedPromptsPerSkill)
	if len(items) != expectedItems {
		b.Fatalf("expected %d catalog items in prompt-heavy fixture, got %d", expectedItems, len(items))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for run := 0; run < b.N; run++ {
		if err := manager.RebuildIndex(); err != nil {
			b.Fatalf("rebuild index failed at run %d: %v", run, err)
		}
	}
}
