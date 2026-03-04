package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
)

const searchResultSizeLimit = 100

// Searcher handles full-text search using bleve
type Searcher struct {
	indexPath string
	index     bleve.Index
}

// NewSearcher creates a new Searcher with a bleve index
func NewSearcher(skillsDir string) (*Searcher, error) {
	indexPath := filepath.Join(skillsDir, ".index")

	// Try to open existing index
	index, err := bleve.Open(indexPath)
	if err != nil {
		// Create new index if it doesn't exist
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(indexPath, mapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create search index: %w", err)
		}
	}

	return &Searcher{
		indexPath: indexPath,
		index:     index,
	}, nil
}

// IndexSkills indexes a list of skills
func (s *Searcher) IndexSkills(skills []Skill) error {
	documents := make([]catalogIndexDocument, 0, len(skills))
	for _, skill := range skills {
		doc := catalogIndexDocument{
			ID:         skill.Name,
			Classifier: CatalogClassifierSkill,
			Name:       skill.Name,
			Content:    skill.Content,
			ReadOnly:   skill.ReadOnly,
		}
		if skill.Metadata != nil {
			doc.Description = skill.Metadata.Description
			doc.License = skill.Metadata.License
			doc.Compatibility = skill.Metadata.Compatibility
		}
		documents = append(documents, doc)
	}

	return s.rebuildIndex(documents)
}

// IndexCatalogItems indexes catalog items with classifier-aware fields.
func (s *Searcher) IndexCatalogItems(items []CatalogItem) error {
	documents := make([]catalogIndexDocument, 0, len(items))
	for _, item := range items {
		if !item.Classifier.IsValid() {
			return fmt.Errorf("failed to index catalog item %q: invalid classifier %q", item.ID, item.Classifier)
		}

		id := strings.TrimSpace(item.ID)
		if id == "" {
			switch item.Classifier {
			case CatalogClassifierSkill:
				if CanonicalSkillCatalogKey(item.Name) == "" {
					return fmt.Errorf("failed to index skill catalog item with empty name")
				}
				id = BuildSkillCatalogItemID(item.Name)
			case CatalogClassifierPrompt:
				if CanonicalPromptCatalogKey(item.ParentSkillID, item.ResourcePath) == "" {
					return fmt.Errorf("failed to index prompt catalog item with empty parent skill and resource path")
				}
				id = BuildPromptCatalogItemID(item.ParentSkillID, item.ResourcePath)
			}
		}
		if id == "" {
			return fmt.Errorf("failed to index catalog item with empty ID")
		}

		documents = append(documents, catalogIndexDocument{
			ID:            id,
			Classifier:    item.Classifier,
			Name:          item.Name,
			Description:   item.Description,
			Content:       item.Content,
			ParentSkillID: item.ParentSkillID,
			ResourcePath:  item.ResourcePath,
			ReadOnly:      item.ReadOnly,
		})
	}

	return s.rebuildIndex(documents)
}

// SearchCatalog performs full-text search across catalog items with an optional classifier filter.
func (s *Searcher) SearchCatalog(query string, classifier *CatalogClassifier) ([]CatalogItem, error) {
	if s.index == nil {
		return []CatalogItem{}, nil
	}

	normalizedQuery := strings.TrimSpace(query)
	if normalizedQuery == "" {
		return []CatalogItem{}, nil
	}

	if classifier != nil && !classifier.IsValid() {
		return nil, fmt.Errorf("invalid classifier filter %q", *classifier)
	}

	searchQuery := buildContentQuery(normalizedQuery)
	if classifier != nil {
		classifierQuery := bleve.NewTermQuery(string(*classifier))
		classifierQuery.SetField("classifier")
		searchQuery = bleve.NewConjunctionQuery(searchQuery, classifierQuery)
	}

	req := bleve.NewSearchRequest(searchQuery)
	req.Size = searchResultSizeLimit
	req.Fields = []string{
		"classifier",
		"name",
		"description",
		"content",
		"parent_skill_id",
		"resource_path",
		"read_only",
	}

	searchResults, err := s.index.Search(req)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var catalogItems []CatalogItem
	for _, hit := range searchResults.Hits {
		item := CatalogItem{
			ID:   hit.ID,
			Name: hit.ID,
		}

		if classifier != nil {
			item.Classifier = *classifier
		}
		if rawClassifier, ok := fieldAsString(hit.Fields["classifier"]); ok {
			item.Classifier = CatalogClassifier(rawClassifier)
		}
		if rawName, ok := fieldAsString(hit.Fields["name"]); ok && rawName != "" {
			item.Name = rawName
		}
		if rawDescription, ok := fieldAsString(hit.Fields["description"]); ok {
			item.Description = rawDescription
		}
		if rawContent, ok := fieldAsString(hit.Fields["content"]); ok {
			item.Content = rawContent
		}
		if rawParentSkillID, ok := fieldAsString(hit.Fields["parent_skill_id"]); ok {
			item.ParentSkillID = rawParentSkillID
		}
		if rawResourcePath, ok := fieldAsString(hit.Fields["resource_path"]); ok {
			item.ResourcePath = rawResourcePath
		}
		if rawReadOnly, ok := fieldAsBool(hit.Fields["read_only"]); ok {
			item.ReadOnly = rawReadOnly
		}

		catalogItems = append(catalogItems, item)
	}

	return catalogItems, nil
}

