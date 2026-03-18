package domain

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrCatalogItemNotFound indicates that a catalog item could not be resolved.
	ErrCatalogItemNotFound = errors.New("catalog item not found")
)

type catalogItemLookupManager interface {
	ListCatalogItems() ([]CatalogItem, error)
}

type catalogItemLookupMetadataReader interface {
	List(ctx context.Context, filter CatalogEffectiveListFilter) ([]CatalogItem, error)
}

// GetCatalogItemByID resolves one catalog item by exact ID, returning effective metadata when available.
func GetCatalogItemByID(
	ctx context.Context,
	itemID string,
	manager catalogItemLookupManager,
	metadata catalogItemLookupMetadataReader,
) (CatalogItem, error) {
	normalizedItemID, err := NormalizeCatalogItemID(itemID)
	if err != nil {
		return CatalogItem{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if metadata != nil {
		items, err := metadata.List(ctx, CatalogEffectiveListFilter{ItemID: normalizedItemID})
		if err != nil {
			return CatalogItem{}, fmt.Errorf("list effective catalog item %q: %w", normalizedItemID, err)
		}
		if len(items) > 0 {
			return items[0], nil
		}
	}

	if manager == nil {
		return CatalogItem{}, fmt.Errorf("catalog item reader is required")
	}

	items, err := manager.ListCatalogItems()
	if err != nil {
		return CatalogItem{}, fmt.Errorf("list catalog items: %w", err)
	}
	for _, item := range items {
		if item.ID == normalizedItemID {
			return item, nil
		}
	}

	return CatalogItem{}, fmt.Errorf("%w: item_id=%q", ErrCatalogItemNotFound, normalizedItemID)
}
