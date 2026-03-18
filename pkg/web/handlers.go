package web

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/mudler/skillserver/pkg/domain"
	"github.com/mudler/skillserver/pkg/git"
	"github.com/mudler/skillserver/pkg/persistence"
)

// SkillResponse represents a skill in API responses
type SkillResponse struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Content       string            `json:"content"`
	Description   string            `json:"description,omitempty"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  string            `json:"allowed-tools,omitempty"`
	SourceRepo    string            `json:"sourceRepo,omitempty"`
	SourcePath    string            `json:"sourcePath,omitempty"`
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
	SourceRepo         string                            `json:"source_repo,omitempty"`
	SourcePath         string                            `json:"source_path,omitempty"`
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
	HasAssignment      bool                              `json:"has_assignment"`
	IsFullyClassified  bool                              `json:"is_fully_classified"`
	MissingFields      []string                          `json:"missing_fields"`
}

// CatalogCollectionResponse represents paginated catalog list/search payloads.
type CatalogCollectionResponse struct {
	Items      []CatalogItemResponse `json:"items"`
	NextCursor *string               `json:"next_cursor,omitempty"`
	HasMore    bool                  `json:"has_more"`
}

// PatchCatalogMetadataRequest represents one metadata overlay mutation request.
type PatchCatalogMetadataRequest struct {
	DisplayName    *string         `json:"display_name"`
	Description    *string         `json:"description"`
	Labels         *[]string       `json:"labels"`
	CustomMetadata *map[string]any `json:"custom_metadata"`
	UpdatedBy      *string         `json:"updated_by,omitempty"`
}

// PatchCatalogRelationshipsRequest represents one relationship mutation request.
type PatchCatalogRelationshipsRequest struct {
	PromptItemID catalogOptionalStringField      `json:"prompt_item_id,omitempty"`
	RuleItemIDs  catalogOptionalStringSliceField `json:"rule_item_ids,omitempty"`
	UpdatedBy    *string                         `json:"updated_by,omitempty"`
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
	AddTagIDs            *[]string `json:"add_tag_ids,omitempty"`
	RemoveTagIDs         *[]string `json:"remove_tag_ids,omitempty"`
	ClearTags            *bool     `json:"clear_tags,omitempty"`
	UpdatedBy            *string   `json:"updated_by,omitempty"`
}

// PatchCatalogItemsTaxonomyRequest describes batch taxonomy assignment mutations.
type PatchCatalogItemsTaxonomyRequest struct {
	Items  []PatchCatalogItemsTaxonomyItemRequest `json:"items"`
	DryRun bool                                   `json:"dry_run,omitempty"`
}

// PatchCatalogItemsTaxonomyItemRequest describes one batch taxonomy assignment mutation.
type PatchCatalogItemsTaxonomyItemRequest struct {
	ItemID               string    `json:"item_id"`
	PrimaryDomainID      *string   `json:"primary_domain_id,omitempty"`
	PrimarySubdomainID   *string   `json:"primary_subdomain_id,omitempty"`
	SecondaryDomainID    *string   `json:"secondary_domain_id,omitempty"`
	SecondarySubdomainID *string   `json:"secondary_subdomain_id,omitempty"`
	TagIDs               *[]string `json:"tag_ids,omitempty"`
	AddTagIDs            *[]string `json:"add_tag_ids,omitempty"`
	RemoveTagIDs         *[]string `json:"remove_tag_ids,omitempty"`
	ClearTags            *bool     `json:"clear_tags,omitempty"`
	UpdatedBy            *string   `json:"updated_by,omitempty"`
}

// CatalogExportResponse represents JSON responses from POST /api/catalog/export.
type CatalogExportResponse struct {
	Format   domain.CatalogExportFormat   `json:"format"`
	DryRun   bool                         `json:"dry_run"`
	Manifest domain.CatalogExportManifest `json:"manifest"`
	Download *CatalogExportDownload       `json:"download,omitempty"`
}

// CatalogExportDownload represents archive metadata for non-dry-run export responses.
type CatalogExportDownload struct {
	FileName          string `json:"file_name"`
	ContentType       string `json:"content_type"`
	ContentLength     int    `json:"content_length"`
	LegacySkillExport string `json:"legacy_skill_export_url,omitempty"`
}

// CatalogMaterializeResponse represents JSON responses from POST /api/catalog/materialize.
type CatalogMaterializeResponse struct {
	DryRun                 bool                                      `json:"dry_run"`
	DestinationDir         string                                    `json:"destination_dir"`
	ResolvedDestinationDir string                                    `json:"resolved_destination_dir"`
	Items                  []domain.CatalogMaterializationItemResult `json:"items"`
}

// CatalogMetadataResponse represents source, overlay, and effective metadata views.
type CatalogMetadataResponse struct {
	ItemID        string                           `json:"item_id"`
	Source        CatalogMetadataSourceResponse    `json:"source"`
	Overlay       CatalogMetadataOverlayResponse   `json:"overlay"`
	Effective     CatalogMetadataEffectiveResponse `json:"effective"`
	Relationships CatalogRelationshipSetResponse   `json:"relationships"`
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
	HasAssignment      bool                              `json:"has_assignment"`
	IsFullyClassified  bool                              `json:"is_fully_classified"`
	MissingFields      []string                          `json:"missing_fields"`
}

// CatalogRelationshipsResponse represents one relationship view payload.
type CatalogRelationshipsResponse struct {
	ItemID        string                         `json:"item_id"`
	Relationships CatalogRelationshipSetResponse `json:"relationships"`
}

// CatalogRelationshipSetResponse captures one normalized prompt/rules/skills relationship envelope.
type CatalogRelationshipSetResponse struct {
	Prompt *CatalogRelationshipItemResponse  `json:"prompt"`
	Rules  []CatalogRelationshipItemResponse `json:"rules"`
	Skills []CatalogRelationshipItemResponse `json:"skills"`
}

// CatalogRelationshipItemResponse represents one related catalog item descriptor.
type CatalogRelationshipItemResponse struct {
	ID            string                   `json:"id"`
	Classifier    domain.CatalogClassifier `json:"classifier"`
	Name          string                   `json:"name"`
	ParentSkillID *string                  `json:"parent_skill_id,omitempty"`
	ResourcePath  *string                  `json:"resource_path,omitempty"`
}

// catalogOptionalStringField preserves field presence semantics for nullable string payload fields.
type catalogOptionalStringField struct {
	Set   bool
	Value *string
}

func (f *catalogOptionalStringField) UnmarshalJSON(data []byte) error {
	f.Set = true
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		f.Value = nil
		return nil
	}

	var value string
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return err
	}
	f.Value = &value
	return nil
}

// catalogOptionalStringSliceField preserves field presence semantics for []string payload fields.
type catalogOptionalStringSliceField struct {
	Set   bool
	Null  bool
	Value []string
}

func (f *catalogOptionalStringSliceField) UnmarshalJSON(data []byte) error {
	f.Set = true
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		f.Null = true
		f.Value = nil
		return nil
	}

	var value []string
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return err
	}
	f.Value = value
	return nil
}

type catalogListRequest struct {
	TaxonomyFilter       domain.CatalogEffectiveListFilter
	Classifier           *domain.CatalogClassifier
	IncludeContent       bool
	UseEnvelope          bool
	Limit                int
	Cursor               string
	Unclassified         bool
	MissingPrimaryDomain bool
	MissingTags          bool
}

type catalogTaxonomyConflictResponse struct {
	Error    string                               `json:"error"`
	Conflict *catalogTaxonomyConflictDetailResult `json:"conflict,omitempty"`
}

type catalogTaxonomyConflictDetailResult struct {
	ObjectType        domain.CatalogTaxonomyObjectType     `json:"object_type"`
	ObjectID          string                               `json:"object_id"`
	Reason            domain.CatalogTaxonomyConflictReason `json:"reason"`
	ReferencedItemIDs []string                             `json:"referenced_item_ids,omitempty"`
}

const (
	catalogExportRequestMaxBodyBytes        = 32 * 1024
	catalogMaterializeRequestMaxBodyBytes   = 32 * 1024
	catalogTaxonomyRequestMaxBodyBytes      = 32 * 1024
	catalogMetadataPatchMaxBodyBytes        = 32 * 1024
	catalogRelationshipPatchMaxBodyBytes    = 32 * 1024
	catalogMetadataDisplayNameMaxChars      = 256
	catalogMetadataDescriptionMaxChars      = 4096
	catalogMetadataMaxLabels                = 64
	catalogMetadataLabelMaxChars            = 64
	catalogMetadataCustomMetadataMaxKeys    = 128
	catalogMetadataCustomMetadataMaxDepth   = 6
	catalogMetadataCustomMetadataMaxArray   = 256
	catalogMetadataCustomMetadataMaxString  = 4096
	catalogMetadataCustomMetadataMaxKeyLen  = 128
	catalogListDefaultLimit                 = 50
	catalogListMaxLimit                     = 200
	catalogTaxonomyUsageDefaultPreviewLimit = 10
	catalogTaxonomyUsageMaxPreviewLimit     = 200
)

var errCatalogTaxonomyFiltersUnavailable = errors.New("catalog taxonomy filters are unavailable")

func catalogResponseFromItem(item domain.CatalogItem) CatalogItemResponse {
	return catalogResponseFromItemWithContent(item, true)
}

func catalogResponseFromItemWithContent(item domain.CatalogItem, includeContent bool) CatalogItemResponse {
	classificationState := deriveCatalogItemClassificationState(item)
	content := ""
	if includeContent {
		content = item.Content
	}

	return CatalogItemResponse{
		ID:                 item.ID,
		Classifier:         item.Classifier,
		Name:               item.Name,
		Description:        item.Description,
		Content:            content,
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
		HasAssignment:      classificationState.HasAssignment,
		IsFullyClassified:  classificationState.IsFullyClassified,
		MissingFields:      append([]string{}, classificationState.MissingFields...),
	}
}

type skillSourceDescriptor struct {
	repo string
	path string
}

func (s *Server) resolveSkillsDir() string {
	if s != nil && s.fsManager != nil {
		return strings.TrimSpace(s.fsManager.GetSkillsDir())
	}
	if s != nil {
		if fsManager, ok := s.skillManager.(*domain.FileSystemManager); ok && fsManager != nil {
			return strings.TrimSpace(fsManager.GetSkillsDir())
		}
	}
	return ""
}

func normalizeRelativeSourcePath(basePath, targetPath string) (string, bool) {
	basePath = strings.TrimSpace(basePath)
	targetPath = strings.TrimSpace(targetPath)
	if basePath == "" || targetPath == "" {
		return "", false
	}

	canonicalBasePath, err := filepath.Abs(basePath)
	if err != nil {
		return "", false
	}
	canonicalTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return "", false
	}

	relativePath, err := filepath.Rel(canonicalBasePath, canonicalTargetPath)
	if err != nil {
		return "", false
	}

	normalizedRelativePath := filepath.ToSlash(filepath.Clean(relativePath))
	if normalizedRelativePath == "." || normalizedRelativePath == "" || normalizedRelativePath == ".." ||
		strings.HasPrefix(normalizedRelativePath, "../") {
		return "", false
	}
	return normalizedRelativePath, true
}

func deriveSkillSourceDescriptor(skill domain.Skill, skillsDir string) skillSourceDescriptor {
	descriptor := skillSourceDescriptor{}
	skillID := strings.TrimSpace(skill.ID)
	if repoName, _, hasRepo := strings.Cut(skillID, "/"); hasRepo {
		descriptor.repo = strings.TrimSpace(repoName)
	}

	sourcePath := strings.TrimSpace(skill.SourcePath)
	if sourcePath == "" {
		return descriptor
	}

	if skillsDir != "" {
		if descriptor.repo != "" {
			if relativePath, ok := normalizeRelativeSourcePath(filepath.Join(skillsDir, descriptor.repo), sourcePath); ok {
				descriptor.path = relativePath
				return descriptor
			}
		}
		if relativePath, ok := normalizeRelativeSourcePath(skillsDir, sourcePath); ok {
			descriptor.path = relativePath
			return descriptor
		}
	}

	return descriptor
}

func (s *Server) skillResponseFromDomainSkill(skill domain.Skill) SkillResponse {
	response := SkillResponse{
		ID:       skill.ID,
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

	sourceDescriptor := deriveSkillSourceDescriptor(skill, s.resolveSkillsDir())
	response.SourceRepo = sourceDescriptor.repo
	response.SourcePath = sourceDescriptor.path
	return response
}

func associatedSkillIDForCatalogItem(item domain.CatalogItem) string {
	if parentSkillID := strings.TrimSpace(item.ParentSkillID); parentSkillID != "" {
		return parentSkillID
	}
	if strings.HasPrefix(item.ID, "skill:") {
		return strings.TrimSpace(strings.TrimPrefix(item.ID, "skill:"))
	}
	return ""
}

func (s *Server) catalogItemResponseFromDomainItem(
	item domain.CatalogItem,
	includeContent bool,
	sourceCache map[string]skillSourceDescriptor,
) CatalogItemResponse {
	response := catalogResponseFromItemWithContent(item, includeContent)
	if sourceCache == nil {
		sourceCache = map[string]skillSourceDescriptor{}
	}

	skillID := associatedSkillIDForCatalogItem(item)
	if skillID == "" {
		return response
	}

	if descriptor, ok := sourceCache[skillID]; ok {
		response.SourceRepo = descriptor.repo
		response.SourcePath = descriptor.path
		return response
	}

	skill, err := s.skillManager.ReadSkill(skillID)
	if err != nil || skill == nil {
		sourceCache[skillID] = skillSourceDescriptor{}
		return response
	}

	descriptor := deriveSkillSourceDescriptor(*skill, s.resolveSkillsDir())
	sourceCache[skillID] = descriptor
	response.SourceRepo = descriptor.repo
	response.SourcePath = descriptor.path
	return response
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
		responses[i] = s.skillResponseFromDomainSkill(skill)
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

	response := s.skillResponseFromDomainSkill(*skill)

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

	response := s.skillResponseFromDomainSkill(*skill)

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

	return c.JSON(http.StatusOK, s.skillResponseFromDomainSkill(*skill))
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
		responses[i] = s.skillResponseFromDomainSkill(skill)
	}

	return c.JSON(http.StatusOK, responses)
}

// listCatalog lists all catalog items (skills and prompts).
func (s *Server) listCatalog(c *echo.Context) error {
	request, err := decodeCatalogListRequest(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	if request.requiresMetadataRuntime() && s.catalogMetadataService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": errCatalogTaxonomyFiltersUnavailable.Error(),
		})
	}

	items, err := s.loadCatalogItems(c.Request().Context(), "", request.Classifier, request.TaxonomyFilter)
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

	return c.JSON(http.StatusOK, s.buildCatalogCollectionResponse(items, request))
}

// searchCatalog searches catalog items by query with an optional classifier filter.
func (s *Server) searchCatalog(c *echo.Context) error {
	query := strings.TrimSpace(c.QueryParam("q"))
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "query parameter 'q' is required",
		})
	}

	request, err := decodeCatalogListRequest(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	if request.requiresMetadataRuntime() && s.catalogMetadataService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": errCatalogTaxonomyFiltersUnavailable.Error(),
		})
	}

	items, err := s.loadCatalogItems(c.Request().Context(), query, request.Classifier, request.TaxonomyFilter)
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

	return c.JSON(http.StatusOK, s.buildCatalogCollectionResponse(items, request))
}

// getCatalogItem returns one catalog item by exact ID, including full content.
func (s *Server) getCatalogItem(c *echo.Context) error {
	itemID, err := decodeCatalogItemIDFromRequest(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	if _, err := domain.NormalizeCatalogItemID(itemID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	var metadataReader interface {
		List(ctx context.Context, filter domain.CatalogEffectiveListFilter) ([]domain.CatalogItem, error)
	}
	if s.catalogMetadataService != nil {
		metadataReader = s.catalogMetadataService
	}

	item, err := domain.GetCatalogItemByID(c.Request().Context(), itemID, s.skillManager, metadataReader)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCatalogItemNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "catalog item not found",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
	}

	return c.JSON(http.StatusOK, s.catalogItemResponseFromDomainItem(item, true, nil))
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
		items, err := s.skillManager.ListCatalogItems()
		if err != nil {
			return nil, err
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

func deriveCatalogItemClassificationState(item domain.CatalogItem) domain.CatalogClassificationState {
	return domain.DeriveCatalogClassificationState(
		item.PrimaryDomain,
		item.PrimarySubdomain,
		item.SecondaryDomain,
		item.SecondarySubdomain,
		item.Tags,
	)
}

func normalizeCatalogItemForResponse(item domain.CatalogItem) domain.CatalogItem {
	classificationState := deriveCatalogItemClassificationState(item)
	item.HasAssignment = classificationState.HasAssignment
	item.IsFullyClassified = classificationState.IsFullyClassified
	item.MissingFields = append([]string{}, classificationState.MissingFields...)
	return item
}

func (s *Server) buildCatalogCollectionResponse(items []domain.CatalogItem, request catalogListRequest) any {
	normalizedItems := make([]domain.CatalogItem, 0, len(items))
	for _, item := range items {
		normalizedItem := normalizeCatalogItemForResponse(item)
		if !request.matchesClassificationState(normalizedItem) {
			continue
		}
		normalizedItems = append(normalizedItems, normalizedItem)
	}

	sort.Slice(normalizedItems, func(i int, j int) bool {
		return normalizedItems[i].ID < normalizedItems[j].ID
	})

	pageItems := normalizedItems
	var (
		nextCursor *string
		hasMore    bool
	)
	if request.UseEnvelope {
		pageItems, nextCursor, hasMore = paginateCatalogItems(normalizedItems, request.Cursor, request.Limit)
	}

	responses := make([]CatalogItemResponse, len(pageItems))
	sourceCache := make(map[string]skillSourceDescriptor, len(pageItems))
	for i, item := range pageItems {
		responses[i] = s.catalogItemResponseFromDomainItem(item, request.IncludeContent, sourceCache)
	}

	if !request.UseEnvelope {
		return responses
	}

	return CatalogCollectionResponse{
		Items:      responses,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}

func paginateCatalogItems(
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

func (r catalogListRequest) matchesClassificationState(item domain.CatalogItem) bool {
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

func (r catalogListRequest) requiresMetadataRuntime() bool {
	return hasCatalogTaxonomyListFilterConstraints(r.TaxonomyFilter) ||
		r.Unclassified ||
		r.MissingPrimaryDomain ||
		r.MissingTags
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

	itemID, err := decodeCatalogItemIDFromRequest(c)
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

	itemID, err := decodeCatalogItemIDFromRequest(c)
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

// patchCatalogRelationships updates one skill-owned relationship projection.
func (s *Server) patchCatalogRelationships(c *echo.Context) error {
	if s.catalogRelationshipService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog relationship API is unavailable",
		})
	}

	itemID, err := decodeCatalogItemIDFromRequest(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	request, err := decodeCatalogRelationshipPatchRequest(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	input, err := normalizeCatalogRelationshipPatchInput(itemID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	view, err := s.catalogRelationshipService.Patch(c.Request().Context(), input)
	if err != nil {
		return encodeCatalogRelationshipServiceError(c, err)
	}

	return c.JSON(http.StatusOK, catalogRelationshipsResponseFromView(view))
}

// getCatalogItemTaxonomy returns one catalog item's taxonomy assignment state.
func (s *Server) getCatalogItemTaxonomy(c *echo.Context) error {
	if s.taxonomyAssignment == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy assignment API is unavailable",
		})
	}

	itemID, err := decodeCatalogItemIDFromRequest(c)
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

	itemID, err := decodeCatalogItemIDFromRequest(c)
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
		AddTagIDs:            request.AddTagIDs,
		RemoveTagIDs:         request.RemoveTagIDs,
		ClearTags:            request.ClearTags,
		UpdatedBy:            request.UpdatedBy,
	}

	assignment, err := s.taxonomyAssignment.Patch(c.Request().Context(), input)
	if err != nil {
		return encodeCatalogTaxonomyAssignmentServiceError(c, err)
	}

	return c.JSON(http.StatusOK, assignment)
}

// patchCatalogItemsTaxonomy patches multiple catalog item taxonomy assignments.
func (s *Server) patchCatalogItemsTaxonomy(c *echo.Context) error {
	if s.taxonomyAssignment == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy assignment API is unavailable",
		})
	}

	request, err := decodeCatalogTaxonomyRequest[PatchCatalogItemsTaxonomyRequest](c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	input := domain.CatalogItemTaxonomyBatchPatchRequest{
		DryRun: request.DryRun,
		Items:  make([]domain.CatalogItemTaxonomyAssignmentPatchInput, 0, len(request.Items)),
	}
	for _, item := range request.Items {
		input.Items = append(input.Items, domain.CatalogItemTaxonomyAssignmentPatchInput{
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

	result, err := s.taxonomyAssignment.PatchBatch(c.Request().Context(), input)
	if err != nil {
		return encodeCatalogTaxonomyAssignmentServiceError(c, err)
	}

	return c.JSON(http.StatusOK, result)
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

// getCatalogTaxonomyDomainUsage returns delete-preflight usage for one taxonomy domain.
func (s *Server) getCatalogTaxonomyDomainUsage(c *echo.Context) error {
	if s.taxonomyUsage == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy usage API is unavailable",
		})
	}

	domainID, err := decodeCatalogTaxonomyObjectIDFromPath(c.Param("id"), "domain_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	previewLimit, err := decodeCatalogTaxonomyUsagePreviewLimit(c.QueryParam("preview_limit"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	summary, err := s.taxonomyUsage.GetDomainUsage(c.Request().Context(), domainID, previewLimit)
	if err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, domain.ErrCatalogTaxonomyDomainNotFound)
	}

	return c.JSON(http.StatusOK, summary)
}

// getCatalogTaxonomySubdomainUsage returns delete-preflight usage for one taxonomy subdomain.
func (s *Server) getCatalogTaxonomySubdomainUsage(c *echo.Context) error {
	if s.taxonomyUsage == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy usage API is unavailable",
		})
	}

	subdomainID, err := decodeCatalogTaxonomyObjectIDFromPath(c.Param("id"), "subdomain_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	previewLimit, err := decodeCatalogTaxonomyUsagePreviewLimit(c.QueryParam("preview_limit"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	summary, err := s.taxonomyUsage.GetSubdomainUsage(c.Request().Context(), subdomainID, previewLimit)
	if err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, domain.ErrCatalogTaxonomySubdomainNotFound)
	}

	return c.JSON(http.StatusOK, summary)
}

// getCatalogTaxonomyTagUsage returns delete-preflight usage for one taxonomy tag.
func (s *Server) getCatalogTaxonomyTagUsage(c *echo.Context) error {
	if s.taxonomyUsage == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "catalog taxonomy usage API is unavailable",
		})
	}

	tagID, err := decodeCatalogTaxonomyObjectIDFromPath(c.Param("id"), "tag_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	previewLimit, err := decodeCatalogTaxonomyUsagePreviewLimit(c.QueryParam("preview_limit"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	summary, err := s.taxonomyUsage.GetTagUsage(c.Request().Context(), tagID, previewLimit)
	if err != nil {
		return encodeCatalogTaxonomyServiceError(c, err, domain.ErrCatalogTaxonomyTagNotFound)
	}

	return c.JSON(http.StatusOK, summary)
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

func decodeCatalogItemIDFromQuery(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("catalog item id is required")
	}

	decoded, err := url.QueryUnescape(trimmed)
	if err != nil {
		return "", fmt.Errorf("catalog item id query is invalid")
	}

	itemID := strings.TrimSpace(decoded)
	if itemID == "" {
		return "", fmt.Errorf("catalog item id is required")
	}

	return itemID, nil
}

func decodeCatalogItemIDFromRequest(c *echo.Context) (string, error) {
	if c == nil {
		return "", fmt.Errorf("catalog item id is required")
	}

	if rawQuery := strings.TrimSpace(c.QueryParam("item_id")); rawQuery != "" {
		return decodeCatalogItemIDFromQuery(rawQuery)
	}

	return decodeCatalogItemIDFromPath(c.Param("id"))
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

func decodeCatalogExportRequest(c *echo.Context) (domain.CatalogExportRequest, error) {
	limitedReader := io.LimitReader(c.Request().Body, catalogExportRequestMaxBodyBytes+1)
	payload, err := io.ReadAll(limitedReader)
	if err != nil {
		return domain.CatalogExportRequest{}, fmt.Errorf("invalid request payload")
	}
	if len(payload) == 0 {
		return domain.CatalogExportRequest{}, fmt.Errorf("request body is required")
	}
	if len(payload) > catalogExportRequestMaxBodyBytes {
		return domain.CatalogExportRequest{}, fmt.Errorf(
			"request payload exceeds %d bytes",
			catalogExportRequestMaxBodyBytes,
		)
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var request domain.CatalogExportRequest
	if err := decoder.Decode(&request); err != nil {
		return domain.CatalogExportRequest{}, fmt.Errorf("invalid request payload")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return domain.CatalogExportRequest{}, fmt.Errorf("invalid request payload")
	}

	return request, nil
}

func decodeCatalogMaterializationRequest(c *echo.Context) (domain.CatalogMaterializationRequest, error) {
	limitedReader := io.LimitReader(c.Request().Body, catalogMaterializeRequestMaxBodyBytes+1)
	payload, err := io.ReadAll(limitedReader)
	if err != nil {
		return domain.CatalogMaterializationRequest{}, fmt.Errorf("invalid request payload")
	}
	if len(payload) == 0 {
		return domain.CatalogMaterializationRequest{}, fmt.Errorf("request body is required")
	}
	if len(payload) > catalogMaterializeRequestMaxBodyBytes {
		return domain.CatalogMaterializationRequest{}, fmt.Errorf(
			"request payload exceeds %d bytes",
			catalogMaterializeRequestMaxBodyBytes,
		)
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var request domain.CatalogMaterializationRequest
	if err := decoder.Decode(&request); err != nil {
		return domain.CatalogMaterializationRequest{}, fmt.Errorf("invalid request payload")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return domain.CatalogMaterializationRequest{}, fmt.Errorf("invalid request payload")
	}

	return request, nil
}

func decodeLegacySkillExportName(rawPath string) (string, error) {
	name := strings.TrimPrefix(strings.TrimSpace(rawPath), "/")
	if name == "" {
		return "", fmt.Errorf("skill name is required")
	}

	decoded, err := url.PathUnescape(name)
	if err != nil {
		return "", fmt.Errorf("skill name path is invalid")
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" {
		return "", fmt.Errorf("skill name is required")
	}

	return decoded, nil
}

func buildCatalogExportResponse(result domain.CatalogExportResult) CatalogExportResponse {
	response := CatalogExportResponse{
		Format:   result.Format,
		DryRun:   result.DryRun,
		Manifest: result.Manifest,
	}

	if result.DryRun {
		return response
	}

	contentType := strings.TrimSpace(result.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	download := &CatalogExportDownload{
		FileName:      result.FileName,
		ContentType:   contentType,
		ContentLength: len(result.ArchiveData),
	}
	if legacyURL := buildLegacySkillExportURL(result.Manifest); legacyURL != "" {
		download.LegacySkillExport = legacyURL
	}

	response.Download = download
	return response
}

func buildLegacySkillExportURL(manifest domain.CatalogExportManifest) string {
	if len(manifest.Items) != 1 {
		return ""
	}

	item := manifest.Items[0]
	if item.Classifier != domain.CatalogClassifierSkill {
		return ""
	}

	sourceRef := strings.TrimSpace(item.SourceRef)
	if sourceRef == "" {
		return ""
	}

	return "/api/skills/export/" + url.PathEscape(sourceRef)
}

func (s *Server) executeCatalogExport(
	ctx context.Context,
	request domain.CatalogExportRequest,
) (domain.CatalogExportResult, error) {
	// Preserve WP-002 single-skill behavior for compatibility callers.
	if shouldUseLegacyCatalogExportPath(request) {
		exportService, err := domain.NewCatalogExportService(s.skillManager)
		if err != nil {
			return domain.CatalogExportResult{}, err
		}
		return exportService.Export(ctx, request)
	}

	return s.executeCatalogExportViaMaterialization(ctx, request)
}

func shouldUseLegacyCatalogExportPath(request domain.CatalogExportRequest) bool {
	itemIDs := normalizeRequestedCatalogExportItemIDs(request.ItemIDs)
	if len(itemIDs) != 1 {
		return false
	}

	classifierToken, _, hasClassifier := strings.Cut(itemIDs[0], ":")
	if !hasClassifier {
		return true
	}

	classifier, err := domain.ParseCatalogClassifier(classifierToken)
	if err != nil {
		return false
	}
	return classifier == domain.CatalogClassifierSkill
}

func normalizeRequestedCatalogExportItemIDs(itemIDs []string) []string {
	normalized := make([]string, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		trimmed := strings.TrimSpace(itemID)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func (s *Server) executeCatalogExportViaMaterialization(
	ctx context.Context,
	request domain.CatalogExportRequest,
) (domain.CatalogExportResult, error) {
	format := request.Format
	if format == "" {
		format = domain.CatalogExportFormatTarGz
	}
	if format != domain.CatalogExportFormatTarGz {
		return domain.CatalogExportResult{}, fmt.Errorf(
			"%w: export format %q is not supported",
			domain.ErrCatalogExportInvalidRequest,
			format,
		)
	}

	stagingRoot, err := os.MkdirTemp("", "skillserver-catalog-export-*")
	if err != nil {
		return domain.CatalogExportResult{}, fmt.Errorf("create export staging root: %w", err)
	}
	defer os.RemoveAll(stagingRoot)

	materializationService, err := domain.NewCatalogMaterializationService(s.skillManager, []string{stagingRoot})
	if err != nil {
		return domain.CatalogExportResult{}, err
	}

	materializedResult, err := materializationService.Materialize(
		ctx,
		domain.CatalogMaterializationRequest{
			ItemIDs:        request.ItemIDs,
			DestinationDir: stagingRoot,
			DryRun:         request.DryRun,
		},
	)
	if err != nil {
		return domain.CatalogExportResult{}, mapCatalogMaterializationErrorToExportError(err)
	}

	manifest := buildCatalogExportManifestFromMaterializationResult(materializedResult)
	result := domain.CatalogExportResult{
		Format:   format,
		DryRun:   request.DryRun,
		Manifest: manifest,
	}
	if request.DryRun {
		return result, nil
	}

	archiveData, err := buildCatalogExportArchiveFromDirectory(stagingRoot)
	if err != nil {
		return domain.CatalogExportResult{}, fmt.Errorf("build catalog export archive: %w", err)
	}

	result.ContentType = "application/gzip"
	result.FileName = buildCatalogExportArchiveFileName(manifest)
	result.ArchiveData = archiveData
	return result, nil
}

func mapCatalogMaterializationErrorToExportError(materializeErr error) error {
	switch {
	case errors.Is(materializeErr, domain.ErrCatalogMaterializationInvalidRequest),
		errors.Is(materializeErr, domain.ErrCatalogMaterializationDestinationOutsideAllowedRoots):
		return fmt.Errorf("%w: %v", domain.ErrCatalogExportInvalidRequest, materializeErr)
	case errors.Is(materializeErr, domain.ErrCatalogMaterializationItemNotFound):
		return fmt.Errorf("%w: %v", domain.ErrCatalogExportItemNotFound, materializeErr)
	case errors.Is(materializeErr, domain.ErrCatalogMaterializationUnsupportedClassifier):
		return fmt.Errorf("%w: %v", domain.ErrCatalogExportUnsupportedClassifier, materializeErr)
	default:
		return materializeErr
	}
}

func buildCatalogExportManifestFromMaterializationResult(
	result domain.CatalogMaterializationResult,
) domain.CatalogExportManifest {
	manifestItems := make([]domain.CatalogExportManifestItem, 0, len(result.Items))
	for _, item := range result.Items {
		archiveRoot := strings.TrimSpace(item.TargetPath)
		if archiveRoot == "" && len(item.Files) > 0 {
			archiveRoot = strings.TrimSpace(item.Files[0].TargetPath)
		}
		if archiveRoot == "" {
			archiveRoot = "."
		}

		sourceRef := strings.TrimSpace(item.SourceRef)
		if sourceRef == "" && item.Classifier == domain.CatalogClassifierSkill {
			sourceRef = strings.TrimSpace(strings.TrimPrefix(item.ItemID, "skill:"))
		}

		manifestItems = append(manifestItems, domain.CatalogExportManifestItem{
			ItemID:          item.ItemID,
			Classifier:      item.Classifier,
			SourceRef:       sourceRef,
			ArchiveRoot:     archiveRoot,
			ArchiveFileName: buildCatalogExportManifestItemFileName(item),
		})
	}
	return domain.CatalogExportManifest{Items: manifestItems}
}

func buildCatalogExportManifestItemFileName(item domain.CatalogMaterializationItemResult) string {
	switch item.Classifier {
	case domain.CatalogClassifierSkill:
		sourceRef := strings.TrimSpace(item.SourceRef)
		if sourceRef == "" {
			sourceRef = strings.TrimSpace(strings.TrimPrefix(item.ItemID, "skill:"))
		}
		if sourceRef == "" {
			return "skill-export.tar.gz"
		}
		replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
		return replacer.Replace(sourceRef) + ".tar.gz"
	default:
		targetPath := strings.TrimSpace(item.TargetPath)
		if targetPath == "" && len(item.Files) > 0 {
			targetPath = strings.TrimSpace(item.Files[0].TargetPath)
		}
		baseName := filepath.Base(filepath.FromSlash(targetPath))
		baseName = strings.TrimSpace(baseName)
		if baseName == "" || baseName == "." || baseName == string(filepath.Separator) {
			baseName = strings.TrimSpace(item.ItemID)
		}
		replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
		normalized := strings.TrimSpace(replacer.Replace(baseName))
		if normalized == "" {
			return "catalog-item"
		}
		return normalized
	}
}

func buildCatalogExportArchiveFileName(manifest domain.CatalogExportManifest) string {
	if len(manifest.Items) == 1 {
		item := manifest.Items[0]
		fileName := strings.TrimSpace(item.ArchiveFileName)
		if fileName == "" {
			return "catalog-export.tar.gz"
		}
		if strings.HasSuffix(fileName, ".tar.gz") {
			return fileName
		}
		return fileName + ".tar.gz"
	}

	return "catalog-export.tar.gz"
}

func buildCatalogExportArchiveFromDirectory(rootDir string) ([]byte, error) {
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	defer gzipWriter.Close()
	defer tarWriter.Close()

	if err := filepath.Walk(rootDir, func(currentPath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(rootDir, currentPath)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		relativePath = strings.TrimPrefix(relativePath, "./")
		if relativePath == "" || relativePath == "." || strings.HasPrefix(relativePath, "../") {
			return fmt.Errorf("invalid archive path %q", relativePath)
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relativePath
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		content, err := os.ReadFile(currentPath)
		if err != nil {
			return err
		}
		if _, err := tarWriter.Write(content); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if err := tarWriter.Close(); err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (s *Server) executeCatalogMaterialization(
	ctx context.Context,
	request domain.CatalogMaterializationRequest,
) (domain.CatalogMaterializationResult, error) {
	materializationService, err := domain.NewCatalogMaterializationService(
		s.skillManager,
		s.mcpRuntimeCapabilities.AllowedDestinationRoots,
	)
	if err != nil {
		return domain.CatalogMaterializationResult{}, err
	}
	return materializationService.Materialize(ctx, request)
}

func encodeCatalogExportServiceError(c *echo.Context, exportErr error) error {
	switch {
	case errors.Is(exportErr, domain.ErrCatalogExportInvalidRequest),
		errors.Is(exportErr, domain.ErrCatalogExportUnsupportedClassifier):
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": exportErr.Error(),
		})
	case errors.Is(exportErr, domain.ErrCatalogExportItemNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": exportErr.Error(),
		})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": exportErr.Error(),
		})
	}
}

func buildCatalogMaterializeResponse(
	result domain.CatalogMaterializationResult,
) CatalogMaterializeResponse {
	return CatalogMaterializeResponse{
		DryRun:                 result.DryRun,
		DestinationDir:         result.DestinationDir,
		ResolvedDestinationDir: result.ResolvedDestinationDir,
		Items:                  result.Items,
	}
}

func encodeCatalogMaterializationServiceError(c *echo.Context, materializeErr error) error {
	switch {
	case errors.Is(materializeErr, domain.ErrCatalogMaterializationInvalidRequest),
		errors.Is(materializeErr, domain.ErrCatalogMaterializationUnsupportedClassifier):
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": materializeErr.Error(),
		})
	case errors.Is(materializeErr, domain.ErrCatalogMaterializationDestinationOutsideAllowedRoots):
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": materializeErr.Error(),
		})
	case errors.Is(materializeErr, domain.ErrCatalogMaterializationItemNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": materializeErr.Error(),
		})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": materializeErr.Error(),
		})
	}
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

func decodeCatalogListRequest(c *echo.Context) (catalogListRequest, error) {
	taxonomyFilter, err := decodeCatalogListTaxonomyFilter(c)
	if err != nil {
		return catalogListRequest{}, err
	}
	classifier, err := decodeCatalogClassifierQueryParam(c.QueryParam("classifier"))
	if err != nil {
		return catalogListRequest{}, err
	}

	includeContent, err := decodeCatalogTaxonomyBoolQueryParam(c.QueryParam("include_content"), "include_content")
	if err != nil {
		return catalogListRequest{}, err
	}
	unclassified, err := decodeCatalogTaxonomyBoolQueryParam(c.QueryParam("unclassified"), "unclassified")
	if err != nil {
		return catalogListRequest{}, err
	}
	missingPrimaryDomain, err := decodeCatalogTaxonomyBoolQueryParam(
		c.QueryParam("missing_primary_domain"),
		"missing_primary_domain",
	)
	if err != nil {
		return catalogListRequest{}, err
	}
	missingTags, err := decodeCatalogTaxonomyBoolQueryParam(c.QueryParam("missing_tags"), "missing_tags")
	if err != nil {
		return catalogListRequest{}, err
	}

	request := catalogListRequest{
		TaxonomyFilter: taxonomyFilter,
		Classifier:     classifier,
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

	limitRaw := strings.TrimSpace(c.QueryParam("limit"))
	cursor := strings.TrimSpace(c.QueryParam("cursor"))
	if limitRaw == "" && cursor == "" {
		return request, nil
	}

	request.UseEnvelope = true
	request.Cursor = cursor
	request.Limit = catalogListDefaultLimit
	if limitRaw == "" {
		return request, nil
	}

	limit, err := strconv.Atoi(limitRaw)
	if err != nil || limit <= 0 || limit > catalogListMaxLimit {
		return catalogListRequest{}, fmt.Errorf(
			"query parameter %q must be an integer between 1 and %d",
			"limit",
			catalogListMaxLimit,
		)
	}
	request.Limit = limit
	return request, nil
}

func decodeCatalogClassifierQueryParam(raw string) (*domain.CatalogClassifier, error) {
	classifierRaw := strings.TrimSpace(raw)
	if classifierRaw == "" {
		return nil, nil
	}

	parsedClassifier, err := domain.ParseCatalogClassifier(classifierRaw)
	if err != nil {
		return nil, err
	}
	return &parsedClassifier, nil
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

func decodeCatalogTaxonomyUsagePreviewLimit(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return catalogTaxonomyUsageDefaultPreviewLimit, nil
	}

	limit, err := strconv.Atoi(trimmed)
	if err != nil || limit < 0 || limit > catalogTaxonomyUsageMaxPreviewLimit {
		return 0, fmt.Errorf(
			"query parameter %q must be an integer between 0 and %d",
			"preview_limit",
			catalogTaxonomyUsageMaxPreviewLimit,
		)
	}

	return limit, nil
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
		return c.JSON(http.StatusConflict, buildCatalogTaxonomyConflictResponse(serviceErr))
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
		return c.JSON(http.StatusConflict, buildCatalogTaxonomyConflictResponse(serviceErr))
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": serviceErr.Error(),
		})
	}
}

func buildCatalogTaxonomyConflictResponse(serviceErr error) catalogTaxonomyConflictResponse {
	response := catalogTaxonomyConflictResponse{
		Error: serviceErr.Error(),
	}

	var conflictErr *domain.CatalogTaxonomyConflictError
	if !errors.As(serviceErr, &conflictErr) {
		return response
	}

	response.Conflict = &catalogTaxonomyConflictDetailResult{
		ObjectType:        conflictErr.ObjectType,
		ObjectID:          conflictErr.ObjectID,
		Reason:            conflictErr.Reason,
		ReferencedItemIDs: append([]string{}, conflictErr.ReferencedItemIDs...),
	}
	return response
}

func encodeCatalogRelationshipServiceError(c *echo.Context, serviceErr error) error {
	switch {
	case errors.Is(serviceErr, domain.ErrCatalogRelationshipReadOnlySurface):
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": serviceErr.Error(),
		})
	case errors.Is(serviceErr, domain.ErrCatalogRelationshipItemNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "catalog item not found",
		})
	case errors.Is(serviceErr, domain.ErrCatalogRelationshipValidation):
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": serviceErr.Error(),
		})
	case isCatalogRelationshipConflictError(serviceErr):
		return c.JSON(http.StatusConflict, map[string]string{
			"error": serviceErr.Error(),
		})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": serviceErr.Error(),
		})
	}
}

func isCatalogRelationshipConflictError(err error) bool {
	if err == nil {
		return false
	}

	lowered := strings.ToLower(err.Error())
	return strings.Contains(lowered, "unique constraint failed") ||
		strings.Contains(lowered, "constraint failed")
}

func decodeCatalogRelationshipPatchRequest(c *echo.Context) (PatchCatalogRelationshipsRequest, error) {
	limitedReader := io.LimitReader(c.Request().Body, catalogRelationshipPatchMaxBodyBytes+1)
	payload, err := io.ReadAll(limitedReader)
	if err != nil {
		return PatchCatalogRelationshipsRequest{}, fmt.Errorf("invalid request payload")
	}
	if len(payload) == 0 {
		return PatchCatalogRelationshipsRequest{}, fmt.Errorf("request body is required")
	}
	if len(payload) > catalogRelationshipPatchMaxBodyBytes {
		return PatchCatalogRelationshipsRequest{}, fmt.Errorf(
			"request payload exceeds %d bytes",
			catalogRelationshipPatchMaxBodyBytes,
		)
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var request PatchCatalogRelationshipsRequest
	if err := decoder.Decode(&request); err != nil {
		return PatchCatalogRelationshipsRequest{}, fmt.Errorf("invalid request payload")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return PatchCatalogRelationshipsRequest{}, fmt.Errorf("invalid request payload")
	}

	return request, nil
}

func normalizeCatalogRelationshipPatchInput(
	itemID string,
	request PatchCatalogRelationshipsRequest,
) (domain.CatalogRelationshipPatchInput, error) {
	normalized := domain.CatalogRelationshipPatchInput{
		ItemID: strings.TrimSpace(itemID),
	}

	if request.PromptItemID.Set {
		normalized.PromptItemIDSet = true
		if request.PromptItemID.Value != nil {
			promptItemID := strings.TrimSpace(*request.PromptItemID.Value)
			if promptItemID == "" {
				return domain.CatalogRelationshipPatchInput{}, fmt.Errorf("prompt_item_id cannot be empty")
			}
			normalized.PromptItemID = &promptItemID
		}
	}

	if request.RuleItemIDs.Set {
		if request.RuleItemIDs.Null {
			return domain.CatalogRelationshipPatchInput{}, fmt.Errorf("rule_item_ids must be an array when provided")
		}

		ruleItemIDs := make([]string, 0, len(request.RuleItemIDs.Value))
		for index, rawRuleItemID := range request.RuleItemIDs.Value {
			ruleItemID := strings.TrimSpace(rawRuleItemID)
			if ruleItemID == "" {
				return domain.CatalogRelationshipPatchInput{}, fmt.Errorf("rule_item_ids[%d] cannot be empty", index)
			}
			ruleItemIDs = append(ruleItemIDs, ruleItemID)
		}
		normalized.RuleItemIDs = &ruleItemIDs
	}

	if request.UpdatedBy != nil {
		updatedBy := strings.TrimSpace(*request.UpdatedBy)
		if updatedBy != "" {
			normalized.UpdatedBy = &updatedBy
		}
	}

	if !normalized.PromptItemIDSet && normalized.RuleItemIDs == nil {
		return domain.CatalogRelationshipPatchInput{}, fmt.Errorf(
			"at least one of prompt_item_id or rule_item_ids is required",
		)
	}

	return normalized, nil
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
			HasAssignment:      view.Effective.HasAssignment,
			IsFullyClassified:  view.Effective.IsFullyClassified,
			MissingFields:      append([]string{}, view.Effective.MissingFields...),
		},
		Relationships: catalogRelationshipSetResponseFromDomain(view.Relationships),
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
	if response.Relationships.Rules == nil {
		response.Relationships.Rules = []CatalogRelationshipItemResponse{}
	}
	if response.Relationships.Skills == nil {
		response.Relationships.Skills = []CatalogRelationshipItemResponse{}
	}

	return response
}

func catalogRelationshipsResponseFromView(view domain.CatalogRelationshipView) CatalogRelationshipsResponse {
	response := CatalogRelationshipsResponse{
		ItemID:        view.ItemID,
		Relationships: catalogRelationshipSetResponseFromDomain(view.Relationships),
	}
	if response.Relationships.Rules == nil {
		response.Relationships.Rules = []CatalogRelationshipItemResponse{}
	}
	if response.Relationships.Skills == nil {
		response.Relationships.Skills = []CatalogRelationshipItemResponse{}
	}
	return response
}

func catalogRelationshipSetResponseFromDomain(
	relationships domain.CatalogRelationshipSet,
) CatalogRelationshipSetResponse {
	response := CatalogRelationshipSetResponse{
		Rules:  make([]CatalogRelationshipItemResponse, 0, len(relationships.Rules)),
		Skills: make([]CatalogRelationshipItemResponse, 0, len(relationships.Skills)),
	}

	if relationships.Prompt != nil {
		promptItem := catalogRelationshipItemResponseFromDomain(*relationships.Prompt)
		response.Prompt = &promptItem
	}
	for _, item := range relationships.Rules {
		response.Rules = append(response.Rules, catalogRelationshipItemResponseFromDomain(item))
	}
	for _, item := range relationships.Skills {
		response.Skills = append(response.Skills, catalogRelationshipItemResponseFromDomain(item))
	}

	return response
}

func catalogRelationshipItemResponseFromDomain(
	item domain.CatalogRelationshipItem,
) CatalogRelationshipItemResponse {
	response := CatalogRelationshipItemResponse{
		ID:         item.ID,
		Classifier: item.Classifier,
		Name:       item.Name,
	}
	if item.ParentSkillID != nil {
		parentSkillID := strings.TrimSpace(*item.ParentSkillID)
		if parentSkillID != "" {
			response.ParentSkillID = &parentSkillID
		}
	}
	if item.ResourcePath != nil {
		resourcePath := strings.TrimSpace(*item.ResourcePath)
		if resourcePath != "" {
			response.ResourcePath = &resourcePath
		}
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

// exportCatalog handles additive export planning/execution requests.
func (s *Server) exportCatalog(c *echo.Context) error {
	request, err := decodeCatalogExportRequest(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	result, err := s.executeCatalogExport(c.Request().Context(), request)
	if err != nil {
		return encodeCatalogExportServiceError(c, err)
	}

	return c.JSON(http.StatusOK, buildCatalogExportResponse(result))
}

// materializeCatalog handles project-folder materialization planning/execution requests.
func (s *Server) materializeCatalog(c *echo.Context) error {
	if !s.mcpRuntimeCapabilities.MaterializationEnabled {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "catalog materialization capability is disabled",
		})
	}

	request, err := decodeCatalogMaterializationRequest(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	result, err := s.executeCatalogMaterialization(c.Request().Context(), request)
	if err != nil {
		return encodeCatalogMaterializationServiceError(c, err)
	}

	return c.JSON(http.StatusOK, buildCatalogMaterializeResponse(result))
}

// exportSkill exports a skill as a compressed archive.
//
// This route is retained for backward compatibility and now delegates to the
// shared catalog export service introduced in WP-001.
func (s *Server) exportSkill(c *echo.Context) error {
	name, err := decodeLegacySkillExportName(c.Param("*"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	result, err := s.executeCatalogExport(c.Request().Context(), domain.CatalogExportRequest{
		ItemIDs: []string{domain.BuildSkillCatalogItemID(name)},
	})
	if err != nil {
		return encodeCatalogExportServiceError(c, err)
	}

	if len(result.ArchiveData) == 0 {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "catalog export completed without archive data",
		})
	}

	contentType := strings.TrimSpace(result.ContentType)
	if contentType == "" {
		contentType = "application/gzip"
	}

	// Keep the legacy filename semantics for callers that depend on this header.
	fileName := name + ".tar.gz"
	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Response().Header().Set("Content-Length", strconv.Itoa(len(result.ArchiveData)))

	return c.Blob(http.StatusOK, contentType, result.ArchiveData)
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

	return c.JSON(http.StatusCreated, s.skillResponseFromDomainSkill(*skill))
}

// Git repository management handlers

// GitRepoResponse represents a git repository in API responses
type GitRepoResponse struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	// AuthMode exposes the configured repo authentication mode without secret material.
	AuthMode string `json:"auth_mode"`
	// CredentialSource exposes where credentials resolve from without secret material.
	CredentialSource string `json:"credential_source"`
	// HasCredentials indicates whether the configured auth mode has complete non-secret refs or stored secret material.
	HasCredentials bool `json:"has_credentials"`
	// StoredCredentialsEnabled indicates whether stored-secret git workflows are allowed at runtime.
	StoredCredentialsEnabled bool `json:"stored_credentials_enabled"`
	// LastSyncStatus reflects redacted sync status state from the syncer.
	LastSyncStatus string `json:"last_sync_status"`
	// LastSyncError reflects a redacted sync error, if present.
	LastSyncError string `json:"last_sync_error,omitempty"`
}

// GitRepoAuthRequest represents non-secret auth descriptor metadata for add/update requests.
type GitRepoAuthRequest struct {
	Mode          string `json:"mode,omitempty"`
	Source        string `json:"source,omitempty"`
	ReferenceID   string `json:"reference_id,omitempty"`
	UsernameRef   string `json:"username_ref,omitempty"`
	PasswordRef   string `json:"password_ref,omitempty"`
	TokenRef      string `json:"token_ref,omitempty"`
	KeyRef        string `json:"key_ref,omitempty"`
	KnownHostsRef string `json:"known_hosts_ref,omitempty"`
}

// GitRepoStoredCredentialWriteRequest contains write-only secret fields accepted only when auth.source=stored.
type GitRepoStoredCredentialWriteRequest struct {
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Token      string `json:"token,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
	KnownHosts string `json:"known_hosts,omitempty"`
}

