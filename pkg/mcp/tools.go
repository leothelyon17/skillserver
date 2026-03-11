package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mudler/skillserver/pkg/domain"
)

// ListSkillsInput is the input for list_skills tool
type ListSkillsInput struct{}

// ListSkillsOutput is the output for list_skills tool
type ListSkillsOutput struct {
	Skills []SkillInfo `json:"skills"`
}

// SkillInfo represents basic information about a skill
type SkillInfo struct {
	ID          string `json:"id"`   // Canonical skill catalog item ID (skill:<skill-id>)
	Name        string `json:"name"` // Display name
	Description string `json:"description,omitempty"`
}

// ReadSkillInput is the input for read_skill tool
type ReadSkillInput struct {
	ID string `json:"id" jsonschema:"The skill ID returned by list_skills or search_skills. Accepts bare skill IDs ('skill-name', 'repoName/skill-name') and canonical skill item IDs ('skill:<skill-id>')."`
}

// ReadSkillOutput is the output for read_skill tool
type ReadSkillOutput struct {
	Content string `json:"content"`
}

// SearchSkillsInput is the input for search_skills tool
type SearchSkillsInput struct {
	Query string `json:"query" jsonschema:"The search query"`
}

// SearchSkillsOutput is the output for search_skills tool
type SearchSkillsOutput struct {
	Results []SearchResult `json:"results"`
}

// SearchResult represents a search result
type SearchResult struct {
	ID      string `json:"id"`   // Canonical skill catalog item ID (skill:<skill-id>)
	Name    string `json:"name"` // Display name
	Content string `json:"content"`
	Snippet string `json:"snippet,omitempty"`
}

const (
	catalogListDefaultLimit                 = 50
	catalogListMaxLimit                     = 200
	catalogTaxonomyUsageDefaultPreviewLimit = 10
	catalogTaxonomyUsageMaxPreviewLimit     = 200
)

// listSkills lists all available skills
func listSkills(ctx context.Context, req *mcp.CallToolRequest, input ListSkillsInput, manager domain.SkillManager) (
	*mcp.CallToolResult,
	ListSkillsOutput,
	error,
) {
	skills, err := manager.ListSkills()
	if err != nil {
		return nil, ListSkillsOutput{}, fmt.Errorf("failed to list skills: %w", err)
	}

	skillInfos := make([]SkillInfo, len(skills))
	for i, skill := range skills {
		skillInfos[i] = SkillInfo{
			ID:   domain.BuildSkillCatalogItemID(skill.ID),
			Name: resolveSkillDisplayName(skill),
		}
		if skill.Metadata != nil {
			skillInfos[i].Description = skill.Metadata.Description
		}
	}

	return nil, ListSkillsOutput{Skills: skillInfos}, nil
}

// readSkill reads the full content of a skill
func readSkill(ctx context.Context, req *mcp.CallToolRequest, input ReadSkillInput, manager domain.SkillManager) (
	*mcp.CallToolResult,
	ReadSkillOutput,
	error,
) {
	skillID, err := normalizeSkillToolInput(input.ID, "id")
	if err != nil {
		return nil, ReadSkillOutput{}, err
	}

	skill, err := manager.ReadSkill(skillID)
	if err != nil {
		return nil, ReadSkillOutput{}, fmt.Errorf("failed to read skill: %w", err)
	}

	return nil, ReadSkillOutput{Content: skill.Content}, nil
}

// searchSkills searches for skills matching the query
func searchSkills(ctx context.Context, req *mcp.CallToolRequest, input SearchSkillsInput, manager domain.SkillManager) (
	*mcp.CallToolResult,
	SearchSkillsOutput,
	error,
) {
	skills, err := manager.SearchSkills(input.Query)
	if err != nil {
		return nil, SearchSkillsOutput{}, fmt.Errorf("failed to search skills: %w", err)
	}

	results := make([]SearchResult, len(skills))
	for i, skill := range skills {
		// Create a snippet (first 200 characters)
		snippet := skill.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}

		results[i] = SearchResult{
			ID:      domain.BuildSkillCatalogItemID(skill.ID),
			Name:    resolveSkillDisplayName(skill),
			Content: skill.Content,
			Snippet: snippet,
		}
	}

	return nil, SearchSkillsOutput{Results: results}, nil
}

// ListCatalogInput is the input for list_catalog tool.
type ListCatalogInput struct {
	Classifier        string   `json:"classifier,omitempty" jsonschema:"Optional catalog classifier filter ('skill', 'prompt', or 'rule')"`
	PrimaryDomainID   string   `json:"primary_domain_id,omitempty" jsonschema:"Optional primary domain selector"`
	SecondaryDomainID string   `json:"secondary_domain_id,omitempty" jsonschema:"Optional secondary domain selector"`
	SubdomainID       string   `json:"subdomain_id,omitempty" jsonschema:"Optional subdomain selector (matches primary or secondary subdomain)"`
	TagIDs            []string `json:"tag_ids,omitempty" jsonschema:"Optional tag selectors"`
	TagMatch          string   `json:"tag_match,omitempty" jsonschema:"Optional tag match mode ('any' or 'all')"`
	IncludeContent    *bool    `json:"include_content,omitempty" jsonschema:"Optional content inclusion flag. Defaults to false."`
	Limit             *int     `json:"limit,omitempty" jsonschema:"Optional result limit (1-200). Defaults to 50."`
	Cursor            string   `json:"cursor,omitempty" jsonschema:"Optional exclusive after-item_id cursor for deterministic pagination."`
	Unclassified      *bool    `json:"unclassified,omitempty" jsonschema:"Optional effective-state filter. When true, includes only items with has_assignment=false."`
	MissingPrimary    *bool    `json:"missing_primary_domain,omitempty" jsonschema:"Optional effective-state filter. When true, includes only items with no primary_domain."`
	MissingTags       *bool    `json:"missing_tags,omitempty" jsonschema:"Optional effective-state filter. When true, includes only items with zero tags."`
}

// ListCatalogOutput is the output for list_catalog tool.
type ListCatalogOutput struct {
	Items      []CatalogItemInfo `json:"items"`
	NextCursor *string           `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
}

// SearchCatalogInput is the input for search_catalog tool.
type SearchCatalogInput struct {
	Query             string   `json:"query" jsonschema:"The search query"`
	Classifier        string   `json:"classifier,omitempty" jsonschema:"Optional catalog classifier filter ('skill', 'prompt', or 'rule')"`
	PrimaryDomainID   string   `json:"primary_domain_id,omitempty" jsonschema:"Optional primary domain selector"`
	SecondaryDomainID string   `json:"secondary_domain_id,omitempty" jsonschema:"Optional secondary domain selector"`
	SubdomainID       string   `json:"subdomain_id,omitempty" jsonschema:"Optional subdomain selector (matches primary or secondary subdomain)"`
	TagIDs            []string `json:"tag_ids,omitempty" jsonschema:"Optional tag selectors"`
	TagMatch          string   `json:"tag_match,omitempty" jsonschema:"Optional tag match mode ('any' or 'all')"`
	IncludeContent    *bool    `json:"include_content,omitempty" jsonschema:"Optional content inclusion flag. Defaults to false."`
	Limit             *int     `json:"limit,omitempty" jsonschema:"Optional result limit (1-200). Defaults to 50."`
	Cursor            string   `json:"cursor,omitempty" jsonschema:"Optional exclusive after-item_id cursor for deterministic pagination."`
	Unclassified      *bool    `json:"unclassified,omitempty" jsonschema:"Optional effective-state filter. When true, includes only items with has_assignment=false."`
	MissingPrimary    *bool    `json:"missing_primary_domain,omitempty" jsonschema:"Optional effective-state filter. When true, includes only items with no primary_domain."`
	MissingTags       *bool    `json:"missing_tags,omitempty" jsonschema:"Optional effective-state filter. When true, includes only items with zero tags."`
}

// SearchCatalogOutput is the output for search_catalog tool.
type SearchCatalogOutput struct {
	Results    []CatalogItemInfo `json:"results"`
	NextCursor *string           `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
}

var errCatalogTaxonomyFiltersUnavailable = errors.New("catalog taxonomy filters are unavailable")

