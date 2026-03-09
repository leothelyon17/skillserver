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
	CatalogClassificationState
}

// CatalogItemTaxonomyAssignmentPatchInput describes a partial assignment mutation request.
type CatalogItemTaxonomyAssignmentPatchInput struct {
	ItemID               string
	PrimaryDomainID      *string
	PrimarySubdomainID   *string
	SecondaryDomainID    *string
	SecondarySubdomainID *string
	TagIDs               *[]string
	AddTagIDs            *[]string
	RemoveTagIDs         *[]string
	ClearTags            *bool
	UpdatedBy            *string
	UpdatedAt            *time.Time
}

// CatalogItemTaxonomyBatchPatchRequest describes one batch taxonomy mutation.
type CatalogItemTaxonomyBatchPatchRequest struct {
	Items  []CatalogItemTaxonomyAssignmentPatchInput `json:"items"`
	DryRun bool                                      `json:"dry_run,omitempty"`
}

// CatalogItemTaxonomyBatchPatchStatus summarizes one batch item outcome.
type CatalogItemTaxonomyBatchPatchStatus string

const (
	CatalogItemTaxonomyBatchPatchStatusPlanned   CatalogItemTaxonomyBatchPatchStatus = "planned"
	CatalogItemTaxonomyBatchPatchStatusUpdated   CatalogItemTaxonomyBatchPatchStatus = "updated"
	CatalogItemTaxonomyBatchPatchStatusUnchanged CatalogItemTaxonomyBatchPatchStatus = "unchanged"
	CatalogItemTaxonomyBatchPatchStatusInvalid   CatalogItemTaxonomyBatchPatchStatus = "invalid"
	CatalogItemTaxonomyBatchPatchStatusNotFound  CatalogItemTaxonomyBatchPatchStatus = "not_found"
)

// CatalogItemTaxonomyBatchPatchItemResult captures one batch item outcome.
type CatalogItemTaxonomyBatchPatchItemResult struct {
	RequestedItemID string                              `json:"requested_item_id,omitempty"`
	ItemID          string                              `json:"item_id,omitempty"`
	Status          CatalogItemTaxonomyBatchPatchStatus `json:"status"`
	Assignment      *CatalogItemTaxonomyAssignment      `json:"assignment,omitempty"`
	Error           string                              `json:"error,omitempty"`
}

// CatalogItemTaxonomyBatchPatchResult captures one dry-run/apply batch result set.
type CatalogItemTaxonomyBatchPatchResult struct {
	DryRun bool                                      `json:"dry_run"`
	Items  []CatalogItemTaxonomyBatchPatchItemResult `json:"items"`
}