// AddGitRepoRequest represents a request to add a git repository
type AddGitRepoRequest struct {
	URL              string                               `json:"url"`
	Enabled          *bool                                `json:"enabled,omitempty"`
	Auth             *GitRepoAuthRequest                  `json:"auth,omitempty"`
	StoredCredential *GitRepoStoredCredentialWriteRequest `json:"stored_credential,omitempty"`
}

// UpdateGitRepoRequest represents a request to update a git repository
type UpdateGitRepoRequest struct {
	URL              string                               `json:"url"`
	Enabled          *bool                                `json:"enabled,omitempty"`
	Auth             *GitRepoAuthRequest                  `json:"auth,omitempty"`
	StoredCredential *GitRepoStoredCredentialWriteRequest `json:"stored_credential,omitempty"`
}

// RuntimeCapabilitiesResponse represents runtime capability gates exposed to API/UI clients.
type RuntimeCapabilitiesResponse struct {
	Git     GitRuntimeCapabilities     `json:"git"`
	Catalog CatalogRuntimeCapabilities `json:"catalog"`
	MCP     MCPRuntimeCapabilities     `json:"mcp"`
}

// getRuntimeCapabilities returns runtime capability state needed by the repo API/UI.
func (s *Server) getRuntimeCapabilities(c *echo.Context) error {
	return c.JSON(http.StatusOK, RuntimeCapabilitiesResponse{
		Git:     s.gitRuntimeCapabilities,
		Catalog: s.catalogRuntimeCapabilities,
		MCP:     s.mcpRuntimeCapabilities,
	})
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
		response, responseErr := s.buildGitRepoResponse(repo)
		if responseErr != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("failed to build repo response: %v", responseErr),
			})
		}
		repos[i] = response
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
	if s.configManager == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "config manager not available",
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

	authConfig := normalizeGitRepoAuthRequest(req.Auth)
	hasStoredCredentialInput := hasStoredCredentialWriteInput(req.StoredCredential)
	if err := validateGitRepoAuthRequest(
		authConfig,
		hasStoredCredentialInput,
		s.gitRuntimeCapabilities.StoredCredentialsEnabled,
	); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	storedCredentialPayload, hasStoredCredentialPayload, err := buildStoredCredentialPayload(
		authConfig.Mode,
		req.StoredCredential,
	)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	canonicalRepoURL, err := git.CanonicalizeRepoURL(req.URL)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid URL: %v", err),
		})
	}

	// Load current config
	configRepos, err := s.configManager.LoadConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to load config: %v", err),
		})
	}

	// Check if repo already exists (canonical URL match)
	for _, repo := range configRepos {
		if repo.URL == canonicalRepoURL {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "repository already exists",
			})
		}
	}

	// Add new repo to config (enabled by default)
	newRepo := git.GitRepoConfig{
		ID:      git.GenerateID(canonicalRepoURL),
		URL:     canonicalRepoURL,
		Name:    git.ResolveCheckoutName(canonicalRepoURL),
		Enabled: true,
		Auth:    authConfig,
	}
	if req.Enabled != nil {
		newRepo.Enabled = *req.Enabled
	}
	if repoAuthSourceForAPI(newRepo.Auth.Mode, newRepo.Auth.Source) == git.GitRepoAuthSourceStored &&
		!hasStoredCredentialPayload {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "stored_credential payload is required when auth.source=\"stored\" for new repositories",
		})
	}
	configRepos = append(configRepos, newRepo)
	enabledGitRepoNames := enabledGitRepoCheckoutNames(configRepos)

	// Save config
	if err := s.configManager.SaveConfig(configRepos); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to save config: %v", err),
		})
	}

	var createdStoredCredential bool
	if hasStoredCredentialPayload {
		createdStoredCredential, err = s.upsertStoredCredential(newRepo, storedCredentialPayload)
		if err != nil {
			rollbackRepos := removeRepoByID(configRepos, newRepo.ID)
			_ = s.configManager.SaveConfig(rollbackRepos)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("failed to persist stored credentials: %v", err),
			})
		}
	}

	// Add repo to syncer and update FileSystemManager
	if s.gitSyncer != nil {
		// Update git repo visibility before AddRepo so sync-triggered rebuilds in persistence mode
		// discover newly enabled repositories in the same request.
		if s.fsManager != nil {
			s.fsManager.UpdateGitRepos(enabledGitRepoNames)
		}

		if err := s.gitSyncer.AddRepo(newRepo); err != nil {
			// Remove from config if sync failed
			rollbackRepos := removeRepoByID(configRepos, newRepo.ID)
			_ = s.configManager.SaveConfig(rollbackRepos)
			if createdStoredCredential {
				_ = s.deleteStoredCredentialByReferenceID(resolveStoredCredentialReferenceID(newRepo.Auth, newRepo.ID))
			}
			if s.fsManager != nil {
				s.fsManager.UpdateGitRepos(enabledGitRepoCheckoutNames(rollbackRepos))
			}
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": git.RedactGitAuthError(err),
			})
		}

		// Ensure the catalog index reflects the newly enabled repo set.
		if err := s.skillManager.RebuildIndex(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to rebuild index",
			})
		}
	}

	response, responseErr := s.buildGitRepoResponse(newRepo)
	if responseErr != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to build repo response: %v", responseErr),
		})
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
	if s.configManager == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "config manager not available",
		})
	}

	id := c.Param("id")

	var req UpdateGitRepoRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	configRepos, err := s.configManager.LoadConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to load config: %v", err),
		})
	}

	repoIndex := -1
	for i := range configRepos {
		if configRepos[i].ID == id {
			repoIndex = i
			break
		}
	}
	if repoIndex < 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "repository not found",
		})
	}

	updatedRepo := configRepos[repoIndex]
	updatedURL := updatedRepo.URL
	if strings.TrimSpace(req.URL) != "" {
		updatedURL, err = git.CanonicalizeRepoURL(req.URL)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("invalid URL: %v", err),
			})
		}
	}
	if req.Enabled != nil {
		updatedRepo.Enabled = *req.Enabled
	}
	if req.Auth != nil {
		updatedRepo.Auth = normalizeGitRepoAuthRequest(req.Auth)
	}

	hasStoredCredentialInput := hasStoredCredentialWriteInput(req.StoredCredential)
	if err := validateGitRepoAuthRequest(
		updatedRepo.Auth,
		hasStoredCredentialInput,
		s.gitRuntimeCapabilities.StoredCredentialsEnabled,
	); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	storedCredentialPayload, hasStoredCredentialPayload, err := buildStoredCredentialPayload(
		updatedRepo.Auth.Mode,
		req.StoredCredential,
	)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Duplicate detection uses canonical URL semantics, excluding the repo being updated.
	for i, repo := range configRepos {
		if i == repoIndex {
			continue
		}
		if repo.URL == updatedURL {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "repository already exists",
			})
		}
	}

	originalRepo := configRepos[repoIndex]
	updatedRepo.URL = updatedURL
	updatedRepo.ID = git.GenerateID(updatedURL)
	updatedRepo.Name = git.ResolveCheckoutName(updatedURL)
	if req.Auth == nil {
		updatedRepo.Auth = originalRepo.Auth
	}
	if req.Enabled == nil {
		updatedRepo.Enabled = originalRepo.Enabled
	}
	if repoAuthSourceForAPI(updatedRepo.Auth.Mode, updatedRepo.Auth.Source) == git.GitRepoAuthSourceStored &&
		!hasStoredCredentialPayload {
		hasConfiguredStoredCredentials, lookupErr := s.hasStoredCredentialForRepo(updatedRepo)
		if lookupErr != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("failed to verify stored credential state: %v", lookupErr),
			})
		}
		if !hasConfiguredStoredCredentials {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "stored_credential payload is required when auth.source=\"stored\" has no configured credential",
			})
		}
	}
	configRepos[repoIndex] = updatedRepo

	// Save updated config before applying runtime state; on runtime errors we roll back.
	if err := s.configManager.SaveConfig(configRepos); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to save config: %v", err),
		})
	}

	storedCredentialCreated := false
	if hasStoredCredentialPayload {
		storedCredentialCreated, err = s.upsertStoredCredential(updatedRepo, storedCredentialPayload)
		if err != nil {
			configRepos[repoIndex] = originalRepo
			_ = s.configManager.SaveConfig(configRepos)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("failed to persist stored credentials: %v", err),
			})
		}
	}

	enabledRepos := make([]git.GitRepoConfig, 0, len(configRepos))
	for _, repo := range configRepos {
		if repo.Enabled {
			enabledRepos = append(enabledRepos, repo)
		}
	}

	if s.gitSyncer != nil {
		if err := s.gitSyncer.UpdateRepos(enabledRepos); err != nil {
			// Best-effort rollback to preserve a coherent config/runtime state.
			configRepos[repoIndex] = originalRepo
			_ = s.configManager.SaveConfig(configRepos)

			rollbackEnabledRepos := make([]git.GitRepoConfig, 0, len(configRepos))
			for _, repo := range configRepos {
				if repo.Enabled {
					rollbackEnabledRepos = append(rollbackEnabledRepos, repo)
				}
			}
			_ = s.gitSyncer.UpdateRepos(rollbackEnabledRepos)
			if storedCredentialCreated {
				_ = s.deleteStoredCredentialByReferenceID(
					resolveStoredCredentialReferenceID(updatedRepo.Auth, updatedRepo.ID),
				)
			}

			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("failed to update syncer: %v", err),
			})
		}

		if s.fsManager != nil {
			gitRepoNames := make([]string, len(enabledRepos))
			for i, repo := range enabledRepos {
				gitRepoNames[i] = git.ResolveRepoCheckoutName(repo)
			}
			s.fsManager.UpdateGitRepos(gitRepoNames)
		}

		if err := s.skillManager.RebuildIndex(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to rebuild index",
			})
		}
	}

	if shouldDeleteImplicitStoredCredential(originalRepo, updatedRepo) {
		_ = s.deleteStoredCredentialByReferenceID(resolveStoredCredentialReferenceID(originalRepo.Auth, originalRepo.ID))
	}

	response, responseErr := s.buildGitRepoResponse(updatedRepo)
	if responseErr != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to build repo response: %v", responseErr),
		})
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
	repoName := git.ResolveRepoCheckoutName(*foundRepo)

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
	if err := s.gitSyncer.RemoveRepo(foundRepo.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": git.RedactGitAuthError(err),
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
		enabledRepos := make([]git.GitRepoConfig, 0)
		for _, repo := range updatedConfigs {
			if repo.Enabled {
				enabledRepos = append(enabledRepos, repo)
			}
		}
		gitRepoNames := make([]string, len(enabledRepos))
		for i, repo := range enabledRepos {
			gitRepoNames[i] = git.ResolveRepoCheckoutName(repo)
		}
		s.fsManager.UpdateGitRepos(gitRepoNames)
	}

	// Trigger re-indexing
	if err := s.skillManager.RebuildIndex(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to rebuild index",
		})
	}

	if usesImplicitStoredCredentialReference(*foundRepo) {
		_ = s.deleteStoredCredentialByReferenceID(
			resolveStoredCredentialReferenceID(foundRepo.Auth, foundRepo.ID),
		)
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
	if err := s.gitSyncer.SyncRepo(foundRepo.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": git.RedactGitAuthError(err),
		})
	}
	if s.manualRepoSyncHook != nil {
		if err := s.manualRepoSyncHook(*foundRepo); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": git.RedactGitAuthError(err),
			})
		}
	} else if err := s.skillManager.RebuildIndex(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to rebuild index",
		})
	}

	response, responseErr := s.buildGitRepoResponse(*foundRepo)
	if responseErr != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to build repo response: %v", responseErr),
		})
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
		enabledRepos := make([]git.GitRepoConfig, 0)
		for _, repo := range configRepos {
			if repo.Enabled {
				enabledRepos = append(enabledRepos, repo)
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
			for i, repo := range enabledRepos {
				gitRepoNames[i] = git.ResolveRepoCheckoutName(repo)
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

	response, responseErr := s.buildGitRepoResponse(*foundRepo)
	if responseErr != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to build repo response: %v", responseErr),
		})
	}

	return c.JSON(http.StatusOK, response)
}

