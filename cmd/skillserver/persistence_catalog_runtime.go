package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/mudler/skillserver/pkg/domain"
	"github.com/mudler/skillserver/pkg/persistence"
)

type catalogPersistenceRuntime struct {
	db                      *sql.DB
	sourceRepo              *persistence.CatalogSourceRepository
	overlayRepo             *persistence.CatalogMetadataOverlayRepository
	taxonomyAssignment      *domain.CatalogTaxonomyAssignmentService
	taxonomyRegistryService *domain.CatalogTaxonomyRegistryService
	coordinator             *catalogPersistenceCoordinator
}

type catalogPersistenceCoordinator struct {
	fsManager        *domain.FileSystemManager
	syncService      *domain.CatalogSyncService
	backfillService  *domain.CatalogTaxonomyLegacyLabelBackfillService
	effectiveService *domain.CatalogEffectiveService
	logger           *log.Logger
}

func bootstrapCatalogPersistenceRuntime(
	ctx context.Context,
	cfg PersistenceRuntimeConfig,
	fsManager *domain.FileSystemManager,
	logger *log.Logger,
) (*catalogPersistenceRuntime, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if fsManager == nil {
		return nil, fmt.Errorf("file system manager is required for persistence runtime")
	}
	if strings.TrimSpace(cfg.DBPath) == "" {
		return nil, fmt.Errorf("persistence database path is required when persistence mode is enabled")
	}

	db, err := persistence.BootstrapSQLite(ctx, cfg.DBPath, persistence.SQLiteBootstrapConfig{})
	if err != nil {
		return nil, fmt.Errorf("bootstrap persistence sqlite at %q: %w", cfg.DBPath, err)
	}

	sourceRepo, err := persistence.NewCatalogSourceRepository(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize catalog source repository: %w", err)
	}
	overlayRepo, err := persistence.NewCatalogMetadataOverlayRepository(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize catalog overlay repository: %w", err)
	}
	taxonomyAssignmentRepo, err := persistence.NewCatalogItemTaxonomyAssignmentRepository(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize catalog taxonomy assignment repository: %w", err)
	}
	tagAssignmentRepo, err := persistence.NewCatalogItemTagAssignmentRepository(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize catalog tag assignment repository: %w", err)
	}
	domainRepo, err := persistence.NewCatalogDomainRepository(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize catalog domain repository: %w", err)
	}
	subdomainRepo, err := persistence.NewCatalogSubdomainRepository(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize catalog subdomain repository: %w", err)
	}
	tagRepo, err := persistence.NewCatalogTagRepository(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize catalog tag repository: %w", err)
	}

	coordinator, err := newCatalogPersistenceCoordinator(
		fsManager,
		sourceRepo,
		overlayRepo,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
		domainRepo,
		subdomainRepo,
		tagRepo,
		logger,
	)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	taxonomyRegistryService, err := domain.NewCatalogTaxonomyRegistryService(
		domainRepo,
		subdomainRepo,
		tagRepo,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
		domain.CatalogTaxonomyRegistryServiceOptions{},
	)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize catalog taxonomy registry service: %w", err)
	}

	taxonomyAssignmentService, err := domain.NewCatalogTaxonomyAssignmentService(
		sourceRepo,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
		domainRepo,
		subdomainRepo,
		tagRepo,
		domain.CatalogTaxonomyAssignmentServiceOptions{},
	)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize catalog taxonomy assignment service: %w", err)
	}

	return &catalogPersistenceRuntime{
		db:                      db,
		sourceRepo:              sourceRepo,
		overlayRepo:             overlayRepo,
		taxonomyAssignment:      taxonomyAssignmentService,
		taxonomyRegistryService: taxonomyRegistryService,
		coordinator:             coordinator,
	}, nil
}