// Search performs a full-text search and returns matching skills.
// This remains a skill-only compatibility wrapper for existing callers.
func (s *Searcher) Search(query string) ([]Skill, error) {
	skillClassifier := CatalogClassifierSkill
	results, err := s.SearchCatalog(query, &skillClassifier)
	if err != nil {
		return nil, err
	}

	var skills []Skill
	for _, result := range results {
		skillID := resolveLegacySkillID(result)
		if skillID == "" {
			continue
		}
		skills = append(skills, Skill{
			Name: skillID,
			ID:   skillID,
		})
	}

	return skills, nil
}

// Close closes the search index
func (s *Searcher) Close() error {
	if s.index != nil {
		return s.index.Close()
	}
	return nil
}

type catalogIndexDocument struct {
	ID            string
	Classifier    CatalogClassifier
	Name          string
	Description   string
	Content       string
	ParentSkillID string
	ResourcePath  string
	ReadOnly      bool
	License       string
	Compatibility string
}

func (s *Searcher) rebuildIndex(documents []catalogIndexDocument) error {
	if s.index != nil {
		if err := s.index.Close(); err != nil {
			return fmt.Errorf("failed to close existing index: %w", err)
		}
	}

	if err := os.RemoveAll(s.indexPath); err != nil {
		return fmt.Errorf("failed to remove existing index: %w", err)
	}

	index, err := bleve.New(s.indexPath, bleve.NewIndexMapping())
	if err != nil {
		return fmt.Errorf("failed to recreate index: %w", err)
	}

	for _, document := range documents {
		if !document.Classifier.IsValid() {
			return fmt.Errorf("failed to index document %q: invalid classifier %q", document.ID, document.Classifier)
		}
		if strings.TrimSpace(document.ID) == "" {
			return fmt.Errorf("failed to index document with empty ID")
		}

		if err := index.Index(document.ID, document.toMap()); err != nil {
			return fmt.Errorf("failed to index document %s: %w", document.ID, err)
		}
	}

	s.index = index
	return nil
}

func (d catalogIndexDocument) toMap() map[string]any {
	doc := map[string]any{
		"classifier": string(d.Classifier),
		"name":       d.Name,
		"content":    d.Content,
		"read_only":  d.ReadOnly,
	}

	if d.Description != "" {
		doc["description"] = d.Description
	}
	if d.ParentSkillID != "" {
		doc["parent_skill_id"] = d.ParentSkillID
	}
	if d.ResourcePath != "" {
		doc["resource_path"] = d.ResourcePath
	}
	if d.License != "" {
		doc["license"] = d.License
	}
	if d.Compatibility != "" {
		doc["compatibility"] = d.Compatibility
	}

	return doc
}

func buildContentQuery(text string) query.Query {
	contentQuery := bleve.NewMatchQuery(text)
	contentQuery.SetField("content")

	nameQuery := bleve.NewMatchQuery(text)
	nameQuery.SetField("name")

	descQuery := bleve.NewMatchQuery(text)
	descQuery.SetField("description")

	licenseQuery := bleve.NewMatchQuery(text)
	licenseQuery.SetField("license")

	compatibilityQuery := bleve.NewMatchQuery(text)
	compatibilityQuery.SetField("compatibility")

	parentSkillIDQuery := bleve.NewMatchQuery(text)
	parentSkillIDQuery.SetField("parent_skill_id")

	resourcePathQuery := bleve.NewMatchQuery(text)
	resourcePathQuery.SetField("resource_path")

	return bleve.NewDisjunctionQuery(
		contentQuery,
		nameQuery,
		descQuery,
		licenseQuery,
		compatibilityQuery,
		parentSkillIDQuery,
		resourcePathQuery,
	)
}

func resolveLegacySkillID(item CatalogItem) string {
	if strings.TrimSpace(item.ParentSkillID) != "" {
		return item.ParentSkillID
	}

	id := strings.TrimSpace(item.ID)
	if strings.HasPrefix(id, skillCatalogIDPrefix) {
		skillID := strings.TrimSpace(strings.TrimPrefix(id, skillCatalogIDPrefix))
		if skillID != "" {
			return skillID
		}
	}
	return id
}

func fieldAsString(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case []byte:
		return string(typed), true
	default:
		return "", false
	}
}

func fieldAsBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed, err := strconv.ParseBool(typed)
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}