// CatalogItemInfo represents a classifier-aware catalog item in MCP responses.
type CatalogItemInfo struct {
	ID                 string                            `json:"id"`
	Classifier         domain.CatalogClassifier          `json:"classifier"`
	Name               string                            `json:"name"`
	Description        string                            `json:"description,omitempty"`
	Content            string                            `json:"content,omitempty"`
	ParentSkillID      string                            `json:"parent_skill_id,omitempty"`
	ResourcePath       string                            `json:"resource_path,omitempty"`
	PrimaryDomain      *domain.CatalogTaxonomyReference  `json:"primary_domain,omitempty"`
	PrimarySubdomain   *domain.CatalogTaxonomyReference  `json:"primary_subdomain,omitempty"`
	SecondaryDomain    *domain.CatalogTaxonomyReference  `json:"secondary_domain,omitempty"`
	SecondarySubdomain *domain.CatalogTaxonomyReference  `json:"secondary_subdomain,omitempty"`
	Tags               []domain.CatalogTaxonomyReference `json:"tags,omitempty"`
	ReadOnly           bool                              `json:"read_only"`
	HasAssignment      bool                              `json:"has_assignment"`
	IsFullyClassified  bool                              `json:"is_fully_classified"`
	MissingFields      []string                          `json:"missing_fields"`
}

// listCatalog lists unified catalog items with an optional classifier filter.
func listCatalog(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListCatalogInput,
	manager domain.SkillManager,
	catalogMetadata CatalogMetadataReader,
) (
	*mcp.CallToolResult,
	ListCatalogOutput,
	error,
) {
	classifier, err := parseCatalogClassifierFilter(input.Classifier)
	if err != nil {
		return nil, ListCatalogOutput{}, err
	}

	taxonomyFilter, err := parseCatalogFilterTaxonomyInput(catalogFilterTaxonomyInput{
		PrimaryDomainID:   input.PrimaryDomainID,
		SecondaryDomainID: input.SecondaryDomainID,
		SubdomainID:       input.SubdomainID,
		TagIDs:            input.TagIDs,
		TagMatch:          input.TagMatch,
	})
	if err != nil {
		return nil, ListCatalogOutput{}, err
	}

	request, err := newCatalogToolCollectionRequest(
		input.IncludeContent,
		input.Limit,
		input.Cursor,
		input.Unclassified,
		input.MissingPrimary,
		input.MissingTags,
	)
	if err != nil {
		return nil, ListCatalogOutput{}, err
	}
	if request.requiresMetadataRuntime() && catalogMetadata == nil {
		return nil, ListCatalogOutput{}, errCatalogTaxonomyFiltersUnavailable
	}

	items, err := loadCatalogItems(ctx, "", classifier, taxonomyFilter, manager, catalogMetadata)
	if err != nil {
		return nil, ListCatalogOutput{}, err
	}

	page := buildCatalogToolCollection(items, request)
	return nil, ListCatalogOutput{
		Items:      page.Items,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	}, nil
}

// searchCatalog searches unified catalog items with an optional classifier filter.
func searchCatalog(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SearchCatalogInput,
	manager domain.SkillManager,
	catalogMetadata CatalogMetadataReader,
) (
	*mcp.CallToolResult,
	SearchCatalogOutput,
	error,
) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, SearchCatalogOutput{}, fmt.Errorf("query is required")
	}

	classifier, err := parseCatalogClassifierFilter(input.Classifier)
	if err != nil {
		return nil, SearchCatalogOutput{}, err
	}

	taxonomyFilter, err := parseCatalogFilterTaxonomyInput(catalogFilterTaxonomyInput{
		PrimaryDomainID:   input.PrimaryDomainID,
		SecondaryDomainID: input.SecondaryDomainID,
		SubdomainID:       input.SubdomainID,
		TagIDs:            input.TagIDs,
		TagMatch:          input.TagMatch,
	})
	if err != nil {
		return nil, SearchCatalogOutput{}, err
	}

	request, err := newCatalogToolCollectionRequest(
		input.IncludeContent,
		input.Limit,
		input.Cursor,
		input.Unclassified,
		input.MissingPrimary,
		input.MissingTags,
	)
	if err != nil {
		return nil, SearchCatalogOutput{}, err
	}
	if request.requiresMetadataRuntime() && catalogMetadata == nil {
		return nil, SearchCatalogOutput{}, errCatalogTaxonomyFiltersUnavailable
	}

	items, err := loadCatalogItems(ctx, query, classifier, taxonomyFilter, manager, catalogMetadata)
	if err != nil {
		return nil, SearchCatalogOutput{}, err
	}

	page := buildCatalogToolCollection(items, request)
	return nil, SearchCatalogOutput{
		Results:    page.Items,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	}, nil
}

type catalogToolCollectionRequest struct {
	IncludeContent       bool
	Limit                int
	Cursor               string
	Unclassified         bool
	MissingPrimaryDomain bool
	MissingTags          bool
}

type catalogToolCollectionResult struct {
	Items      []CatalogItemInfo
	NextCursor *string
	HasMore    bool
}

func newCatalogToolCollectionRequest(
	includeContent *bool,
	limit *int,
	cursor string,
	unclassified *bool,
	missingPrimaryDomain *bool,
	missingTags *bool,
) (catalogToolCollectionRequest, error) {
	request := catalogToolCollectionRequest{
		Limit:  catalogListDefaultLimit,
		Cursor: strings.TrimSpace(cursor),
	}
	if includeContent != nil {
		request.IncludeContent = *includeContent
	}
	if unclassified != nil {
		request.Unclassified = *unclassified
	}
	if missingPrimaryDomain != nil {
		request.MissingPrimaryDomain = *missingPrimaryDomain
	}
	if missingTags != nil {
		request.MissingTags = *missingTags
	}
	if limit == nil {
		return request, nil
	}
	if *limit <= 0 || *limit > catalogListMaxLimit {
		return catalogToolCollectionRequest{}, fmt.Errorf(
			"limit must be an integer between 1 and %d",
			catalogListMaxLimit,
		)
	}
	request.Limit = *limit
	return request, nil
}

func (r catalogToolCollectionRequest) matchesClassificationState(item domain.CatalogItem) bool {
	if r.Unclassified && item.HasAssignment {
		return false
	}
	if r.MissingPrimaryDomain && item.PrimaryDomain != nil {
		return false
	}
	if r.MissingTags && len(item.Tags) > 0 {
		return false
	}
	return true
}

func (r catalogToolCollectionRequest) requiresMetadataRuntime() bool {
	return r.Unclassified || r.MissingPrimaryDomain || r.MissingTags
}

