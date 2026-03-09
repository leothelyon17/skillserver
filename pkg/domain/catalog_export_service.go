package domain

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	// ErrCatalogExportInvalidRequest indicates malformed export inputs.
	ErrCatalogExportInvalidRequest = errors.New("catalog export request is invalid")
	// ErrCatalogExportItemNotFound indicates a requested export item was not resolved.
	ErrCatalogExportItemNotFound = errors.New("catalog export item not found")
	// ErrCatalogExportUnsupportedClassifier indicates the requested classifier is not implemented yet.
	ErrCatalogExportUnsupportedClassifier = errors.New("catalog export classifier is not supported")
)

// CatalogExportFormat identifies the output artifact format.
type CatalogExportFormat string

const (
	// CatalogExportFormatTarGz is the legacy-compatible compressed archive format.
	CatalogExportFormatTarGz CatalogExportFormat = "tar.gz"
)

// CatalogExportRequest describes one export operation for one or more catalog items.
type CatalogExportRequest struct {
	ItemIDs []string            `json:"item_ids"`
	Format  CatalogExportFormat `json:"format,omitempty"`
	DryRun  bool                `json:"dry_run,omitempty"`
}

// CatalogExportManifestItem describes the planned export output for one catalog item.
type CatalogExportManifestItem struct {
	ItemID          string            `json:"item_id"`
	Classifier      CatalogClassifier `json:"classifier"`
	SourceRef       string            `json:"source_ref,omitempty"`
	ArchiveRoot     string            `json:"archive_root"`
	ArchiveFileName string            `json:"archive_file_name"`
}

// CatalogExportManifest is the dry-run/execution plan shared across adapters.
type CatalogExportManifest struct {
	Items []CatalogExportManifestItem `json:"items"`
}

// CatalogExportResult contains manifest metadata and optional archive payload bytes.
type CatalogExportResult struct {
	Format      CatalogExportFormat   `json:"format"`
	DryRun      bool                  `json:"dry_run"`
	ContentType string                `json:"content_type,omitempty"`
	FileName    string                `json:"file_name,omitempty"`
	ArchiveData []byte                `json:"-"`
	Manifest    CatalogExportManifest `json:"manifest"`
}

type catalogExportSkillReader interface {
	ReadSkill(name string) (*Skill, error)
}

// CatalogExportService provides classifier-aware catalog export orchestration.
//
// WP-001 intentionally implements skill export first while keeping request/result
// models classifier-agnostic for later prompt/rule expansion.
type CatalogExportService struct {
	skillReader catalogExportSkillReader
}

// NewCatalogExportService constructs a shared catalog export service seam.
func NewCatalogExportService(skillReader catalogExportSkillReader) (*CatalogExportService, error) {
	if skillReader == nil {
		return nil, fmt.Errorf("catalog export skill reader is required")
	}

	return &CatalogExportService{
		skillReader: skillReader,
	}, nil
}

// Export executes one catalog export request and returns payload/manifest data.
func (s *CatalogExportService) Export(ctx context.Context, request CatalogExportRequest) (CatalogExportResult, error) {
	if s == nil {
		return CatalogExportResult{}, fmt.Errorf("catalog export service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return CatalogExportResult{}, ctx.Err()
	default:
	}

	format := request.Format
	if format == "" {
		format = CatalogExportFormatTarGz
	}
	if format != CatalogExportFormatTarGz {
		return CatalogExportResult{}, fmt.Errorf(
			"%w: export format %q is not supported",
			ErrCatalogExportInvalidRequest,
			format,
		)
	}

	itemIDs := normalizeCatalogExportItemIDs(request.ItemIDs)
	if len(itemIDs) == 0 {
		return CatalogExportResult{}, fmt.Errorf(
			"%w: at least one item id is required",
			ErrCatalogExportInvalidRequest,
		)
	}

	// Batching is additive work for later WPs; keep current behavior explicit.
	if len(itemIDs) > 1 {
		return CatalogExportResult{}, fmt.Errorf(
			"%w: batch export is not supported in this phase",
			ErrCatalogExportInvalidRequest,
		)
	}

	manifestItem, skill, err := s.resolveSkillExportItem(itemIDs[0])
	if err != nil {
		return CatalogExportResult{}, err
	}

	result := CatalogExportResult{
		Format: format,
		DryRun: request.DryRun,
		Manifest: CatalogExportManifest{
			Items: []CatalogExportManifestItem{manifestItem},
		},
	}

	if request.DryRun {
		return result, nil
	}

	archiveData, err := buildSkillArchive(skill.SourcePath)
	if err != nil {
		return CatalogExportResult{}, fmt.Errorf(
			"build archive for item %q: %w",
			manifestItem.ItemID,
			err,
		)
	}

	result.ContentType = catalogExportFormatContentType(format)
	result.FileName = manifestItem.ArchiveFileName
	result.ArchiveData = archiveData
	return result, nil
}

func (s *CatalogExportService) resolveSkillExportItem(itemID string) (CatalogExportManifestItem, *Skill, error) {
	reference, err := NormalizeCatalogItemReference(itemID)
	if err != nil {
		return CatalogExportManifestItem{}, nil, fmt.Errorf("%w: %v", ErrCatalogExportInvalidRequest, err)
	}

	if reference.Classifier != CatalogClassifierSkill {
		return CatalogExportManifestItem{}, nil, fmt.Errorf(
			"%w: classifier %q",
			ErrCatalogExportUnsupportedClassifier,
			reference.Classifier,
		)
	}

	skill, err := s.skillReader.ReadSkill(reference.SkillID)
	if err != nil {
		return CatalogExportManifestItem{}, nil, fmt.Errorf(
			"%w: %q",
			ErrCatalogExportItemNotFound,
			itemID,
		)
	}

	archiveRoot := filepath.Base(skill.SourcePath)
	if archiveRoot == "." || archiveRoot == string(filepath.Separator) {
		archiveRoot = filepath.Base(reference.SkillID)
	}

	return CatalogExportManifestItem{
		ItemID:          reference.ItemID,
		Classifier:      CatalogClassifierSkill,
		SourceRef:       reference.SkillID,
		ArchiveRoot:     archiveRoot,
		ArchiveFileName: buildSkillArchiveFileName(skill.ID),
	}, skill, nil
}

func normalizeCatalogExportItemIDs(itemIDs []string) []string {
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

func buildSkillArchiveFileName(skillID string) string {
	normalized := CanonicalSkillCatalogKey(skillID)
	if normalized == "" {
		return "skill-export.tar.gz"
	}

	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	return replacer.Replace(normalized) + ".tar.gz"
}

func catalogExportFormatContentType(format CatalogExportFormat) string {
	switch format {
	case CatalogExportFormatTarGz:
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}
