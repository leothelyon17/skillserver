package persistence

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewCatalogMetadataOverlayRepository_WithNilExecutor_ReturnsError(t *testing.T) {
	_, err := NewCatalogMetadataOverlayRepository(nil)
	if err == nil {
		t.Fatalf("expected nil executor error, got nil")
	}
}

func TestCatalogMetadataOverlayRepository_UpsertAndGetByItemID_RoundTripsJSON(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	overlayRepo := newCatalogMetadataOverlayRepositoryForTest(t, db)

	itemID := "skill:planner"
	mustUpsertCatalogSourceRow(t, ctx, sourceRepo, CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       CatalogClassifierSkill,
		SourceType:       CatalogSourceTypeLocal,
		Name:             "planner",
		Description:      "planner skill",
		Content:          "content",
		ContentHash:      "sha256:planner",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 4, 18, 0, 0, 0, time.UTC),
	})

	expectedUpdatedAt := time.Date(2026, time.March, 4, 18, 30, 0, 0, time.UTC)
	if err := overlayRepo.Upsert(ctx, CatalogMetadataOverlayRow{
		ItemID:              itemID,
		DisplayNameOverride: stringPointer("Planner - Platform"),
		DescriptionOverride: stringPointer("Updated planner description"),
		CustomMetadata: map[string]any{
			"owner": "platform",
			"priority": map[string]any{
				"label": "high",
				"rank":  1,
			},
		},
		Labels:    []string{"internal", "stable"},
		UpdatedAt: expectedUpdatedAt,
		UpdatedBy: stringPointer("wp-003"),
	}); err != nil {
		t.Fatalf("expected overlay upsert to succeed, got %v", err)
	}

	actual, err := overlayRepo.GetByItemID(ctx, itemID)
	if err != nil {
		t.Fatalf("expected overlay lookup to succeed, got %v", err)
	}

	if actual.DisplayNameOverride == nil || *actual.DisplayNameOverride != "Planner - Platform" {
		t.Fatalf("expected display_name_override to round-trip, got %+v", actual.DisplayNameOverride)
	}
	if actual.DescriptionOverride == nil || *actual.DescriptionOverride != "Updated planner description" {
		t.Fatalf("expected description_override to round-trip, got %+v", actual.DescriptionOverride)
	}
	if gotOwner, ok := actual.CustomMetadata["owner"].(string); !ok || gotOwner != "platform" {
		t.Fatalf("expected custom metadata owner to round-trip, got %+v", actual.CustomMetadata)
	}
	priority, ok := actual.CustomMetadata["priority"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested custom metadata to round-trip, got %+v", actual.CustomMetadata)
	}
	if rank, ok := priority["rank"].(float64); !ok || rank != 1 {
		t.Fatalf("expected nested rank metadata to round-trip, got %+v", priority)
	}
	if len(actual.Labels) != 2 || actual.Labels[0] != "internal" || actual.Labels[1] != "stable" {
		t.Fatalf("expected labels to round-trip, got %+v", actual.Labels)
	}
	if !actual.UpdatedAt.Equal(expectedUpdatedAt) {
		t.Fatalf("expected updated_at %s, got %s", expectedUpdatedAt, actual.UpdatedAt)
	}
	if actual.UpdatedBy == nil || *actual.UpdatedBy != "wp-003" {
		t.Fatalf("expected updated_by to round-trip, got %+v", actual.UpdatedBy)
	}
}