func buildCatalogToolCollection(
	items []domain.CatalogItem,
	request catalogToolCollectionRequest,
) catalogToolCollectionResult {
	normalizedItems := make([]domain.CatalogItem, 0, len(items))
	for _, item := range items {
		normalizedItem := normalizeCatalogItemForToolResponse(item)
		if !request.matchesClassificationState(normalizedItem) {
			continue
		}
		normalizedItems = append(normalizedItems, normalizedItem)
	}

	sort.Slice(normalizedItems, func(i int, j int) bool {
		return normalizedItems[i].ID < normalizedItems[j].ID
	})

	pageItems, nextCursor, hasMore := paginateCatalogToolItems(
		normalizedItems,
		request.Cursor,
		request.Limit,
	)

	return catalogToolCollectionResult{
		Items:      buildCatalogItemInfos(pageItems, request.IncludeContent),
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}

func paginateCatalogToolItems(
	items []domain.CatalogItem,
	cursor string,
	limit int,
) ([]domain.CatalogItem, *string, bool) {
	if len(items) == 0 {
		return []domain.CatalogItem{}, nil, false
	}

	start := 0
	normalizedCursor := strings.TrimSpace(cursor)
	if normalizedCursor != "" {
		for start < len(items) && strings.Compare(items[start].ID, normalizedCursor) <= 0 {
			start++
		}
	}
	if start >= len(items) {
		return []domain.CatalogItem{}, nil, false
	}

	pageItems := items[start:]
	if limit <= 0 || len(pageItems) <= limit {
		return append([]domain.CatalogItem{}, pageItems...), nil, false
	}

	pageItems = append([]domain.CatalogItem{}, pageItems[:limit]...)
	nextCursor := pageItems[len(pageItems)-1].ID
	return pageItems, &nextCursor, true
}

type catalogFilterTaxonomyInput struct {
	PrimaryDomainID   string
	SecondaryDomainID string
	SubdomainID       string
	TagIDs            []string
	TagMatch          string
}

func parseCatalogFilterTaxonomyInput(input catalogFilterTaxonomyInput) (domain.CatalogEffectiveListFilter, error) {
	filter := domain.CatalogEffectiveListFilter{
		PrimaryDomainID:   strings.TrimSpace(input.PrimaryDomainID),
		SecondaryDomainID: strings.TrimSpace(input.SecondaryDomainID),
		SubdomainID:       strings.TrimSpace(input.SubdomainID),
		TagIDs:            normalizeStringList(input.TagIDs),
	}

	rawTagMatch := strings.TrimSpace(input.TagMatch)
	if rawTagMatch == "" {
		return filter, nil
	}

	tagMatch := domain.CatalogTagMatchMode(strings.ToLower(rawTagMatch))
	if !tagMatch.IsValid() {
		return domain.CatalogEffectiveListFilter{}, fmt.Errorf("tag_match must be one of: any, all")
	}

	filter.TagMatch = tagMatch
	return filter, nil
}

func hasCatalogTaxonomyFilterConstraints(filter domain.CatalogEffectiveListFilter) bool {
	return strings.TrimSpace(filter.PrimaryDomainID) != "" ||
		strings.TrimSpace(filter.SecondaryDomainID) != "" ||
		strings.TrimSpace(filter.SubdomainID) != "" ||
		len(filter.TagIDs) > 0 ||
		strings.TrimSpace(string(filter.TagMatch)) != ""
}

func loadCatalogItems(
	ctx context.Context,
	query string,
	classifier *domain.CatalogClassifier,
	taxonomyFilter domain.CatalogEffectiveListFilter,
	manager domain.SkillManager,
	catalogMetadata CatalogMetadataReader,
) ([]domain.CatalogItem, error) {
	normalizedQuery := strings.TrimSpace(query)
	if catalogMetadata != nil {
		taxonomyFilter.Classifier = classifier
		items, err := catalogMetadata.List(ctx, taxonomyFilter)
		if err != nil {
			return nil, fmt.Errorf("list effective catalog items: %w", err)
		}
		if normalizedQuery == "" {
			return items, nil
		}
		return filterCatalogItemsByQuery(items, normalizedQuery), nil
	}

	if hasCatalogTaxonomyFilterConstraints(taxonomyFilter) {
		return nil, errCatalogTaxonomyFiltersUnavailable
	}

	if manager == nil {
		return nil, fmt.Errorf("skill manager is required")
	}

	if normalizedQuery == "" {
		items, err := manager.ListCatalogItems()
		if err != nil {
			return nil, fmt.Errorf("failed to list catalog items: %w", err)
		}
		if classifier == nil {
			return items, nil
		}

		filtered := make([]domain.CatalogItem, 0, len(items))
		for _, item := range items {
			if item.Classifier == *classifier {
				filtered = append(filtered, item)
			}
		}
		return filtered, nil
	}

	items, err := manager.SearchCatalogItems(normalizedQuery, classifier)
	if err != nil {
		return nil, fmt.Errorf("failed to search catalog items: %w", err)
	}
	return items, nil
}

func parseCatalogClassifierFilter(raw string) (*domain.CatalogClassifier, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return nil, nil
	}

	classifier, err := domain.ParseCatalogClassifier(normalized)
	if err != nil {
		return nil, err
	}

	return &classifier, nil
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	sort.Strings(normalized)
	return normalized
}

func filterCatalogItemsByQuery(items []domain.CatalogItem, query string) []domain.CatalogItem {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	if normalizedQuery == "" {
		return items
	}

	matches := make([]domain.CatalogItem, 0, len(items))
	for _, item := range items {
		if catalogItemMatchesQuery(item, normalizedQuery) {
			matches = append(matches, item)
		}
	}

	return matches
}

func catalogItemMatchesQuery(item domain.CatalogItem, normalizedQuery string) bool {
	if normalizedQuery == "" {
		return true
	}

	parts := []string{
		item.Name,
		item.Description,
		item.Content,
		item.ParentSkillID,
		item.ResourcePath,
	}
	parts = append(parts, item.Labels...)
	if item.PrimaryDomain != nil {
		parts = append(parts, item.PrimaryDomain.ID, item.PrimaryDomain.Key, item.PrimaryDomain.Name)
	}
	if item.PrimarySubdomain != nil {
		parts = append(parts, item.PrimarySubdomain.ID, item.PrimarySubdomain.Key, item.PrimarySubdomain.Name)
	}
	if item.SecondaryDomain != nil {
		parts = append(parts, item.SecondaryDomain.ID, item.SecondaryDomain.Key, item.SecondaryDomain.Name)
	}
	if item.SecondarySubdomain != nil {
		parts = append(parts, item.SecondarySubdomain.ID, item.SecondarySubdomain.Key, item.SecondarySubdomain.Name)
	}
	for _, tag := range item.Tags {
		parts = append(parts, tag.ID, tag.Key, tag.Name)
	}

	if len(item.CustomMetadata) > 0 {
		customMetadataJSON, err := json.Marshal(item.CustomMetadata)
		if err == nil {
			parts = append(parts, string(customMetadataJSON))
		}
	}

	haystack := strings.ToLower(strings.Join(parts, " "))
	return strings.Contains(haystack, normalizedQuery)
}

func buildCatalogItemInfos(items []domain.CatalogItem, includeContent bool) []CatalogItemInfo {
	results := make([]CatalogItemInfo, len(items))
	for i, item := range items {
		classificationState := deriveCatalogItemClassificationState(item)
		content := ""
		if includeContent {
			content = item.Content
		}

		results[i] = CatalogItemInfo{
			ID:                 item.ID,
			Classifier:         item.Classifier,
			Name:               item.Name,
			Description:        item.Description,
			Content:            content,
			ParentSkillID:      item.ParentSkillID,
			ResourcePath:       item.ResourcePath,
			PrimaryDomain:      copyCatalogItemTaxonomyReference(item.PrimaryDomain),
			PrimarySubdomain:   copyCatalogItemTaxonomyReference(item.PrimarySubdomain),
			SecondaryDomain:    copyCatalogItemTaxonomyReference(item.SecondaryDomain),
			SecondarySubdomain: copyCatalogItemTaxonomyReference(item.SecondarySubdomain),
			Tags:               copyCatalogItemTaxonomyReferences(item.Tags),
			ReadOnly:           item.ReadOnly,
			HasAssignment:      classificationState.HasAssignment,
			IsFullyClassified:  classificationState.IsFullyClassified,
			MissingFields:      append([]string{}, classificationState.MissingFields...),
		}
	}

	return results
}

func normalizeCatalogItemForToolResponse(item domain.CatalogItem) domain.CatalogItem {
	classificationState := deriveCatalogItemClassificationState(item)
	item.HasAssignment = classificationState.HasAssignment
	item.IsFullyClassified = classificationState.IsFullyClassified
	item.MissingFields = append([]string{}, classificationState.MissingFields...)
	return item
}

func deriveCatalogItemClassificationState(item domain.CatalogItem) domain.CatalogClassificationState {
	return domain.DeriveCatalogClassificationState(
		item.PrimaryDomain,
		item.PrimarySubdomain,
		item.SecondaryDomain,
		item.SecondarySubdomain,
		item.Tags,
	)
}

func copyCatalogItemTaxonomyReference(
	reference *domain.CatalogTaxonomyReference,
) *domain.CatalogTaxonomyReference {
	if reference == nil {
		return nil
	}
	copied := *reference
	return &copied
}

func copyCatalogItemTaxonomyReferences(
	references []domain.CatalogTaxonomyReference,
) []domain.CatalogTaxonomyReference {
	if len(references) == 0 {
		return []domain.CatalogTaxonomyReference{}
	}
	copied := make([]domain.CatalogTaxonomyReference, len(references))
	copy(copied, references)
	return copied
}

// ListTaxonomyDomainsInput is the input for list_taxonomy_domains tool.
type ListTaxonomyDomainsInput struct {
	DomainID  string   `json:"domain_id,omitempty" jsonschema:"Optional domain identifier filter"`
	DomainIDs []string `json:"domain_ids,omitempty" jsonschema:"Optional domain identifier filters"`
	Key       string   `json:"key,omitempty" jsonschema:"Optional taxonomy domain key filter"`
	Keys      []string `json:"keys,omitempty" jsonschema:"Optional taxonomy domain key filters"`
	Active    *bool    `json:"active,omitempty" jsonschema:"Optional active status filter"`
}

// ListTaxonomyDomainsOutput is the output for list_taxonomy_domains tool.
type ListTaxonomyDomainsOutput struct {
	Domains []domain.CatalogTaxonomyDomain `json:"domains"`
}

