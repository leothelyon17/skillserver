package mcp

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mudler/skillserver/pkg/domain"
)

// ExportCatalogItemsInput is the input for export_catalog_items.
type ExportCatalogItemsInput struct {
	ItemIDs              []string `json:"item_ids" jsonschema:"Catalog item identifiers to export (skill/prompt/rule item IDs)."`
	Format               string   `json:"format,omitempty" jsonschema:"Optional export format ('tar.gz')."`
	DryRun               bool     `json:"dry_run,omitempty" jsonschema:"When true, returns only export planning metadata without archive bytes."`
	ArchiveRootMode      string   `json:"archive_root_mode,omitempty" jsonschema:"Optional archive root mode ('flat' or 'materialized'). Defaults to 'flat'."`
	IncludeArchiveBase64 *bool    `json:"include_archive_base64,omitempty" jsonschema:"When true, include archive bytes inline as base64. Defaults to false."`
}

// ExportCatalogItemsDownload describes archive output metadata.
type ExportCatalogItemsDownload struct {
	FileName      string `json:"file_name"`
	ContentType   string `json:"content_type"`
	ContentLength int    `json:"content_length"`
	ArchiveBase64 string `json:"archive_base64,omitempty"`
}

// ExportCatalogItemsOutput is the output for export_catalog_items.
type ExportCatalogItemsOutput struct {
	Format   domain.CatalogExportFormat   `json:"format"`
	DryRun   bool                         `json:"dry_run"`
	Manifest domain.CatalogExportManifest `json:"manifest"`
	Download *ExportCatalogItemsDownload  `json:"download,omitempty"`
}

type catalogExportArchiveRootMode string

const (
	catalogExportArchiveRootModeFlat         catalogExportArchiveRootMode = "flat"
	catalogExportArchiveRootModeMaterialized catalogExportArchiveRootMode = "materialized"
)

// MaterializeCatalogItemsInput is the input for materialize_catalog_items.
type MaterializeCatalogItemsInput struct {
	ItemIDs        []string `json:"item_ids" jsonschema:"Catalog item identifiers to materialize (skill/prompt/rule item IDs)."`
	DestinationDir string   `json:"destination_dir" jsonschema:"Absolute destination directory where catalog items should be materialized."`
	ConflictPolicy string   `json:"conflict_policy,omitempty" jsonschema:"Optional conflict policy ('error', 'overwrite', or 'skip')."`
	DryRun         bool     `json:"dry_run,omitempty" jsonschema:"When true, returns only planning results and writes no files."`
}

// MaterializeCatalogItemsOutput is the output for materialize_catalog_items.
type MaterializeCatalogItemsOutput = domain.CatalogMaterializationResult

func exportCatalogItems(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ExportCatalogItemsInput,
	manager domain.SkillManager,
) (
	*mcp.CallToolResult,
	ExportCatalogItemsOutput,
	error,
) {
	if manager == nil {
		return nil, ExportCatalogItemsOutput{}, fmt.Errorf("skill manager is required")
	}

	format := domain.CatalogExportFormat(strings.TrimSpace(input.Format))
	result, err := executeCatalogExportViaMaterialization(
		ctx,
		manager,
		domain.CatalogExportRequest{
			ItemIDs: input.ItemIDs,
			Format:  format,
			DryRun:  input.DryRun,
		},
		input.ArchiveRootMode,
	)
	if err != nil {
		return nil, ExportCatalogItemsOutput{}, err
	}

	output := ExportCatalogItemsOutput{
		Format:   result.Format,
		DryRun:   result.DryRun,
		Manifest: result.Manifest,
	}

	includeArchiveBase64 := input.IncludeArchiveBase64 != nil && *input.IncludeArchiveBase64
	if !result.DryRun {
		contentType := strings.TrimSpace(result.ContentType)
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		download := &ExportCatalogItemsDownload{
			FileName:      result.FileName,
			ContentType:   contentType,
			ContentLength: len(result.ArchiveData),
		}
		if includeArchiveBase64 {
			download.ArchiveBase64 = base64.StdEncoding.EncodeToString(result.ArchiveData)
		}
		output.Download = download
	}

	return nil, output, nil
}

func materializeCatalogItems(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input MaterializeCatalogItemsInput,
	manager domain.SkillManager,
	allowedDestinationRoots []string,
) (
	*mcp.CallToolResult,
	MaterializeCatalogItemsOutput,
	error,
) {
	if manager == nil {
		return nil, MaterializeCatalogItemsOutput{}, fmt.Errorf("skill manager is required")
	}

	materializationService, err := domain.NewCatalogMaterializationService(
		manager,
		allowedDestinationRoots,
	)
	if err != nil {
		return nil, MaterializeCatalogItemsOutput{}, err
	}

	request := domain.CatalogMaterializationRequest{
		ItemIDs:        input.ItemIDs,
		DestinationDir: strings.TrimSpace(input.DestinationDir),
		ConflictPolicy: domain.CatalogMaterializeConflictPolicy(strings.TrimSpace(input.ConflictPolicy)),
		DryRun:         input.DryRun,
	}
	result, err := materializationService.Materialize(ctx, request)
	if err != nil {
		return nil, MaterializeCatalogItemsOutput{}, err
	}

	return nil, result, nil
}

