package domain

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mudler/skillserver/pkg/persistence"
)

var (
	// ErrCatalogTaxonomyAssignmentItemNotFound indicates that a catalog item does not exist.
	ErrCatalogTaxonomyAssignmentItemNotFound = errors.New("catalog taxonomy assignment item not found")
)

type catalogTaxonomyAssignmentSourceRepository interface {
	GetByItemID(ctx context.Context, itemID string) (persistence.CatalogSourceRow, error)
}

type catalogTaxonomyAssignmentRepositoryWriter interface {
	GetByItemID(ctx context.Context, itemID string) (persistence.CatalogItemTaxonomyAssignmentRow, error)
	Upsert(ctx context.Context, row persistence.CatalogItemTaxonomyAssignmentRow) error
	DeleteByItemID(ctx context.Context, itemID string) (bool, error)
}

type catalogTagAssignmentRepositoryWriter interface {
	ListByItemID(ctx context.Context, itemID string) ([]persistence.CatalogItemTagAssignmentRow, error)
	ReplaceForItemID(ctx context.Context, itemID string, tagIDs []string, createdAt time.Time) error
}

type catalogTaxonomyAssignmentDomainRepository interface {
	GetByDomainID(ctx context.Context, domainID string) (persistence.CatalogDomainRow, error)
}

type catalogTaxonomyAssignmentSubdomainRepository interface {
	GetBySubdomainID(ctx context.Context, subdomainID string) (persistence.CatalogSubdomainRow, error)
}

type catalogTaxonomyAssignmentTagRepository interface {
	List(ctx context.Context, filter persistence.CatalogTagListFilter) ([]persistence.CatalogTagRow, error)
}

// CatalogTaxonomyAssignmentServiceOptions configures assignment service behavior.
type CatalogTaxonomyAssignmentServiceOptions struct {
	Now func() time.Time
}

// CatalogItemTaxonomyAssignment is the service-layer item taxonomy assignment view.
type CatalogItemTaxonomyAssignment struct {
	ItemID             string                     `json:"item_id"`
	PrimaryDomain      *CatalogTaxonomyReference  `json:"primary_domain,omitempty"`
	PrimarySubdomain   *CatalogTaxonomyReference  `json:"primary_subdomain,omitempty"`
	SecondaryDomain    *CatalogTaxonomyReference  `json:"secondary_domain,omitempty"`
	SecondarySubdomain *CatalogTaxonomyReference  `json:"secondary_subdomain,omitempty"`
	Tags               []CatalogTaxonomyReference `json:"tags"`
	UpdatedAt          *time.Time                 `json:"updated_at,omitempty"`
	UpdatedBy          *string                    `json:"updated_by,omitempty"`
}

// CatalogItemTaxonomyAssignmentPatchInput describes a partial assignment mutation request.
type CatalogItemTaxonomyAssignmentPatchInput struct {
	ItemID               string
	PrimaryDomainID      *string
	PrimarySubdomainID   *string
	SecondaryDomainID    *string
	SecondarySubdomainID *string
	TagIDs               *[]string
	UpdatedBy            *string
	UpdatedAt            *time.Time
}

// CatalogTaxonomyAssignmentService handles taxonomy assignment patch/get flows for one catalog item.
type CatalogTaxonomyAssignmentService struct {
	sourceRepo        catalogTaxonomyAssignmentSourceRepository
	assignmentRepo    catalogTaxonomyAssignmentRepositoryWriter
	tagAssignmentRepo catalogTagAssignmentRepositoryWriter
	domainRepo        catalogTaxonomyAssignmentDomainRepository
	subdomainRepo     catalogTaxonomyAssignmentSubdomainRepository
	tagRepo           catalogTaxonomyAssignmentTagRepository
	now               func() time.Time
}

