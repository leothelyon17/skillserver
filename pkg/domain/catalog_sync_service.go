package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/mudler/skillserver/pkg/persistence"
)

type catalogSyncSourceRepository interface {
	Upsert(ctx context.Context, row persistence.CatalogSourceRow) error
	List(ctx context.Context, filter persistence.CatalogSourceListFilter) ([]persistence.CatalogSourceRow, error)
	SoftDeleteByItemID(ctx context.Context, itemID string, deletedAt time.Time) (bool, error)
}

// CatalogSyncServiceOptions configures runtime behavior for catalog synchronization.
type CatalogSyncServiceOptions struct {
	Logger *log.Logger
	Now    func() time.Time
}

// CatalogSyncService synchronizes discovered catalog items into persistence snapshots.
type CatalogSyncService struct {
	sourceRepo catalogSyncSourceRepository
	logger     *log.Logger
	now        func() time.Time
}

type catalogSyncScope struct {
	mode     string
	repoName string
}

type catalogSyncSummary struct {
	scope          catalogSyncScope
	discovered     int
	scopedExisting int
	upserted       int
	tombstoned     int
	restored       int
	unchanged      int
}

const (
	catalogSyncModeFull = "full"
	catalogSyncModeRepo = "repo"
)

// NewCatalogSyncService creates a catalog synchronization orchestrator.
func NewCatalogSyncService(
	sourceRepo catalogSyncSourceRepository,
	options CatalogSyncServiceOptions,
) (*CatalogSyncService, error) {
	if sourceRepo == nil {
		return nil, fmt.Errorf("catalog sync source repository is required")
	}

	logger := options.Logger
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	now := options.Now
	if now == nil {
		now = time.Now
	}

	return &CatalogSyncService{
		sourceRepo: sourceRepo,
		logger:     logger,
		now:        now,
	}, nil
}

// SyncAll reconciles persistence state to the current discovered catalog snapshot.
func (s *CatalogSyncService) SyncAll(discovered []CatalogItem) error {
	if s == nil {
		return fmt.Errorf("catalog sync service is required")
	}

	summary, err := s.sync(catalogSyncScope{mode: catalogSyncModeFull}, discovered)
	if err != nil {
		return fmt.Errorf("sync all catalog items: %w", err)
	}
	s.logSummary(summary)
	return nil
}

// SyncRepo reconciles persistence state for one Git repository only.
func (s *CatalogSyncService) SyncRepo(repoName string, discovered []CatalogItem) error {
	if s == nil {
		return fmt.Errorf("catalog sync service is required")
	}

	normalizedRepoName := strings.TrimSpace(repoName)
	if normalizedRepoName == "" {
		return fmt.Errorf("catalog sync repository name is required")
	}

	summary, err := s.sync(catalogSyncScope{
		mode:     catalogSyncModeRepo,
		repoName: normalizedRepoName,
	}, discovered)
	if err != nil {
		return fmt.Errorf("sync catalog repository %q: %w", normalizedRepoName, err)
	}
	s.logSummary(summary)
	return nil
}

func (s *CatalogSyncService) sync(
	scope catalogSyncScope,
	discovered []CatalogItem,
) (catalogSyncSummary, error) {
	syncedAt := s.now().UTC()

	discoveredRowsByID, err := buildDiscoveredSourceRows(scope, discovered, syncedAt)
	if err != nil {
		return catalogSyncSummary{}, err
	}

	existingRows, err := s.listScopedExistingRows(scope)
	if err != nil {
		return catalogSyncSummary{}, err
	}

	summary := catalogSyncSummary{
		scope:          scope,
		discovered:     len(discoveredRowsByID),
		scopedExisting: len(existingRows),
	}

	existingByID := make(map[string]persistence.CatalogSourceRow, len(existingRows))
	for _, row := range existingRows {
		existingByID[row.ItemID] = row
	}

	discoveredIDs := sortedCatalogSourceRowIDs(discoveredRowsByID)
	for _, itemID := range discoveredIDs {
		discoveredRow := discoveredRowsByID[itemID]
		existingRow, exists := existingByID[itemID]
		if exists && catalogSourceRowsEqualForSync(existingRow, discoveredRow) {
			summary.unchanged++
			delete(existingByID, itemID)
			continue
		}

		if err := s.sourceRepo.Upsert(context.Background(), discoveredRow); err != nil {
			return summary, fmt.Errorf("upsert source row %q: %w", itemID, err)
		}

		summary.upserted++
		if exists && existingRow.DeletedAt != nil {
			summary.restored++
		}
		delete(existingByID, itemID)
	}

	missingIDs := sortedCatalogSourceRowIDs(existingByID)
	for _, itemID := range missingIDs {
		existingRow := existingByID[itemID]
		if existingRow.DeletedAt != nil {
			summary.unchanged++
			continue
		}

		deleted, err := s.sourceRepo.SoftDeleteByItemID(context.Background(), itemID, syncedAt)
		if err != nil {
			return summary, fmt.Errorf("soft-delete missing source row %q: %w", itemID, err)
		}
		if deleted {
			summary.tombstoned++
			continue
		}
		summary.unchanged++
	}

	return summary, nil
}