// ListTaxonomySubdomainsInput is the input for list_taxonomy_subdomains tool.
type ListTaxonomySubdomainsInput struct {
	SubdomainID  string   `json:"subdomain_id,omitempty" jsonschema:"Optional subdomain identifier filter"`
	SubdomainIDs []string `json:"subdomain_ids,omitempty" jsonschema:"Optional subdomain identifier filters"`
	DomainID     string   `json:"domain_id,omitempty" jsonschema:"Optional parent domain identifier filter"`
	DomainIDs    []string `json:"domain_ids,omitempty" jsonschema:"Optional parent domain identifier filters"`
	Key          string   `json:"key,omitempty" jsonschema:"Optional taxonomy subdomain key filter"`
	Keys         []string `json:"keys,omitempty" jsonschema:"Optional taxonomy subdomain key filters"`
	Active       *bool    `json:"active,omitempty" jsonschema:"Optional active status filter"`
}

// ListTaxonomySubdomainsOutput is the output for list_taxonomy_subdomains tool.
type ListTaxonomySubdomainsOutput struct {
	Subdomains []domain.CatalogTaxonomySubdomain `json:"subdomains"`
}

// ListTaxonomyTagsInput is the input for list_taxonomy_tags tool.
type ListTaxonomyTagsInput struct {
	TagID  string   `json:"tag_id,omitempty" jsonschema:"Optional tag identifier filter"`
	TagIDs []string `json:"tag_ids,omitempty" jsonschema:"Optional tag identifier filters"`
	Key    string   `json:"key,omitempty" jsonschema:"Optional taxonomy tag key filter"`
	Keys   []string `json:"keys,omitempty" jsonschema:"Optional taxonomy tag key filters"`
	Active *bool    `json:"active,omitempty" jsonschema:"Optional active status filter"`
}

// ListTaxonomyTagsOutput is the output for list_taxonomy_tags tool.
type ListTaxonomyTagsOutput struct {
	Tags []domain.CatalogTaxonomyTag `json:"tags"`
}

// GetCatalogItemTaxonomyInput is the input for get_catalog_item_taxonomy tool.
type GetCatalogItemTaxonomyInput struct {
	ItemID string `json:"item_id" jsonschema:"Catalog item identifier. Accepts bare skill IDs only for skill items; prompt/rule inputs must be canonical."`
}

// GetCatalogItemTaxonomyOutput is the output for get_catalog_item_taxonomy tool.
type GetCatalogItemTaxonomyOutput = domain.CatalogItemTaxonomyAssignment

// GetCatalogItemRelationshipsInput is the input for get_catalog_item_relationships tool.
type GetCatalogItemRelationshipsInput struct {
	ItemID string `json:"item_id" jsonschema:"Catalog item identifier. Accepts bare skill IDs only for skill items; prompt/rule inputs must be canonical."`
}

// GetCatalogItemRelationshipsOutput is the output for get_catalog_item_relationships tool.
type GetCatalogItemRelationshipsOutput = domain.CatalogRelationshipView

func listTaxonomyDomains(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListTaxonomyDomainsInput,
	registry CatalogTaxonomyRegistryReader,
) (
	*mcp.CallToolResult,
	ListTaxonomyDomainsOutput,
	error,
) {
	if registry == nil {
		return nil, ListTaxonomyDomainsOutput{}, fmt.Errorf("catalog taxonomy registry API is unavailable")
	}

	domains, err := registry.ListDomains(ctx, domain.CatalogTaxonomyDomainListFilter{
		DomainID:  strings.TrimSpace(input.DomainID),
		DomainIDs: normalizeStringList(input.DomainIDs),
		Key:       strings.TrimSpace(input.Key),
		Keys:      normalizeStringList(input.Keys),
		Active:    input.Active,
	})
	if err != nil {
		return nil, ListTaxonomyDomainsOutput{}, fmt.Errorf("list taxonomy domains: %w", err)
	}

	return nil, ListTaxonomyDomainsOutput{Domains: domains}, nil
}

func listTaxonomySubdomains(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListTaxonomySubdomainsInput,
	registry CatalogTaxonomyRegistryReader,
) (
	*mcp.CallToolResult,
	ListTaxonomySubdomainsOutput,
	error,
) {
	if registry == nil {
		return nil, ListTaxonomySubdomainsOutput{}, fmt.Errorf("catalog taxonomy registry API is unavailable")
	}

	subdomains, err := registry.ListSubdomains(ctx, domain.CatalogTaxonomySubdomainListFilter{
		SubdomainID:  strings.TrimSpace(input.SubdomainID),
		SubdomainIDs: normalizeStringList(input.SubdomainIDs),
		DomainID:     strings.TrimSpace(input.DomainID),
		DomainIDs:    normalizeStringList(input.DomainIDs),
		Key:          strings.TrimSpace(input.Key),
		Keys:         normalizeStringList(input.Keys),
		Active:       input.Active,
	})
	if err != nil {
		return nil, ListTaxonomySubdomainsOutput{}, fmt.Errorf("list taxonomy subdomains: %w", err)
	}

	return nil, ListTaxonomySubdomainsOutput{Subdomains: subdomains}, nil
}

func listTaxonomyTags(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListTaxonomyTagsInput,
	registry CatalogTaxonomyRegistryReader,
) (
	*mcp.CallToolResult,
	ListTaxonomyTagsOutput,
	error,
) {
	if registry == nil {
		return nil, ListTaxonomyTagsOutput{}, fmt.Errorf("catalog taxonomy registry API is unavailable")
	}

	tags, err := registry.ListTags(ctx, domain.CatalogTaxonomyTagListFilter{
		TagID:  strings.TrimSpace(input.TagID),
		TagIDs: normalizeStringList(input.TagIDs),
		Key:    strings.TrimSpace(input.Key),
		Keys:   normalizeStringList(input.Keys),
		Active: input.Active,
	})
	if err != nil {
		return nil, ListTaxonomyTagsOutput{}, fmt.Errorf("list taxonomy tags: %w", err)
	}

	return nil, ListTaxonomyTagsOutput{Tags: tags}, nil
}

func getCatalogItemTaxonomy(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetCatalogItemTaxonomyInput,
	assignment CatalogTaxonomyAssignmentReader,
) (
	*mcp.CallToolResult,
	GetCatalogItemTaxonomyOutput,
	error,
) {
	if assignment == nil {
		return nil, GetCatalogItemTaxonomyOutput{}, fmt.Errorf("catalog taxonomy assignment API is unavailable")
	}

	itemID := strings.TrimSpace(input.ItemID)
	if itemID == "" {
		return nil, GetCatalogItemTaxonomyOutput{}, fmt.Errorf("item_id is required")
	}

	value, err := assignment.Get(ctx, itemID)
	if err != nil {
		return nil, GetCatalogItemTaxonomyOutput{}, fmt.Errorf("get catalog item taxonomy for %q: %w", itemID, err)
	}

	return nil, value, nil
}

func getCatalogItemRelationships(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetCatalogItemRelationshipsInput,
	relationships CatalogRelationshipReader,
) (
	*mcp.CallToolResult,
	GetCatalogItemRelationshipsOutput,
	error,
) {
	if relationships == nil {
		return nil, GetCatalogItemRelationshipsOutput{}, fmt.Errorf("catalog relationship API is unavailable")
	}

	itemID := strings.TrimSpace(input.ItemID)
	if itemID == "" {
		return nil, GetCatalogItemRelationshipsOutput{}, fmt.Errorf("item_id is required")
	}

	value, err := relationships.Get(ctx, itemID)
	if err != nil {
		return nil, GetCatalogItemRelationshipsOutput{}, fmt.Errorf(
			"get catalog item relationships for %q: %w",
			itemID,
			err,
		)
	}

	return nil, value, nil
}

// CreateTaxonomyDomainInput is the input for create_taxonomy_domain.
type CreateTaxonomyDomainInput struct {
	DomainID    string `json:"domain_id" jsonschema:"Catalog taxonomy domain identifier"`
	Key         string `json:"key" jsonschema:"Catalog taxonomy domain key"`
	Name        string `json:"name" jsonschema:"Catalog taxonomy domain display name"`
	Description string `json:"description,omitempty" jsonschema:"Optional catalog taxonomy domain description"`
	Active      *bool  `json:"active,omitempty" jsonschema:"Optional active flag override (defaults true)"`
}

// CreateTaxonomyDomainOutput is the output for create_taxonomy_domain.
type CreateTaxonomyDomainOutput = domain.CatalogTaxonomyDomain

// UpdateTaxonomyDomainInput is the input for update_taxonomy_domain.
type UpdateTaxonomyDomainInput struct {
	DomainID    string  `json:"domain_id" jsonschema:"Catalog taxonomy domain identifier"`
	Key         *string `json:"key,omitempty" jsonschema:"Optional updated domain key"`
	Name        *string `json:"name,omitempty" jsonschema:"Optional updated domain display name"`
	Description *string `json:"description,omitempty" jsonschema:"Optional updated domain description"`
	Active      *bool   `json:"active,omitempty" jsonschema:"Optional updated active flag"`
}