// NewCatalogTaxonomyAssignmentService constructs the taxonomy assignment service.
func NewCatalogTaxonomyAssignmentService(
	sourceRepo catalogTaxonomyAssignmentSourceRepository,
	assignmentRepo catalogTaxonomyAssignmentRepositoryWriter,
	tagAssignmentRepo catalogTagAssignmentRepositoryWriter,
	domainRepo catalogTaxonomyAssignmentDomainRepository,
	subdomainRepo catalogTaxonomyAssignmentSubdomainRepository,
	tagRepo catalogTaxonomyAssignmentTagRepository,
	options CatalogTaxonomyAssignmentServiceOptions,
) (*CatalogTaxonomyAssignmentService, error) {
	if sourceRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy assignment source repository is required")
	}
	if assignmentRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy assignment repository is required")
	}
	if tagAssignmentRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy tag assignment repository is required")
	}
	if domainRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy assignment domain repository is required")
	}
	if subdomainRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy assignment subdomain repository is required")
	}
	if tagRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy assignment tag repository is required")
	}

	now := options.Now
	if now == nil {
		now = time.Now
	}

	return &CatalogTaxonomyAssignmentService{
		sourceRepo:        sourceRepo,
		assignmentRepo:    assignmentRepo,
		tagAssignmentRepo: tagAssignmentRepo,
		domainRepo:        domainRepo,
		subdomainRepo:     subdomainRepo,
		tagRepo:           tagRepo,
		now:               now,
	}, nil
}

