package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/mudler/skillserver/pkg/domain"
	"github.com/mudler/skillserver/pkg/git"
)

// SkillResponse represents a skill in API responses
type SkillResponse struct {
	Name          string            `json:"name"`
	Content       string            `json:"content"`
	Description   string            `json:"description,omitempty"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  string            `json:"allowed-tools,omitempty"`
	ReadOnly      bool              `json:"readOnly"`
}

// CreateSkillRequest represents a request to create a skill
type CreateSkillRequest struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Content       string            `json:"content"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  string            `json:"allowed-tools,omitempty"`
}

// UpdateSkillRequest represents a request to update a skill
type UpdateSkillRequest struct {
	Description   string            `json:"description"`
	Content       string            `json:"content"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  string            `json:"allowed-tools,omitempty"`
}

// CatalogItemResponse represents a catalog entry in API responses.
type CatalogItemResponse struct {
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
	CustomMetadata     map[string]any                    `json:"custom_metadata,omitempty"`
	Labels             []string                          `json:"labels,omitempty"`
	ContentWritable    bool                              `json:"content_writable"`
	MetadataWritable   bool                              `json:"metadata_writable"`
	ReadOnly           bool                              `json:"read_only"`
}

// PatchCatalogMetadataRequest represents one metadata overlay mutation request.
type PatchCatalogMetadataRequest struct {
	DisplayName    *string         `json:"display_name"`
	Description    *string         `json:"description"`
	Labels         *[]string       `json:"labels"`
	CustomMetadata *map[string]any `json:"custom_metadata"`
	UpdatedBy      *string         `json:"updated_by,omitempty"`
}

// CatalogTaxonomyDomainCreateRequest describes domain create payloads.
type CatalogTaxonomyDomainCreateRequest struct {
	DomainID    string `json:"domain_id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Active      *bool  `json:"active,omitempty"`
}

// CatalogTaxonomyDomainUpdateRequest describes domain patch payloads.
type CatalogTaxonomyDomainUpdateRequest struct {
	Key         *string `json:"key,omitempty"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Active      *bool   `json:"active,omitempty"`
}

// CatalogTaxonomySubdomainCreateRequest describes subdomain create payloads.
type CatalogTaxonomySubdomainCreateRequest struct {
	SubdomainID string `json:"subdomain_id"`
	DomainID    string `json:"domain_id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Active      *bool  `json:"active,omitempty"`
}

// CatalogTaxonomySubdomainUpdateRequest describes subdomain patch payloads.
type CatalogTaxonomySubdomainUpdateRequest struct {
	DomainID    *string `json:"domain_id,omitempty"`
	Key         *string `json:"key,omitempty"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Active      *bool   `json:"active,omitempty"`
}

// CatalogTaxonomyTagCreateRequest describes tag create payloads.
type CatalogTaxonomyTagCreateRequest struct {
	TagID       string `json:"tag_id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
	Active      *bool  `json:"active,omitempty"`
}

