package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrCatalogDomainNotFound indicates that a domain row does not exist.
	ErrCatalogDomainNotFound = errors.New("catalog domain row not found")
	// ErrCatalogSubdomainNotFound indicates that a subdomain row does not exist.
	ErrCatalogSubdomainNotFound = errors.New("catalog subdomain row not found")
	// ErrCatalogTagNotFound indicates that a tag row does not exist.
	ErrCatalogTagNotFound = errors.New("catalog tag row not found")
	// ErrCatalogItemTaxonomyAssignmentNotFound indicates that an item taxonomy assignment row does not exist.
	ErrCatalogItemTaxonomyAssignmentNotFound = errors.New("catalog item taxonomy assignment row not found")
)

// CatalogDomainRow mirrors one row in catalog_domains.
type CatalogDomainRow struct {
	DomainID    string
	Key         string
	Name        string
	Description string
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CatalogDomainListFilter constrains domain list queries.
type CatalogDomainListFilter struct {
	DomainID  string
	DomainIDs []string
	Key       string
	Keys      []string
	Active    *bool
}

// CatalogSubdomainRow mirrors one row in catalog_subdomains.
type CatalogSubdomainRow struct {
	SubdomainID string
	DomainID    string
	Key         string
	Name        string
	Description string
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CatalogSubdomainListFilter constrains subdomain list queries.
type CatalogSubdomainListFilter struct {
	SubdomainID  string
	SubdomainIDs []string
	DomainID     string
	DomainIDs    []string
	Key          string
	Keys         []string
	Active       *bool
}

// CatalogTagRow mirrors one row in catalog_tags.
type CatalogTagRow struct {
	TagID       string
	Key         string
	Name        string
	Description string
	Color       string
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CatalogTagListFilter constrains tag list queries.
type CatalogTagListFilter struct {
	TagID  string
	TagIDs []string
	Key    string
	Keys   []string
	Active *bool
}

// CatalogItemTaxonomyAssignmentRow mirrors one row in catalog_item_taxonomy_assignments.
type CatalogItemTaxonomyAssignmentRow struct {
	ItemID               string
	PrimaryDomainID      *string
	PrimarySubdomainID   *string
	SecondaryDomainID    *string
	SecondarySubdomainID *string
	UpdatedAt            time.Time
	UpdatedBy            *string
}

// CatalogItemTaxonomyAssignmentListFilter constrains item taxonomy assignment list queries.
type CatalogItemTaxonomyAssignmentListFilter struct {
	ItemID               string
	ItemIDs              []string
	PrimaryDomainID      *string
	SecondaryDomainID    *string
	DomainID             *string
	PrimarySubdomainID   *string
	SecondarySubdomainID *string
	SubdomainID          *string
}

// CatalogItemTagAssignmentRow mirrors one row in catalog_item_tag_assignments.
type CatalogItemTagAssignmentRow struct {
	ItemID    string
	TagID     string
	CreatedAt time.Time
}

// CatalogItemTagAssignmentListFilter constrains item tag assignment list queries.
type CatalogItemTagAssignmentListFilter struct {
	ItemID  string
	ItemIDs []string
	TagID   string
	TagIDs  []string
}

func validateCatalogDomainCreateRow(row CatalogDomainRow) (CatalogDomainRow, error) {
	return validateCatalogDomainWriteRow(row, true)
}

func validateCatalogDomainUpdateRow(row CatalogDomainRow) (CatalogDomainRow, error) {
	return validateCatalogDomainWriteRow(row, false)
}

func validateCatalogDomainWriteRow(row CatalogDomainRow, includeCreatedAt bool) (CatalogDomainRow, error) {
	domainID, err := normalizeRequiredID(row.DomainID, "catalog domain domain_id")
	if err != nil {
		return CatalogDomainRow{}, err
	}
	row.DomainID = domainID

	key, err := normalizeRequiredID(row.Key, "catalog domain key")
	if err != nil {
		return CatalogDomainRow{}, err
	}
	row.Key = key

	name, err := normalizeRequiredID(row.Name, "catalog domain name")
	if err != nil {
		return CatalogDomainRow{}, err
	}
	row.Name = name
	row.Description = strings.TrimSpace(row.Description)

	now := time.Now().UTC()
	if includeCreatedAt {
		if row.CreatedAt.IsZero() {
			row.CreatedAt = now
		} else {
			row.CreatedAt = row.CreatedAt.UTC()
		}
	}

	if row.UpdatedAt.IsZero() {
		if includeCreatedAt {
			row.UpdatedAt = row.CreatedAt
		} else {
			row.UpdatedAt = now
		}
	} else {
		row.UpdatedAt = row.UpdatedAt.UTC()
	}

	return row, nil
}

func validateCatalogSubdomainCreateRow(row CatalogSubdomainRow) (CatalogSubdomainRow, error) {
	return validateCatalogSubdomainWriteRow(row, true)
}

func validateCatalogSubdomainUpdateRow(row CatalogSubdomainRow) (CatalogSubdomainRow, error) {
	return validateCatalogSubdomainWriteRow(row, false)
}

func validateCatalogSubdomainWriteRow(row CatalogSubdomainRow, includeCreatedAt bool) (CatalogSubdomainRow, error) {
	subdomainID, err := normalizeRequiredID(row.SubdomainID, "catalog subdomain subdomain_id")
	if err != nil {
		return CatalogSubdomainRow{}, err
	}
	row.SubdomainID = subdomainID

	domainID, err := normalizeRequiredID(row.DomainID, "catalog subdomain domain_id")
	if err != nil {
		return CatalogSubdomainRow{}, err
	}
	row.DomainID = domainID

	key, err := normalizeRequiredID(row.Key, "catalog subdomain key")
	if err != nil {
		return CatalogSubdomainRow{}, err
	}
	row.Key = key

	name, err := normalizeRequiredID(row.Name, "catalog subdomain name")
	if err != nil {
		return CatalogSubdomainRow{}, err
	}
	row.Name = name
	row.Description = strings.TrimSpace(row.Description)

	now := time.Now().UTC()
	if includeCreatedAt {
		if row.CreatedAt.IsZero() {
			row.CreatedAt = now
		} else {
			row.CreatedAt = row.CreatedAt.UTC()
		}
	}

	if row.UpdatedAt.IsZero() {
		if includeCreatedAt {
			row.UpdatedAt = row.CreatedAt
		} else {
			row.UpdatedAt = now
		}
	} else {
		row.UpdatedAt = row.UpdatedAt.UTC()
	}

	return row, nil
}

func validateCatalogTagCreateRow(row CatalogTagRow) (CatalogTagRow, error) {
	return validateCatalogTagWriteRow(row, true)
}

func validateCatalogTagUpdateRow(row CatalogTagRow) (CatalogTagRow, error) {
	return validateCatalogTagWriteRow(row, false)
}

func validateCatalogTagWriteRow(row CatalogTagRow, includeCreatedAt bool) (CatalogTagRow, error) {
	tagID, err := normalizeRequiredID(row.TagID, "catalog tag tag_id")
	if err != nil {
		return CatalogTagRow{}, err
	}
	row.TagID = tagID

	key, err := normalizeRequiredID(row.Key, "catalog tag key")
	if err != nil {
		return CatalogTagRow{}, err
	}
	row.Key = key

	name, err := normalizeRequiredID(row.Name, "catalog tag name")
	if err != nil {
		return CatalogTagRow{}, err
	}
	row.Name = name
	row.Description = strings.TrimSpace(row.Description)
	row.Color = strings.TrimSpace(row.Color)

	now := time.Now().UTC()
	if includeCreatedAt {
		if row.CreatedAt.IsZero() {
			row.CreatedAt = now
		} else {
			row.CreatedAt = row.CreatedAt.UTC()
		}
	}

	if row.UpdatedAt.IsZero() {
		if includeCreatedAt {
			row.UpdatedAt = row.CreatedAt
		} else {
			row.UpdatedAt = now
		}
	} else {
		row.UpdatedAt = row.UpdatedAt.UTC()
	}

	return row, nil
}

func validateCatalogItemTaxonomyAssignmentUpsertRow(
	row CatalogItemTaxonomyAssignmentRow,
) (CatalogItemTaxonomyAssignmentRow, error) {
	itemID, err := normalizeRequiredID(row.ItemID, "catalog item taxonomy assignment item_id")
	if err != nil {
		return CatalogItemTaxonomyAssignmentRow{}, err
	}
	row.ItemID = itemID

	row.PrimaryDomainID = normalizeOptionalForeignID(row.PrimaryDomainID)
	row.PrimarySubdomainID = normalizeOptionalForeignID(row.PrimarySubdomainID)
	row.SecondaryDomainID = normalizeOptionalForeignID(row.SecondaryDomainID)
	row.SecondarySubdomainID = normalizeOptionalForeignID(row.SecondarySubdomainID)
	row.UpdatedBy = normalizeOptionalForeignID(row.UpdatedBy)

	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = time.Now().UTC()
	} else {
		row.UpdatedAt = row.UpdatedAt.UTC()
	}

	return row, nil
}

func validateCatalogItemTagAssignmentRow(row CatalogItemTagAssignmentRow) (CatalogItemTagAssignmentRow, error) {
	itemID, err := normalizeRequiredID(row.ItemID, "catalog item tag assignment item_id")
	if err != nil {
		return CatalogItemTagAssignmentRow{}, err
	}
	row.ItemID = itemID

	tagID, err := normalizeRequiredID(row.TagID, "catalog item tag assignment tag_id")
	if err != nil {
		return CatalogItemTagAssignmentRow{}, err
	}
	row.TagID = tagID

	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	} else {
		row.CreatedAt = row.CreatedAt.UTC()
	}

	return row, nil
}

func scanCatalogDomainRow(scanner rowScanner) (CatalogDomainRow, error) {
	var (
		domainID     string
		key          string
		name         string
		description  string
		activeRaw    int
		createdAtRaw string
		updatedAtRaw string
	)

	if err := scanner.Scan(
		&domainID,
		&key,
		&name,
		&description,
		&activeRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return CatalogDomainRow{}, err
	}

	active, err := parseSQLiteBoolean(activeRaw, "catalog domain active")
	if err != nil {
		return CatalogDomainRow{}, err
	}

	createdAt, err := parseCatalogTimestamp(createdAtRaw)
	if err != nil {
		return CatalogDomainRow{}, fmt.Errorf("parse catalog domain created_at: %w", err)
	}
	updatedAt, err := parseCatalogTimestamp(updatedAtRaw)
	if err != nil {
		return CatalogDomainRow{}, fmt.Errorf("parse catalog domain updated_at: %w", err)
	}

	return CatalogDomainRow{
		DomainID:    domainID,
		Key:         key,
		Name:        name,
		Description: description,
		Active:      active,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func scanCatalogSubdomainRow(scanner rowScanner) (CatalogSubdomainRow, error) {
	var (
		subdomainID  string
		domainID     string
		key          string
		name         string
		description  string
		activeRaw    int
		createdAtRaw string
		updatedAtRaw string
	)

	if err := scanner.Scan(
		&subdomainID,
		&domainID,
		&key,
		&name,
		&description,
		&activeRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return CatalogSubdomainRow{}, err
	}

	active, err := parseSQLiteBoolean(activeRaw, "catalog subdomain active")
	if err != nil {
		return CatalogSubdomainRow{}, err
	}

	createdAt, err := parseCatalogTimestamp(createdAtRaw)
	if err != nil {
		return CatalogSubdomainRow{}, fmt.Errorf("parse catalog subdomain created_at: %w", err)
	}
	updatedAt, err := parseCatalogTimestamp(updatedAtRaw)
	if err != nil {
		return CatalogSubdomainRow{}, fmt.Errorf("parse catalog subdomain updated_at: %w", err)
	}

	return CatalogSubdomainRow{
		SubdomainID: subdomainID,
		DomainID:    domainID,
		Key:         key,
		Name:        name,
		Description: description,
		Active:      active,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func scanCatalogTagRow(scanner rowScanner) (CatalogTagRow, error) {
	var (
		tagID        string
		key          string
		name         string
		description  string
		color        string
		activeRaw    int
		createdAtRaw string
		updatedAtRaw string
	)

	if err := scanner.Scan(
		&tagID,
		&key,
		&name,
		&description,
		&color,
		&activeRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return CatalogTagRow{}, err
	}

	active, err := parseSQLiteBoolean(activeRaw, "catalog tag active")
	if err != nil {
		return CatalogTagRow{}, err
	}

	createdAt, err := parseCatalogTimestamp(createdAtRaw)
	if err != nil {
		return CatalogTagRow{}, fmt.Errorf("parse catalog tag created_at: %w", err)
	}
	updatedAt, err := parseCatalogTimestamp(updatedAtRaw)
	if err != nil {
		return CatalogTagRow{}, fmt.Errorf("parse catalog tag updated_at: %w", err)
	}

	return CatalogTagRow{
		TagID:       tagID,
		Key:         key,
		Name:        name,
		Description: description,
		Color:       color,
		Active:      active,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func scanCatalogItemTaxonomyAssignmentRow(scanner rowScanner) (CatalogItemTaxonomyAssignmentRow, error) {
	var (
		itemID               string
		primaryDomainID      sql.NullString
		primarySubdomainID   sql.NullString
		secondaryDomainID    sql.NullString
		secondarySubdomainID sql.NullString
		updatedAtRaw         string
		updatedBy            sql.NullString
	)

	if err := scanner.Scan(
		&itemID,
		&primaryDomainID,
		&primarySubdomainID,
		&secondaryDomainID,
		&secondarySubdomainID,
		&updatedAtRaw,
		&updatedBy,
	); err != nil {
		return CatalogItemTaxonomyAssignmentRow{}, err
	}

	updatedAt, err := parseCatalogTimestamp(updatedAtRaw)
	if err != nil {
		return CatalogItemTaxonomyAssignmentRow{}, fmt.Errorf("parse catalog item taxonomy assignment updated_at: %w", err)
	}

	return CatalogItemTaxonomyAssignmentRow{
		ItemID:               itemID,
		PrimaryDomainID:      nullStringToPointer(primaryDomainID),
		PrimarySubdomainID:   nullStringToPointer(primarySubdomainID),
		SecondaryDomainID:    nullStringToPointer(secondaryDomainID),
		SecondarySubdomainID: nullStringToPointer(secondarySubdomainID),
		UpdatedAt:            updatedAt,
		UpdatedBy:            nullStringToPointer(updatedBy),
	}, nil
}

func scanCatalogItemTagAssignmentRow(scanner rowScanner) (CatalogItemTagAssignmentRow, error) {
	var (
		itemID       string
		tagID        string
		createdAtRaw string
	)

	if err := scanner.Scan(&itemID, &tagID, &createdAtRaw); err != nil {
		return CatalogItemTagAssignmentRow{}, err
	}

	createdAt, err := parseCatalogTimestamp(createdAtRaw)
	if err != nil {
		return CatalogItemTagAssignmentRow{}, fmt.Errorf("parse catalog item tag assignment created_at: %w", err)
	}

	return CatalogItemTagAssignmentRow{
		ItemID:    itemID,
		TagID:     tagID,
		CreatedAt: createdAt,
	}, nil
}

func parseSQLiteBoolean(raw int, fieldName string) (bool, error) {
	switch raw {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("%s must be 0 or 1, got %d", fieldName, raw)
	}
}

func normalizeOptionalForeignID(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