func normalizeGitRepoAuthRequest(request *GitRepoAuthRequest) git.GitRepoAuthConfig {
	if request == nil {
		return git.GitRepoAuthConfig{
			Mode:   git.GitRepoAuthModeNone,
			Source: git.GitRepoAuthSourceNone,
		}
	}

	normalized := git.GitRepoAuthConfig{
		Mode:          strings.TrimSpace(strings.ToLower(request.Mode)),
		Source:        strings.TrimSpace(strings.ToLower(request.Source)),
		ReferenceID:   strings.TrimSpace(request.ReferenceID),
		UsernameRef:   strings.TrimSpace(request.UsernameRef),
		PasswordRef:   strings.TrimSpace(request.PasswordRef),
		TokenRef:      strings.TrimSpace(request.TokenRef),
		KeyRef:        strings.TrimSpace(request.KeyRef),
		KnownHostsRef: strings.TrimSpace(request.KnownHostsRef),
	}
	if normalized.Mode == "" {
		normalized.Mode = git.GitRepoAuthModeNone
	}
	if normalized.Source == "" && normalized.Mode == git.GitRepoAuthModeNone {
		normalized.Source = git.GitRepoAuthSourceNone
	}

	return normalized
}

func validateGitRepoAuthRequest(
	auth git.GitRepoAuthConfig,
	hasStoredCredentialInput bool,
	storedCredentialsEnabled bool,
) error {
	mode := repoAuthModeForAPI(auth.Mode)
	source := repoAuthSourceForAPI(mode, auth.Source)

	switch mode {
	case git.GitRepoAuthModeNone:
		if source != "" && source != git.GitRepoAuthSourceNone {
			return fmt.Errorf("auth mode %q does not support source %q", mode, source)
		}
		if hasAnyAuthReferenceFields(auth) {
			return fmt.Errorf("auth mode %q does not support credential reference fields", mode)
		}
		if hasStoredCredentialInput {
			return fmt.Errorf("auth mode %q does not support stored_credential payload", mode)
		}
		return nil
	case git.GitRepoAuthModeHTTPSToken, git.GitRepoAuthModeHTTPSBasic, git.GitRepoAuthModeSSHKey:
	default:
		return fmt.Errorf("unsupported auth mode %q", mode)
	}

	switch source {
	case git.GitRepoAuthSourceEnv, git.GitRepoAuthSourceFile:
		if strings.TrimSpace(auth.ReferenceID) != "" {
			return fmt.Errorf("auth source %q does not support reference_id", source)
		}
		if hasStoredCredentialInput {
			return fmt.Errorf("stored_credential payload is supported only when auth.source=%q", git.GitRepoAuthSourceStored)
		}
		return git.ValidateGitRepoAuthConfig(git.GitRepoAuthConfig{
			Mode:          mode,
			Source:        source,
			UsernameRef:   strings.TrimSpace(auth.UsernameRef),
			PasswordRef:   strings.TrimSpace(auth.PasswordRef),
			TokenRef:      strings.TrimSpace(auth.TokenRef),
			KeyRef:        strings.TrimSpace(auth.KeyRef),
			KnownHostsRef: strings.TrimSpace(auth.KnownHostsRef),
		})

	case git.GitRepoAuthSourceStored:
		if !storedCredentialsEnabled {
			return fmt.Errorf("stored credentials are disabled by server configuration")
		}
		if hasEnvFileReferenceFields(auth) {
			return fmt.Errorf("auth source %q does not support *_ref credential references", source)
		}
		return nil

	case "", git.GitRepoAuthSourceNone:
		return fmt.Errorf(
			"auth mode %q requires source %q, %q, or %q",
			mode,
			git.GitRepoAuthSourceEnv,
			git.GitRepoAuthSourceFile,
			git.GitRepoAuthSourceStored,
		)

	default:
		return fmt.Errorf("auth mode %q does not support source %q", mode, source)
	}
}