func (s *CatalogSyncService) listScopedExistingRows(scope catalogSyncScope) ([]persistence.CatalogSourceRow, error) {
	if scope.mode == catalogSyncModeRepo {
		gitSourceType := persistence.CatalogSourceTypeGit
		repoName := scope.repoName
		rows, err := s.sourceRepo.List(context.Background(), persistence.CatalogSourceListFilter{
			SourceType:     &gitSourceType,
			SourceRepo:     &repoName,
			IncludeDeleted: true,
		})
		if err != nil {
			return nil, fmt.Errorf("list existing source rows for repo %q: %w", scope.repoName, err)
		}
		return rows, nil
	}

	rows, err := s.sourceRepo.List(context.Background(), persistence.CatalogSourceListFilter{
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, fmt.Errorf("list existing source rows: %w", err)
	}
	return rows, nil
}

func buildDiscoveredSourceRows(
	scope catalogSyncScope,
	discovered []CatalogItem,
	syncedAt time.Time,
) (map[string]persistence.CatalogSourceRow, error) {
	rowsByID := make(map[string]persistence.CatalogSourceRow, len(discovered))

	for index, item := range discovered {
		row, err := mapCatalogItemToSourceRow(item, syncedAt)
		if err != nil {
			return nil, fmt.Errorf("map discovered item at index %d (%q): %w", index, item.ID, err)
		}

		if scope.mode == catalogSyncModeRepo {
			if row.SourceType != persistence.CatalogSourceTypeGit {
				continue
			}
			if row.SourceRepo == nil || strings.TrimSpace(*row.SourceRepo) != scope.repoName {
				continue
			}
		}

		existingRow, exists := rowsByID[row.ItemID]
		if !exists {
			rowsByID[row.ItemID] = row
			continue
		}

		if !catalogSourceRowsEqualForSync(existingRow, row) {
			return nil, fmt.Errorf("duplicate discovered catalog item %q has conflicting snapshots", row.ItemID)
		}
	}

	return rowsByID, nil
}

func mapCatalogItemToSourceRow(item CatalogItem, syncedAt time.Time) (persistence.CatalogSourceRow, error) {
	itemID := strings.TrimSpace(item.ID)
	if itemID == "" {
		return persistence.CatalogSourceRow{}, fmt.Errorf("catalog item id is required")
	}

	classifier, err := mapCatalogClassifier(item.Classifier, itemID)
	if err != nil {
		return persistence.CatalogSourceRow{}, err
	}

	skillID := resolveCatalogSkillID(item)
	repoName := resolveCatalogRepoName(skillID)

	sourceType := persistence.CatalogSourceTypeLocal
	if item.ReadOnly {
		sourceType = persistence.CatalogSourceTypeGit
	} else if isFileImportCatalogItem(skillID) {
		sourceType = persistence.CatalogSourceTypeFileImport
	}

	name := strings.TrimSpace(item.Name)
	if name == "" {
		name = deriveCatalogItemName(item, skillID)
	}
	if name == "" {
		return persistence.CatalogSourceRow{}, fmt.Errorf("catalog item name is required")
	}

	description := strings.TrimSpace(item.Description)
	resourcePath := normalizeCatalogOptionalPath(item.ResourcePath)

	var parentSkillID *string
	if classifier == persistence.CatalogClassifierPrompt {
		if skillID == "" {
			return persistence.CatalogSourceRow{}, fmt.Errorf("prompt catalog item parent skill id is required")
		}
		parentSkillID = stringPointer(skillID)
	}

	var sourceRepo *string
	if sourceType == persistence.CatalogSourceTypeGit {
		sourceRepo = repoName
	}

	return persistence.CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       classifier,
		SourceType:       sourceType,
		SourceRepo:       sourceRepo,
		ParentSkillID:    parentSkillID,
		ResourcePath:     resourcePath,
		Name:             name,
		Description:      description,
		Content:          item.Content,
		ContentHash:      buildCatalogContentHash(item.Content),
		ContentWritable:  !item.ReadOnly,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
		DeletedAt:        nil,
	}, nil
}

func mapCatalogClassifier(
	classifier CatalogClassifier,
	itemID string,
) (persistence.CatalogClassifier, error) {
	switch classifier {
	case CatalogClassifierSkill:
		return persistence.CatalogClassifierSkill, nil
	case CatalogClassifierPrompt:
		return persistence.CatalogClassifierPrompt, nil
	}

	trimmedID := strings.TrimSpace(itemID)
	switch {
	case strings.HasPrefix(trimmedID, skillCatalogIDPrefix):
		return persistence.CatalogClassifierSkill, nil
	case strings.HasPrefix(trimmedID, promptCatalogIDPrefix):
		return persistence.CatalogClassifierPrompt, nil
	default:
		return "", fmt.Errorf("catalog classifier %q is invalid", classifier)
	}
}