// CatalogTaxonomyTagUpdateRequest describes tag patch payloads.
type CatalogTaxonomyTagUpdateRequest struct {
	Key         *string `json:"key,omitempty"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
	Active      *bool   `json:"active,omitempty"`
}

// PatchCatalogItemTaxonomyRequest describes item taxonomy assignment patch payloads.
type PatchCatalogItemTaxonomyRequest struct {
	PrimaryDomainID      *string   `json:"primary_domain_id,omitempty"`
	PrimarySubdomainID   *string   `json:"primary_subdomain_id,omitempty"`
	SecondaryDomainID    *string   `json:"secondary_domain_id,omitempty"`
	SecondarySubdomainID *string   `json:"secondary_subdomain_id,omitempty"`
	TagIDs               *[]string `json:"tag_ids,omitempty"`
	UpdatedBy            *string   `json:"updated_by,omitempty"`
}

// CatalogMetadataResponse represents source, overlay, and effective metadata views.
type CatalogMetadataResponse struct {
	ItemID    string                           `json:"item_id"`
	Source    CatalogMetadataSourceResponse    `json:"source"`
	Overlay   CatalogMetadataOverlayResponse   `json:"overlay"`
	Effective CatalogMetadataEffectiveResponse `json:"effective"`
}

// CatalogMetadataSourceResponse represents immutable source snapshot metadata.
type CatalogMetadataSourceResponse struct {
	ItemID           string                   `json:"item_id"`
	Classifier       domain.CatalogClassifier `json:"classifier"`
	SourceType       string                   `json:"source_type"`
	SourceRepo       *string                  `json:"source_repo,omitempty"`
	ParentSkillID    *string                  `json:"parent_skill_id,omitempty"`
	ResourcePath     *string                  `json:"resource_path,omitempty"`
	Name             string                   `json:"name"`
	Description      string                   `json:"description,omitempty"`
	ContentWritable  bool                     `json:"content_writable"`
	MetadataWritable bool                     `json:"metadata_writable"`
	ReadOnly         bool                     `json:"read_only"`
}

// CatalogMetadataOverlayResponse represents user-owned overlay metadata.
type CatalogMetadataOverlayResponse struct {
	DisplayName    *string        `json:"display_name,omitempty"`
	Description    *string        `json:"description,omitempty"`
	CustomMetadata map[string]any `json:"custom_metadata"`
	Labels         []string       `json:"labels"`
	UpdatedAt      *string        `json:"updated_at,omitempty"`
	UpdatedBy      *string        `json:"updated_by,omitempty"`
}

// CatalogMetadataEffectiveResponse represents merged source + overlay metadata.
type CatalogMetadataEffectiveResponse struct {
	Name               string                            `json:"name"`
	Description        string                            `json:"description,omitempty"`
	PrimaryDomain      *domain.CatalogTaxonomyReference  `json:"primary_domain,omitempty"`
	PrimarySubdomain   *domain.CatalogTaxonomyReference  `json:"primary_subdomain,omitempty"`
	SecondaryDomain    *domain.CatalogTaxonomyReference  `json:"secondary_domain,omitempty"`
	SecondarySubdomain *domain.CatalogTaxonomyReference  `json:"secondary_subdomain,omitempty"`
	Tags               []domain.CatalogTaxonomyReference `json:"tags"`
	CustomMetadata     map[string]any                    `json:"custom_metadata"`
	Labels             []string                          `json:"labels"`
	ContentWritable    bool                              `json:"content_writable"`
	MetadataWritable   bool                              `json:"metadata_writable"`
	ReadOnly           bool                              `json:"read_only"`
}

const (
	catalogTaxonomyRequestMaxBodyBytes     = 32 * 1024
	catalogMetadataPatchMaxBodyBytes       = 32 * 1024
	catalogMetadataDisplayNameMaxChars     = 256
	catalogMetadataDescriptionMaxChars     = 4096
	catalogMetadataMaxLabels               = 64
	catalogMetadataLabelMaxChars           = 64
	catalogMetadataCustomMetadataMaxKeys   = 128
	catalogMetadataCustomMetadataMaxDepth  = 6
	catalogMetadataCustomMetadataMaxArray  = 256
	catalogMetadataCustomMetadataMaxString = 4096
	catalogMetadataCustomMetadataMaxKeyLen = 128
)

var errCatalogTaxonomyFiltersUnavailable = errors.New("catalog taxonomy filters are unavailable")

func catalogResponseFromItem(item domain.CatalogItem) CatalogItemResponse {
	return CatalogItemResponse{
		ID:                 item.ID,
		Classifier:         item.Classifier,
		Name:               item.Name,
		Description:        item.Description,
		Content:            item.Content,
		ParentSkillID:      item.ParentSkillID,
		ResourcePath:       item.ResourcePath,
		PrimaryDomain:      cloneCatalogTaxonomyReference(item.PrimaryDomain),
		PrimarySubdomain:   cloneCatalogTaxonomyReference(item.PrimarySubdomain),
		SecondaryDomain:    cloneCatalogTaxonomyReference(item.SecondaryDomain),
		SecondarySubdomain: cloneCatalogTaxonomyReference(item.SecondarySubdomain),
		Tags:               cloneCatalogTaxonomyReferences(item.Tags),
		CustomMetadata:     cloneCatalogMetadataMap(item.CustomMetadata),
		Labels:             append([]string{}, item.Labels...),
		ContentWritable:    item.ContentWritable,
		MetadataWritable:   item.MetadataWritable,
		ReadOnly:           item.ReadOnly,
	}
}

func skillNameFromRoute(c *echo.Context) string {
	repo := strings.TrimSpace(c.Param("repo"))
	name := strings.TrimSpace(c.Param("name"))
	if repo != "" && name != "" {
		return repo + "/" + name
	}
	return name
}

// listSkills lists all skills
func (s *Server) listSkills(c *echo.Context) error {
	skills, err := s.skillManager.ListSkills()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	responses := make([]SkillResponse, len(skills))
	for i, skill := range skills {
		responses[i] = SkillResponse{
			Name:     skill.Name,
			Content:  skill.Content,
			ReadOnly: skill.ReadOnly,
		}
		if skill.Metadata != nil {
			responses[i].Description = skill.Metadata.Description
			responses[i].License = skill.Metadata.License
			responses[i].Compatibility = skill.Metadata.Compatibility
			responses[i].Metadata = skill.Metadata.Metadata
			responses[i].AllowedTools = skill.Metadata.AllowedTools
		}
	}

	return c.JSON(http.StatusOK, responses)
}

// getSkill gets a single skill by name
func (s *Server) getSkill(c *echo.Context) error {
	name := skillNameFromRoute(c)
	skill, err := s.skillManager.ReadSkill(name)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "skill not found",
		})
	}

	response := SkillResponse{
		Name:     skill.Name,
		Content:  skill.Content,
		ReadOnly: skill.ReadOnly,
	}
	if skill.Metadata != nil {
		response.Description = skill.Metadata.Description
		response.License = skill.Metadata.License
		response.Compatibility = skill.Metadata.Compatibility
		response.Metadata = skill.Metadata.Metadata
		response.AllowedTools = skill.Metadata.AllowedTools
	}

	return c.JSON(http.StatusOK, response)
}

// createSkill creates a new skill
func (s *Server) createSkill(c *echo.Context) error {
	var req CreateSkillRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	// Validate name
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "name is required",
		})
	}

	// Validate name according to Agent Skills spec
	if err := domain.ValidateSkillName(req.Name); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Validate description
	if req.Description == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "description is required",
		})
	}
	if len(req.Description) > 1024 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "description must be 1-1024 characters",
		})
	}

	// Validate compatibility if provided
	if req.Compatibility != "" && len(req.Compatibility) > 500 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "compatibility must be max 500 characters",
		})
	}

	// Get the skills directory from the manager
	fsManager, ok := s.skillManager.(*domain.FileSystemManager)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "unsupported manager type",
		})
	}

	// Create skill directory
	skillsDir := fsManager.GetSkillsDir()
	skillDir := filepath.Join(skillsDir, req.Name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to create skill directory: %v", err),
		})
	}

	// Build frontmatter
	frontmatter := fmt.Sprintf("---\nname: %s\ndescription: %s\n", req.Name, req.Description)
	if req.License != "" {
		frontmatter += fmt.Sprintf("license: %s\n", req.License)
	}
	if req.Compatibility != "" {
		frontmatter += fmt.Sprintf("compatibility: %s\n", req.Compatibility)
	}
	if len(req.Metadata) > 0 {
		frontmatter += "metadata:\n"
		for k, v := range req.Metadata {
			frontmatter += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}
	if req.AllowedTools != "" {
		frontmatter += fmt.Sprintf("allowed-tools: %s\n", req.AllowedTools)
	}
	frontmatter += "---\n\n"

	// Write SKILL.md file
	skillMdPath := filepath.Join(skillDir, "SKILL.md")
	fullContent := frontmatter + req.Content
	if err := writeFile(skillMdPath, fullContent); err != nil {
		os.RemoveAll(skillDir) // Clean up on error
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Rebuild index
	if err := s.skillManager.RebuildIndex(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to rebuild index",
		})
	}

	skill, err := s.skillManager.ReadSkill(req.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to read created skill",
		})
	}

	response := SkillResponse{
		Name:     skill.Name,
		Content:  skill.Content,
		ReadOnly: skill.ReadOnly,
	}
	if skill.Metadata != nil {
		response.Description = skill.Metadata.Description
		response.License = skill.Metadata.License
		response.Compatibility = skill.Metadata.Compatibility
		response.Metadata = skill.Metadata.Metadata
		response.AllowedTools = skill.Metadata.AllowedTools
	}

	return c.JSON(http.StatusCreated, response)
}

// updateSkill updates an existing skill
func (s *Server) updateSkill(c *echo.Context) error {
	name := skillNameFromRoute(c)

	// Check if skill exists and is read-only
	existingSkill, err := s.skillManager.ReadSkill(name)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "skill not found",
		})
	}
	if existingSkill.ReadOnly {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "cannot update read-only skill from git repository",
		})
	}

	var req UpdateSkillRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	// Validate description
	if req.Description == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "description is required",
		})
	}
	if len(req.Description) > 1024 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "description must be 1-1024 characters",
		})
	}

	// Validate compatibility if provided
	if req.Compatibility != "" && len(req.Compatibility) > 500 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "compatibility must be max 500 characters",
		})
	}

	// Get the skills directory from the manager
	fsManager, ok := s.skillManager.(*domain.FileSystemManager)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "unsupported manager type",
		})
	}

	// Build frontmatter (name must match directory name)
	skillDir := filepath.Join(fsManager.GetSkillsDir(), name)
	frontmatter := fmt.Sprintf("---\nname: %s\ndescription: %s\n", name, req.Description)
	if req.License != "" {
		frontmatter += fmt.Sprintf("license: %s\n", req.License)
	}
	if req.Compatibility != "" {
		frontmatter += fmt.Sprintf("compatibility: %s\n", req.Compatibility)
	}
	if len(req.Metadata) > 0 {
		frontmatter += "metadata:\n"
		for k, v := range req.Metadata {
			frontmatter += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}
	if req.AllowedTools != "" {
		frontmatter += fmt.Sprintf("allowed-tools: %s\n", req.AllowedTools)
	}
	frontmatter += "---\n\n"

	// Write SKILL.md file
	skillMdPath := filepath.Join(skillDir, "SKILL.md")
	fullContent := frontmatter + req.Content
	if err := writeFile(skillMdPath, fullContent); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Rebuild index
	if err := s.skillManager.RebuildIndex(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to rebuild index",
		})
	}

	skill, err := s.skillManager.ReadSkill(name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to read updated skill",
		})
	}

	response := SkillResponse{
		Name:     skill.Name,
		Content:  skill.Content,
		ReadOnly: skill.ReadOnly,
	}
	if skill.Metadata != nil {
		response.Description = skill.Metadata.Description
		response.License = skill.Metadata.License
		response.Compatibility = skill.Metadata.Compatibility
		response.Metadata = skill.Metadata.Metadata
		response.AllowedTools = skill.Metadata.AllowedTools
	}

	return c.JSON(http.StatusOK, response)
}

// deleteSkill deletes a skill
func (s *Server) deleteSkill(c *echo.Context) error {
	name := skillNameFromRoute(c)

	// Check if skill exists and is read-only
	existingSkill, err := s.skillManager.ReadSkill(name)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "skill not found",
		})
	}
	if existingSkill.ReadOnly {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "cannot delete read-only skill from git repository",
		})
	}

	// Get the skills directory from the manager
	fsManager, ok := s.skillManager.(*domain.FileSystemManager)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "unsupported manager type",
		})
	}

	// Delete the skill directory
	skillsDir := fsManager.GetSkillsDir()
	skillDir := filepath.Join(skillsDir, name)
	if err := os.RemoveAll(skillDir); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Rebuild index
	if err := s.skillManager.RebuildIndex(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to rebuild index",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// searchSkills searches for skills
func (s *Server) searchSkills(c *echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "query parameter 'q' is required",
		})
	}

	skills, err := s.skillManager.SearchSkills(query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	responses := make([]SkillResponse, len(skills))
	for i, skill := range skills {
		responses[i] = SkillResponse{
			Name:     skill.Name,
			Content:  skill.Content,
			ReadOnly: skill.ReadOnly,
		}
		if skill.Metadata != nil {
			responses[i].Description = skill.Metadata.Description
			responses[i].License = skill.Metadata.License
			responses[i].Compatibility = skill.Metadata.Compatibility
			responses[i].Metadata = skill.Metadata.Metadata
			responses[i].AllowedTools = skill.Metadata.AllowedTools
		}
	}

	return c.JSON(http.StatusOK, responses)
}

// listCatalog lists all catalog items (skills and prompts).
func (s *Server) listCatalog(c *echo.Context) error {
	taxonomyFilter, err := decodeCatalogListTaxonomyFilter(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	items, err := s.loadCatalogItems(c.Request().Context(), "", nil, taxonomyFilter)
	if err != nil {
		if errors.Is(err, errCatalogTaxonomyFiltersUnavailable) {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	responses := make([]CatalogItemResponse, len(items))
	for i, item := range items {
		responses[i] = catalogResponseFromItem(item)
	}

	return c.JSON(http.StatusOK, responses)
}

// searchCatalog searches catalog items by query with an optional classifier filter.
func (s *Server) searchCatalog(c *echo.Context) error {
	query := strings.TrimSpace(c.QueryParam("q"))
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "query parameter 'q' is required",
		})
	}

	var classifier *domain.CatalogClassifier
	classifierRaw := strings.TrimSpace(c.QueryParam("classifier"))
	if classifierRaw != "" {
		parsedClassifier, err := domain.ParseCatalogClassifier(classifierRaw)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}
		classifier = &parsedClassifier
	}

	taxonomyFilter, err := decodeCatalogListTaxonomyFilter(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	items, err := s.loadCatalogItems(c.Request().Context(), query, classifier, taxonomyFilter)
	if err != nil {
		if errors.Is(err, errCatalogTaxonomyFiltersUnavailable) {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	responses := make([]CatalogItemResponse, len(items))
	for i, item := range items {
		responses[i] = catalogResponseFromItem(item)
	}

	return c.JSON(http.StatusOK, responses)
}

func (s *Server) loadCatalogItems(
	ctx context.Context,
	query string,
	classifier *domain.CatalogClassifier,
	taxonomyFilter domain.CatalogEffectiveListFilter,
) ([]domain.CatalogItem, error) {
	normalizedQuery := strings.TrimSpace(query)
	if s.catalogMetadataService != nil {
		taxonomyFilter.Classifier = classifier
		items, err := s.catalogMetadataService.List(ctx, taxonomyFilter)
		if err != nil {
			return nil, err
		}
		if normalizedQuery == "" {
			return items, nil
		}

		return filterCatalogItemsByQuery(items, normalizedQuery), nil
	}

	if hasCatalogTaxonomyListFilterConstraints(taxonomyFilter) {
		return nil, errCatalogTaxonomyFiltersUnavailable
	}

	if normalizedQuery == "" {
		return s.skillManager.ListCatalogItems()
	}
	return s.skillManager.SearchCatalogItems(normalizedQuery, classifier)
}

func cloneCatalogMetadataMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}

	copied := make(map[string]any, len(input))
	for key, value := range input {
		copied[key] = value
	}
	return copied
}

func cloneCatalogTaxonomyReference(
	reference *domain.CatalogTaxonomyReference,
) *domain.CatalogTaxonomyReference {
	if reference == nil {
		return nil
	}
	copied := *reference
	return &copied
}

func cloneCatalogTaxonomyReferences(
	references []domain.CatalogTaxonomyReference,
) []domain.CatalogTaxonomyReference {
	if len(references) == 0 {
		return []domain.CatalogTaxonomyReference{}
	}
	copied := make([]domain.CatalogTaxonomyReference, len(references))
	copy(copied, references)
	return copied
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

// getCatalogMetadata returns source + overlay + effective metadata for one catalog item.
func (s *Server) getCatalogMetadata(c *echo.Context) error {
	if s.catalogMetadataService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog metadata API is unavailable",
		})
	}

	itemID, err := decodeCatalogItemIDFromPath(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	view, err := s.catalogMetadataService.Get(c.Request().Context(), itemID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCatalogMetadataItemNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "catalog item not found",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
	}

	return c.JSON(http.StatusOK, catalogMetadataResponseFromView(view))
}

// patchCatalogMetadata updates metadata overlays for one catalog item.
func (s *Server) patchCatalogMetadata(c *echo.Context) error {
	if s.catalogMetadataService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog metadata API is unavailable",
		})
	}

	itemID, err := decodeCatalogItemIDFromPath(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	request, err := decodeCatalogMetadataPatchRequest(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	input, err := normalizeCatalogMetadataPatchInput(itemID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	view, err := s.catalogMetadataService.Patch(c.Request().Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCatalogMetadataItemNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "catalog item not found",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
	}

	return c.JSON(http.StatusOK, catalogMetadataResponseFromView(view))
}

// getCatalogItemTaxonomy returns one catalog item's taxonomy assignment state.
func (s *Server) getCatalogItemTaxonomy(c *echo.Context) error {
	if s.taxonomyAssignment == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy assignment API is unavailable",
		})
	}

	itemID, err := decodeCatalogItemIDFromPath(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	assignment, err := s.taxonomyAssignment.Get(c.Request().Context(), itemID)
	if err != nil {
		return encodeCatalogTaxonomyAssignmentServiceError(c, err)
	}

	return c.JSON(http.StatusOK, assignment)
}

// patchCatalogItemTaxonomy patches one catalog item's taxonomy assignment state.
func (s *Server) patchCatalogItemTaxonomy(c *echo.Context) error {
	if s.taxonomyAssignment == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy assignment API is unavailable",
		})
	}

	itemID, err := decodeCatalogItemIDFromPath(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	request, err := decodeCatalogTaxonomyRequest[PatchCatalogItemTaxonomyRequest](c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	input := domain.CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:               itemID,
		PrimaryDomainID:      request.PrimaryDomainID,
		PrimarySubdomainID:   request.PrimarySubdomainID,
		SecondaryDomainID:    request.SecondaryDomainID,
		SecondarySubdomainID: request.SecondarySubdomainID,
		TagIDs:               request.TagIDs,
		UpdatedBy:            request.UpdatedBy,
	}

	assignment, err := s.taxonomyAssignment.Patch(c.Request().Context(), input)
	if err != nil {
		return encodeCatalogTaxonomyAssignmentServiceError(c, err)
	}

	return c.JSON(http.StatusOK, assignment)
}

// listCatalogTaxonomyDomains returns taxonomy domain objects.
func (s *Server) listCatalogTaxonomyDomains(c *echo.Context) error {
	if s.taxonomyRegistry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy registry API is unavailable",
		})
	}

	filter, err := decodeCatalogTaxonomyDomainListFilter(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	domains, err := s.taxonomyRegistry.ListDomains(c.Request().Context(), filter)
	if err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, nil)
	}

	return c.JSON(http.StatusOK, domains)
}

// createCatalogTaxonomyDomain creates one taxonomy domain object.
func (s *Server) createCatalogTaxonomyDomain(c *echo.Context) error {
	if s.taxonomyRegistry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy registry API is unavailable",
		})
	}

	request, err := decodeCatalogTaxonomyRequest[CatalogTaxonomyDomainCreateRequest](c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	created, err := s.taxonomyRegistry.CreateDomain(
		c.Request().Context(),
		domain.CatalogTaxonomyDomainCreateInput{
			DomainID:    request.DomainID,
			Key:         request.Key,
			Name:        request.Name,
			Description: request.Description,
			Active:      request.Active,
		},
	)
	if err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, nil)
	}

	return c.JSON(http.StatusCreated, created)
}

// updateCatalogTaxonomyDomain patches one taxonomy domain object by ID.
func (s *Server) updateCatalogTaxonomyDomain(c *echo.Context) error {
	if s.taxonomyRegistry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy registry API is unavailable",
		})
	}

	domainID, err := decodeCatalogTaxonomyObjectIDFromPath(c.Param("id"), "domain_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	request, err := decodeCatalogTaxonomyRequest[CatalogTaxonomyDomainUpdateRequest](c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	if !hasCatalogTaxonomyDomainUpdateValues(request) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "at least one of key, name, description, or active is required",
		})
	}

	updated, err := s.taxonomyRegistry.UpdateDomain(
		c.Request().Context(),
		domain.CatalogTaxonomyDomainUpdateInput{
			DomainID:    domainID,
			Key:         request.Key,
			Name:        request.Name,
			Description: request.Description,
			Active:      request.Active,
		},
	)
	if err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, domain.ErrCatalogTaxonomyDomainNotFound)
	}

	return c.JSON(http.StatusOK, updated)
}

// deleteCatalogTaxonomyDomain deletes one taxonomy domain object by ID.
func (s *Server) deleteCatalogTaxonomyDomain(c *echo.Context) error {
	if s.taxonomyRegistry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy registry API is unavailable",
		})
	}

	domainID, err := decodeCatalogTaxonomyObjectIDFromPath(c.Param("id"), "domain_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if err := s.taxonomyRegistry.DeleteDomain(c.Request().Context(), domainID); err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, domain.ErrCatalogTaxonomyDomainNotFound)
	}

	return c.NoContent(http.StatusNoContent)
}

// listCatalogTaxonomySubdomains returns taxonomy subdomain objects.
func (s *Server) listCatalogTaxonomySubdomains(c *echo.Context) error {
	if s.taxonomyRegistry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy registry API is unavailable",
		})
	}

	filter, err := decodeCatalogTaxonomySubdomainListFilter(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	subdomains, err := s.taxonomyRegistry.ListSubdomains(c.Request().Context(), filter)
	if err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, nil)
	}

	return c.JSON(http.StatusOK, subdomains)
}

// createCatalogTaxonomySubdomain creates one taxonomy subdomain object.
func (s *Server) createCatalogTaxonomySubdomain(c *echo.Context) error {
	if s.taxonomyRegistry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy registry API is unavailable",
		})
	}

	request, err := decodeCatalogTaxonomyRequest[CatalogTaxonomySubdomainCreateRequest](c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	created, err := s.taxonomyRegistry.CreateSubdomain(
		c.Request().Context(),
		domain.CatalogTaxonomySubdomainCreateInput{
			SubdomainID: request.SubdomainID,
			DomainID:    request.DomainID,
			Key:         request.Key,
			Name:        request.Name,
			Description: request.Description,
			Active:      request.Active,
		},
	)
	if err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, nil)
	}

	return c.JSON(http.StatusCreated, created)
}

// updateCatalogTaxonomySubdomain patches one taxonomy subdomain object by ID.
func (s *Server) updateCatalogTaxonomySubdomain(c *echo.Context) error {
	if s.taxonomyRegistry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy registry API is unavailable",
		})
	}

	subdomainID, err := decodeCatalogTaxonomyObjectIDFromPath(c.Param("id"), "subdomain_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	request, err := decodeCatalogTaxonomyRequest[CatalogTaxonomySubdomainUpdateRequest](c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	if !hasCatalogTaxonomySubdomainUpdateValues(request) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "at least one of domain_id, key, name, description, or active is required",
		})
	}

	updated, err := s.taxonomyRegistry.UpdateSubdomain(
		c.Request().Context(),
		domain.CatalogTaxonomySubdomainUpdateInput{
			SubdomainID: subdomainID,
			DomainID:    request.DomainID,
			Key:         request.Key,
			Name:        request.Name,
			Description: request.Description,
			Active:      request.Active,
		},
	)
	if err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, domain.ErrCatalogTaxonomySubdomainNotFound)
	}

	return c.JSON(http.StatusOK, updated)
}

// deleteCatalogTaxonomySubdomain deletes one taxonomy subdomain object by ID.
func (s *Server) deleteCatalogTaxonomySubdomain(c *echo.Context) error {
	if s.taxonomyRegistry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy registry API is unavailable",
		})
	}

	subdomainID, err := decodeCatalogTaxonomyObjectIDFromPath(c.Param("id"), "subdomain_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if err := s.taxonomyRegistry.DeleteSubdomain(c.Request().Context(), subdomainID); err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, domain.ErrCatalogTaxonomySubdomainNotFound)
	}

	return c.NoContent(http.StatusNoContent)
}

// listCatalogTaxonomyTags returns taxonomy tag objects.
func (s *Server) listCatalogTaxonomyTags(c *echo.Context) error {
	if s.taxonomyRegistry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy registry API is unavailable",
		})
	}

	filter, err := decodeCatalogTaxonomyTagListFilter(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	tags, err := s.taxonomyRegistry.ListTags(c.Request().Context(), filter)
	if err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, nil)
	}

	return c.JSON(http.StatusOK, tags)
}

// createCatalogTaxonomyTag creates one taxonomy tag object.
func (s *Server) createCatalogTaxonomyTag(c *echo.Context) error {
	if s.taxonomyRegistry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy registry API is unavailable",
		})
	}

	request, err := decodeCatalogTaxonomyRequest[CatalogTaxonomyTagCreateRequest](c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	created, err := s.taxonomyRegistry.CreateTag(
		c.Request().Context(),
		domain.CatalogTaxonomyTagCreateInput{
			TagID:       request.TagID,
			Key:         request.Key,
			Name:        request.Name,
			Description: request.Description,
			Color:       request.Color,
			Active:      request.Active,
		},
	)
	if err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, nil)
	}

	return c.JSON(http.StatusCreated, created)
}

// updateCatalogTaxonomyTag patches one taxonomy tag object by ID.
func (s *Server) updateCatalogTaxonomyTag(c *echo.Context) error {
	if s.taxonomyRegistry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy registry API is unavailable",
		})
	}

	tagID, err := decodeCatalogTaxonomyObjectIDFromPath(c.Param("id"), "tag_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	request, err := decodeCatalogTaxonomyRequest[CatalogTaxonomyTagUpdateRequest](c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	if !hasCatalogTaxonomyTagUpdateValues(request) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "at least one of key, name, description, color, or active is required",
		})
	}

	updated, err := s.taxonomyRegistry.UpdateTag(
		c.Request().Context(),
		domain.CatalogTaxonomyTagUpdateInput{
			TagID:       tagID,
			Key:         request.Key,
			Name:        request.Name,
			Description: request.Description,
			Color:       request.Color,
			Active:      request.Active,
		},
	)
	if err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, domain.ErrCatalogTaxonomyTagNotFound)
	}

	return c.JSON(http.StatusOK, updated)
}

// deleteCatalogTaxonomyTag deletes one taxonomy tag object by ID.
func (s *Server) deleteCatalogTaxonomyTag(c *echo.Context) error {
	if s.taxonomyRegistry == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy registry API is unavailable",
		})
	}

	tagID, err := decodeCatalogTaxonomyObjectIDFromPath(c.Param("id"), "tag_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if err := s.taxonomyRegistry.DeleteTag(c.Request().Context(), tagID); err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, domain.ErrCatalogTaxonomyTagNotFound)
	}

	return c.NoContent(http.StatusNoContent)
}

func decodeCatalogItemIDFromPath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("catalog item id is required")
	}

	decoded, err := url.PathUnescape(trimmed)
	if err != nil {
		return "", fmt.Errorf("catalog item id path is invalid")
	}

	itemID := strings.TrimSpace(decoded)
	if itemID == "" {
		return "", fmt.Errorf("catalog item id is required")
	}

	return itemID, nil
}

func decodeCatalogTaxonomyObjectIDFromPath(raw string, field string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required", field)
	}

	decoded, err := url.PathUnescape(trimmed)
	if err != nil {
		return "", fmt.Errorf("%s path is invalid", field)
	}

	normalized := strings.TrimSpace(decoded)
	if normalized == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return normalized, nil
}

func decodeCatalogTaxonomyRequest[T any](c *echo.Context) (T, error) {
	var zero T

	limitedReader := io.LimitReader(c.Request().Body, catalogTaxonomyRequestMaxBodyBytes+1)
	payload, err := io.ReadAll(limitedReader)
	if err != nil {
		return zero, fmt.Errorf("invalid request payload")
	}
	if len(payload) == 0 {
		return zero, fmt.Errorf("request body is required")
	}
	if len(payload) > catalogTaxonomyRequestMaxBodyBytes {
		return zero, fmt.Errorf("request payload exceeds %d bytes", catalogTaxonomyRequestMaxBodyBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var request T
	if err := decoder.Decode(&request); err != nil {
		return zero, fmt.Errorf("invalid request payload")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return zero, fmt.Errorf("invalid request payload")
	}

	return request, nil
}

func decodeCatalogTaxonomyDomainListFilter(
	c *echo.Context,
) (domain.CatalogTaxonomyDomainListFilter, error) {
	active, err := decodeCatalogTaxonomyBoolQueryParam(c.QueryParam("active"), "active")
	if err != nil {
		return domain.CatalogTaxonomyDomainListFilter{}, err
	}

	return domain.CatalogTaxonomyDomainListFilter{
		DomainID:  strings.TrimSpace(c.QueryParam("domain_id")),
		DomainIDs: decodeCatalogTaxonomyCSVQueryParam(c.QueryParam("domain_ids")),
		Key:       strings.TrimSpace(c.QueryParam("key")),
		Keys:      decodeCatalogTaxonomyCSVQueryParam(c.QueryParam("keys")),
		Active:    active,
	}, nil
}

func decodeCatalogTaxonomySubdomainListFilter(
	c *echo.Context,
) (domain.CatalogTaxonomySubdomainListFilter, error) {
	active, err := decodeCatalogTaxonomyBoolQueryParam(c.QueryParam("active"), "active")
	if err != nil {
		return domain.CatalogTaxonomySubdomainListFilter{}, err
	}

	return domain.CatalogTaxonomySubdomainListFilter{
		SubdomainID:  strings.TrimSpace(c.QueryParam("subdomain_id")),
		SubdomainIDs: decodeCatalogTaxonomyCSVQueryParam(c.QueryParam("subdomain_ids")),
		DomainID:     strings.TrimSpace(c.QueryParam("domain_id")),
		DomainIDs:    decodeCatalogTaxonomyCSVQueryParam(c.QueryParam("domain_ids")),
		Key:          strings.TrimSpace(c.QueryParam("key")),
		Keys:         decodeCatalogTaxonomyCSVQueryParam(c.QueryParam("keys")),
		Active:       active,
	}, nil
}

func decodeCatalogTaxonomyTagListFilter(c *echo.Context) (domain.CatalogTaxonomyTagListFilter, error) {
	active, err := decodeCatalogTaxonomyBoolQueryParam(c.QueryParam("active"), "active")
	if err != nil {
		return domain.CatalogTaxonomyTagListFilter{}, err
	}

	return domain.CatalogTaxonomyTagListFilter{
		TagID:  strings.TrimSpace(c.QueryParam("tag_id")),
		TagIDs: decodeCatalogTaxonomyCSVQueryParam(c.QueryParam("tag_ids")),
		Key:    strings.TrimSpace(c.QueryParam("key")),
		Keys:   decodeCatalogTaxonomyCSVQueryParam(c.QueryParam("keys")),
		Active: active,
	}, nil
}

func decodeCatalogListTaxonomyFilter(c *echo.Context) (domain.CatalogEffectiveListFilter, error) {
	filter := domain.CatalogEffectiveListFilter{
		PrimaryDomainID:   strings.TrimSpace(c.QueryParam("primary_domain_id")),
		SecondaryDomainID: strings.TrimSpace(c.QueryParam("secondary_domain_id")),
		SubdomainID:       strings.TrimSpace(c.QueryParam("subdomain_id")),
		TagIDs:            decodeCatalogTaxonomyCSVQueryParam(c.QueryParam("tag_ids")),
	}

	tagMatchRaw := strings.TrimSpace(c.QueryParam("tag_match"))
	if tagMatchRaw == "" {
		return filter, nil
	}

	tagMatch := domain.CatalogTagMatchMode(strings.ToLower(tagMatchRaw))
	if !tagMatch.IsValid() {
		return domain.CatalogEffectiveListFilter{}, fmt.Errorf(
			"query parameter %q must be one of: any, all",
			"tag_match",
		)
	}
	filter.TagMatch = tagMatch
	return filter, nil
}

func hasCatalogTaxonomyListFilterConstraints(filter domain.CatalogEffectiveListFilter) bool {
	return strings.TrimSpace(filter.PrimaryDomainID) != "" ||
		strings.TrimSpace(filter.SecondaryDomainID) != "" ||
		strings.TrimSpace(filter.SubdomainID) != "" ||
		len(filter.TagIDs) > 0 ||
		strings.TrimSpace(string(filter.TagMatch)) != ""
}

func decodeCatalogTaxonomyBoolQueryParam(
	raw string,
	field string,
) (*bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseBool(trimmed)
	if err != nil {
		return nil, fmt.Errorf("query parameter %q must be a boolean", field)
	}

	return &parsed, nil
}

func decodeCatalogTaxonomyCSVQueryParam(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	parts := strings.Split(trimmed, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		result = append(result, candidate)
	}

	return result
}

func hasCatalogTaxonomyDomainUpdateValues(request CatalogTaxonomyDomainUpdateRequest) bool {
	return request.Key != nil || request.Name != nil || request.Description != nil || request.Active != nil
}

func hasCatalogTaxonomySubdomainUpdateValues(request CatalogTaxonomySubdomainUpdateRequest) bool {
	return request.DomainID != nil ||
		request.Key != nil ||
		request.Name != nil ||
		request.Description != nil ||
		request.Active != nil
}

func hasCatalogTaxonomyTagUpdateValues(request CatalogTaxonomyTagUpdateRequest) bool {
	return request.Key != nil ||
		request.Name != nil ||
		request.Description != nil ||
		request.Color != nil ||
		request.Active != nil
}

func encodeCatalogTaxonomyServiceError(
	c *echo.Context,
	serviceErr error,
	notFoundSentinel error,
) error {
	switch {
	case notFoundSentinel != nil && errors.Is(serviceErr, notFoundSentinel):
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": serviceErr.Error(),
		})
	case errors.Is(serviceErr, domain.ErrCatalogTaxonomyValidation),
		errors.Is(serviceErr, domain.ErrCatalogTaxonomyInvalidRelationship):
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": serviceErr.Error(),
		})
	case errors.Is(serviceErr, domain.ErrCatalogTaxonomyConflict):
		return c.JSON(http.StatusConflict, map[string]string{
			"error": serviceErr.Error(),
		})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": serviceErr.Error(),
		})
	}
}

func encodeCatalogTaxonomyAssignmentServiceError(c *echo.Context, serviceErr error) error {
	switch {
	case errors.Is(serviceErr, domain.ErrCatalogTaxonomyAssignmentItemNotFound),
		errors.Is(serviceErr, domain.ErrCatalogTaxonomyDomainNotFound),
		errors.Is(serviceErr, domain.ErrCatalogTaxonomySubdomainNotFound),
		errors.Is(serviceErr, domain.ErrCatalogTaxonomyTagNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": serviceErr.Error(),
		})
	case errors.Is(serviceErr, domain.ErrCatalogTaxonomyValidation),
		errors.Is(serviceErr, domain.ErrCatalogTaxonomyInvalidRelationship):
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": serviceErr.Error(),
		})
	case errors.Is(serviceErr, domain.ErrCatalogTaxonomyConflict):
		return c.JSON(http.StatusConflict, map[string]string{
			"error": serviceErr.Error(),
		})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": serviceErr.Error(),
		})
	}
}

func decodeCatalogMetadataPatchRequest(c *echo.Context) (PatchCatalogMetadataRequest, error) {
	limitedReader := io.LimitReader(c.Request().Body, catalogMetadataPatchMaxBodyBytes+1)
	payload, err := io.ReadAll(limitedReader)
	if err != nil {
		return PatchCatalogMetadataRequest{}, fmt.Errorf("invalid request payload")
	}
	if len(payload) == 0 {
		return PatchCatalogMetadataRequest{}, fmt.Errorf("request body is required")
	}
	if len(payload) > catalogMetadataPatchMaxBodyBytes {
		return PatchCatalogMetadataRequest{}, fmt.Errorf("request payload exceeds %d bytes", catalogMetadataPatchMaxBodyBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var request PatchCatalogMetadataRequest
	if err := decoder.Decode(&request); err != nil {
		return PatchCatalogMetadataRequest{}, fmt.Errorf("invalid request payload")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return PatchCatalogMetadataRequest{}, fmt.Errorf("invalid request payload")
	}

	return request, nil
}

func normalizeCatalogMetadataPatchInput(
	itemID string,
	request PatchCatalogMetadataRequest,
) (domain.CatalogMetadataPatchInput, error) {
	normalized := domain.CatalogMetadataPatchInput{
		ItemID: itemID,
	}

	if request.DisplayName != nil {
		displayName := strings.TrimSpace(*request.DisplayName)
		if len(displayName) > catalogMetadataDisplayNameMaxChars {
			return domain.CatalogMetadataPatchInput{}, fmt.Errorf(
				"display_name must be <= %d characters",
				catalogMetadataDisplayNameMaxChars,
			)
		}
		normalized.DisplayNameOverride = &displayName
	}
	if request.Description != nil {
		description := strings.TrimSpace(*request.Description)
		if len(description) > catalogMetadataDescriptionMaxChars {
			return domain.CatalogMetadataPatchInput{}, fmt.Errorf(
				"description must be <= %d characters",
				catalogMetadataDescriptionMaxChars,
			)
		}
		normalized.DescriptionOverride = &description
	}
	if request.UpdatedBy != nil {
		updatedBy := strings.TrimSpace(*request.UpdatedBy)
		if updatedBy != "" {
			normalized.UpdatedBy = &updatedBy
		}
	}
	if request.Labels != nil {
		labels, err := normalizeCatalogMetadataLabels(*request.Labels)
		if err != nil {
			return domain.CatalogMetadataPatchInput{}, err
		}
		normalized.Labels = &labels
	}
	if request.CustomMetadata != nil {
		customMetadata, err := normalizeCatalogMetadataMap(*request.CustomMetadata)
		if err != nil {
			return domain.CatalogMetadataPatchInput{}, err
		}
		normalized.CustomMetadata = &customMetadata
	}

	if normalized.DisplayNameOverride == nil &&
		normalized.DescriptionOverride == nil &&
		normalized.Labels == nil &&
		normalized.CustomMetadata == nil {
		return domain.CatalogMetadataPatchInput{}, fmt.Errorf(
			"at least one of display_name, description, labels, or custom_metadata is required",
		)
	}

	return normalized, nil
}

func normalizeCatalogMetadataLabels(rawLabels []string) ([]string, error) {
	if len(rawLabels) > catalogMetadataMaxLabels {
		return nil, fmt.Errorf("labels must include <= %d entries", catalogMetadataMaxLabels)
	}

	labels := make([]string, 0, len(rawLabels))
	seen := make(map[string]struct{}, len(rawLabels))
	for _, rawLabel := range rawLabels {
		label := strings.TrimSpace(rawLabel)
		if label == "" {
			return nil, fmt.Errorf("labels cannot contain empty values")
		}
		if len(label) > catalogMetadataLabelMaxChars {
			return nil, fmt.Errorf("labels entries must be <= %d characters", catalogMetadataLabelMaxChars)
		}

		key := strings.ToLower(label)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		labels = append(labels, label)
	}

	return labels, nil
}

func normalizeCatalogMetadataMap(raw map[string]any) (map[string]any, error) {
	if len(raw) > catalogMetadataCustomMetadataMaxKeys {
		return nil, fmt.Errorf("custom_metadata must include <= %d top-level keys", catalogMetadataCustomMetadataMaxKeys)
	}

	normalized := make(map[string]any, len(raw))
	for key, value := range raw {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			return nil, fmt.Errorf("custom_metadata keys cannot be empty")
		}
		if len(normalizedKey) > catalogMetadataCustomMetadataMaxKeyLen {
			return nil, fmt.Errorf(
				"custom_metadata keys must be <= %d characters",
				catalogMetadataCustomMetadataMaxKeyLen,
			)
		}
		if err := validateCatalogMetadataValue(value, 1); err != nil {
			return nil, err
		}
		normalized[normalizedKey] = value
	}

	return normalized, nil
}

func validateCatalogMetadataValue(value any, depth int) error {
	if depth > catalogMetadataCustomMetadataMaxDepth {
		return fmt.Errorf("custom_metadata nesting exceeds max depth %d", catalogMetadataCustomMetadataMaxDepth)
	}

	switch typed := value.(type) {
	case nil, bool, float64:
		return nil
	case string:
		if len(typed) > catalogMetadataCustomMetadataMaxString {
			return fmt.Errorf(
				"custom_metadata string values must be <= %d characters",
				catalogMetadataCustomMetadataMaxString,
			)
		}
		return nil
	case map[string]any:
		if len(typed) > catalogMetadataCustomMetadataMaxKeys {
			return fmt.Errorf(
				"custom_metadata objects must include <= %d keys",
				catalogMetadataCustomMetadataMaxKeys,
			)
		}
		for key, nested := range typed {
			normalizedKey := strings.TrimSpace(key)
			if normalizedKey == "" {
				return fmt.Errorf("custom_metadata keys cannot be empty")
			}
			if len(normalizedKey) > catalogMetadataCustomMetadataMaxKeyLen {
				return fmt.Errorf(
					"custom_metadata keys must be <= %d characters",
					catalogMetadataCustomMetadataMaxKeyLen,
				)
			}
			if err := validateCatalogMetadataValue(nested, depth+1); err != nil {
				return err
			}
		}
		return nil
	case []any:
		if len(typed) > catalogMetadataCustomMetadataMaxArray {
			return fmt.Errorf(
				"custom_metadata arrays must include <= %d entries",
				catalogMetadataCustomMetadataMaxArray,
			)
		}
		for _, entry := range typed {
			if err := validateCatalogMetadataValue(entry, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("custom_metadata includes unsupported value types")
	}
}

func catalogMetadataResponseFromView(view domain.CatalogMetadataView) CatalogMetadataResponse {
	response := CatalogMetadataResponse{
		ItemID: view.ItemID,
		Source: CatalogMetadataSourceResponse{
			ItemID:           view.Source.ItemID,
			Classifier:       view.Source.Classifier,
			SourceType:       string(view.Source.SourceType),
			SourceRepo:       view.Source.SourceRepo,
			ParentSkillID:    view.Source.ParentSkillID,
			ResourcePath:     view.Source.ResourcePath,
			Name:             view.Source.Name,
			Description:      view.Source.Description,
			ContentWritable:  view.Source.ContentWritable,
			MetadataWritable: view.Source.MetadataWritable,
			ReadOnly:         view.Source.ReadOnly,
		},
		Overlay: CatalogMetadataOverlayResponse{
			DisplayName:    view.Overlay.DisplayNameOverride,
			Description:    view.Overlay.DescriptionOverride,
			CustomMetadata: view.Overlay.CustomMetadata,
			Labels:         view.Overlay.Labels,
			UpdatedBy:      view.Overlay.UpdatedBy,
		},
		Effective: CatalogMetadataEffectiveResponse{
			Name:               view.Effective.Name,
			Description:        view.Effective.Description,
			PrimaryDomain:      cloneCatalogTaxonomyReference(view.Effective.PrimaryDomain),
			PrimarySubdomain:   cloneCatalogTaxonomyReference(view.Effective.PrimarySubdomain),
			SecondaryDomain:    cloneCatalogTaxonomyReference(view.Effective.SecondaryDomain),
			SecondarySubdomain: cloneCatalogTaxonomyReference(view.Effective.SecondarySubdomain),
			Tags:               cloneCatalogTaxonomyReferences(view.Effective.Tags),
			CustomMetadata:     view.Effective.CustomMetadata,
			Labels:             view.Effective.Labels,
			ContentWritable:    view.Effective.ContentWritable,
			MetadataWritable:   view.Effective.MetadataWritable,
			ReadOnly:           view.Effective.ReadOnly,
		},
	}

	if view.Overlay.UpdatedAt != nil {
		formatted := view.Overlay.UpdatedAt.UTC().Format(time.RFC3339)
		response.Overlay.UpdatedAt = &formatted
	}
	if response.Overlay.CustomMetadata == nil {
		response.Overlay.CustomMetadata = map[string]any{}
	}
	if response.Overlay.Labels == nil {
		response.Overlay.Labels = []string{}
	}
	if response.Effective.CustomMetadata == nil {
		response.Effective.CustomMetadata = map[string]any{}
	}
	if response.Effective.Tags == nil {
		response.Effective.Tags = []domain.CatalogTaxonomyReference{}
	}
	if response.Effective.Labels == nil {
		response.Effective.Labels = []string{}
	}

	return response
}

// Resource management handlers

// listSkillResources lists all resources in a skill
func (s *Server) listSkillResources(c *echo.Context) error {
	skillName := skillNameFromRoute(c)

	// Check if skill exists
	skill, err := s.skillManager.ReadSkill(skillName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "skill not found",
		})
	}

	resources, err := s.skillManager.ListSkillResources(skill.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Group resources by type/origin while preserving legacy buckets.
	scripts := []map[string]any{}
	references := []map[string]any{}
	assets := []map[string]any{}
	prompts := []map[string]any{}
	imported := []map[string]any{}

	for _, res := range resources {
		origin := string(res.Origin)
		if origin == "" {
			origin = string(domain.ResourceOriginDirect)
		}

		resourceMap := map[string]any{
			"path":      res.Path,
			"name":      res.Name,
			"size":      res.Size,
			"mime_type": res.MimeType,
			"readable":  res.Readable,
			"origin":    origin,
			"writable":  res.Writable,
			"modified":  res.Modified.Format(time.RFC3339),
		}

		switch res.Type {
		case domain.ResourceTypeScript:
			scripts = append(scripts, resourceMap)
		case domain.ResourceTypeReference:
			references = append(references, resourceMap)
		case domain.ResourceTypePrompt:
			prompts = append(prompts, resourceMap)
		case domain.ResourceTypeAsset:
			assets = append(assets, resourceMap)
		}

		if origin == string(domain.ResourceOriginImported) {
			imported = append(imported, resourceMap)
		}
	}

	response := map[string]any{
		"scripts":    scripts,
		"references": references,
		"assets":     assets,
		"readOnly":   skill.ReadOnly,
		"groups": map[string]any{
			"scripts":    scripts,
			"references": references,
			"assets":     assets,
		},
	}
	if len(prompts) > 0 {
		response["prompts"] = prompts
		response["groups"].(map[string]any)["prompts"] = prompts
	}
	if len(imported) > 0 {
		response["imported"] = imported
		response["groups"].(map[string]any)["imported"] = imported
	}

	return c.JSON(http.StatusOK, response)
}

// getSkillResource gets a specific resource file
func (s *Server) getSkillResource(c *echo.Context) error {
	skillName := skillNameFromRoute(c)
	resourcePath := c.Param("*")

	if resourcePath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "resource path is required",
		})
	}

	// Check if skill exists
	skill, err := s.skillManager.ReadSkill(skillName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "skill not found",
		})
	}

	// Get resource info first
	info, err := s.skillManager.GetSkillResourceInfo(skill.ID, resourcePath)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "resource not found",
		})
	}

	// Read resource content
	content, err := s.skillManager.ReadSkillResource(skill.ID, resourcePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Check if client wants base64 encoding
	encoding := c.QueryParam("encoding")
	if encoding == "base64" || !info.Readable {
		return c.JSON(http.StatusOK, map[string]any{
			"content":   content.Content,
			"encoding":  content.Encoding,
			"mime_type": content.MimeType,
			"size":      content.Size,
		})
	}

	// For text files, return as plain text
	c.Response().Header().Set("Content-Type", content.MimeType)
	return c.String(http.StatusOK, content.Content)
}

// createSkillResource creates/uploads a new resource
func (s *Server) createSkillResource(c *echo.Context) error {
	skillName := skillNameFromRoute(c)

	// Check if skill exists and is not read-only
	skill, err := s.skillManager.ReadSkill(skillName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "skill not found",
		})
	}
	if skill.ReadOnly {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "cannot create resources in read-only skill from git repository",
		})
	}

	fsManager, ok := s.skillManager.(*domain.FileSystemManager)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "unsupported manager type",
		})
	}

	// Check Content-Type to determine if it's multipart/form-data or JSON
	contentType := c.Request().Header.Get("Content-Type")

	var resourcePath string
	var fileContent []byte

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Handle file upload
		file, err := c.FormFile("file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "file is required",
			})
		}

		resourceType := c.FormValue("type")
		if resourceType == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "type is required (script, reference, or asset)",
			})
		}

		pathValue := c.FormValue("path")
		if pathValue != "" {
			resourcePath = pathValue
		} else {
			resourcePath = resourceType + "s/" + file.Filename
		}

		// Validate path starts with correct directory
		if !strings.HasPrefix(resourcePath, resourceType+"s/") {
			resourcePath = resourceType + "s/" + file.Filename
		}

		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to open uploaded file",
			})
		}
		defer src.Close()

		fileContent, err = io.ReadAll(src)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to read uploaded file",
			})
		}
	} else {
		// Handle JSON request for text files
		var req struct {
			Type    string `json:"type"`
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request",
			})
		}

		resourcePath = req.Path
		if req.Type != "" && !strings.HasPrefix(resourcePath, req.Type+"/") {
			resourcePath = req.Type + "/" + resourcePath
		}
		fileContent = []byte(req.Content)
	}

	if domain.IsImportedResourcePath(resourcePath) {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "cannot create imported read-only resources",
		})
	}

	// Validate path
	if err := domain.ValidateResourcePath(resourcePath); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Check size limit (10MB)
	const maxFileSize = 10 * 1024 * 1024
	if len(fileContent) > maxFileSize {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("file too large (max %d bytes)", maxFileSize),
		})
	}

	// Write file
	skillsDir := fsManager.GetSkillsDir()
	skillDir := filepath.Join(skillsDir, skillName)
	fullPath := filepath.Join(skillDir, resourcePath)

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to create directory: %v", err),
		})
	}

	if err := os.WriteFile(fullPath, fileContent, 0644); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Return resource info
	info, err := s.skillManager.GetSkillResourceInfo(skill.ID, resourcePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to read created resource",
		})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"path":      info.Path,
		"name":      info.Name,
		"size":      info.Size,
		"mime_type": info.MimeType,
		"readable":  info.Readable,
		"origin":    string(info.Origin),
		"writable":  info.Writable,
		"modified":  info.Modified.Format(time.RFC3339),
	})
}

// updateSkillResource updates an existing resource
func (s *Server) updateSkillResource(c *echo.Context) error {
	skillName := skillNameFromRoute(c)
	resourcePath := c.Param("*")

	if resourcePath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "resource path is required",
		})
	}

	// Check if skill exists and is not read-only
	skill, err := s.skillManager.ReadSkill(skillName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "skill not found",
		})
	}
	if skill.ReadOnly {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "cannot update resources in read-only skill from git repository",
		})
	}
	if domain.IsImportedResourcePath(resourcePath) {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "cannot update imported read-only resources",
		})
	}

	// Validate path
	if err := domain.ValidateResourcePath(resourcePath); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Read request body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "failed to read request body",
		})
	}

	// Check size limit
	const maxFileSize = 10 * 1024 * 1024
	if len(body) > maxFileSize {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("file too large (max %d bytes)", maxFileSize),
		})
	}

	fsManager, ok := s.skillManager.(*domain.FileSystemManager)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "unsupported manager type",
		})
	}

	// Write file
	skillsDir := fsManager.GetSkillsDir()
	skillDir := filepath.Join(skillsDir, skillName)
	fullPath := filepath.Join(skillDir, resourcePath)

	if err := os.WriteFile(fullPath, body, 0644); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Return resource info
	info, err := s.skillManager.GetSkillResourceInfo(skill.ID, resourcePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to read updated resource",
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"path":      info.Path,
		"name":      info.Name,
		"size":      info.Size,
		"mime_type": info.MimeType,
		"readable":  info.Readable,
		"origin":    string(info.Origin),
		"writable":  info.Writable,
		"modified":  info.Modified.Format(time.RFC3339),
	})
}

// deleteSkillResource deletes a resource
func (s *Server) deleteSkillResource(c *echo.Context) error {
	skillName := skillNameFromRoute(c)
	resourcePath := c.Param("*")

	if resourcePath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "resource path is required",
		})
	}

	// Check if skill exists and is not read-only
	skill, err := s.skillManager.ReadSkill(skillName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "skill not found",
		})
	}
	if skill.ReadOnly {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "cannot delete resources from read-only skill from git repository",
		})
	}
	if domain.IsImportedResourcePath(resourcePath) {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "cannot delete imported read-only resources",
		})
	}

	// Validate path
	if err := domain.ValidateResourcePath(resourcePath); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	fsManager, ok := s.skillManager.(*domain.FileSystemManager)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "unsupported manager type",
		})
	}

	// Delete file
	skillsDir := fsManager.GetSkillsDir()
	skillDir := filepath.Join(skillsDir, skillName)
	fullPath := filepath.Join(skillDir, resourcePath)

	if err := os.Remove(fullPath); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// exportSkill exports a skill as a compressed archive
func (s *Server) exportSkill(c *echo.Context) error {
	// Get skill name from wildcard path (handles names with slashes like "repoName/skillName")
	// The route is /skills/export/*, so * captures the skill name
	name := c.Param("*")
	// Remove leading slash if present
	name = strings.TrimPrefix(name, "/")
	// URL decode the name (Echo should do this automatically, but be explicit)
	if decoded, err := url.PathUnescape(name); err == nil {
		name = decoded
	}

	// Check if skill exists
	skill, err := s.skillManager.ReadSkill(name)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "skill not found",
		})
	}

	// Get the skills directory from the manager
	fsManager, ok := s.skillManager.(*domain.FileSystemManager)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "unsupported manager type",
		})
	}

	// Create archive
	archiveData, err := domain.ExportSkill(skill.ID, fsManager.GetSkillsDir())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to create archive: %v", err),
		})
	}

	// Set headers for file download
	c.Response().Header().Set("Content-Type", "application/gzip")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.tar.gz\"", name))
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", len(archiveData)))

	return c.Blob(http.StatusOK, "application/gzip", archiveData)
}

// importSkill imports a skill from a compressed archive
func (s *Server) importSkill(c *echo.Context) error {
	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "file is required",
		})
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "failed to open uploaded file",
		})
	}
	defer src.Close()

	// Read file content
	const maxArchiveSize = 50 * 1024 * 1024 // 50MB limit
	archiveData := make([]byte, file.Size)
	if file.Size > maxArchiveSize {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("archive too large (max %d bytes)", maxArchiveSize),
		})
	}

	n, err := io.ReadFull(src, archiveData)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "failed to read uploaded file",
		})
	}
	archiveData = archiveData[:n]

	// Get the skills directory from the manager
	fsManager, ok := s.skillManager.(*domain.FileSystemManager)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "unsupported manager type",
		})
	}

	// Import skill
	skillName, err := domain.ImportSkill(archiveData, fsManager.GetSkillsDir())
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Rebuild index
	if err := s.skillManager.RebuildIndex(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to rebuild index",
		})
	}

	// Read the imported skill
	skill, err := s.skillManager.ReadSkill(skillName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to read imported skill",
		})
	}

	response := SkillResponse{
		Name:     skill.Name,
		Content:  skill.Content,
		ReadOnly: skill.ReadOnly,
	}
	if skill.Metadata != nil {
		response.Description = skill.Metadata.Description
		response.License = skill.Metadata.License
		response.Compatibility = skill.Metadata.Compatibility
		response.Metadata = skill.Metadata.Metadata
		response.AllowedTools = skill.Metadata.AllowedTools
	}

	return c.JSON(http.StatusCreated, response)
}

// Git repository management handlers

// GitRepoResponse represents a git repository in API responses
type GitRepoResponse struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// AddGitRepoRequest represents a request to add a git repository
type AddGitRepoRequest struct {
	URL string `json:"url"`
}

// UpdateGitRepoRequest represents a request to update a git repository
type UpdateGitRepoRequest struct {
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// listGitRepos lists all configured git repositories
func (s *Server) listGitRepos(c *echo.Context) error {
	if s.configManager == nil {
		return c.JSON(http.StatusOK, []GitRepoResponse{})
	}

	// Load repos from config file
	configRepos, err := s.configManager.LoadConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to load config: %v", err),
		})
	}

	// Convert to response format
	repos := make([]GitRepoResponse, len(configRepos))
	for i, repo := range configRepos {
		repos[i] = GitRepoResponse{
			ID:      repo.ID,
			URL:     repo.URL,
			Name:    repo.Name,
			Enabled: repo.Enabled,
		}
	}

	return c.JSON(http.StatusOK, repos)
}

// addGitRepo adds a new git repository
func (s *Server) addGitRepo(c *echo.Context) error {
	if s.gitSyncer == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "git syncer not available",
		})
	}

	var req AddGitRepoRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	if req.URL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "URL is required",
		})
	}

	// Validate URL format (basic check)
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") && !strings.HasPrefix(req.URL, "git@") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid URL format",
		})
	}

	// Load current config
	configRepos, err := s.configManager.LoadConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to load config: %v", err),
		})
	}

	// Check if repo already exists
	for _, repo := range configRepos {
		if repo.URL == req.URL {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "repository already exists",
			})
		}
	}

	// Add new repo to config (enabled by default)
	newRepo := git.GitRepoConfig{
		ID:      git.GenerateID(req.URL),
		URL:     req.URL,
		Name:    git.ExtractRepoName(req.URL),
		Enabled: true,
	}
	configRepos = append(configRepos, newRepo)

	// Save config
	if err := s.configManager.SaveConfig(configRepos); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to save config: %v", err),
		})
	}

	// Add repo to syncer and update FileSystemManager
	if s.gitSyncer != nil {
		if err := s.gitSyncer.AddRepo(req.URL); err != nil {
			// Remove from config if sync failed
			for i, repo := range configRepos {
				if repo.URL == req.URL {
					configRepos = append(configRepos[:i], configRepos[i+1:]...)
					s.configManager.SaveConfig(configRepos)
					break
				}
			}
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}

		// Update FileSystemManager's git repos list for read-only detection
		if s.fsManager != nil {
			enabledRepos := make([]string, 0)
			for _, repo := range configRepos {
				if repo.Enabled {
					enabledRepos = append(enabledRepos, repo.URL)
				}
			}
			gitRepoNames := make([]string, len(enabledRepos))
			for i, url := range enabledRepos {
				gitRepoNames[i] = git.ExtractRepoName(url)
			}
			s.fsManager.UpdateGitRepos(gitRepoNames)
		}

		// Ensure the catalog index reflects the newly enabled repo set.
		if err := s.skillManager.RebuildIndex(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to rebuild index",
			})
		}
	}

	response := GitRepoResponse{
		ID:      git.GenerateID(req.URL),
		URL:     req.URL,
		Name:    git.ExtractRepoName(req.URL),
		Enabled: true,
	}

	return c.JSON(http.StatusCreated, response)
}

// updateGitRepo updates a git repository
func (s *Server) updateGitRepo(c *echo.Context) error {
	if s.gitSyncer == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "git syncer not available",
		})
	}

	id := c.Param("id")

	var req UpdateGitRepoRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	// Find repo by ID
	repos := s.gitSyncer.GetRepos()
	var foundURL string
	for _, url := range repos {
		if git.GenerateID(url) == id {
			foundURL = url
			break
		}
	}

	if foundURL == "" {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "repository not found",
		})
	}

	// If URL changed, remove old and add new
	if req.URL != "" && req.URL != foundURL {
		if err := s.gitSyncer.RemoveRepo(foundURL); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
		if err := s.gitSyncer.AddRepo(req.URL); err != nil {
			// Try to restore old repo on error
			s.gitSyncer.AddRepo(foundURL)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}
		foundURL = req.URL
	}

	// Update FileSystemManager's git repos list for read-only detection
	if s.fsManager != nil {
		repos := s.gitSyncer.GetRepos()
		gitRepoNames := make([]string, len(repos))
		for i, url := range repos {
			gitRepoNames[i] = git.ExtractRepoName(url)
		}
		s.fsManager.UpdateGitRepos(gitRepoNames)
	}

	// Save config
	if s.configManager != nil {
		repos := s.gitSyncer.GetRepos()
		configs := make([]git.GitRepoConfig, len(repos))
		for i, url := range repos {
			configs[i] = git.GitRepoConfig{
				ID:      git.GenerateID(url),
				URL:     url,
				Name:    git.ExtractRepoName(url),
				Enabled: true,
			}
		}
		if err := s.configManager.SaveConfig(configs); err != nil {
			fmt.Printf("Warning: failed to save config: %v\n", err)
		}
	}

	// Load current config
	configRepos, err := s.configManager.LoadConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to load config: %v", err),
		})
	}

	// Update enabled status for the repo
	for i := range configRepos {
		if configRepos[i].ID == id {
			configRepos[i].Enabled = req.Enabled
			break
		}
	}

	// Save updated config
	if err := s.configManager.SaveConfig(configRepos); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to save config: %v", err),
		})
	}

	// Update syncer and FileSystemManager based on enabled repos
	if s.gitSyncer != nil {
		enabledRepos := make([]string, 0)
		for _, repo := range configRepos {
			if repo.Enabled {
				enabledRepos = append(enabledRepos, repo.URL)
			}
		}

		// Update syncer repos
		if err := s.gitSyncer.UpdateRepos(enabledRepos); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("failed to update syncer: %v", err),
			})
		}

		// Update FileSystemManager's git repos list
		if s.fsManager != nil {
			gitRepoNames := make([]string, len(enabledRepos))
			for i, url := range enabledRepos {
				gitRepoNames[i] = git.ExtractRepoName(url)
			}
			s.fsManager.UpdateGitRepos(gitRepoNames)
		}

		// Rebuild index
		if err := s.skillManager.RebuildIndex(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to rebuild index",
			})
		}
	}

	// Find updated repo for response
	var response GitRepoResponse
	for _, repo := range configRepos {
		if repo.ID == id {
			response = GitRepoResponse{
				ID:      repo.ID,
				URL:     repo.URL,
				Name:    repo.Name,
				Enabled: repo.Enabled,
			}
			break
		}
	}

	return c.JSON(http.StatusOK, response)
}

// deleteGitRepo deletes a git repository
func (s *Server) deleteGitRepo(c *echo.Context) error {
	if s.gitSyncer == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "git syncer not available",
		})
	}

	id := c.Param("id")

	if s.configManager == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "config manager not available",
		})
	}

	// Load current config
	configRepos, err := s.configManager.LoadConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to load config: %v", err),
		})
	}

	// Find repo by ID
	var foundRepo *git.GitRepoConfig
	for _, repo := range configRepos {
		if repo.ID == id {
			foundRepo = &repo
			break
		}
	}

	if foundRepo == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "repository not found",
		})
	}

	// Get repo name to delete the directory
	repoName := foundRepo.Name
	foundURL := foundRepo.URL

	// Remove repo from config (we already have configRepos loaded above)
	updatedConfigs := make([]git.GitRepoConfig, 0, len(configRepos)-1)
	for _, repo := range configRepos {
		if repo.ID != id {
			updatedConfigs = append(updatedConfigs, repo)
		}
	}

	// Save updated config
	if err := s.configManager.SaveConfig(updatedConfigs); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to save config: %v", err),
		})
	}

	// Remove repo from syncer
	if err := s.gitSyncer.RemoveRepo(foundURL); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Delete the repository directory and all its contents
	skillsDir := s.gitSyncer.GetSkillsDir()
	repoDir := filepath.Join(skillsDir, repoName)
	if err := os.RemoveAll(repoDir); err != nil {
		// Log error but don't fail the request - repo is already removed from config
		fmt.Printf("Warning: failed to delete repository directory %s: %v\n", repoDir, err)
	}

	// Update FileSystemManager's git repos list for read-only detection
	if s.fsManager != nil {
		enabledRepos := make([]string, 0)
		for _, repo := range updatedConfigs {
			if repo.Enabled {
				enabledRepos = append(enabledRepos, repo.URL)
			}
		}
		gitRepoNames := make([]string, len(enabledRepos))
		for i, url := range enabledRepos {
			gitRepoNames[i] = git.ExtractRepoName(url)
		}
		s.fsManager.UpdateGitRepos(gitRepoNames)
	}

	// Trigger re-indexing
	if err := s.skillManager.RebuildIndex(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to rebuild index",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// syncGitRepo manually syncs a git repository
func (s *Server) syncGitRepo(c *echo.Context) error {
	if s.gitSyncer == nil || s.configManager == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "git syncer or config manager not available",
		})
	}

	id := c.Param("id")

	// Load config to find repo
	configRepos, err := s.configManager.LoadConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to load config: %v", err),
		})
	}

	// Find repo by ID
	var foundRepo *git.GitRepoConfig
	for i := range configRepos {
		if configRepos[i].ID == id {
			foundRepo = &configRepos[i]
			break
		}
	}

	if foundRepo == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "repository not found",
		})
	}

	// Check if repo is enabled
	if !foundRepo.Enabled {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "cannot sync disabled repository",
		})
	}

	// Sync the repo
	if err := s.gitSyncer.SyncRepo(foundRepo.URL); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	if s.manualRepoSyncHook != nil {
		if err := s.manualRepoSyncHook(*foundRepo); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
	} else if err := s.skillManager.RebuildIndex(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to rebuild index",
		})
	}

	response := GitRepoResponse{
		ID:      foundRepo.ID,
		URL:     foundRepo.URL,
		Name:    foundRepo.Name,
		Enabled: foundRepo.Enabled,
	}

	return c.JSON(http.StatusOK, response)
}

// toggleGitRepo toggles the enabled status of a git repository
func (s *Server) toggleGitRepo(c *echo.Context) error {
	if s.configManager == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "config manager not available",
		})
	}

	id := c.Param("id")

	// Load current config
	configRepos, err := s.configManager.LoadConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to load config: %v", err),
		})
	}

	// Find and toggle the repo
	var foundRepo *git.GitRepoConfig
	for i := range configRepos {
		if configRepos[i].ID == id {
			configRepos[i].Enabled = !configRepos[i].Enabled
			foundRepo = &configRepos[i]
			break
		}
	}

	if foundRepo == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "repository not found",
		})
	}

	// Save updated config
	if err := s.configManager.SaveConfig(configRepos); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to save config: %v", err),
		})
	}

	// Update syncer and FileSystemManager based on enabled repos
	if s.gitSyncer != nil {
		enabledRepos := make([]string, 0)
		for _, repo := range configRepos {
			if repo.Enabled {
				enabledRepos = append(enabledRepos, repo.URL)
			}
		}

		// Update syncer repos
		if err := s.gitSyncer.UpdateRepos(enabledRepos); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("failed to update syncer: %v", err),
			})
		}

		// Update FileSystemManager's git repos list
		if s.fsManager != nil {
			gitRepoNames := make([]string, len(enabledRepos))
			for i, url := range enabledRepos {
				gitRepoNames[i] = git.ExtractRepoName(url)
			}
			s.fsManager.UpdateGitRepos(gitRepoNames)
		}

		// Rebuild index
		if err := s.skillManager.RebuildIndex(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to rebuild index",
			})
		}
	}

	response := GitRepoResponse{
		ID:      foundRepo.ID,
		URL:     foundRepo.URL,
		Name:    foundRepo.Name,
		Enabled: foundRepo.Enabled,
	}

	return c.JSON(http.StatusOK, response)
}

// Helper functions

func writeFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}

func deleteFile(filename string) error {
	return os.Remove(filename)
}