func buildStoredCredentialPayload(
	authMode string,
	input *GitRepoStoredCredentialWriteRequest,
) (persistence.GitRepoCredentialSecretPayload, bool, error) {
	if !hasStoredCredentialWriteInput(input) {
		return persistence.GitRepoCredentialSecretPayload{}, false, nil
	}

	mode := repoAuthModeForAPI(authMode)
	if input == nil {
		return persistence.GitRepoCredentialSecretPayload{}, false, fmt.Errorf("stored_credential payload is required")
	}

	switch mode {
	case git.GitRepoAuthModeHTTPSToken:
		if strings.TrimSpace(input.Token) == "" {
			return persistence.GitRepoCredentialSecretPayload{}, false, fmt.Errorf("stored_credential.token is required for auth mode %q", mode)
		}
		return persistence.GitRepoCredentialSecretPayload{
			Type:     persistence.GitRepoCredentialSecretTypeHTTPSToken,
			Username: strings.TrimSpace(input.Username),
			Token:    input.Token,
		}, true, nil

	case git.GitRepoAuthModeHTTPSBasic:
		if strings.TrimSpace(input.Username) == "" {
			return persistence.GitRepoCredentialSecretPayload{}, false, fmt.Errorf("stored_credential.username is required for auth mode %q", mode)
		}
		if strings.TrimSpace(input.Password) == "" {
			return persistence.GitRepoCredentialSecretPayload{}, false, fmt.Errorf("stored_credential.password is required for auth mode %q", mode)
		}
		return persistence.GitRepoCredentialSecretPayload{
			Type:     persistence.GitRepoCredentialSecretTypeHTTPSBasic,
			Username: strings.TrimSpace(input.Username),
			Password: input.Password,
		}, true, nil

	case git.GitRepoAuthModeSSHKey:
		if strings.TrimSpace(input.PrivateKey) == "" {
			return persistence.GitRepoCredentialSecretPayload{}, false, fmt.Errorf("stored_credential.private_key is required for auth mode %q", mode)
		}
		if strings.TrimSpace(input.KnownHosts) == "" {
			return persistence.GitRepoCredentialSecretPayload{}, false, fmt.Errorf("stored_credential.known_hosts is required for auth mode %q", mode)
		}
		return persistence.GitRepoCredentialSecretPayload{
			Type:       persistence.GitRepoCredentialSecretTypeSSHKey,
			PrivateKey: input.PrivateKey,
			Passphrase: input.Passphrase,
			KnownHosts: input.KnownHosts,
		}, true, nil
	}

	return persistence.GitRepoCredentialSecretPayload{}, false, fmt.Errorf(
		"stored_credential payload is not supported for auth mode %q",
		mode,
	)
}