// UpdateTaxonomyDomainOutput is the output for update_taxonomy_domain.
type UpdateTaxonomyDomainOutput = domain.CatalogTaxonomyDomain

// DeleteTaxonomyDomainInput is the input for delete_taxonomy_domain.
type DeleteTaxonomyDomainInput struct {
	DomainID string `json:"domain_id" jsonschema:"Catalog taxonomy domain identifier"`
}

// DeleteTaxonomyDomainOutput is the output for delete_taxonomy_domain.
type DeleteTaxonomyDomainOutput struct {
	Deleted bool `json:"deleted"`
}

// CreateTaxonomySubdomainInput is the input for create_taxonomy_subdomain.
type CreateTaxonomySubdomainInput struct {
	SubdomainID string `json:"subdomain_id" jsonschema:"Catalog taxonomy subdomain identifier"`
	DomainID    string `json:"domain_id" jsonschema:"Parent catalog taxonomy domain identifier"`
	Key         string `json:"key" jsonschema:"Catalog taxonomy subdomain key"`
	Name        string `json:"name" jsonschema:"Catalog taxonomy subdomain display name"`
	Description string `json:"description,omitempty" jsonschema:"Optional catalog taxonomy subdomain description"`
	Active      *bool  `json:"active,omitempty" jsonschema:"Optional active flag override (defaults true)"`
}

// CreateTaxonomySubdomainOutput is the output for create_taxonomy_subdomain.
type CreateTaxonomySubdomainOutput = domain.CatalogTaxonomySubdomain

// UpdateTaxonomySubdomainInput is the input for update_taxonomy_subdomain.
type UpdateTaxonomySubdomainInput struct {
	SubdomainID string  `json:"subdomain_id" jsonschema:"Catalog taxonomy subdomain identifier"`
	DomainID    *string `json:"domain_id,omitempty" jsonschema:"Optional updated parent domain identifier"`
	Key         *string `json:"key,omitempty" jsonschema:"Optional updated subdomain key"`
	Name        *string `json:"name,omitempty" jsonschema:"Optional updated subdomain display name"`
	Description *string `json:"description,omitempty" jsonschema:"Optional updated subdomain description"`
	Active      *bool   `json:"active,omitempty" jsonschema:"Optional updated active flag"`
}

// UpdateTaxonomySubdomainOutput is the output for update_taxonomy_subdomain.
type UpdateTaxonomySubdomainOutput = domain.CatalogTaxonomySubdomain

// DeleteTaxonomySubdomainInput is the input for delete_taxonomy_subdomain.
type DeleteTaxonomySubdomainInput struct {
	SubdomainID string `json:"subdomain_id" jsonschema:"Catalog taxonomy subdomain identifier"`
}

// DeleteTaxonomySubdomainOutput is the output for delete_taxonomy_subdomain.
type DeleteTaxonomySubdomainOutput struct {
	Deleted bool `json:"deleted"`
}

// CreateTaxonomyTagInput is the input for create_taxonomy_tag.
type CreateTaxonomyTagInput struct {
	TagID       string `json:"tag_id" jsonschema:"Catalog taxonomy tag identifier"`
	Key         string `json:"key" jsonschema:"Catalog taxonomy tag key"`
	Name        string `json:"name" jsonschema:"Catalog taxonomy tag display name"`
	Description string `json:"description,omitempty" jsonschema:"Optional catalog taxonomy tag description"`
	Color       string `json:"color,omitempty" jsonschema:"Optional catalog taxonomy tag color"`
	Active      *bool  `json:"active,omitempty" jsonschema:"Optional active flag override (defaults true)"`
}

// CreateTaxonomyTagOutput is the output for create_taxonomy_tag.
type CreateTaxonomyTagOutput = domain.CatalogTaxonomyTag

// UpdateTaxonomyTagInput is the input for update_taxonomy_tag.
type UpdateTaxonomyTagInput struct {
	TagID       string  `json:"tag_id" jsonschema:"Catalog taxonomy tag identifier"`
	Key         *string `json:"key,omitempty" jsonschema:"Optional updated tag key"`
	Name        *string `json:"name,omitempty" jsonschema:"Optional updated tag display name"`
	Description *string `json:"description,omitempty" jsonschema:"Optional updated tag description"`
	Color       *string `json:"color,omitempty" jsonschema:"Optional updated tag color"`
	Active      *bool   `json:"active,omitempty" jsonschema:"Optional updated active flag"`
}

// UpdateTaxonomyTagOutput is the output for update_taxonomy_tag.
type UpdateTaxonomyTagOutput = domain.CatalogTaxonomyTag

// DeleteTaxonomyTagInput is the input for delete_taxonomy_tag.
type DeleteTaxonomyTagInput struct {
	TagID string `json:"tag_id" jsonschema:"Catalog taxonomy tag identifier"`
}

// DeleteTaxonomyTagOutput is the output for delete_taxonomy_tag.
type DeleteTaxonomyTagOutput struct {
	Deleted bool `json:"deleted"`
}

// PatchCatalogItemTaxonomyInput is the input for patch_catalog_item_taxonomy.
type PatchCatalogItemTaxonomyInput struct {
	ItemID               string    `json:"item_id" jsonschema:"Catalog item identifier"`
	PrimaryDomainID      *string   `json:"primary_domain_id,omitempty" jsonschema:"Optional primary domain identifier"`
	PrimarySubdomainID   *string   `json:"primary_subdomain_id,omitempty" jsonschema:"Optional primary subdomain identifier"`
	SecondaryDomainID    *string   `json:"secondary_domain_id,omitempty" jsonschema:"Optional secondary domain identifier"`
	SecondarySubdomainID *string   `json:"secondary_subdomain_id,omitempty" jsonschema:"Optional secondary subdomain identifier"`
	TagIDs               *[]string `json:"tag_ids,omitempty" jsonschema:"Optional full replacement tag id list"`
	AddTagIDs            *[]string `json:"add_tag_ids,omitempty" jsonschema:"Optional additive tag id list"`
	RemoveTagIDs         *[]string `json:"remove_tag_ids,omitempty" jsonschema:"Optional tag ids to remove"`
	ClearTags            *bool     `json:"clear_tags,omitempty" jsonschema:"Optional flag to clear all existing tags"`
	UpdatedBy            *string   `json:"updated_by,omitempty" jsonschema:"Optional updater identity"`
}

// PatchCatalogItemTaxonomyOutput is the output for patch_catalog_item_taxonomy.
type PatchCatalogItemTaxonomyOutput = domain.CatalogItemTaxonomyAssignment

// PatchCatalogItemsTaxonomyInput is the input for patch_catalog_items_taxonomy.
type PatchCatalogItemsTaxonomyInput struct {
	Items  []PatchCatalogItemsTaxonomyItemInput `json:"items" jsonschema:"Batch taxonomy mutation items"`
	DryRun bool                                 `json:"dry_run,omitempty" jsonschema:"When true, returns only planned statuses without writing changes."`
}

// PatchCatalogItemsTaxonomyItemInput is one batch taxonomy mutation item.
type PatchCatalogItemsTaxonomyItemInput struct {
	ItemID               string    `json:"item_id" jsonschema:"Catalog item identifier"`
	PrimaryDomainID      *string   `json:"primary_domain_id,omitempty" jsonschema:"Optional primary domain identifier"`
	PrimarySubdomainID   *string   `json:"primary_subdomain_id,omitempty" jsonschema:"Optional primary subdomain identifier"`
	SecondaryDomainID    *string   `json:"secondary_domain_id,omitempty" jsonschema:"Optional secondary domain identifier"`
	SecondarySubdomainID *string   `json:"secondary_subdomain_id,omitempty" jsonschema:"Optional secondary subdomain identifier"`
	TagIDs               *[]string `json:"tag_ids,omitempty" jsonschema:"Optional full replacement tag id list"`
	AddTagIDs            *[]string `json:"add_tag_ids,omitempty" jsonschema:"Optional additive tag id list"`
	RemoveTagIDs         *[]string `json:"remove_tag_ids,omitempty" jsonschema:"Optional tag ids to remove"`
	ClearTags            *bool     `json:"clear_tags,omitempty" jsonschema:"Optional flag to clear all existing tags"`
	UpdatedBy            *string   `json:"updated_by,omitempty" jsonschema:"Optional updater identity"`
}