func resolveCatalogSkillID(item CatalogItem) string {
	if trimmed := CanonicalSkillCatalogKey(item.ParentSkillID); trimmed != "" {
		return trimmed
	}

	itemID := strings.TrimSpace(item.ID)
	if itemID == "" {
		return ""
	}

	if strings.HasPrefix(itemID, skillCatalogIDPrefix) {
		return strings.TrimPrefix(itemID, skillCatalogIDPrefix)
	}

	if strings.HasPrefix(itemID, promptCatalogIDPrefix) {
		payload := strings.TrimPrefix(itemID, promptCatalogIDPrefix)
		parts := strings.SplitN(payload, ":", 2)
		if len(parts) == 2 {
			return parts[0]
		}
	}

	return ""
}

func resolveCatalogRepoName(skillID string) *string {
	trimmedSkillID := strings.TrimSpace(skillID)
	if trimmedSkillID == "" {
		return nil
	}

	parts := strings.SplitN(trimmedSkillID, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	repoName := strings.TrimSpace(parts[0])
	if repoName == "" {
		return nil
	}
	return &repoName
}

func deriveCatalogItemName(item CatalogItem, skillID string) string {
	if normalizedPath := CanonicalPromptCatalogResourcePath(item.ResourcePath); normalizedPath != "" {
		baseName := strings.TrimSpace(path.Base(normalizedPath))
		if baseName != "" && baseName != "." && baseName != "/" {
			return baseName
		}
	}

	if skillID != "" {
		baseName := strings.TrimSpace(path.Base(skillID))
		if baseName != "" && baseName != "." && baseName != "/" {
			return baseName
		}
	}

	trimmedID := strings.TrimSpace(item.ID)
	if trimmedID != "" {
		return trimmedID
	}
	return ""
}

func normalizeCatalogOptionalPath(raw string) *string {
	normalized := CanonicalPromptCatalogResourcePath(raw)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func isFileImportCatalogItem(skillID string) bool {
	normalized := strings.ToLower(strings.TrimSpace(skillID))
	switch {
	case strings.HasPrefix(normalized, "file-import/"):
		return true
	case strings.HasPrefix(normalized, "file_import/"):
		return true
	case strings.HasPrefix(normalized, "imported/"):
		return true
	default:
		return false
	}
}

func buildCatalogContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func catalogSourceRowsEqualForSync(
	existing persistence.CatalogSourceRow,
	discovered persistence.CatalogSourceRow,
) bool {
	if existing.ItemID != discovered.ItemID {
		return false
	}
	if existing.Classifier != discovered.Classifier {
		return false
	}
	if existing.SourceType != discovered.SourceType {
		return false
	}
	if !optionalCatalogStringEquals(existing.SourceRepo, discovered.SourceRepo) {
		return false
	}
	if !optionalCatalogStringEquals(existing.ParentSkillID, discovered.ParentSkillID) {
		return false
	}
	if !optionalCatalogStringEquals(existing.ResourcePath, discovered.ResourcePath) {
		return false
	}
	if existing.Name != discovered.Name {
		return false
	}
	if existing.Description != discovered.Description {
		return false
	}
	if existing.Content != discovered.Content {
		return false
	}
	if existing.ContentHash != discovered.ContentHash {
		return false
	}
	if existing.ContentWritable != discovered.ContentWritable {
		return false
	}
	if existing.MetadataWritable != discovered.MetadataWritable {
		return false
	}
	if existing.DeletedAt != nil || discovered.DeletedAt != nil {
		return false
	}
	return true
}

func optionalCatalogStringEquals(left *string, right *string) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func sortedCatalogSourceRowIDs(rowsByID map[string]persistence.CatalogSourceRow) []string {
	ids := make([]string, 0, len(rowsByID))
	for itemID := range rowsByID {
		ids = append(ids, itemID)
	}
	sort.Strings(ids)
	return ids
}

func (s *CatalogSyncService) logSummary(summary catalogSyncSummary) {
	repoName := "*"
	if summary.scope.repoName != "" {
		repoName = summary.scope.repoName
	}

	s.logger.Printf(
		"catalog sync completed mode=%s repo=%s discovered=%d existing=%d upserted=%d tombstoned=%d restored=%d unchanged=%d",
		summary.scope.mode,
		repoName,
		summary.discovered,
		summary.scopedExisting,
		summary.upserted,
		summary.tombstoned,
		summary.restored,
		summary.unchanged,
	)
}

func stringPointer(value string) *string {
	return &value
}