type catalogTaxonomyAssignmentPatchPlan struct {
	requestedItemID      string
	itemID               string
	currentHasAssignment bool
	desiredHasAssignment bool
	assignmentChanged    bool
	tagChanged           bool
	assignmentRow        persistence.CatalogItemTaxonomyAssignmentRow
	tagIDs               []string
	result               CatalogItemTaxonomyAssignment
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
	normalizedItemID, err := normalizeCatalogTaxonomyAssignmentItemID(normalizedItemID)
	if err != nil {
		return CatalogItemTaxonomyAssignment{}, err
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

	plan, err := s.planPatch(ctx, input)
	if err != nil {
		return CatalogItemTaxonomyAssignment{}, err
	}
	if !plan.assignmentChanged && !plan.tagChanged {
		return plan.result, nil
	}

	if err := s.applyPatchPlan(ctx, plan); err != nil {
		return CatalogItemTaxonomyAssignment{}, err
	}

	return plan.result, nil
}

// PatchBatch applies one dry-run or write batch of taxonomy assignment mutations.
func (s *CatalogTaxonomyAssignmentService) PatchBatch(
	ctx context.Context,
	request CatalogItemTaxonomyBatchPatchRequest,
) (CatalogItemTaxonomyBatchPatchResult, error) {
	if s == nil {
		return CatalogItemTaxonomyBatchPatchResult{}, fmt.Errorf("catalog taxonomy assignment service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateCatalogTaxonomyBatchPatchRequest(request); err != nil {
		return CatalogItemTaxonomyBatchPatchResult{}, err
	}

	result := CatalogItemTaxonomyBatchPatchResult{
		DryRun: request.DryRun,
		Items:  make([]CatalogItemTaxonomyBatchPatchItemResult, 0, len(request.Items)),
	}
	plans := make([]catalogTaxonomyAssignmentPatchPlan, 0, len(request.Items))

	for _, item := range request.Items {
		plan, err := s.planPatch(ctx, item)
		if err != nil {
			if !isCatalogTaxonomyBatchItemError(err) {
				return CatalogItemTaxonomyBatchPatchResult{}, err
			}

			result.Items = append(result.Items, newCatalogTaxonomyBatchPatchItemErrorResult(item.ItemID, err))
			continue
		}

		status := CatalogItemTaxonomyBatchPatchStatusUnchanged
		if plan.assignmentChanged || plan.tagChanged {
			if request.DryRun {
				status = CatalogItemTaxonomyBatchPatchStatusPlanned
			} else {
				status = CatalogItemTaxonomyBatchPatchStatusUpdated
			}
		}

		assignment := plan.result
		result.Items = append(result.Items, CatalogItemTaxonomyBatchPatchItemResult{
			RequestedItemID: plan.requestedItemID,
			ItemID:          plan.itemID,
			Status:          status,
			Assignment:      &assignment,
		})
		if plan.assignmentChanged || plan.tagChanged {
			plans = append(plans, plan)
		}
	}

	if request.DryRun {
		return result, nil
	}

	for _, plan := range plans {
		if err := s.applyPatchPlan(ctx, plan); err != nil {
			return CatalogItemTaxonomyBatchPatchResult{}, err
		}
	}

	return result, nil
}

func catalogTaxonomyAssignmentPatchIncludesChanges(input CatalogItemTaxonomyAssignmentPatchInput) bool {
	return input.PrimaryDomainID != nil ||
		input.PrimarySubdomainID != nil ||
		input.SecondaryDomainID != nil ||
		input.SecondarySubdomainID != nil ||
		input.TagIDs != nil ||
		input.AddTagIDs != nil ||
		input.RemoveTagIDs != nil ||
		catalogTaxonomyAssignmentClearTagsRequested(input.ClearTags)
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

func catalogTaxonomyAssignmentClearTagsRequested(clearTags *bool) bool {
	return clearTags != nil && *clearTags
}

func validateCatalogTaxonomyBatchPatchRequest(request CatalogItemTaxonomyBatchPatchRequest) error {
	if len(request.Items) == 0 {
		return &CatalogTaxonomyValidationError{
			Field:  "items",
			Detail: "must include at least one patch item",
		}
	}

	seen := make(map[string]struct{}, len(request.Items))
	for _, item := range request.Items {
		normalizedItemID, err := normalizeCatalogTaxonomyAssignmentItemID(strings.TrimSpace(item.ItemID))
		if err != nil {
			continue
		}
		if _, exists := seen[normalizedItemID]; exists {
			return &CatalogTaxonomyValidationError{
				Field:  "items",
				Detail: fmt.Sprintf("contains duplicate item_id %q", normalizedItemID),
			}
		}
		seen[normalizedItemID] = struct{}{}
	}

	return nil
}

func normalizeCatalogTaxonomyAssignmentItemID(rawItemID string) (string, error) {
	normalizedItemID, err := NormalizeCatalogItemID(rawItemID)
	if err != nil {
		return "", &CatalogTaxonomyValidationError{
			Field:  "item_id",
			Detail: "is invalid",
		}
	}
	return normalizedItemID, nil
}

func isCatalogTaxonomyBatchItemError(err error) bool {
	return errors.Is(err, ErrCatalogTaxonomyValidation) ||
		errors.Is(err, ErrCatalogTaxonomyInvalidRelationship) ||
		errors.Is(err, ErrCatalogTaxonomyAssignmentItemNotFound) ||
		errors.Is(err, ErrCatalogTaxonomyDomainNotFound) ||
		errors.Is(err, ErrCatalogTaxonomySubdomainNotFound) ||
		errors.Is(err, ErrCatalogTaxonomyTagNotFound)
}

func newCatalogTaxonomyBatchPatchItemErrorResult(
	rawItemID string,
	err error,
) CatalogItemTaxonomyBatchPatchItemResult {
	result := CatalogItemTaxonomyBatchPatchItemResult{
		RequestedItemID: strings.TrimSpace(rawItemID),
		Status:          CatalogItemTaxonomyBatchPatchStatusInvalid,
		Error:           err.Error(),
	}
	if errors.Is(err, ErrCatalogTaxonomyAssignmentItemNotFound) ||
		errors.Is(err, ErrCatalogTaxonomyDomainNotFound) ||
		errors.Is(err, ErrCatalogTaxonomySubdomainNotFound) ||
		errors.Is(err, ErrCatalogTaxonomyTagNotFound) {
		result.Status = CatalogItemTaxonomyBatchPatchStatusNotFound
	}

	if normalizedItemID, normalizeErr := normalizeCatalogTaxonomyAssignmentItemID(result.RequestedItemID); normalizeErr == nil {
		result.ItemID = normalizedItemID
	}

	return result
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

func (s *CatalogTaxonomyAssignmentService) planPatch(
	ctx context.Context,
	input CatalogItemTaxonomyAssignmentPatchInput,
) (catalogTaxonomyAssignmentPatchPlan, error) {
	requestedItemID := strings.TrimSpace(input.ItemID)
	if requestedItemID == "" {
		return catalogTaxonomyAssignmentPatchPlan{}, &CatalogTaxonomyValidationError{
			Field:  "item_id",
			Detail: "is required",
		}
	}

	normalizedItemID, err := normalizeCatalogTaxonomyAssignmentItemID(requestedItemID)
	if err != nil {
		return catalogTaxonomyAssignmentPatchPlan{}, err
	}
	if !catalogTaxonomyAssignmentPatchIncludesChanges(input) {
		return catalogTaxonomyAssignmentPatchPlan{}, &CatalogTaxonomyValidationError{
			Field:  "taxonomy_patch",
			Detail: "must include at least one taxonomy field",
		}
	}
	if err := validateCatalogTaxonomyTagMutationInputs(input); err != nil {
		return catalogTaxonomyAssignmentPatchPlan{}, err
	}
	if err := s.ensureAssignableItemExists(ctx, normalizedItemID); err != nil {
		return catalogTaxonomyAssignmentPatchPlan{}, err
	}

	existingAssignment, hasExistingAssignment, err := s.getAssignmentRow(ctx, normalizedItemID)
	if err != nil {
		return catalogTaxonomyAssignmentPatchPlan{}, err
	}
	existingTagRows, err := s.tagAssignmentRepo.ListByItemID(ctx, normalizedItemID)
	if err != nil {
		return catalogTaxonomyAssignmentPatchPlan{}, fmt.Errorf(
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
			return catalogTaxonomyAssignmentPatchPlan{}, err
		}
	}
	if secondaryDomainID != nil {
		if err := s.ensureDomainExists(ctx, *secondaryDomainID); err != nil {
			return catalogTaxonomyAssignmentPatchPlan{}, err
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
		return catalogTaxonomyAssignmentPatchPlan{}, err
	}
	secondaryDomainID, secondarySubdomainID, err = s.validateAndNormalizeSubdomainAssignment(
		ctx,
		normalizedItemID,
		"secondary",
		secondaryDomainID,
		secondarySubdomainID,
	)
	if err != nil {
		return catalogTaxonomyAssignmentPatchPlan{}, err
	}

	desiredTagIDs, referencedTagIDs := resolveCatalogTaxonomyPatchTagIDs(existingTagRows, input)
	if err := s.ensureTagsExist(ctx, referencedTagIDs); err != nil {
		return catalogTaxonomyAssignmentPatchPlan{}, err
	}

	existingTagIDs := catalogTaxonomyAssignmentTagIDs(existingTagRows)
	desiredHasAssignment := primaryDomainID != nil ||
		primarySubdomainID != nil ||
		secondaryDomainID != nil ||
		secondarySubdomainID != nil
	assignmentChanged := hasExistingAssignment != desiredHasAssignment
	if !assignmentChanged && desiredHasAssignment {
		assignmentChanged = !catalogTaxonomyAssignmentOptionalIDsEqual(existingAssignment.PrimaryDomainID, primaryDomainID) ||
			!catalogTaxonomyAssignmentOptionalIDsEqual(existingAssignment.PrimarySubdomainID, primarySubdomainID) ||
			!catalogTaxonomyAssignmentOptionalIDsEqual(existingAssignment.SecondaryDomainID, secondaryDomainID) ||
			!catalogTaxonomyAssignmentOptionalIDsEqual(existingAssignment.SecondarySubdomainID, secondarySubdomainID)
	}
	tagChanged := !slices.Equal(existingTagIDs, desiredTagIDs)

	plannedAssignmentRow := existingAssignment
	if desiredHasAssignment && (assignmentChanged || tagChanged) {
		updatedAt := s.now().UTC()
		if input.UpdatedAt != nil && !input.UpdatedAt.IsZero() {
			updatedAt = input.UpdatedAt.UTC()
		}
		updatedBy := mergeCatalogTaxonomyID(
			catalogOptionalString(existingAssignment.UpdatedBy, hasExistingAssignment),
			input.UpdatedBy,
		)
		plannedAssignmentRow = persistence.CatalogItemTaxonomyAssignmentRow{
			ItemID:               normalizedItemID,
			PrimaryDomainID:      primaryDomainID,
			PrimarySubdomainID:   primarySubdomainID,
			SecondaryDomainID:    secondaryDomainID,
			SecondarySubdomainID: secondarySubdomainID,
			UpdatedAt:            updatedAt,
			UpdatedBy:            updatedBy,
		}
	}

	view, err := s.buildAssignmentView(
		ctx,
		normalizedItemID,
		plannedAssignmentRow,
		desiredHasAssignment,
		buildCatalogTaxonomyTagAssignmentRows(normalizedItemID, desiredTagIDs),
	)
	if err != nil {
		return catalogTaxonomyAssignmentPatchPlan{}, err
	}

	return catalogTaxonomyAssignmentPatchPlan{
		requestedItemID:      requestedItemID,
		itemID:               normalizedItemID,
		currentHasAssignment: hasExistingAssignment,
		desiredHasAssignment: desiredHasAssignment,
		assignmentChanged:    assignmentChanged,
		tagChanged:           tagChanged,
		assignmentRow:        plannedAssignmentRow,
		tagIDs:               desiredTagIDs,
		result:               view,
	}, nil
}

func (s *CatalogTaxonomyAssignmentService) applyPatchPlan(
	ctx context.Context,
	plan catalogTaxonomyAssignmentPatchPlan,
) error {
	if plan.assignmentChanged || plan.tagChanged {
		if plan.desiredHasAssignment {
			if err := s.assignmentRepo.Upsert(ctx, plan.assignmentRow); err != nil {
				return fmt.Errorf("upsert catalog taxonomy assignment for %q: %w", plan.itemID, err)
			}
		} else if plan.currentHasAssignment {
			if _, err := s.assignmentRepo.DeleteByItemID(ctx, plan.itemID); err != nil {
				return fmt.Errorf("delete catalog taxonomy assignment for %q: %w", plan.itemID, err)
			}
		}
	}

	if plan.tagChanged {
		createdAt := plan.assignmentRow.UpdatedAt
		if createdAt.IsZero() {
			createdAt = s.now().UTC()
		}
		if err := s.tagAssignmentRepo.ReplaceForItemID(ctx, plan.itemID, plan.tagIDs, createdAt); err != nil {
			return fmt.Errorf("replace catalog taxonomy tag assignments for %q: %w", plan.itemID, err)
		}
	}

	return nil
}

func validateCatalogTaxonomyTagMutationInputs(input CatalogItemTaxonomyAssignmentPatchInput) error {
	clearTags := catalogTaxonomyAssignmentClearTagsRequested(input.ClearTags)
	if input.TagIDs != nil && (input.AddTagIDs != nil || input.RemoveTagIDs != nil || clearTags) {
		return &CatalogTaxonomyValidationError{
			Field:  "tag_ids",
			Detail: "cannot be combined with add_tag_ids, remove_tag_ids, or clear_tags",
		}
	}
	if clearTags && (input.AddTagIDs != nil || input.RemoveTagIDs != nil) {
		return &CatalogTaxonomyValidationError{
			Field:  "clear_tags",
			Detail: "cannot be combined with add_tag_ids or remove_tag_ids",
		}
	}

	addTagIDs := normalizeCatalogOptionalIDList(derefCatalogTaxonomyStringSlice(input.AddTagIDs))
	removeTagIDs := normalizeCatalogOptionalIDList(derefCatalogTaxonomyStringSlice(input.RemoveTagIDs))
	for _, tagID := range addTagIDs {
		if slices.Contains(removeTagIDs, tagID) {
			return &CatalogTaxonomyValidationError{
				Field:  "add_tag_ids",
				Detail: "and remove_tag_ids must be disjoint",
			}
		}
	}

	return nil
}

func resolveCatalogTaxonomyPatchTagIDs(
	existingTagRows []persistence.CatalogItemTagAssignmentRow,
	input CatalogItemTaxonomyAssignmentPatchInput,
) ([]string, []string) {
	if input.TagIDs != nil {
		tagIDs := normalizeCatalogOptionalIDList(*input.TagIDs)
		return tagIDs, tagIDs
	}
	if catalogTaxonomyAssignmentClearTagsRequested(input.ClearTags) {
		return nil, nil
	}

	existingTagIDs := catalogTaxonomyAssignmentTagIDs(existingTagRows)
	addTagIDs := normalizeCatalogOptionalIDList(derefCatalogTaxonomyStringSlice(input.AddTagIDs))
	removeTagIDs := normalizeCatalogOptionalIDList(derefCatalogTaxonomyStringSlice(input.RemoveTagIDs))
	if len(addTagIDs) == 0 && len(removeTagIDs) == 0 {
		return existingTagIDs, nil
	}

	desiredSet := make(map[string]struct{}, len(existingTagIDs)+len(addTagIDs))
	for _, tagID := range existingTagIDs {
		desiredSet[tagID] = struct{}{}
	}
	for _, tagID := range addTagIDs {
		desiredSet[tagID] = struct{}{}
	}
	for _, tagID := range removeTagIDs {
		delete(desiredSet, tagID)
	}

	desiredTagIDs := make([]string, 0, len(desiredSet))
	for tagID := range desiredSet {
		desiredTagIDs = append(desiredTagIDs, tagID)
	}
	slices.Sort(desiredTagIDs)

	referencedTagIDs := append([]string{}, addTagIDs...)
	referencedTagIDs = append(referencedTagIDs, removeTagIDs...)
	referencedTagIDs = normalizeCatalogOptionalIDList(referencedTagIDs)

	return desiredTagIDs, referencedTagIDs
}

func derefCatalogTaxonomyStringSlice(values *[]string) []string {
	if values == nil {
		return nil
	}
	return *values
}

func catalogTaxonomyAssignmentTagIDs(rows []persistence.CatalogItemTagAssignmentRow) []string {
	tagIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		tagIDs = append(tagIDs, row.TagID)
	}
	return normalizeCatalogOptionalIDList(tagIDs)
}

func catalogTaxonomyAssignmentOptionalIDsEqual(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func buildCatalogTaxonomyTagAssignmentRows(
	itemID string,
	tagIDs []string,
) []persistence.CatalogItemTagAssignmentRow {
	rows := make([]persistence.CatalogItemTagAssignmentRow, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		rows = append(rows, persistence.CatalogItemTagAssignmentRow{
			ItemID: itemID,
			TagID:  tagID,
		})
	}
	return rows
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

	if len(tagAssignmentRows) > 0 {
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
	}

	view.CatalogClassificationState = DeriveCatalogClassificationState(
		view.PrimaryDomain,
		view.PrimarySubdomain,
		view.SecondaryDomain,
		view.SecondarySubdomain,
		view.Tags,
	)

	return view, nil
}