func TestCatalogMetadataOverlayRepository_Upsert_WithNilMetadataAndLabels_PersistsEmptyContainers(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	overlayRepo := newCatalogMetadataOverlayRepositoryForTest(t, db)

	itemID := "skill:nil-metadata"
	mustUpsertCatalogSourceRow(t, ctx, sourceRepo, CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       CatalogClassifierSkill,
		SourceType:       CatalogSourceTypeLocal,
		Name:             "nil-metadata",
		Description:      "nil metadata skill",
		Content:          "content",
		ContentHash:      "sha256:nil-metadata",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 4, 19, 0, 0, 0, time.UTC),
	})

	if err := overlayRepo.Upsert(ctx, CatalogMetadataOverlayRow{ItemID: itemID}); err != nil {
		t.Fatalf("expected overlay upsert with nil metadata to succeed, got %v", err)
	}

	overlayRow, err := overlayRepo.GetByItemID(ctx, itemID)
	if err != nil {
		t.Fatalf("expected overlay lookup to succeed, got %v", err)
	}
	if overlayRow.DisplayNameOverride != nil {
		t.Fatalf("expected display_name_override nil, got %+v", overlayRow.DisplayNameOverride)
	}
	if overlayRow.DescriptionOverride != nil {
		t.Fatalf("expected description_override nil, got %+v", overlayRow.DescriptionOverride)
	}
	if overlayRow.CustomMetadata == nil || len(overlayRow.CustomMetadata) != 0 {
		t.Fatalf("expected custom metadata to be an empty map, got %+v", overlayRow.CustomMetadata)
	}
	if overlayRow.Labels == nil || len(overlayRow.Labels) != 0 {
		t.Fatalf("expected labels to be an empty array, got %+v", overlayRow.Labels)
	}
}

func TestCatalogMetadataOverlayRepository_ListDeleteAndMissingLookup(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	overlayRepo := newCatalogMetadataOverlayRepositoryForTest(t, db)

	items := []string{"skill:alpha", "skill:beta", "skill:gamma"}
	for _, itemID := range items {
		mustUpsertCatalogSourceRow(t, ctx, sourceRepo, CatalogSourceRow{
			ItemID:           itemID,
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeLocal,
			Name:             strings.TrimPrefix(itemID, "skill:"),
			Description:      "test skill",
			Content:          "content",
			ContentHash:      "sha256:" + itemID,
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     time.Date(2026, time.March, 4, 20, 0, 0, 0, time.UTC),
		})

		if err := overlayRepo.Upsert(ctx, CatalogMetadataOverlayRow{
			ItemID:              itemID,
			DisplayNameOverride: stringPointer("overlay " + itemID),
			CustomMetadata: map[string]any{
				"id": itemID,
			},
			Labels: []string{"label-" + itemID},
		}); err != nil {
			t.Fatalf("expected overlay upsert for %q to succeed, got %v", itemID, err)
		}
	}

	allOverlays, err := overlayRepo.List(ctx, CatalogMetadataOverlayListFilter{})
	if err != nil {
		t.Fatalf("expected overlay list query to succeed, got %v", err)
	}
	if len(allOverlays) != 3 {
		t.Fatalf("expected 3 overlays, got %d", len(allOverlays))
	}
	if allOverlays[0].ItemID != "skill:alpha" || allOverlays[1].ItemID != "skill:beta" || allOverlays[2].ItemID != "skill:gamma" {
		t.Fatalf("expected deterministic ordering by item_id, got %+v", allOverlays)
	}

	subset, err := overlayRepo.List(ctx, CatalogMetadataOverlayListFilter{ItemIDs: []string{"skill:gamma", "skill:alpha"}})
	if err != nil {
		t.Fatalf("expected overlay subset query to succeed, got %v", err)
	}
	if len(subset) != 2 || subset[0].ItemID != "skill:alpha" || subset[1].ItemID != "skill:gamma" {
		t.Fatalf("expected deterministic subset ordering, got %+v", subset)
	}

	deleted, err := overlayRepo.DeleteByItemID(ctx, "skill:beta")
	if err != nil {
		t.Fatalf("expected overlay delete to succeed, got %v", err)
	}
	if !deleted {
		t.Fatalf("expected overlay delete to affect one row")
	}

	_, err = overlayRepo.GetByItemID(ctx, "skill:beta")
	if !errors.Is(err, ErrCatalogMetadataOverlayNotFound) {
		t.Fatalf("expected missing overlay not found error, got %v", err)
	}
}