// Get returns one item's taxonomy assignment state.
func (s *CatalogTaxonomyAssignmentService) Get(ctx context.Context, itemID string) (CatalogItemTaxonomyAssignment, error) {
	if s == nil {
		return CatalogItemTaxonomyAssignment{}, fmt.Errorf("catalog taxonomy assignment service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	normalizedItemID := strings.TrimSpace(itemID)
	if normalizedItemID == "" {
		return CatalogItemTaxonomyAssignment{}, &CatalogTaxonomyValidationError{
			Field:  "item_id",
			Detail: "is required",
		}
	}

	if err := s.ensureAssignableItemExists(ctx, normalizedItemID); err != nil {
		return CatalogItemTaxonomyAssignment{}, err
	}

	assignmentRow, hasAssignment, err := s.getAssignmentRow(ctx, normalizedItemID)
	if err != nil {
		return CatalogItemTaxonomyAssignment{}, err
	}

	tagAssignmentRows, err := s.tagAssignmentRepo.ListByItemID(ctx, normalizedItemID)
	if err != nil {
		return CatalogItemTaxonomyAssignment{}, fmt.Errorf(
			"list catalog taxonomy tag assignments for %q: %w",
			normalizedItemID,
			err,
		)
	}

	return s.buildAssignmentView(ctx, normalizedItemID, assignmentRow, hasAssignment, tagAssignmentRows)
}

// Patch applies a partial taxonomy assignment update for one item.
func (s *CatalogTaxonomyAssignmentService) Patch(
	ctx context.Context,
	input CatalogItemTaxonomyAssignmentPatchInput,
) (CatalogItemTaxonomyAssignment, error) {
	if s == nil {
		return CatalogItemTaxonomyAssignment{}, fmt.Errorf("catalog taxonomy assignment service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	normalizedItemID := strings.TrimSpace(input.ItemID)
	if normalizedItemID == "" {
		return CatalogItemTaxonomyAssignment{}, &CatalogTaxonomyValidationError{
			Field:  "item_id",
			Detail: "is required",
		}
	}
	if !catalogTaxonomyAssignmentPatchIncludesChanges(input) {
		return CatalogItemTaxonomyAssignment{}, &CatalogTaxonomyValidationError{
			Field:  "taxonomy_patch",
			Detail: "must include at least one taxonomy field",
		}
	}

	if err := s.ensureAssignableItemExists(ctx, normalizedItemID); err != nil {
		return CatalogItemTaxonomyAssignment{}, err
	}

	existingAssignment, hasExistingAssignment, err := s.getAssignmentRow(ctx, normalizedItemID)
	if err != nil {
		return CatalogItemTaxonomyAssignment{}, err
	}

	existingTagRows, err := s.tagAssignmentRepo.ListByItemID(ctx, normalizedItemID)
	if err != nil {
		return CatalogItemTaxonomyAssignment{}, fmt.Errorf(
			"list catalog taxonomy tag assignments for %q: %w",
			normalizedItemID,
			err,
		)
	}

	primaryDomainID := mergeCatalogTaxonomyID(
		catalogOptionalString(existingAssignment.PrimaryDomainID, hasExistingAssignment),
		input.PrimaryDomainID,
	)
	primarySubdomainID := mergeCatalogTaxonomyID(
		catalogOptionalString(existingAssignment.PrimarySubdomainID, hasExistingAssignment),
		input.PrimarySubdomainID,
	)
	secondaryDomainID := mergeCatalogTaxonomyID(
		catalogOptionalString(existingAssignment.SecondaryDomainID, hasExistingAssignment),
		input.SecondaryDomainID,
	)
	secondarySubdomainID := mergeCatalogTaxonomyID(
		catalogOptionalString(existingAssignment.SecondarySubdomainID, hasExistingAssignment),
		input.SecondarySubdomainID,
	)

	if primaryDomainID != nil {
		if err := s.ensureDomainExists(ctx, *primaryDomainID); err != nil {
			return CatalogItemTaxonomyAssignment{}, err
		}
	}
	if secondaryDomainID != nil {
		if err := s.ensureDomainExists(ctx, *secondaryDomainID); err != nil {
			return CatalogItemTaxonomyAssignment{}, err
		}
	}

	primaryDomainID, primarySubdomainID, err = s.validateAndNormalizeSubdomainAssignment(
		ctx,
		normalizedItemID,
		"primary",
		primaryDomainID,
		primarySubdomainID,
	)
	if err != nil {
		return CatalogItemTaxonomyAssignment{}, err
	}

	secondaryDomainID, secondarySubdomainID, err = s.validateAndNormalizeSubdomainAssignment(
		ctx,
		normalizedItemID,
		"secondary",
		secondaryDomainID,
		secondarySubdomainID,
	)
	if err != nil {
		return CatalogItemTaxonomyAssignment{}, err
	}

	tagIDs := make([]string, 0, len(existingTagRows))
	if input.TagIDs == nil {
		for _, existingTagRow := range existingTagRows {
			tagIDs = append(tagIDs, existingTagRow.TagID)
		}
	} else {
		tagIDs = normalizeCatalogOptionalIDList(*input.TagIDs)
	}
	if err := s.ensureTagsExist(ctx, tagIDs); err != nil {
		return CatalogItemTaxonomyAssignment{}, err
	}

	updatedAt := s.now().UTC()
	if input.UpdatedAt != nil && !input.UpdatedAt.IsZero() {
		updatedAt = input.UpdatedAt.UTC()
	}
	updatedBy := mergeCatalogTaxonomyID(catalogOptionalString(existingAssignment.UpdatedBy, hasExistingAssignment), input.UpdatedBy)

	shouldPersistAssignment := primaryDomainID != nil ||
		primarySubdomainID != nil ||
		secondaryDomainID != nil ||
		secondarySubdomainID != nil

	if shouldPersistAssignment {
		if err := s.assignmentRepo.Upsert(ctx, persistence.CatalogItemTaxonomyAssignmentRow{
			ItemID:               normalizedItemID,
			PrimaryDomainID:      primaryDomainID,
			PrimarySubdomainID:   primarySubdomainID,
			SecondaryDomainID:    secondaryDomainID,
			SecondarySubdomainID: secondarySubdomainID,
			UpdatedAt:            updatedAt,
			UpdatedBy:            updatedBy,
		}); err != nil {
			return CatalogItemTaxonomyAssignment{}, fmt.Errorf(
				"upsert catalog taxonomy assignment for %q: %w",
				normalizedItemID,
				err,
			)
		}
	} else {
		if _, err := s.assignmentRepo.DeleteByItemID(ctx, normalizedItemID); err != nil {
			return CatalogItemTaxonomyAssignment{}, fmt.Errorf(
				"delete catalog taxonomy assignment for %q: %w",
				normalizedItemID,
				err,
			)
		}
	}

	if input.TagIDs != nil {
		if err := s.tagAssignmentRepo.ReplaceForItemID(ctx, normalizedItemID, tagIDs, updatedAt); err != nil {
			return CatalogItemTaxonomyAssignment{}, fmt.Errorf(
				"replace catalog taxonomy tag assignments for %q: %w",
				normalizedItemID,
				err,
			)
		}
	}

	return s.Get(ctx, normalizedItemID)
}

func catalogTaxonomyAssignmentPatchIncludesChanges(input CatalogItemTaxonomyAssignmentPatchInput) bool {
	return input.PrimaryDomainID != nil ||
		input.PrimarySubdomainID != nil ||
		input.SecondaryDomainID != nil ||
		input.SecondarySubdomainID != nil ||
		input.TagIDs != nil
}

func catalogOptionalString(value *string, enabled bool) *string {
	if !enabled || value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func mergeCatalogTaxonomyID(existing *string, patch *string) *string {
	if patch == nil {
		return existing
	}

	normalized := strings.TrimSpace(*patch)
	if normalized == "" {
		return nil
	}

	return &normalized
}

func (s *CatalogTaxonomyAssignmentService) ensureAssignableItemExists(
	ctx context.Context,
	itemID string,
) error {
	row, err := s.sourceRepo.GetByItemID(ctx, itemID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogSourceNotFound) {
			return fmt.Errorf("%w: item_id=%q", ErrCatalogTaxonomyAssignmentItemNotFound, itemID)
		}
		return fmt.Errorf("get catalog source item %q: %w", itemID, err)
	}
	if row.DeletedAt != nil {
		return fmt.Errorf("%w: item_id=%q", ErrCatalogTaxonomyAssignmentItemNotFound, itemID)
	}
	return nil
}

func (s *CatalogTaxonomyAssignmentService) getAssignmentRow(
	ctx context.Context,
	itemID string,
) (persistence.CatalogItemTaxonomyAssignmentRow, bool, error) {
	row, err := s.assignmentRepo.GetByItemID(ctx, itemID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogItemTaxonomyAssignmentNotFound) {
			return persistence.CatalogItemTaxonomyAssignmentRow{}, false, nil
		}
		return persistence.CatalogItemTaxonomyAssignmentRow{}, false, fmt.Errorf(
			"get catalog taxonomy assignment for %q: %w",
			itemID,
			err,
		)
	}
	return row, true, nil
}

func (s *CatalogTaxonomyAssignmentService) ensureDomainExists(ctx context.Context, domainID string) error {
	if _, err := s.domainRepo.GetByDomainID(ctx, domainID); err != nil {
		if errors.Is(err, persistence.ErrCatalogDomainNotFound) {
			return fmt.Errorf("%w: domain_id=%q", ErrCatalogTaxonomyDomainNotFound, domainID)
		}
		return fmt.Errorf("get catalog taxonomy domain %q: %w", domainID, err)
	}
	return nil
}

func (s *CatalogTaxonomyAssignmentService) validateAndNormalizeSubdomainAssignment(
	ctx context.Context,
	itemID string,
	slot string,
	domainID *string,
	subdomainID *string,
) (*string, *string, error) {
	if subdomainID == nil {
		return domainID, nil, nil
	}

	subdomainRow, err := s.subdomainRepo.GetBySubdomainID(ctx, *subdomainID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogSubdomainNotFound) {
			return nil, nil, fmt.Errorf("%w: subdomain_id=%q", ErrCatalogTaxonomySubdomainNotFound, *subdomainID)
		}
		return nil, nil, fmt.Errorf("get catalog taxonomy subdomain %q: %w", *subdomainID, err)
	}

	if domainID == nil {
		autoDomainID := subdomainRow.DomainID
		domainID = &autoDomainID
	} else if *domainID != subdomainRow.DomainID {
		return nil, nil, &CatalogTaxonomyInvalidRelationshipError{
			ObjectType:       CatalogTaxonomyObjectSubdomain,
			ObjectID:         itemID,
			Relationship:     slot + "_domain_id",
			ReferencedType:   CatalogTaxonomyObjectDomain,
			ReferencedObject: *domainID,
		}
	}

	return domainID, subdomainID, nil
}

func (s *CatalogTaxonomyAssignmentService) ensureTagsExist(ctx context.Context, tagIDs []string) error {
	if len(tagIDs) == 0 {
		return nil
	}

	rows, err := s.tagRepo.List(ctx, persistence.CatalogTagListFilter{TagIDs: tagIDs})
	if err != nil {
		return fmt.Errorf("list catalog taxonomy tags by id: %w", err)
	}

	found := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		found[row.TagID] = struct{}{}
	}

	missing := make([]string, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		if _, exists := found[tagID]; !exists {
			missing = append(missing, tagID)
		}
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		return fmt.Errorf("%w: tag_ids=%s", ErrCatalogTaxonomyTagNotFound, strings.Join(missing, ","))
	}

	return nil
}

func (s *CatalogTaxonomyAssignmentService) buildAssignmentView(
	ctx context.Context,
	itemID string,
	assignmentRow persistence.CatalogItemTaxonomyAssignmentRow,
	hasAssignment bool,
	tagAssignmentRows []persistence.CatalogItemTagAssignmentRow,
) (CatalogItemTaxonomyAssignment, error) {
	view := CatalogItemTaxonomyAssignment{
		ItemID: itemID,
		Tags:   []CatalogTaxonomyReference{},
	}

	if hasAssignment {
		if assignmentRow.PrimaryDomainID != nil {
			domainRow, err := s.domainRepo.GetByDomainID(ctx, *assignmentRow.PrimaryDomainID)
			if err != nil {
				return CatalogItemTaxonomyAssignment{}, fmt.Errorf(
					"get primary catalog taxonomy domain %q: %w",
					*assignmentRow.PrimaryDomainID,
					err,
				)
			}
			view.PrimaryDomain = mapCatalogDomainReference(domainRow)
		}

		if assignmentRow.PrimarySubdomainID != nil {
			subdomainRow, err := s.subdomainRepo.GetBySubdomainID(ctx, *assignmentRow.PrimarySubdomainID)
			if err != nil {
				return CatalogItemTaxonomyAssignment{}, fmt.Errorf(
					"get primary catalog taxonomy subdomain %q: %w",
					*assignmentRow.PrimarySubdomainID,
					err,
				)
			}
			view.PrimarySubdomain = mapCatalogSubdomainReference(subdomainRow)
		}

		if assignmentRow.SecondaryDomainID != nil {
			domainRow, err := s.domainRepo.GetByDomainID(ctx, *assignmentRow.SecondaryDomainID)
			if err != nil {
				return CatalogItemTaxonomyAssignment{}, fmt.Errorf(
					"get secondary catalog taxonomy domain %q: %w",
					*assignmentRow.SecondaryDomainID,
					err,
				)
			}
			view.SecondaryDomain = mapCatalogDomainReference(domainRow)
		}

		if assignmentRow.SecondarySubdomainID != nil {
			subdomainRow, err := s.subdomainRepo.GetBySubdomainID(ctx, *assignmentRow.SecondarySubdomainID)
			if err != nil {
				return CatalogItemTaxonomyAssignment{}, fmt.Errorf(
					"get secondary catalog taxonomy subdomain %q: %w",
					*assignmentRow.SecondarySubdomainID,
					err,
				)
			}
			view.SecondarySubdomain = mapCatalogSubdomainReference(subdomainRow)
		}

		updatedAt := assignmentRow.UpdatedAt.UTC()
		view.UpdatedAt = &updatedAt
		view.UpdatedBy = assignmentRow.UpdatedBy
	}

	if len(tagAssignmentRows) == 0 {
		return view, nil
	}

	tagIDs := make([]string, 0, len(tagAssignmentRows))
	for _, tagAssignmentRow := range tagAssignmentRows {
		tagIDs = append(tagIDs, tagAssignmentRow.TagID)
	}

	tagRows, err := s.tagRepo.List(ctx, persistence.CatalogTagListFilter{TagIDs: tagIDs})
	if err != nil {
		return CatalogItemTaxonomyAssignment{}, fmt.Errorf("list catalog taxonomy tags for assignment view: %w", err)
	}

	tagByID := make(map[string]persistence.CatalogTagRow, len(tagRows))
	for _, tagRow := range tagRows {
		tagByID[tagRow.TagID] = tagRow
	}

	view.Tags = make([]CatalogTaxonomyReference, 0, len(tagAssignmentRows))
	for _, tagID := range tagIDs {
		tagRow, exists := tagByID[tagID]
		if !exists {
			continue
		}
		view.Tags = append(view.Tags, *mapCatalogTagReference(tagRow))
	}

	return view, nil
}