// PatchCatalogItemsTaxonomyOutput is the output for patch_catalog_items_taxonomy.
type PatchCatalogItemsTaxonomyOutput = domain.CatalogItemTaxonomyBatchPatchResult

// GetTaxonomyDomainUsageInput is the input for get_taxonomy_domain_usage.
type GetTaxonomyDomainUsageInput struct {
	DomainID     string `json:"domain_id" jsonschema:"Catalog taxonomy domain identifier"`
	PreviewLimit *int   `json:"preview_limit,omitempty" jsonschema:"Optional impacted-item preview limit (0-200). Defaults to 10."`
}

// GetTaxonomyDomainUsageOutput is the output for get_taxonomy_domain_usage.
type GetTaxonomyDomainUsageOutput = domain.CatalogTaxonomyUsageSummary

// GetTaxonomySubdomainUsageInput is the input for get_taxonomy_subdomain_usage.
type GetTaxonomySubdomainUsageInput struct {
	SubdomainID  string `json:"subdomain_id" jsonschema:"Catalog taxonomy subdomain identifier"`
	PreviewLimit *int   `json:"preview_limit,omitempty" jsonschema:"Optional impacted-item preview limit (0-200). Defaults to 10."`
}

// GetTaxonomySubdomainUsageOutput is the output for get_taxonomy_subdomain_usage.
type GetTaxonomySubdomainUsageOutput = domain.CatalogTaxonomyUsageSummary

// GetTaxonomyTagUsageInput is the input for get_taxonomy_tag_usage.
type GetTaxonomyTagUsageInput struct {
	TagID        string `json:"tag_id" jsonschema:"Catalog taxonomy tag identifier"`
	PreviewLimit *int   `json:"preview_limit,omitempty" jsonschema:"Optional impacted-item preview limit (0-200). Defaults to 10."`
}

// GetTaxonomyTagUsageOutput is the output for get_taxonomy_tag_usage.
type GetTaxonomyTagUsageOutput = domain.CatalogTaxonomyUsageSummary

func createTaxonomyDomain(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateTaxonomyDomainInput,
	registry CatalogTaxonomyRegistryWriter,
) (
	*mcp.CallToolResult,
	CreateTaxonomyDomainOutput,
	error,
) {
	if registry == nil {
		return nil, CreateTaxonomyDomainOutput{}, fmt.Errorf("catalog taxonomy registry API is unavailable")
	}

	domainID := strings.TrimSpace(input.DomainID)
	value, err := registry.CreateDomain(ctx, domain.CatalogTaxonomyDomainCreateInput{
		DomainID:    domainID,
		Key:         input.Key,
		Name:        input.Name,
		Description: input.Description,
		Active:      input.Active,
	})
	if err != nil {
		return nil, CreateTaxonomyDomainOutput{}, fmt.Errorf("create taxonomy domain %q: %w", domainID, err)
	}

	return nil, value, nil
}

func updateTaxonomyDomain(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdateTaxonomyDomainInput,
	registry CatalogTaxonomyRegistryWriter,
) (
	*mcp.CallToolResult,
	UpdateTaxonomyDomainOutput,
	error,
) {
	if registry == nil {
		return nil, UpdateTaxonomyDomainOutput{}, fmt.Errorf("catalog taxonomy registry API is unavailable")
	}

	domainID := strings.TrimSpace(input.DomainID)
	if domainID == "" {
		return nil, UpdateTaxonomyDomainOutput{}, fmt.Errorf("domain_id is required")
	}
	if !hasTaxonomyDomainUpdateValues(input) {
		return nil, UpdateTaxonomyDomainOutput{}, fmt.Errorf(
			"at least one of key, name, description, or active is required",
		)
	}

	value, err := registry.UpdateDomain(ctx, domain.CatalogTaxonomyDomainUpdateInput{
		DomainID:    domainID,
		Key:         input.Key,
		Name:        input.Name,
		Description: input.Description,
		Active:      input.Active,
	})
	if err != nil {
		return nil, UpdateTaxonomyDomainOutput{}, fmt.Errorf("update taxonomy domain %q: %w", domainID, err)
	}

	return nil, value, nil
}

func deleteTaxonomyDomain(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DeleteTaxonomyDomainInput,
	registry CatalogTaxonomyRegistryWriter,
) (
	*mcp.CallToolResult,
	DeleteTaxonomyDomainOutput,
	error,
) {
	if registry == nil {
		return nil, DeleteTaxonomyDomainOutput{}, fmt.Errorf("catalog taxonomy registry API is unavailable")
	}

	domainID := strings.TrimSpace(input.DomainID)
	if domainID == "" {
		return nil, DeleteTaxonomyDomainOutput{}, fmt.Errorf("domain_id is required")
	}
	if err := registry.DeleteDomain(ctx, domainID); err != nil {
		return nil, DeleteTaxonomyDomainOutput{}, fmt.Errorf("delete taxonomy domain %q: %w", domainID, err)
	}

	return nil, DeleteTaxonomyDomainOutput{Deleted: true}, nil
}

func createTaxonomySubdomain(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateTaxonomySubdomainInput,
	registry CatalogTaxonomyRegistryWriter,
) (
	*mcp.CallToolResult,
	CreateTaxonomySubdomainOutput,
	error,
) {
	if registry == nil {
		return nil, CreateTaxonomySubdomainOutput{}, fmt.Errorf("catalog taxonomy registry API is unavailable")
	}

	subdomainID := strings.TrimSpace(input.SubdomainID)
	value, err := registry.CreateSubdomain(ctx, domain.CatalogTaxonomySubdomainCreateInput{
		SubdomainID: subdomainID,
		DomainID:    input.DomainID,
		Key:         input.Key,
		Name:        input.Name,
		Description: input.Description,
		Active:      input.Active,
	})
	if err != nil {
		return nil, CreateTaxonomySubdomainOutput{}, fmt.Errorf(
			"create taxonomy subdomain %q: %w",
			subdomainID,
			err,
		)
	}

	return nil, value, nil
}

func updateTaxonomySubdomain(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdateTaxonomySubdomainInput,
	registry CatalogTaxonomyRegistryWriter,
) (
	*mcp.CallToolResult,
	UpdateTaxonomySubdomainOutput,
	error,
) {
	if registry == nil {
		return nil, UpdateTaxonomySubdomainOutput{}, fmt.Errorf("catalog taxonomy registry API is unavailable")
	}

	subdomainID := strings.TrimSpace(input.SubdomainID)
	if subdomainID == "" {
		return nil, UpdateTaxonomySubdomainOutput{}, fmt.Errorf("subdomain_id is required")
	}
	if !hasTaxonomySubdomainUpdateValues(input) {
		return nil, UpdateTaxonomySubdomainOutput{}, fmt.Errorf(
			"at least one of domain_id, key, name, description, or active is required",
		)
	}

	value, err := registry.UpdateSubdomain(ctx, domain.CatalogTaxonomySubdomainUpdateInput{
		SubdomainID: subdomainID,
		DomainID:    input.DomainID,
		Key:         input.Key,
		Name:        input.Name,
		Description: input.Description,
		Active:      input.Active,
	})
	if err != nil {
		return nil, UpdateTaxonomySubdomainOutput{}, fmt.Errorf(
			"update taxonomy subdomain %q: %w",
			subdomainID,
			err,
		)
	}

	return nil, value, nil
}

func deleteTaxonomySubdomain(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DeleteTaxonomySubdomainInput,
	registry CatalogTaxonomyRegistryWriter,
) (
	*mcp.CallToolResult,
	DeleteTaxonomySubdomainOutput,
	error,
) {
	if registry == nil {
		return nil, DeleteTaxonomySubdomainOutput{}, fmt.Errorf("catalog taxonomy registry API is unavailable")
	}

	subdomainID := strings.TrimSpace(input.SubdomainID)
	if subdomainID == "" {
		return nil, DeleteTaxonomySubdomainOutput{}, fmt.Errorf("subdomain_id is required")
	}
	if err := registry.DeleteSubdomain(ctx, subdomainID); err != nil {
		return nil, DeleteTaxonomySubdomainOutput{}, fmt.Errorf(
			"delete taxonomy subdomain %q: %w",
			subdomainID,
			err,
		)
	}

	return nil, DeleteTaxonomySubdomainOutput{Deleted: true}, nil
}