func (s *Server) buildGitRepoResponse(repo git.GitRepoConfig) (GitRepoResponse, error) {
	mode := repoAuthModeForAPI(repo.Auth.Mode)
	source := repoAuthSourceForAPI(mode, repo.Auth.Source)
	hasCredentials, err := s.hasCredentialsConfigured(repo)
	if err != nil {
		return GitRepoResponse{}, err
	}

	lastSyncStatus := string(git.RepoSyncStateNever)
	lastSyncError := ""
	if s.gitSyncer != nil {
		if status, ok := s.gitSyncer.GetRepoSyncStatus(repo.ID); ok {
			if strings.TrimSpace(string(status.State)) != "" {
				lastSyncStatus = string(status.State)
			}
			lastSyncError = strings.TrimSpace(status.LastError)
		}
	}

	return GitRepoResponse{
		ID:                       repo.ID,
		URL:                      repo.URL,
		Name:                     repo.Name,
		Enabled:                  repo.Enabled,
		AuthMode:                 mode,
		CredentialSource:         source,
		HasCredentials:           hasCredentials,
		StoredCredentialsEnabled: s.gitRuntimeCapabilities.StoredCredentialsEnabled,
		LastSyncStatus:           lastSyncStatus,
		LastSyncError:            lastSyncError,
	}, nil
}