func TestCatalogMetadataOverlayRepository_GetByItemID_RejectsMalformedJSON(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	overlayRepo := newCatalogMetadataOverlayRepositoryForTest(t, db)

	mustUpsertCatalogSourceRow(t, ctx, sourceRepo, CatalogSourceRow{
		ItemID:           "skill:malformed-custom-metadata",
		Classifier:       CatalogClassifierSkill,
		SourceType:       CatalogSourceTypeLocal,
		Name:             "malformed-custom-metadata",
		Description:      "test",
		Content:          "content",
		ContentHash:      "sha256:a",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 4, 21, 0, 0, 0, time.UTC),
	})

	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO catalog_metadata_overlays (
			item_id,
			custom_metadata_json,
			labels_json,
			updated_at
		) VALUES (?, ?, ?, ?);`,
		"skill:malformed-custom-metadata",
		`{"owner":`,
		`[]`,
		formatCatalogTimestamp(time.Date(2026, time.March, 4, 21, 1, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("expected malformed custom metadata fixture insert to succeed, got %v", err)
	}

	_, err := overlayRepo.GetByItemID(ctx, "skill:malformed-custom-metadata")
	if err == nil {
		t.Fatalf("expected malformed custom metadata JSON to be rejected, got nil")
	}
	if !strings.Contains(err.Error(), "custom_metadata_json") {
		t.Fatalf("expected malformed custom metadata error to mention custom_metadata_json, got %v", err)
	}

	mustUpsertCatalogSourceRow(t, ctx, sourceRepo, CatalogSourceRow{
		ItemID:           "skill:malformed-labels",
		Classifier:       CatalogClassifierSkill,
		SourceType:       CatalogSourceTypeLocal,
		Name:             "malformed-labels",
		Description:      "test",
		Content:          "content",
		ContentHash:      "sha256:b",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 4, 21, 5, 0, 0, time.UTC),
	})

	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO catalog_metadata_overlays (
			item_id,
			custom_metadata_json,
			labels_json,
			updated_at
		) VALUES (?, ?, ?, ?);`,
		"skill:malformed-labels",
		`{}`,
		`{"not":"an-array"}`,
		formatCatalogTimestamp(time.Date(2026, time.March, 4, 21, 6, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("expected malformed labels fixture insert to succeed, got %v", err)
	}

	_, err = overlayRepo.GetByItemID(ctx, "skill:malformed-labels")
	if err == nil {
		t.Fatalf("expected malformed labels JSON to be rejected, got nil")
	}
	if !strings.Contains(err.Error(), "labels_json") {
		t.Fatalf("expected malformed labels error to mention labels_json, got %v", err)
	}
}

func TestCatalogMetadataOverlayRepository_Upsert_DoesNotMutateSourceRow(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	overlayRepo := newCatalogMetadataOverlayRepositoryForTest(t, db)

	itemID := "skill:source-isolation"
	originalSource := CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       CatalogClassifierSkill,
		SourceType:       CatalogSourceTypeLocal,
		Name:             "source-isolation",
		Description:      "Original source description",
		Content:          "Original source content",
		ContentHash:      "sha256:source-isolation",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 4, 22, 0, 0, 0, time.UTC),
	}
	mustUpsertCatalogSourceRow(t, ctx, sourceRepo, originalSource)

	if err := overlayRepo.Upsert(ctx, CatalogMetadataOverlayRow{
		ItemID:              itemID,
		DisplayNameOverride: stringPointer("Overlay Name"),
		DescriptionOverride: stringPointer("Overlay Description"),
		CustomMetadata: map[string]any{
			"owner": "overlay-only",
		},
		Labels: []string{"overlay"},
	}); err != nil {
		t.Fatalf("expected overlay upsert to succeed, got %v", err)
	}

	afterOverlaySource, err := sourceRepo.GetByItemID(ctx, itemID)
	if err != nil {
		t.Fatalf("expected source row lookup after overlay mutation to succeed, got %v", err)
	}

	assertCatalogSourceRowEqual(t, originalSource, afterOverlaySource)
}

func TestCatalogMetadataOverlayRepository_List_InvalidFilter_ReturnsError(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogMetadataOverlayRepositoryForTest(t, db)

	_, err := repo.List(ctx, CatalogMetadataOverlayListFilter{
		ItemID:  "skill:a",
		ItemIDs: []string{"skill:b"},
	})
	if err == nil {
		t.Fatalf("expected invalid filter error, got nil")
	}
}