func createTaxonomyTag(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateTaxonomyTagInput,
	registry CatalogTaxonomyRegistryWriter,
) (
	*mcp.CallToolResult,
	CreateTaxonomyTagOutput,
	error,
) {
	if registry == nil {
		return nil, CreateTaxonomyTagOutput{}, fmt.Errorf("catalog taxonomy registry API is unavailable")
	}

	tagID := strings.TrimSpace(input.TagID)
	value, err := registry.CreateTag(ctx, domain.CatalogTaxonomyTagCreateInput{
		TagID:       tagID,
		Key:         input.Key,
		Name:        input.Name,
		Description: input.Description,
		Color:       input.Color,
		Active:      input.Active,
	})
	if err != nil {
		return nil, CreateTaxonomyTagOutput{}, fmt.Errorf("create taxonomy tag %q: %w", tagID, err)
	}

	return nil, value, nil
}

func updateTaxonomyTag(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdateTaxonomyTagInput,
	registry CatalogTaxonomyRegistryWriter,
) (
	*mcp.CallToolResult,
	UpdateTaxonomyTagOutput,
	error,
) {
	if registry == nil {
		return nil, UpdateTaxonomyTagOutput{}, fmt.Errorf("catalog taxonomy registry API is unavailable")
	}

	tagID := strings.TrimSpace(input.TagID)
	if tagID == "" {
		return nil, UpdateTaxonomyTagOutput{}, fmt.Errorf("tag_id is required")
	}
	if !hasTaxonomyTagUpdateValues(input) {
		return nil, UpdateTaxonomyTagOutput{}, fmt.Errorf(
			"at least one of key, name, description, color, or active is required",
		)
	}

	value, err := registry.UpdateTag(ctx, domain.CatalogTaxonomyTagUpdateInput{
		TagID:       tagID,
		Key:         input.Key,
		Name:        input.Name,
		Description: input.Description,
		Color:       input.Color,
		Active:      input.Active,
	})
	if err != nil {
		return nil, UpdateTaxonomyTagOutput{}, fmt.Errorf("update taxonomy tag %q: %w", tagID, err)
	}

	return nil, value, nil
}

func deleteTaxonomyTag(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DeleteTaxonomyTagInput,
	registry CatalogTaxonomyRegistryWriter,
) (
	*mcp.CallToolResult,
	DeleteTaxonomyTagOutput,
	error,
) {
	if registry == nil {
		return nil, DeleteTaxonomyTagOutput{}, fmt.Errorf("catalog taxonomy registry API is unavailable")
	}

	tagID := strings.TrimSpace(input.TagID)
	if tagID == "" {
		return nil, DeleteTaxonomyTagOutput{}, fmt.Errorf("tag_id is required")
	}
	if err := registry.DeleteTag(ctx, tagID); err != nil {
		return nil, DeleteTaxonomyTagOutput{}, fmt.Errorf("delete taxonomy tag %q: %w", tagID, err)
	}

	return nil, DeleteTaxonomyTagOutput{Deleted: true}, nil
}

func patchCatalogItemTaxonomy(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input PatchCatalogItemTaxonomyInput,
	assignment CatalogTaxonomyAssignmentWriter,
) (
	*mcp.CallToolResult,
	PatchCatalogItemTaxonomyOutput,
	error,
) {
	if assignment == nil {
		return nil, PatchCatalogItemTaxonomyOutput{}, fmt.Errorf("catalog taxonomy assignment API is unavailable")
	}

	itemID := strings.TrimSpace(input.ItemID)
	if itemID == "" {
		return nil, PatchCatalogItemTaxonomyOutput{}, fmt.Errorf("item_id is required")
	}

	value, err := assignment.Patch(ctx, domain.CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:               itemID,
		PrimaryDomainID:      input.PrimaryDomainID,
		PrimarySubdomainID:   input.PrimarySubdomainID,
		SecondaryDomainID:    input.SecondaryDomainID,
		SecondarySubdomainID: input.SecondarySubdomainID,
		TagIDs:               input.TagIDs,
		AddTagIDs:            input.AddTagIDs,
		RemoveTagIDs:         input.RemoveTagIDs,
		ClearTags:            input.ClearTags,
		UpdatedBy:            input.UpdatedBy,
	})
	if err != nil {
		return nil, PatchCatalogItemTaxonomyOutput{}, fmt.Errorf(
			"patch catalog item taxonomy for %q: %w",
			itemID,
			err,
		)
	}

	return nil, value, nil
}

func patchCatalogItemsTaxonomy(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input PatchCatalogItemsTaxonomyInput,
	assignment CatalogTaxonomyAssignmentWriter,
) (
	*mcp.CallToolResult,
	PatchCatalogItemsTaxonomyOutput,
	error,
) {
	if assignment == nil {
		return nil, PatchCatalogItemsTaxonomyOutput{}, fmt.Errorf("catalog taxonomy assignment API is unavailable")
	}

	request := domain.CatalogItemTaxonomyBatchPatchRequest{
		DryRun: input.DryRun,
		Items:  make([]domain.CatalogItemTaxonomyAssignmentPatchInput, 0, len(input.Items)),
	}
	for _, item := range input.Items {
		request.Items = append(request.Items, domain.CatalogItemTaxonomyAssignmentPatchInput{
			ItemID:               item.ItemID,
			PrimaryDomainID:      item.PrimaryDomainID,
			PrimarySubdomainID:   item.PrimarySubdomainID,
			SecondaryDomainID:    item.SecondaryDomainID,
			SecondarySubdomainID: item.SecondarySubdomainID,
			TagIDs:               item.TagIDs,
			AddTagIDs:            item.AddTagIDs,
			RemoveTagIDs:         item.RemoveTagIDs,
			ClearTags:            item.ClearTags,
			UpdatedBy:            item.UpdatedBy,
		})
	}

	value, err := assignment.PatchBatch(ctx, request)
	if err != nil {
		return nil, PatchCatalogItemsTaxonomyOutput{}, fmt.Errorf("patch catalog items taxonomy: %w", err)
	}

	return nil, value, nil
}

func getTaxonomyDomainUsage(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetTaxonomyDomainUsageInput,
	usage CatalogTaxonomyUsageReader,
) (
	*mcp.CallToolResult,
	GetTaxonomyDomainUsageOutput,
	error,
) {
	if usage == nil {
		return nil, GetTaxonomyDomainUsageOutput{}, fmt.Errorf("catalog taxonomy usage API is unavailable")
	}

	domainID := strings.TrimSpace(input.DomainID)
	if domainID == "" {
		return nil, GetTaxonomyDomainUsageOutput{}, fmt.Errorf("domain_id is required")
	}

	previewLimit, err := normalizeCatalogTaxonomyUsagePreviewLimit(input.PreviewLimit)
	if err != nil {
		return nil, GetTaxonomyDomainUsageOutput{}, err
	}

	value, err := usage.GetDomainUsage(ctx, domainID, previewLimit)
	if err != nil {
		return nil, GetTaxonomyDomainUsageOutput{}, fmt.Errorf("get taxonomy domain usage for %q: %w", domainID, err)
	}

	return nil, value, nil
}

func getTaxonomySubdomainUsage(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetTaxonomySubdomainUsageInput,
	usage CatalogTaxonomyUsageReader,
) (
	*mcp.CallToolResult,
	GetTaxonomySubdomainUsageOutput,
	error,
) {
	if usage == nil {
		return nil, GetTaxonomySubdomainUsageOutput{}, fmt.Errorf("catalog taxonomy usage API is unavailable")
	}

	subdomainID := strings.TrimSpace(input.SubdomainID)
	if subdomainID == "" {
		return nil, GetTaxonomySubdomainUsageOutput{}, fmt.Errorf("subdomain_id is required")
	}

	previewLimit, err := normalizeCatalogTaxonomyUsagePreviewLimit(input.PreviewLimit)
	if err != nil {
		return nil, GetTaxonomySubdomainUsageOutput{}, err
	}

	value, err := usage.GetSubdomainUsage(ctx, subdomainID, previewLimit)
	if err != nil {
		return nil, GetTaxonomySubdomainUsageOutput{}, fmt.Errorf(
			"get taxonomy subdomain usage for %q: %w",
			subdomainID,
			err,
		)
	}

	return nil, value, nil
}

func getTaxonomyTagUsage(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetTaxonomyTagUsageInput,
	usage CatalogTaxonomyUsageReader,
) (
	*mcp.CallToolResult,
	GetTaxonomyTagUsageOutput,
	error,
) {
	if usage == nil {
		return nil, GetTaxonomyTagUsageOutput{}, fmt.Errorf("catalog taxonomy usage API is unavailable")
	}

	tagID := strings.TrimSpace(input.TagID)
	if tagID == "" {
		return nil, GetTaxonomyTagUsageOutput{}, fmt.Errorf("tag_id is required")
	}

	previewLimit, err := normalizeCatalogTaxonomyUsagePreviewLimit(input.PreviewLimit)
	if err != nil {
		return nil, GetTaxonomyTagUsageOutput{}, err
	}

	value, err := usage.GetTagUsage(ctx, tagID, previewLimit)
	if err != nil {
		return nil, GetTaxonomyTagUsageOutput{}, fmt.Errorf("get taxonomy tag usage for %q: %w", tagID, err)
	}

	return nil, value, nil
}