func (s *Server) hasCredentialsConfigured(repo git.GitRepoConfig) (bool, error) {
	mode := repoAuthModeForAPI(repo.Auth.Mode)
	if mode == git.GitRepoAuthModeNone {
		return false, nil
	}

	source := repoAuthSourceForAPI(mode, repo.Auth.Source)
	switch source {
	case git.GitRepoAuthSourceEnv, git.GitRepoAuthSourceFile:
		return hasEnvFileCredentialsConfigured(mode, repo.Auth), nil
	case git.GitRepoAuthSourceStored:
		return s.hasStoredCredentialForRepo(repo)
	default:
		return false, nil
	}
}

func (s *Server) hasStoredCredentialForRepo(repo git.GitRepoConfig) (bool, error) {
	if s.gitCredentialStore == nil {
		return false, nil
	}

	referenceID := resolveStoredCredentialReferenceID(repo.Auth, repo.ID)
	if strings.TrimSpace(referenceID) == "" {
		return false, nil
	}

	_, err := s.gitCredentialStore.GetEncryptedByRepoID(context.Background(), referenceID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, persistence.ErrGitRepoCredentialNotFound) {
		return false, nil
	}
	return false, err
}

func (s *Server) upsertStoredCredential(
	repo git.GitRepoConfig,
	payload persistence.GitRepoCredentialSecretPayload,
) (bool, error) {
	if s.gitCredentialStore == nil {
		return false, fmt.Errorf("stored credential persistence is not available")
	}

	referenceID := resolveStoredCredentialReferenceID(repo.Auth, repo.ID)
	_, existingErr := s.gitCredentialStore.GetEncryptedByRepoID(context.Background(), referenceID)
	if existingErr != nil && !errors.Is(existingErr, persistence.ErrGitRepoCredentialNotFound) {
		return false, existingErr
	}
	existed := existingErr == nil

	if err := s.gitCredentialStore.ReplaceCredential(
		context.Background(),
		referenceID,
		payload,
		time.Now().UTC(),
	); err != nil {
		return false, err
	}

	return !existed, nil
}