func newCatalogPersistenceCoordinator(
	fsManager *domain.FileSystemManager,
	sourceRepo *persistence.CatalogSourceRepository,
	overlayRepo *persistence.CatalogMetadataOverlayRepository,
	taxonomyAssignmentRepo *persistence.CatalogItemTaxonomyAssignmentRepository,
	tagAssignmentRepo *persistence.CatalogItemTagAssignmentRepository,
	domainRepo *persistence.CatalogDomainRepository,
	subdomainRepo *persistence.CatalogSubdomainRepository,
	tagRepo *persistence.CatalogTagRepository,
	logger *log.Logger,
) (*catalogPersistenceCoordinator, error) {
	if fsManager == nil {
		return nil, fmt.Errorf("file system manager is required for persistence synchronization")
	}
	if sourceRepo == nil {
		return nil, fmt.Errorf("catalog source repository is required for persistence synchronization")
	}
	if overlayRepo == nil {
		return nil, fmt.Errorf("catalog overlay repository is required for persistence synchronization")
	}
	if taxonomyAssignmentRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy assignment repository is required for persistence synchronization")
	}
	if tagAssignmentRepo == nil {
		return nil, fmt.Errorf("catalog tag assignment repository is required for persistence synchronization")
	}
	if domainRepo == nil {
		return nil, fmt.Errorf("catalog domain repository is required for persistence synchronization")
	}
	if subdomainRepo == nil {
		return nil, fmt.Errorf("catalog subdomain repository is required for persistence synchronization")
	}
	if tagRepo == nil {
		return nil, fmt.Errorf("catalog tag repository is required for persistence synchronization")
	}

	resolvedLogger := logger
	if resolvedLogger == nil {
		resolvedLogger = log.New(io.Discard, "", 0)
	}

	syncService, err := domain.NewCatalogSyncService(sourceRepo, domain.CatalogSyncServiceOptions{
		Logger: resolvedLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize catalog sync service: %w", err)
	}
	effectiveService, err := domain.NewCatalogEffectiveService(
		sourceRepo,
		overlayRepo,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
		domainRepo,
		subdomainRepo,
		tagRepo,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize catalog effective service: %w", err)
	}
	backfillService, err := domain.NewCatalogTaxonomyLegacyLabelBackfillService(
		sourceRepo,
		overlayRepo,
		tagRepo,
		tagAssignmentRepo,
		domain.CatalogTaxonomyLegacyLabelBackfillServiceOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("initialize catalog taxonomy legacy label backfill service: %w", err)
	}

	return &catalogPersistenceCoordinator{
		fsManager:        fsManager,
		syncService:      syncService,
		backfillService:  backfillService,
		effectiveService: effectiveService,
		logger:           resolvedLogger,
	}, nil
}

func (c *catalogPersistenceCoordinator) FullSyncAndRebuild(ctx context.Context) error {
	return c.syncAndRebuild(ctx, func(discovered []domain.CatalogItem) error {
		return c.syncService.SyncAll(discovered)
	})
}

func (c *catalogPersistenceCoordinator) RepoSyncAndRebuild(ctx context.Context, repoName string) error {
	return c.syncAndRebuild(ctx, func(discovered []domain.CatalogItem) error {
		return c.syncService.SyncRepo(repoName, discovered)
	})
}

func (c *catalogPersistenceCoordinator) syncAndRebuild(
	ctx context.Context,
	syncFn func(discovered []domain.CatalogItem) error,
) error {
	if c == nil {
		return fmt.Errorf("catalog persistence coordinator is required")
	}
	if syncFn == nil {
		return fmt.Errorf("catalog persistence sync function is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	discovered, err := c.fsManager.ListCatalogItems()
	if err != nil {
		return fmt.Errorf("discover catalog items for persistence synchronization: %w", err)
	}

	if err := syncFn(discovered); err != nil {
		return fmt.Errorf("persist synchronized catalog snapshot: %w", err)
	}
	backfillReport, err := c.backfillService.BackfillFromLegacyLabels(ctx)
	if err != nil {
		return fmt.Errorf("backfill legacy labels into taxonomy tags: %w", err)
	}
	if c.logger != nil && backfillReport.ItemsWithLegacyLabels > 0 {
		c.logger.Printf(
			"Catalog taxonomy legacy label backfill completed: items_scanned=%d items_with_legacy_labels=%d tags_created=%d item_assignments_updated=%d normalization_collisions=%d",
			backfillReport.ItemsScanned,
			backfillReport.ItemsWithLegacyLabels,
			backfillReport.TagsCreated,
			backfillReport.ItemAssignmentsUpdated,
			len(backfillReport.NormalizationCollisions),
		)
	}

	effectiveItems, err := c.effectiveService.List(ctx, domain.CatalogEffectiveListFilter{})
	if err != nil {
		return fmt.Errorf("load effective catalog snapshot: %w", err)
	}

	if err := c.fsManager.RebuildIndexFromCatalogItems(effectiveItems); err != nil {
		return fmt.Errorf("rebuild search index from effective catalog snapshot: %w", err)
	}

	return nil
}

func (r *catalogPersistenceRuntime) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return persistence.CloseSQLite(r.db)
}