func executeCatalogExportViaMaterialization(
	ctx context.Context,
	manager domain.SkillManager,
	request domain.CatalogExportRequest,
	rawArchiveRootMode string,
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

	archiveRootMode, err := normalizeCatalogExportArchiveRootMode(rawArchiveRootMode)
	if err != nil {
		return domain.CatalogExportResult{}, err
	}

	stagingRoot, err := os.MkdirTemp("", "skillserver-mcp-export-*")
	if err != nil {
		return domain.CatalogExportResult{}, fmt.Errorf("create export staging root: %w", err)
	}
	defer os.RemoveAll(stagingRoot)

	materializationService, err := domain.NewCatalogMaterializationService(manager, []string{stagingRoot})
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

	manifest, err := buildCatalogExportManifestFromMaterializationResult(materializedResult, archiveRootMode)
	if err != nil {
		return domain.CatalogExportResult{}, fmt.Errorf("build catalog export manifest: %w", err)
	}
	result := domain.CatalogExportResult{
		Format:   format,
		DryRun:   request.DryRun,
		Manifest: manifest,
	}
	if request.DryRun {
		return result, nil
	}

	archiveData, err := buildCatalogExportArchiveFromMaterializationResult(
		materializedResult,
		archiveRootMode,
	)
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
	mode catalogExportArchiveRootMode,
) (domain.CatalogExportManifest, error) {
	manifestItems := make([]domain.CatalogExportManifestItem, 0, len(result.Items))
	for _, item := range result.Items {
		archiveRoot := strings.TrimSpace(item.TargetPath)
		if archiveRoot == "" && len(item.Files) > 0 {
			archiveRoot = strings.TrimSpace(item.Files[0].TargetPath)
		}
		archiveRoot, err := transformCatalogExportArchivePath(archiveRoot, mode)
		if err != nil {
			return domain.CatalogExportManifest{}, err
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
	return domain.CatalogExportManifest{Items: manifestItems}, nil
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

func buildCatalogExportArchiveFromMaterializationResult(
	result domain.CatalogMaterializationResult,
	mode catalogExportArchiveRootMode,
) ([]byte, error) {
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	writtenPaths := make(map[string]struct{})
	for _, item := range result.Items {
		for _, file := range item.Files {
			archivePath, err := transformCatalogExportArchivePath(file.TargetPath, mode)
			if err != nil {
				_ = tarWriter.Close()
				_ = gzipWriter.Close()
				return nil, err
			}
			if archivePath == "" || archivePath == "." {
				_ = tarWriter.Close()
				_ = gzipWriter.Close()
				return nil, fmt.Errorf("invalid archive path %q", archivePath)
			}
			if _, exists := writtenPaths[archivePath]; exists {
				_ = tarWriter.Close()
				_ = gzipWriter.Close()
				return nil, fmt.Errorf("duplicate archive path %q", archivePath)
			}
			writtenPaths[archivePath] = struct{}{}

			info, err := os.Stat(file.ResolvedPath)
			if err != nil {
				_ = tarWriter.Close()
				_ = gzipWriter.Close()
				return nil, err
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				_ = tarWriter.Close()
				_ = gzipWriter.Close()
				return nil, err
			}
			header.Name = archivePath
			if err := tarWriter.WriteHeader(header); err != nil {
				_ = tarWriter.Close()
				_ = gzipWriter.Close()
				return nil, err
			}

			content, err := os.ReadFile(file.ResolvedPath)
			if err != nil {
				_ = tarWriter.Close()
				_ = gzipWriter.Close()
				return nil, err
			}
			if _, err := tarWriter.Write(content); err != nil {
				_ = tarWriter.Close()
				_ = gzipWriter.Close()
				return nil, err
			}
		}
	}

	if err := tarWriter.Close(); err != nil {
		_ = gzipWriter.Close()
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func normalizeCatalogExportArchiveRootMode(raw string) (catalogExportArchiveRootMode, error) {
	normalized := catalogExportArchiveRootMode(strings.ToLower(strings.TrimSpace(raw)))
	if normalized == "" {
		return catalogExportArchiveRootModeFlat, nil
	}
	switch normalized {
	case catalogExportArchiveRootModeFlat, catalogExportArchiveRootModeMaterialized:
		return normalized, nil
	default:
		return "", fmt.Errorf(
			"%w: archive_root_mode must be one of: flat, materialized",
			domain.ErrCatalogExportInvalidRequest,
		)
	}
}

func transformCatalogExportArchivePath(
	targetPath string,
	mode catalogExportArchiveRootMode,
) (string, error) {
	normalized := filepath.ToSlash(filepath.Clean(strings.TrimSpace(targetPath)))
	normalized = strings.TrimPrefix(normalized, "./")
	if normalized == "" || normalized == "." || strings.HasPrefix(normalized, "../") {
		return "", fmt.Errorf("invalid archive path %q", targetPath)
	}
	if mode != catalogExportArchiveRootModeFlat {
		return normalized, nil
	}

	parts := strings.Split(normalized, "/")
	if len(parts) == 0 {
		return normalized, nil
	}
	switch parts[0] {
	case "skills", "prompts", "rules":
		if len(parts) == 1 {
			return "", fmt.Errorf("invalid archive path %q", targetPath)
		}
		normalized = strings.Join(parts[1:], "/")
	}
	if normalized == "" || normalized == "." || strings.HasPrefix(normalized, "../") {
		return "", fmt.Errorf("invalid archive path %q", targetPath)
	}
	return normalized, nil
}