func (s *Server) deleteStoredCredentialByReferenceID(referenceID string) error {
	if s.gitCredentialStore == nil || strings.TrimSpace(referenceID) == "" {
		return nil
	}
	_, err := s.gitCredentialStore.DeleteByRepoID(context.Background(), referenceID)
	return err
}

func repoAuthModeForAPI(mode string) string {
	normalized := strings.TrimSpace(strings.ToLower(mode))
	if normalized == "" {
		return git.GitRepoAuthModeNone
	}
	return normalized
}

func repoAuthSourceForAPI(mode, source string) string {
	normalized := strings.TrimSpace(strings.ToLower(source))
	if normalized == "" && mode == git.GitRepoAuthModeNone {
		return git.GitRepoAuthSourceNone
	}
	return normalized
}

func hasAnyAuthReferenceFields(auth git.GitRepoAuthConfig) bool {
	return strings.TrimSpace(auth.ReferenceID) != "" ||
		hasEnvFileReferenceFields(auth)
}

func hasEnvFileReferenceFields(auth git.GitRepoAuthConfig) bool {
	return strings.TrimSpace(auth.UsernameRef) != "" ||
		strings.TrimSpace(auth.PasswordRef) != "" ||
		strings.TrimSpace(auth.TokenRef) != "" ||
		strings.TrimSpace(auth.KeyRef) != "" ||
		strings.TrimSpace(auth.KnownHostsRef) != ""
}

