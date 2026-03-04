package domain

import (
	"os"
	"testing"
)

func TestFieldAsBool(t *testing.T) {
	cases := []struct {
		name     string
		input    any
		expected bool
		ok       bool
	}{
		{
			name:     "bool true",
			input:    true,
			expected: true,
			ok:       true,
		},
		{
			name:     "string true",
			input:    "true",
			expected: true,
			ok:       true,
		},
		{
			name:     "invalid string",
			input:    "not-a-bool",
			expected: false,
			ok:       false,
		},
		{
			name:     "unsupported type",
			input:    1,
			expected: false,
			ok:       false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value, ok := fieldAsBool(tc.input)
			if ok != tc.ok {
				t.Fatalf("fieldAsBool(%v) ok = %v, want %v", tc.input, ok, tc.ok)
			}
			if value != tc.expected {
				t.Fatalf("fieldAsBool(%v) value = %v, want %v", tc.input, value, tc.expected)
			}
		})
	}
}

func TestFieldAsString(t *testing.T) {
	text, ok := fieldAsString("value")
	if !ok || text != "value" {
		t.Fatalf("fieldAsString(string) = (%q, %v), want (%q, true)", text, ok, "value")
	}

	text, ok = fieldAsString([]byte("bytes"))
	if !ok || text != "bytes" {
		t.Fatalf("fieldAsString([]byte) = (%q, %v), want (%q, true)", text, ok, "bytes")
	}

	text, ok = fieldAsString(123)
	if ok || text != "" {
		t.Fatalf("fieldAsString(123) = (%q, %v), want (\"\", false)", text, ok)
	}
}

func TestResolveLegacySkillID(t *testing.T) {
	cases := []struct {
		name     string
		item     CatalogItem
		expected string
	}{
		{
			name: "parent skill id wins",
			item: CatalogItem{
				ID:            BuildSkillCatalogItemID("docker"),
				ParentSkillID: "docker-parent",
			},
			expected: "docker-parent",
		},
		{
			name: "skill prefix is stripped",
			item: CatalogItem{
				ID: BuildSkillCatalogItemID("docker"),
			},
			expected: "docker",
		},
		{
			name: "raw id is returned",
			item: CatalogItem{
				ID: "docker",
			},
			expected: "docker",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := resolveLegacySkillID(tc.item); actual != tc.expected {
				t.Fatalf("resolveLegacySkillID() = %q, want %q", actual, tc.expected)
			}
		})
	}
}

func TestIndexCatalogItemsDerivesDeterministicIDs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "search-internal-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	searcher, err := NewSearcher(tempDir)
	if err != nil {
		t.Fatalf("failed to create searcher: %v", err)
	}
	defer searcher.Close()

	items := []CatalogItem{
		{
			Classifier: CatalogClassifierSkill,
			Name:       "docker",
			Content:    "Container skill content",
		},
		{
			Classifier:    CatalogClassifierPrompt,
			Name:          "assistant.md",
			Content:       "Assistant prompt content",
			ParentSkillID: "docker",
			ResourcePath:  "prompts/assistant.md",
		},
	}

	if err := searcher.IndexCatalogItems(items); err != nil {
		t.Fatalf("failed to index catalog items: %v", err)
	}

	skillClassifier := CatalogClassifierSkill
	skillResults, err := searcher.SearchCatalog("Container", &skillClassifier)
	if err != nil {
		t.Fatalf("failed to search skill catalog items: %v", err)
	}
	if len(skillResults) != 1 {
		t.Fatalf("expected 1 skill result, got %d", len(skillResults))
	}
	if skillResults[0].ID != BuildSkillCatalogItemID("docker") {
		t.Fatalf("unexpected skill ID %q", skillResults[0].ID)
	}

	promptClassifier := CatalogClassifierPrompt
	promptResults, err := searcher.SearchCatalog("Assistant", &promptClassifier)
	if err != nil {
		t.Fatalf("failed to search prompt catalog items: %v", err)
	}
	if len(promptResults) != 1 {
		t.Fatalf("expected 1 prompt result, got %d", len(promptResults))
	}
	if promptResults[0].ID != BuildPromptCatalogItemID("docker", "prompts/assistant.md") {
		t.Fatalf("unexpected prompt ID %q", promptResults[0].ID)
	}
}

func TestSearcherCloseWithNilIndex(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "search-close-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	searcher, err := NewSearcher(tempDir)
	if err != nil {
		t.Fatalf("failed to create searcher: %v", err)
	}

	searcher.index = nil
	if err := searcher.Close(); err != nil {
		t.Fatalf("close returned unexpected error: %v", err)
	}
}