func normalizeCatalogTaxonomyUsagePreviewLimit(limit *int) (int, error) {
	if limit == nil {
		return catalogTaxonomyUsageDefaultPreviewLimit, nil
	}
	if *limit < 0 || *limit > catalogTaxonomyUsageMaxPreviewLimit {
		return 0, fmt.Errorf(
			"preview_limit must be an integer between 0 and %d",
			catalogTaxonomyUsageMaxPreviewLimit,
		)
	}
	return *limit, nil
}

func hasTaxonomyDomainUpdateValues(input UpdateTaxonomyDomainInput) bool {
	return input.Key != nil || input.Name != nil || input.Description != nil || input.Active != nil
}

func hasTaxonomySubdomainUpdateValues(input UpdateTaxonomySubdomainInput) bool {
	return input.DomainID != nil ||
		input.Key != nil ||
		input.Name != nil ||
		input.Description != nil ||
		input.Active != nil
}

func hasTaxonomyTagUpdateValues(input UpdateTaxonomyTagInput) bool {
	return input.Key != nil ||
		input.Name != nil ||
		input.Description != nil ||
		input.Color != nil ||
		input.Active != nil
}

// ListSkillResourcesInput is the input for list_skill_resources tool
type ListSkillResourcesInput struct {
	SkillID string `json:"skill_id" jsonschema:"The skill ID returned by list_skills or search_skills. Accepts bare skill IDs and canonical skill item IDs."`
}

// ListSkillResourcesOutput is the output for list_skill_resources tool
type ListSkillResourcesOutput struct {
	Resources []SkillResourceInfo `json:"resources"`
}

// SkillResourceInfo represents resource information in MCP responses
type SkillResourceInfo struct {
	Type     string `json:"type"`      // "script", "reference", "prompt", or "asset"
	Path     string `json:"path"`      // Relative path from skill root
	Name     string `json:"name"`      // Filename only
	Size     int64  `json:"size"`      // File size in bytes
	MimeType string `json:"mime_type"` // MIME type
	Readable bool   `json:"readable"`  // true if text file, false if binary
	Origin   string `json:"origin"`    // "direct" or "imported"
	Writable bool   `json:"writable"`  // true if resource can be modified
}

// ReadSkillResourceInput is the input for read_skill_resource tool
type ReadSkillResourceInput struct {
	SkillID      string `json:"skill_id"`      // Bare skill ID or canonical skill item ID
	ResourcePath string `json:"resource_path"` // Relative path from skill root (e.g., "scripts/script.py")
}

// ReadSkillResourceOutput is the output for read_skill_resource tool
type ReadSkillResourceOutput struct {
	Content  string `json:"content"`  // UTF-8 for text, base64 for binary
	Encoding string `json:"encoding"` // "utf-8" or "base64"
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

// GetSkillResourceInfoInput is the input for get_skill_resource_info tool
type GetSkillResourceInfoInput struct {
	SkillID      string `json:"skill_id"`
	ResourcePath string `json:"resource_path"`
}

// GetSkillResourceInfoOutput is the output for get_skill_resource_info tool
type GetSkillResourceInfoOutput struct {
	Exists   bool   `json:"exists"`
	Type     string `json:"type"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	Readable bool   `json:"readable"`
	Origin   string `json:"origin"`
	Writable bool   `json:"writable"`
}

// listSkillResources lists all resources in a skill's optional directories
func listSkillResources(ctx context.Context, req *mcp.CallToolRequest, input ListSkillResourcesInput, manager domain.SkillManager) (
	*mcp.CallToolResult,
	ListSkillResourcesOutput,
	error,
) {
	skillID, err := normalizeSkillToolInput(input.SkillID, "skill_id")
	if err != nil {
		return nil, ListSkillResourcesOutput{}, err
	}

	resources, err := manager.ListSkillResources(skillID)
	if err != nil {
		return nil, ListSkillResourcesOutput{}, fmt.Errorf("failed to list skill resources: %w", err)
	}

	resourceInfos := make([]SkillResourceInfo, len(resources))
	for i, res := range resources {
		resourceInfos[i] = SkillResourceInfo{
			Type:     string(res.Type),
			Path:     res.Path,
			Name:     res.Name,
			Size:     res.Size,
			MimeType: res.MimeType,
			Readable: res.Readable,
			Origin:   resolveResourceOrigin(res.Origin),
			Writable: res.Writable,
		}
	}

	return nil, ListSkillResourcesOutput{Resources: resourceInfos}, nil
}

// readSkillResource reads the content of a skill resource file
func readSkillResource(ctx context.Context, req *mcp.CallToolRequest, input ReadSkillResourceInput, manager domain.SkillManager) (
	*mcp.CallToolResult,
	ReadSkillResourceOutput,
	error,
) {
	skillID, err := normalizeSkillToolInput(input.SkillID, "skill_id")
	if err != nil {
		return nil, ReadSkillResourceOutput{}, err
	}

	// Check file size limit (1MB for MCP)
	info, err := manager.GetSkillResourceInfo(skillID, input.ResourcePath)
	if err != nil {
		return nil, ReadSkillResourceOutput{}, fmt.Errorf("failed to get resource info: %w", err)
	}

	const maxMCPFileSize = 1024 * 1024 // 1MB
	if info.Size > maxMCPFileSize {
		return nil, ReadSkillResourceOutput{}, fmt.Errorf("file too large (%d bytes, max %d). Use web UI to download", info.Size, maxMCPFileSize)
	}

	content, err := manager.ReadSkillResource(skillID, input.ResourcePath)
	if err != nil {
		return nil, ReadSkillResourceOutput{}, fmt.Errorf("failed to read resource: %w", err)
	}

	return nil, ReadSkillResourceOutput{
		Content:  content.Content,
		Encoding: content.Encoding,
		MimeType: content.MimeType,
		Size:     content.Size,
	}, nil
}

// getSkillResourceInfo gets metadata about a specific resource without reading content
func getSkillResourceInfo(ctx context.Context, req *mcp.CallToolRequest, input GetSkillResourceInfoInput, manager domain.SkillManager) (
	*mcp.CallToolResult,
	GetSkillResourceInfoOutput,
	error,
) {
	skillID, err := normalizeSkillToolInput(input.SkillID, "skill_id")
	if err != nil {
		return nil, GetSkillResourceInfoOutput{}, err
	}

	info, err := manager.GetSkillResourceInfo(skillID, input.ResourcePath)
	if err != nil {
		// Resource doesn't exist
		return nil, GetSkillResourceInfoOutput{
			Exists: false,
		}, nil
	}

	return nil, GetSkillResourceInfoOutput{
		Exists:   true,
		Type:     string(info.Type),
		Path:     info.Path,
		Name:     info.Name,
		Size:     info.Size,
		MimeType: info.MimeType,
		Readable: info.Readable,
		Origin:   resolveResourceOrigin(info.Origin),
		Writable: info.Writable,
	}, nil
}

func resolveResourceOrigin(origin domain.ResourceOrigin) string {
	if origin == "" {
		return string(domain.ResourceOriginDirect)
	}

	return string(origin)
}

func normalizeSkillToolInput(raw string, fieldName string) (string, error) {
	reference, err := domain.NormalizeCatalogItemReference(raw)
	if err != nil {
		return "", fmt.Errorf("%s is invalid: %w", fieldName, err)
	}
	if reference.Classifier != domain.CatalogClassifierSkill {
		return "", fmt.Errorf("%s must reference a skill item", fieldName)
	}
	if strings.TrimSpace(reference.SkillID) == "" {
		return "", fmt.Errorf("%s is invalid", fieldName)
	}
	return reference.SkillID, nil
}

func resolveSkillDisplayName(skill domain.Skill) string {
	if trimmed := strings.TrimSpace(skill.Name); trimmed != "" {
		return trimmed
	}
	if skill.Metadata != nil {
		if trimmed := strings.TrimSpace(skill.Metadata.Name); trimmed != "" {
			return trimmed
		}
	}
	if trimmed := strings.TrimSpace(skill.ID); trimmed != "" {
		return trimmed
	}
	return domain.CanonicalSkillCatalogKey(skill.ID)
}