func hasEnvFileCredentialsConfigured(mode string, auth git.GitRepoAuthConfig) bool {
	switch mode {
	case git.GitRepoAuthModeHTTPSToken:
		return strings.TrimSpace(auth.TokenRef) != ""
	case git.GitRepoAuthModeHTTPSBasic:
		return strings.TrimSpace(auth.UsernameRef) != "" &&
			strings.TrimSpace(auth.PasswordRef) != ""
	case git.GitRepoAuthModeSSHKey:
		return strings.TrimSpace(auth.KeyRef) != "" &&
			strings.TrimSpace(auth.KnownHostsRef) != ""
	default:
		return false
	}
}

func hasStoredCredentialWriteInput(input *GitRepoStoredCredentialWriteRequest) bool {
	if input == nil {
		return false
	}
	return strings.TrimSpace(input.Username) != "" ||
		strings.TrimSpace(input.Password) != "" ||
		strings.TrimSpace(input.Token) != "" ||
		strings.TrimSpace(input.PrivateKey) != "" ||
		strings.TrimSpace(input.Passphrase) != "" ||
		strings.TrimSpace(input.KnownHosts) != ""
}

func resolveStoredCredentialReferenceID(auth git.GitRepoAuthConfig, repoID string) string {
	if ref := strings.TrimSpace(auth.ReferenceID); ref != "" {
		return ref
	}
	return strings.TrimSpace(repoID)
}

func usesImplicitStoredCredentialReference(repo git.GitRepoConfig) bool {
	return repoAuthSourceForAPI(repoAuthModeForAPI(repo.Auth.Mode), repo.Auth.Source) == git.GitRepoAuthSourceStored &&
		strings.TrimSpace(repo.Auth.ReferenceID) == ""
}

func shouldDeleteImplicitStoredCredential(oldRepo, newRepo git.GitRepoConfig) bool {
	if !usesImplicitStoredCredentialReference(oldRepo) {
		return false
	}
	oldReferenceID := resolveStoredCredentialReferenceID(oldRepo.Auth, oldRepo.ID)
	if oldReferenceID == "" {
		return false
	}

	newSource := repoAuthSourceForAPI(repoAuthModeForAPI(newRepo.Auth.Mode), newRepo.Auth.Source)
	if newSource != git.GitRepoAuthSourceStored {
		return true
	}

	newReferenceID := resolveStoredCredentialReferenceID(newRepo.Auth, newRepo.ID)
	return oldReferenceID != newReferenceID
}

func removeRepoByID(repos []git.GitRepoConfig, repoID string) []git.GitRepoConfig {
	filtered := make([]git.GitRepoConfig, 0, len(repos))
	for _, repo := range repos {
		if repo.ID == repoID {
			continue
		}
		filtered = append(filtered, repo)
	}
	return filtered
}

func enabledGitRepoCheckoutNames(repos []git.GitRepoConfig) []string {
	enabledRepoNames := make([]string, 0, len(repos))
	for _, repo := range repos {
		if !repo.Enabled {
			continue
		}
		enabledRepoNames = append(enabledRepoNames, git.ResolveRepoCheckoutName(repo))
	}
	return enabledRepoNames
}

// Helper functions

func writeFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}

func deleteFile(filename string) error {
	return os.Remove(filename)
}
