package domain

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/mudler/skillserver/pkg/persistence"
)

var (
	// ErrCatalogTaxonomyDomainNotFound indicates that a taxonomy domain does not exist.
	ErrCatalogTaxonomyDomainNotFound = errors.New("catalog taxonomy domain not found")
	// ErrCatalogTaxonomySubdomainNotFound indicates that a taxonomy subdomain does not exist.
	ErrCatalogTaxonomySubdomainNotFound = errors.New("catalog taxonomy subdomain not found")
	// ErrCatalogTaxonomyTagNotFound indicates that a taxonomy tag does not exist.
	ErrCatalogTaxonomyTagNotFound = errors.New("catalog taxonomy tag not found")
	// ErrCatalogTaxonomyConflict indicates a deterministic uniqueness or delete conflict.
	ErrCatalogTaxonomyConflict = errors.New("catalog taxonomy conflict")
	// ErrCatalogTaxonomyInvalidRelationship indicates invalid cross-object relationships.
	ErrCatalogTaxonomyInvalidRelationship = errors.New("catalog taxonomy invalid relationship")
	// ErrCatalogTaxonomyValidation indicates invalid taxonomy inputs.
	ErrCatalogTaxonomyValidation = errors.New("catalog taxonomy validation failed")
)

// CatalogTaxonomyObjectType identifies taxonomy registry object categories.
type CatalogTaxonomyObjectType string

const (
	CatalogTaxonomyObjectDomain    CatalogTaxonomyObjectType = "domain"
	CatalogTaxonomyObjectSubdomain CatalogTaxonomyObjectType = "subdomain"
	CatalogTaxonomyObjectTag       CatalogTaxonomyObjectType = "tag"
)

// CatalogTaxonomyConflictReason provides stable conflict reason codes.
type CatalogTaxonomyConflictReason string

const (
	CatalogTaxonomyConflictReasonDuplicateKey CatalogTaxonomyConflictReason = "duplicate_key"
	CatalogTaxonomyConflictReasonInUse        CatalogTaxonomyConflictReason = "in_use"
	CatalogTaxonomyConflictReasonHasChildren  CatalogTaxonomyConflictReason = "has_children"
	CatalogTaxonomyConflictReasonConstraint   CatalogTaxonomyConflictReason = "constraint"
)

// CatalogTaxonomyConflictError returns actionable conflict context for transport adapters.
type CatalogTaxonomyConflictError struct {
	ObjectType        CatalogTaxonomyObjectType
	ObjectID          string
	Reason            CatalogTaxonomyConflictReason
	Detail            string
	ReferencedItemIDs []string
	Cause             error
}

// Error implements error.
func (e *CatalogTaxonomyConflictError) Error() string {
	if e == nil {
		return ErrCatalogTaxonomyConflict.Error()
	}
	if strings.TrimSpace(e.Detail) != "" {
		return e.Detail
	}

	objectType := strings.TrimSpace(string(e.ObjectType))
	if objectType == "" {
		objectType = "taxonomy object"
	}

	objectID := strings.TrimSpace(e.ObjectID)
	if objectID == "" {
		return fmt.Sprintf("catalog taxonomy %s conflict", objectType)
	}

	return fmt.Sprintf("catalog taxonomy %s %q conflict", objectType, objectID)
}

// Unwrap supports errors.Is(err, ErrCatalogTaxonomyConflict) and preserves the underlying cause.
func (e *CatalogTaxonomyConflictError) Unwrap() error {
	if e == nil || e.Cause == nil {
		return ErrCatalogTaxonomyConflict
	}
	return errors.Join(ErrCatalogTaxonomyConflict, e.Cause)
}

// CatalogTaxonomyInvalidRelationshipError describes invalid parent/child references.
type CatalogTaxonomyInvalidRelationshipError struct {
	ObjectType       CatalogTaxonomyObjectType
	ObjectID         string
	Relationship     string
	ReferencedType   CatalogTaxonomyObjectType
	ReferencedObject string
	Cause            error
}

// Error implements error.
func (e *CatalogTaxonomyInvalidRelationshipError) Error() string {
	if e == nil {
		return ErrCatalogTaxonomyInvalidRelationship.Error()
	}

	objectType := strings.TrimSpace(string(e.ObjectType))
	if objectType == "" {
		objectType = "taxonomy object"
	}

	objectID := strings.TrimSpace(e.ObjectID)
	relation := strings.TrimSpace(e.Relationship)
	referencedType := strings.TrimSpace(string(e.ReferencedType))
	referencedObject := strings.TrimSpace(e.ReferencedObject)
	if relation == "" {
		relation = "relationship"
	}
	if referencedType == "" {
		referencedType = "object"
	}
	if referencedObject == "" {
		return fmt.Sprintf("catalog taxonomy %s %q has invalid %s", objectType, objectID, relation)
	}

	return fmt.Sprintf(
		"catalog taxonomy %s %q has invalid %s reference to %s %q",
		objectType,
		objectID,
		relation,
		referencedType,
		referencedObject,
	)
}

// Unwrap supports errors.Is(err, ErrCatalogTaxonomyInvalidRelationship) and preserves the underlying cause.
func (e *CatalogTaxonomyInvalidRelationshipError) Unwrap() error {
	if e == nil || e.Cause == nil {
		return ErrCatalogTaxonomyInvalidRelationship
	}
	return errors.Join(ErrCatalogTaxonomyInvalidRelationship, e.Cause)
}

// CatalogTaxonomyValidationError provides stable field-level validation failures.
type CatalogTaxonomyValidationError struct {
	Field  string
	Detail string
}

// Error implements error.
func (e *CatalogTaxonomyValidationError) Error() string {
	if e == nil {
		return ErrCatalogTaxonomyValidation.Error()
	}
	if strings.TrimSpace(e.Field) == "" {
		if strings.TrimSpace(e.Detail) != "" {
			return e.Detail
		}
		return ErrCatalogTaxonomyValidation.Error()
	}
	if strings.TrimSpace(e.Detail) == "" {
		return fmt.Sprintf("%s is invalid", e.Field)
	}
	return fmt.Sprintf("%s %s", e.Field, e.Detail)
}

// Unwrap supports errors.Is(err, ErrCatalogTaxonomyValidation).
func (e *CatalogTaxonomyValidationError) Unwrap() error {
	return ErrCatalogTaxonomyValidation
}

type catalogTaxonomyDomainRepository interface {
	Create(ctx context.Context, row persistence.CatalogDomainRow) error
	Update(ctx context.Context, row persistence.CatalogDomainRow) (bool, error)
	GetByDomainID(ctx context.Context, domainID string) (persistence.CatalogDomainRow, error)
	GetByKey(ctx context.Context, key string) (persistence.CatalogDomainRow, error)
	List(ctx context.Context, filter persistence.CatalogDomainListFilter) ([]persistence.CatalogDomainRow, error)
	DeleteByDomainID(ctx context.Context, domainID string) (bool, error)
}

type catalogTaxonomySubdomainRepository interface {
	Create(ctx context.Context, row persistence.CatalogSubdomainRow) error
	Update(ctx context.Context, row persistence.CatalogSubdomainRow) (bool, error)
	GetBySubdomainID(ctx context.Context, subdomainID string) (persistence.CatalogSubdomainRow, error)
	GetByDomainIDAndKey(ctx context.Context, domainID string, key string) (persistence.CatalogSubdomainRow, error)
	List(ctx context.Context, filter persistence.CatalogSubdomainListFilter) ([]persistence.CatalogSubdomainRow, error)
	DeleteBySubdomainID(ctx context.Context, subdomainID string) (bool, error)
}

type catalogTaxonomyTagRepository interface {
	Create(ctx context.Context, row persistence.CatalogTagRow) error
	Update(ctx context.Context, row persistence.CatalogTagRow) (bool, error)
	GetByTagID(ctx context.Context, tagID string) (persistence.CatalogTagRow, error)
	GetByKey(ctx context.Context, key string) (persistence.CatalogTagRow, error)
	List(ctx context.Context, filter persistence.CatalogTagListFilter) ([]persistence.CatalogTagRow, error)
	DeleteByTagID(ctx context.Context, tagID string) (bool, error)
}

type catalogTaxonomyAssignmentRepository interface {
	List(
		ctx context.Context,
		filter persistence.CatalogItemTaxonomyAssignmentListFilter,
	) ([]persistence.CatalogItemTaxonomyAssignmentRow, error)
}

type catalogTagAssignmentRepository interface {
	List(
		ctx context.Context,
		filter persistence.CatalogItemTagAssignmentListFilter,
	) ([]persistence.CatalogItemTagAssignmentRow, error)
}

// CatalogTaxonomyRegistryServiceOptions configures taxonomy registry service behavior.
type CatalogTaxonomyRegistryServiceOptions struct {
	Now func() time.Time
}

// CatalogTaxonomyDomain is the service-layer taxonomy domain representation.
type CatalogTaxonomyDomain struct {
	DomainID    string    `json:"domain_id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CatalogTaxonomySubdomain is the service-layer taxonomy subdomain representation.
type CatalogTaxonomySubdomain struct {
	SubdomainID string    `json:"subdomain_id"`
	DomainID    string    `json:"domain_id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CatalogTaxonomyTag is the service-layer taxonomy tag representation.
type CatalogTaxonomyTag struct {
	TagID       string    `json:"tag_id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Color       string    `json:"color,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CatalogTaxonomyDomainCreateInput describes domain create requests.
type CatalogTaxonomyDomainCreateInput struct {
	DomainID    string
	Key         string
	Name        string
	Description string
	Active      *bool
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

// CatalogTaxonomyDomainUpdateInput describes domain patch requests.
type CatalogTaxonomyDomainUpdateInput struct {
	DomainID    string
	Key         *string
	Name        *string
	Description *string
	Active      *bool
	UpdatedAt   *time.Time
}

// CatalogTaxonomyDomainListFilter constrains domain list queries.
type CatalogTaxonomyDomainListFilter struct {
	DomainID  string
	DomainIDs []string
	Key       string
	Keys      []string
	Active    *bool
}

// CatalogTaxonomySubdomainCreateInput describes subdomain create requests.
type CatalogTaxonomySubdomainCreateInput struct {
	SubdomainID string
	DomainID    string
	Key         string
	Name        string
	Description string
	Active      *bool
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

// CatalogTaxonomySubdomainUpdateInput describes subdomain patch requests.
type CatalogTaxonomySubdomainUpdateInput struct {
	SubdomainID string
	DomainID    *string
	Key         *string
	Name        *string
	Description *string
	Active      *bool
	UpdatedAt   *time.Time
}

// CatalogTaxonomySubdomainListFilter constrains subdomain list queries.
type CatalogTaxonomySubdomainListFilter struct {
	SubdomainID  string
	SubdomainIDs []string
	DomainID     string
	DomainIDs    []string
	Key          string
	Keys         []string
	Active       *bool
}

// CatalogTaxonomyTagCreateInput describes tag create requests.
type CatalogTaxonomyTagCreateInput struct {
	TagID       string
	Key         string
	Name        string
	Description string
	Color       string
	Active      *bool
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

// CatalogTaxonomyTagUpdateInput describes tag patch requests.
type CatalogTaxonomyTagUpdateInput struct {
	TagID       string
	Key         *string
	Name        *string
	Description *string
	Color       *string
	Active      *bool
	UpdatedAt   *time.Time
}

// CatalogTaxonomyTagListFilter constrains tag list queries.
type CatalogTaxonomyTagListFilter struct {
	TagID  string
	TagIDs []string
	Key    string
	Keys   []string
	Active *bool
}

// CatalogTaxonomyRegistryService orchestrates taxonomy object CRUD and validation rules.
type CatalogTaxonomyRegistryService struct {
	domainRepo             catalogTaxonomyDomainRepository
	subdomainRepo          catalogTaxonomySubdomainRepository
	tagRepo                catalogTaxonomyTagRepository
	taxonomyAssignmentRepo catalogTaxonomyAssignmentRepository
	tagAssignmentRepo      catalogTagAssignmentRepository
	now                    func() time.Time
}

// NewCatalogTaxonomyRegistryService constructs the taxonomy registry service.
func NewCatalogTaxonomyRegistryService(
	domainRepo catalogTaxonomyDomainRepository,
	subdomainRepo catalogTaxonomySubdomainRepository,
	tagRepo catalogTaxonomyTagRepository,
	taxonomyAssignmentRepo catalogTaxonomyAssignmentRepository,
	tagAssignmentRepo catalogTagAssignmentRepository,
	options CatalogTaxonomyRegistryServiceOptions,
) (*CatalogTaxonomyRegistryService, error) {
	if domainRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy domain repository is required")
	}
	if subdomainRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy subdomain repository is required")
	}
	if tagRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy tag repository is required")
	}
	if taxonomyAssignmentRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy assignment repository is required")
	}
	if tagAssignmentRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy tag assignment repository is required")
	}

	now := options.Now
	if now == nil {
		now = time.Now
	}

	return &CatalogTaxonomyRegistryService{
		domainRepo:             domainRepo,
		subdomainRepo:          subdomainRepo,
		tagRepo:                tagRepo,
		taxonomyAssignmentRepo: taxonomyAssignmentRepo,
		tagAssignmentRepo:      tagAssignmentRepo,
		now:                    now,
	}, nil
}

// ListDomains returns taxonomy domains matching filter constraints.
func (s *CatalogTaxonomyRegistryService) ListDomains(
	ctx context.Context,
	filter CatalogTaxonomyDomainListFilter,
) ([]CatalogTaxonomyDomain, error) {
	if s == nil {
		return nil, fmt.Errorf("catalog taxonomy registry service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	mappedFilter, err := mapCatalogTaxonomyDomainListFilter(filter)
	if err != nil {
		return nil, err
	}

	rows, err := s.domainRepo.List(ctx, mappedFilter)
	if err != nil {
		return nil, fmt.Errorf("list catalog taxonomy domains: %w", err)
	}

	result := make([]CatalogTaxonomyDomain, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapCatalogTaxonomyDomain(row))
	}
	return result, nil
}

// CreateDomain creates one taxonomy domain with key normalization and conflict checks.
func (s *CatalogTaxonomyRegistryService) CreateDomain(
	ctx context.Context,
	input CatalogTaxonomyDomainCreateInput,
) (CatalogTaxonomyDomain, error) {
	if s == nil {
		return CatalogTaxonomyDomain{}, fmt.Errorf("catalog taxonomy registry service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	row, err := normalizeCatalogTaxonomyDomainCreateInput(input, s.now())
	if err != nil {
		return CatalogTaxonomyDomain{}, err
	}

	if err := s.ensureDomainKeyAvailable(ctx, row.DomainID, row.Key); err != nil {
		return CatalogTaxonomyDomain{}, err
	}

	if err := s.domainRepo.Create(ctx, row); err != nil {
		if isCatalogTaxonomyUniqueConstraintError(err) {
			return CatalogTaxonomyDomain{}, newCatalogTaxonomyDuplicateKeyConflict(
				CatalogTaxonomyObjectDomain,
				row.DomainID,
				row.Key,
				err,
			)
		}
		return CatalogTaxonomyDomain{}, fmt.Errorf("create catalog taxonomy domain %q: %w", row.DomainID, err)
	}

	created, err := s.domainRepo.GetByDomainID(ctx, row.DomainID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogDomainNotFound) {
			return CatalogTaxonomyDomain{}, fmt.Errorf("%w: domain_id=%q", ErrCatalogTaxonomyDomainNotFound, row.DomainID)
		}
		return CatalogTaxonomyDomain{}, fmt.Errorf("get created catalog taxonomy domain %q: %w", row.DomainID, err)
	}

	return mapCatalogTaxonomyDomain(created), nil
}

// UpdateDomain updates one taxonomy domain.
func (s *CatalogTaxonomyRegistryService) UpdateDomain(
	ctx context.Context,
	input CatalogTaxonomyDomainUpdateInput,
) (CatalogTaxonomyDomain, error) {
	if s == nil {
		return CatalogTaxonomyDomain{}, fmt.Errorf("catalog taxonomy registry service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	domainID, err := normalizeCatalogTaxonomyRequiredID(input.DomainID, "domain_id")
	if err != nil {
		return CatalogTaxonomyDomain{}, err
	}

	existing, err := s.domainRepo.GetByDomainID(ctx, domainID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogDomainNotFound) {
			return CatalogTaxonomyDomain{}, fmt.Errorf("%w: domain_id=%q", ErrCatalogTaxonomyDomainNotFound, domainID)
		}
		return CatalogTaxonomyDomain{}, fmt.Errorf("get catalog taxonomy domain %q: %w", domainID, err)
	}

	updatedRow, err := applyCatalogTaxonomyDomainUpdate(existing, input, s.now())
	if err != nil {
		return CatalogTaxonomyDomain{}, err
	}

	if err := s.ensureDomainKeyAvailable(ctx, updatedRow.DomainID, updatedRow.Key); err != nil {
		return CatalogTaxonomyDomain{}, err
	}

	updated, err := s.domainRepo.Update(ctx, updatedRow)
	if err != nil {
		if isCatalogTaxonomyUniqueConstraintError(err) {
			return CatalogTaxonomyDomain{}, newCatalogTaxonomyDuplicateKeyConflict(
				CatalogTaxonomyObjectDomain,
				updatedRow.DomainID,
				updatedRow.Key,
				err,
			)
		}
		return CatalogTaxonomyDomain{}, fmt.Errorf("update catalog taxonomy domain %q: %w", updatedRow.DomainID, err)
	}
	if !updated {
		return CatalogTaxonomyDomain{}, fmt.Errorf("%w: domain_id=%q", ErrCatalogTaxonomyDomainNotFound, updatedRow.DomainID)
	}

	row, err := s.domainRepo.GetByDomainID(ctx, updatedRow.DomainID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogDomainNotFound) {
			return CatalogTaxonomyDomain{}, fmt.Errorf("%w: domain_id=%q", ErrCatalogTaxonomyDomainNotFound, updatedRow.DomainID)
		}
		return CatalogTaxonomyDomain{}, fmt.Errorf("get updated catalog taxonomy domain %q: %w", updatedRow.DomainID, err)
	}

	return mapCatalogTaxonomyDomain(row), nil
}

// DeleteDomain deletes one taxonomy domain after explicit conflict checks.
func (s *CatalogTaxonomyRegistryService) DeleteDomain(ctx context.Context, domainID string) error {
	if s == nil {
		return fmt.Errorf("catalog taxonomy registry service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	normalizedDomainID, err := normalizeCatalogTaxonomyRequiredID(domainID, "domain_id")
	if err != nil {
		return err
	}

	if _, err := s.domainRepo.GetByDomainID(ctx, normalizedDomainID); err != nil {
		if errors.Is(err, persistence.ErrCatalogDomainNotFound) {
			return fmt.Errorf("%w: domain_id=%q", ErrCatalogTaxonomyDomainNotFound, normalizedDomainID)
		}
		return fmt.Errorf("get catalog taxonomy domain %q: %w", normalizedDomainID, err)
	}

	if err := s.assertDomainDeleteAllowed(ctx, normalizedDomainID); err != nil {
		return err
	}

	deleted, err := s.domainRepo.DeleteByDomainID(ctx, normalizedDomainID)
	if err != nil {
		if isCatalogTaxonomyForeignKeyConstraintError(err) {
			return &CatalogTaxonomyConflictError{
				ObjectType: CatalogTaxonomyObjectDomain,
				ObjectID:   normalizedDomainID,
				Reason:     CatalogTaxonomyConflictReasonConstraint,
				Detail: fmt.Sprintf(
					"domain %q cannot be deleted because dependent taxonomy records still exist",
					normalizedDomainID,
				),
				Cause: err,
			}
		}
		return fmt.Errorf("delete catalog taxonomy domain %q: %w", normalizedDomainID, err)
	}
	if !deleted {
		return fmt.Errorf("%w: domain_id=%q", ErrCatalogTaxonomyDomainNotFound, normalizedDomainID)
	}
	return nil
}

// ListSubdomains returns taxonomy subdomains matching filter constraints.
func (s *CatalogTaxonomyRegistryService) ListSubdomains(
	ctx context.Context,
	filter CatalogTaxonomySubdomainListFilter,
) ([]CatalogTaxonomySubdomain, error) {
	if s == nil {
		return nil, fmt.Errorf("catalog taxonomy registry service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	mappedFilter, err := mapCatalogTaxonomySubdomainListFilter(filter)
	if err != nil {
		return nil, err
	}

	rows, err := s.subdomainRepo.List(ctx, mappedFilter)
	if err != nil {
		return nil, fmt.Errorf("list catalog taxonomy subdomains: %w", err)
	}

	result := make([]CatalogTaxonomySubdomain, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapCatalogTaxonomySubdomain(row))
	}
	return result, nil
}

// CreateSubdomain creates one taxonomy subdomain with parent-domain validation.
func (s *CatalogTaxonomyRegistryService) CreateSubdomain(
	ctx context.Context,
	input CatalogTaxonomySubdomainCreateInput,
) (CatalogTaxonomySubdomain, error) {
	if s == nil {
		return CatalogTaxonomySubdomain{}, fmt.Errorf("catalog taxonomy registry service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	row, err := normalizeCatalogTaxonomySubdomainCreateInput(input, s.now())
	if err != nil {
		return CatalogTaxonomySubdomain{}, err
	}

	if err := s.ensureDomainExistsForSubdomain(ctx, row.SubdomainID, row.DomainID); err != nil {
		return CatalogTaxonomySubdomain{}, err
	}
	if err := s.ensureSubdomainKeyAvailable(ctx, row.SubdomainID, row.DomainID, row.Key); err != nil {
		return CatalogTaxonomySubdomain{}, err
	}

	if err := s.subdomainRepo.Create(ctx, row); err != nil {
		if isCatalogTaxonomyUniqueConstraintError(err) {
			return CatalogTaxonomySubdomain{}, newCatalogTaxonomyDuplicateKeyConflict(
				CatalogTaxonomyObjectSubdomain,
				row.SubdomainID,
				row.Key,
				err,
			)
		}
		if isCatalogTaxonomyForeignKeyConstraintError(err) {
			return CatalogTaxonomySubdomain{}, &CatalogTaxonomyInvalidRelationshipError{
				ObjectType:       CatalogTaxonomyObjectSubdomain,
				ObjectID:         row.SubdomainID,
				Relationship:     "domain_id",
				ReferencedType:   CatalogTaxonomyObjectDomain,
				ReferencedObject: row.DomainID,
				Cause:            err,
			}
		}
		return CatalogTaxonomySubdomain{}, fmt.Errorf("create catalog taxonomy subdomain %q: %w", row.SubdomainID, err)
	}

	created, err := s.subdomainRepo.GetBySubdomainID(ctx, row.SubdomainID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogSubdomainNotFound) {
			return CatalogTaxonomySubdomain{}, fmt.Errorf("%w: subdomain_id=%q", ErrCatalogTaxonomySubdomainNotFound, row.SubdomainID)
		}
		return CatalogTaxonomySubdomain{}, fmt.Errorf("get created catalog taxonomy subdomain %q: %w", row.SubdomainID, err)
	}

	return mapCatalogTaxonomySubdomain(created), nil
}

// UpdateSubdomain updates one taxonomy subdomain.
func (s *CatalogTaxonomyRegistryService) UpdateSubdomain(
	ctx context.Context,
	input CatalogTaxonomySubdomainUpdateInput,
) (CatalogTaxonomySubdomain, error) {
	if s == nil {
		return CatalogTaxonomySubdomain{}, fmt.Errorf("catalog taxonomy registry service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	subdomainID, err := normalizeCatalogTaxonomyRequiredID(input.SubdomainID, "subdomain_id")
	if err != nil {
		return CatalogTaxonomySubdomain{}, err
	}

	existing, err := s.subdomainRepo.GetBySubdomainID(ctx, subdomainID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogSubdomainNotFound) {
			return CatalogTaxonomySubdomain{}, fmt.Errorf("%w: subdomain_id=%q", ErrCatalogTaxonomySubdomainNotFound, subdomainID)
		}
		return CatalogTaxonomySubdomain{}, fmt.Errorf("get catalog taxonomy subdomain %q: %w", subdomainID, err)
	}

	updatedRow, err := applyCatalogTaxonomySubdomainUpdate(existing, input, s.now())
	if err != nil {
		return CatalogTaxonomySubdomain{}, err
	}

	if err := s.ensureDomainExistsForSubdomain(ctx, updatedRow.SubdomainID, updatedRow.DomainID); err != nil {
		return CatalogTaxonomySubdomain{}, err
	}
	if err := s.ensureSubdomainKeyAvailable(ctx, updatedRow.SubdomainID, updatedRow.DomainID, updatedRow.Key); err != nil {
		return CatalogTaxonomySubdomain{}, err
	}

	updated, err := s.subdomainRepo.Update(ctx, updatedRow)
	if err != nil {
		if isCatalogTaxonomyUniqueConstraintError(err) {
			return CatalogTaxonomySubdomain{}, newCatalogTaxonomyDuplicateKeyConflict(
				CatalogTaxonomyObjectSubdomain,
				updatedRow.SubdomainID,
				updatedRow.Key,
				err,
			)
		}
		if isCatalogTaxonomyForeignKeyConstraintError(err) {
			return CatalogTaxonomySubdomain{}, &CatalogTaxonomyInvalidRelationshipError{
				ObjectType:       CatalogTaxonomyObjectSubdomain,
				ObjectID:         updatedRow.SubdomainID,
				Relationship:     "domain_id",
				ReferencedType:   CatalogTaxonomyObjectDomain,
				ReferencedObject: updatedRow.DomainID,
				Cause:            err,
			}
		}
		return CatalogTaxonomySubdomain{}, fmt.Errorf("update catalog taxonomy subdomain %q: %w", updatedRow.SubdomainID, err)
	}
	if !updated {
		return CatalogTaxonomySubdomain{}, fmt.Errorf("%w: subdomain_id=%q", ErrCatalogTaxonomySubdomainNotFound, updatedRow.SubdomainID)
	}

	row, err := s.subdomainRepo.GetBySubdomainID(ctx, updatedRow.SubdomainID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogSubdomainNotFound) {
			return CatalogTaxonomySubdomain{}, fmt.Errorf("%w: subdomain_id=%q", ErrCatalogTaxonomySubdomainNotFound, updatedRow.SubdomainID)
		}
		return CatalogTaxonomySubdomain{}, fmt.Errorf("get updated catalog taxonomy subdomain %q: %w", updatedRow.SubdomainID, err)
	}

	return mapCatalogTaxonomySubdomain(row), nil
}

// DeleteSubdomain deletes one taxonomy subdomain after explicit assignment conflict checks.
func (s *CatalogTaxonomyRegistryService) DeleteSubdomain(ctx context.Context, subdomainID string) error {
	if s == nil {
		return fmt.Errorf("catalog taxonomy registry service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	normalizedSubdomainID, err := normalizeCatalogTaxonomyRequiredID(subdomainID, "subdomain_id")
	if err != nil {
		return err
	}

	if _, err := s.subdomainRepo.GetBySubdomainID(ctx, normalizedSubdomainID); err != nil {
		if errors.Is(err, persistence.ErrCatalogSubdomainNotFound) {
			return fmt.Errorf("%w: subdomain_id=%q", ErrCatalogTaxonomySubdomainNotFound, normalizedSubdomainID)
		}
		return fmt.Errorf("get catalog taxonomy subdomain %q: %w", normalizedSubdomainID, err)
	}

	assignments, err := s.taxonomyAssignmentRepo.List(ctx, persistence.CatalogItemTaxonomyAssignmentListFilter{
		SubdomainID: catalogTaxonomyStringPointer(normalizedSubdomainID),
	})
	if err != nil {
		return fmt.Errorf("list taxonomy assignments by subdomain %q: %w", normalizedSubdomainID, err)
	}
	if len(assignments) > 0 {
		return newCatalogTaxonomyInUseConflict(
			CatalogTaxonomyObjectSubdomain,
			normalizedSubdomainID,
			collectTaxonomyItemIDsFromSubdomainAssignments(assignments),
		)
	}

	deleted, err := s.subdomainRepo.DeleteBySubdomainID(ctx, normalizedSubdomainID)
	if err != nil {
		if isCatalogTaxonomyForeignKeyConstraintError(err) {
			return &CatalogTaxonomyConflictError{
				ObjectType: CatalogTaxonomyObjectSubdomain,
				ObjectID:   normalizedSubdomainID,
				Reason:     CatalogTaxonomyConflictReasonConstraint,
				Detail: fmt.Sprintf(
					"subdomain %q cannot be deleted because dependent taxonomy records still exist",
					normalizedSubdomainID,
				),
				Cause: err,
			}
		}
		return fmt.Errorf("delete catalog taxonomy subdomain %q: %w", normalizedSubdomainID, err)
	}
	if !deleted {
		return fmt.Errorf("%w: subdomain_id=%q", ErrCatalogTaxonomySubdomainNotFound, normalizedSubdomainID)
	}
	return nil
}

// ListTags returns taxonomy tags matching filter constraints.
func (s *CatalogTaxonomyRegistryService) ListTags(
	ctx context.Context,
	filter CatalogTaxonomyTagListFilter,
) ([]CatalogTaxonomyTag, error) {
	if s == nil {
		return nil, fmt.Errorf("catalog taxonomy registry service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	mappedFilter, err := mapCatalogTaxonomyTagListFilter(filter)
	if err != nil {
		return nil, err
	}

	rows, err := s.tagRepo.List(ctx, mappedFilter)
	if err != nil {
		return nil, fmt.Errorf("list catalog taxonomy tags: %w", err)
	}

	result := make([]CatalogTaxonomyTag, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapCatalogTaxonomyTag(row))
	}
	return result, nil
}

// CreateTag creates one taxonomy tag with key normalization and conflict checks.
func (s *CatalogTaxonomyRegistryService) CreateTag(
	ctx context.Context,
	input CatalogTaxonomyTagCreateInput,
) (CatalogTaxonomyTag, error) {
	if s == nil {
		return CatalogTaxonomyTag{}, fmt.Errorf("catalog taxonomy registry service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	row, err := normalizeCatalogTaxonomyTagCreateInput(input, s.now())
	if err != nil {
		return CatalogTaxonomyTag{}, err
	}

	if err := s.ensureTagKeyAvailable(ctx, row.TagID, row.Key); err != nil {
		return CatalogTaxonomyTag{}, err
	}

	if err := s.tagRepo.Create(ctx, row); err != nil {
		if isCatalogTaxonomyUniqueConstraintError(err) {
			return CatalogTaxonomyTag{}, newCatalogTaxonomyDuplicateKeyConflict(
				CatalogTaxonomyObjectTag,
				row.TagID,
				row.Key,
				err,
			)
		}
		return CatalogTaxonomyTag{}, fmt.Errorf("create catalog taxonomy tag %q: %w", row.TagID, err)
	}

	created, err := s.tagRepo.GetByTagID(ctx, row.TagID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogTagNotFound) {
			return CatalogTaxonomyTag{}, fmt.Errorf("%w: tag_id=%q", ErrCatalogTaxonomyTagNotFound, row.TagID)
		}
		return CatalogTaxonomyTag{}, fmt.Errorf("get created catalog taxonomy tag %q: %w", row.TagID, err)
	}

	return mapCatalogTaxonomyTag(created), nil
}

// UpdateTag updates one taxonomy tag.
func (s *CatalogTaxonomyRegistryService) UpdateTag(
	ctx context.Context,
	input CatalogTaxonomyTagUpdateInput,
) (CatalogTaxonomyTag, error) {
	if s == nil {
		return CatalogTaxonomyTag{}, fmt.Errorf("catalog taxonomy registry service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	tagID, err := normalizeCatalogTaxonomyRequiredID(input.TagID, "tag_id")
	if err != nil {
		return CatalogTaxonomyTag{}, err
	}

	existing, err := s.tagRepo.GetByTagID(ctx, tagID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogTagNotFound) {
			return CatalogTaxonomyTag{}, fmt.Errorf("%w: tag_id=%q", ErrCatalogTaxonomyTagNotFound, tagID)
		}
		return CatalogTaxonomyTag{}, fmt.Errorf("get catalog taxonomy tag %q: %w", tagID, err)
	}

	updatedRow, err := applyCatalogTaxonomyTagUpdate(existing, input, s.now())
	if err != nil {
		return CatalogTaxonomyTag{}, err
	}

	if err := s.ensureTagKeyAvailable(ctx, updatedRow.TagID, updatedRow.Key); err != nil {
		return CatalogTaxonomyTag{}, err
	}

	updated, err := s.tagRepo.Update(ctx, updatedRow)
	if err != nil {
		if isCatalogTaxonomyUniqueConstraintError(err) {
			return CatalogTaxonomyTag{}, newCatalogTaxonomyDuplicateKeyConflict(
				CatalogTaxonomyObjectTag,
				updatedRow.TagID,
				updatedRow.Key,
				err,
			)
		}
		return CatalogTaxonomyTag{}, fmt.Errorf("update catalog taxonomy tag %q: %w", updatedRow.TagID, err)
	}
	if !updated {
		return CatalogTaxonomyTag{}, fmt.Errorf("%w: tag_id=%q", ErrCatalogTaxonomyTagNotFound, updatedRow.TagID)
	}

	row, err := s.tagRepo.GetByTagID(ctx, updatedRow.TagID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogTagNotFound) {
			return CatalogTaxonomyTag{}, fmt.Errorf("%w: tag_id=%q", ErrCatalogTaxonomyTagNotFound, updatedRow.TagID)
		}
		return CatalogTaxonomyTag{}, fmt.Errorf("get updated catalog taxonomy tag %q: %w", updatedRow.TagID, err)
	}

	return mapCatalogTaxonomyTag(row), nil
}

// DeleteTag deletes one taxonomy tag after explicit assignment conflict checks.
func (s *CatalogTaxonomyRegistryService) DeleteTag(ctx context.Context, tagID string) error {
	if s == nil {
		return fmt.Errorf("catalog taxonomy registry service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	normalizedTagID, err := normalizeCatalogTaxonomyRequiredID(tagID, "tag_id")
	if err != nil {
		return err
	}

	if _, err := s.tagRepo.GetByTagID(ctx, normalizedTagID); err != nil {
		if errors.Is(err, persistence.ErrCatalogTagNotFound) {
			return fmt.Errorf("%w: tag_id=%q", ErrCatalogTaxonomyTagNotFound, normalizedTagID)
		}
		return fmt.Errorf("get catalog taxonomy tag %q: %w", normalizedTagID, err)
	}

	assignments, err := s.tagAssignmentRepo.List(ctx, persistence.CatalogItemTagAssignmentListFilter{
		TagID: normalizedTagID,
	})
	if err != nil {
		return fmt.Errorf("list tag assignments by tag %q: %w", normalizedTagID, err)
	}
	if len(assignments) > 0 {
		return newCatalogTaxonomyInUseConflict(
			CatalogTaxonomyObjectTag,
			normalizedTagID,
			collectTaxonomyItemIDsFromTagAssignments(assignments),
		)
	}

	deleted, err := s.tagRepo.DeleteByTagID(ctx, normalizedTagID)
	if err != nil {
		if isCatalogTaxonomyForeignKeyConstraintError(err) {
			return &CatalogTaxonomyConflictError{
				ObjectType: CatalogTaxonomyObjectTag,
				ObjectID:   normalizedTagID,
				Reason:     CatalogTaxonomyConflictReasonConstraint,
				Detail: fmt.Sprintf(
					"tag %q cannot be deleted because dependent taxonomy records still exist",
					normalizedTagID,
				),
				Cause: err,
			}
		}
		return fmt.Errorf("delete catalog taxonomy tag %q: %w", normalizedTagID, err)
	}
	if !deleted {
		return fmt.Errorf("%w: tag_id=%q", ErrCatalogTaxonomyTagNotFound, normalizedTagID)
	}
	return nil
}

func (s *CatalogTaxonomyRegistryService) assertDomainDeleteAllowed(
	ctx context.Context,
	domainID string,
) error {
	assignments, err := s.taxonomyAssignmentRepo.List(ctx, persistence.CatalogItemTaxonomyAssignmentListFilter{
		DomainID: catalogTaxonomyStringPointer(domainID),
	})
	if err != nil {
		return fmt.Errorf("list taxonomy assignments by domain %q: %w", domainID, err)
	}
	if len(assignments) > 0 {
		return newCatalogTaxonomyInUseConflict(
			CatalogTaxonomyObjectDomain,
			domainID,
			collectTaxonomyItemIDsFromSubdomainAssignments(assignments),
		)
	}

	subdomains, err := s.subdomainRepo.List(ctx, persistence.CatalogSubdomainListFilter{DomainID: domainID})
	if err != nil {
		return fmt.Errorf("list taxonomy subdomains by domain %q: %w", domainID, err)
	}
	if len(subdomains) > 0 {
		subdomainIDs := make([]string, 0, len(subdomains))
		for _, subdomain := range subdomains {
			subdomainIDs = append(subdomainIDs, subdomain.SubdomainID)
		}
		sort.Strings(subdomainIDs)

		return &CatalogTaxonomyConflictError{
			ObjectType: CatalogTaxonomyObjectDomain,
			ObjectID:   domainID,
			Reason:     CatalogTaxonomyConflictReasonHasChildren,
			Detail: fmt.Sprintf(
				"domain %q cannot be deleted because it still owns subdomains: %s",
				domainID,
				strings.Join(subdomainIDs, ", "),
			),
		}
	}

	return nil
}

func (s *CatalogTaxonomyRegistryService) ensureDomainExistsForSubdomain(
	ctx context.Context,
	subdomainID string,
	domainID string,
) error {
	if _, err := s.domainRepo.GetByDomainID(ctx, domainID); err != nil {
		if errors.Is(err, persistence.ErrCatalogDomainNotFound) {
			return &CatalogTaxonomyInvalidRelationshipError{
				ObjectType:       CatalogTaxonomyObjectSubdomain,
				ObjectID:         subdomainID,
				Relationship:     "domain_id",
				ReferencedType:   CatalogTaxonomyObjectDomain,
				ReferencedObject: domainID,
				Cause:            err,
			}
		}
		return fmt.Errorf("get taxonomy domain %q for subdomain %q: %w", domainID, subdomainID, err)
	}
	return nil
}

func (s *CatalogTaxonomyRegistryService) ensureDomainKeyAvailable(
	ctx context.Context,
	domainID string,
	key string,
) error {
	existing, err := s.domainRepo.GetByKey(ctx, key)
	if err == nil {
		if existing.DomainID != domainID {
			return newCatalogTaxonomyDuplicateKeyConflict(
				CatalogTaxonomyObjectDomain,
				domainID,
				key,
				nil,
			)
		}
		return nil
	}
	if errors.Is(err, persistence.ErrCatalogDomainNotFound) {
		return nil
	}
	return fmt.Errorf("check existing taxonomy domain key %q: %w", key, err)
}

func (s *CatalogTaxonomyRegistryService) ensureSubdomainKeyAvailable(
	ctx context.Context,
	subdomainID string,
	domainID string,
	key string,
) error {
	existing, err := s.subdomainRepo.GetByDomainIDAndKey(ctx, domainID, key)
	if err == nil {
		if existing.SubdomainID != subdomainID {
			return newCatalogTaxonomyDuplicateKeyConflict(
				CatalogTaxonomyObjectSubdomain,
				subdomainID,
				key,
				nil,
			)
		}
		return nil
	}
	if errors.Is(err, persistence.ErrCatalogSubdomainNotFound) {
		return nil
	}
	return fmt.Errorf(
		"check existing taxonomy subdomain key %q in domain %q: %w",
		key,
		domainID,
		err,
	)
}

func (s *CatalogTaxonomyRegistryService) ensureTagKeyAvailable(
	ctx context.Context,
	tagID string,
	key string,
) error {
	existing, err := s.tagRepo.GetByKey(ctx, key)
	if err == nil {
		if existing.TagID != tagID {
			return newCatalogTaxonomyDuplicateKeyConflict(
				CatalogTaxonomyObjectTag,
				tagID,
				key,
				nil,
			)
		}
		return nil
	}
	if errors.Is(err, persistence.ErrCatalogTagNotFound) {
		return nil
	}
	return fmt.Errorf("check existing taxonomy tag key %q: %w", key, err)
}

func mapCatalogTaxonomyDomain(row persistence.CatalogDomainRow) CatalogTaxonomyDomain {
	return CatalogTaxonomyDomain{
		DomainID:    row.DomainID,
		Key:         row.Key,
		Name:        row.Name,
		Description: row.Description,
		Active:      row.Active,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapCatalogTaxonomySubdomain(row persistence.CatalogSubdomainRow) CatalogTaxonomySubdomain {
	return CatalogTaxonomySubdomain{
		SubdomainID: row.SubdomainID,
		DomainID:    row.DomainID,
		Key:         row.Key,
		Name:        row.Name,
		Description: row.Description,
		Active:      row.Active,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapCatalogTaxonomyTag(row persistence.CatalogTagRow) CatalogTaxonomyTag {
	return CatalogTaxonomyTag{
		TagID:       row.TagID,
		Key:         row.Key,
		Name:        row.Name,
		Description: row.Description,
		Color:       row.Color,
		Active:      row.Active,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapCatalogTaxonomyDomainListFilter(
	filter CatalogTaxonomyDomainListFilter,
) (persistence.CatalogDomainListFilter, error) {
	mapped := persistence.CatalogDomainListFilter{
		DomainID:  strings.TrimSpace(filter.DomainID),
		DomainIDs: append([]string{}, filter.DomainIDs...),
		Active:    filter.Active,
	}

	if strings.TrimSpace(filter.Key) != "" {
		key, err := normalizeCatalogTaxonomyKey(filter.Key, "domain key filter")
		if err != nil {
			return persistence.CatalogDomainListFilter{}, err
		}
		mapped.Key = key
	}

	keys, err := normalizeCatalogTaxonomyKeyList(filter.Keys, "domain keys filter")
	if err != nil {
		return persistence.CatalogDomainListFilter{}, err
	}
	mapped.Keys = keys

	return mapped, nil
}

func mapCatalogTaxonomySubdomainListFilter(
	filter CatalogTaxonomySubdomainListFilter,
) (persistence.CatalogSubdomainListFilter, error) {
	mapped := persistence.CatalogSubdomainListFilter{
		SubdomainID:  strings.TrimSpace(filter.SubdomainID),
		SubdomainIDs: append([]string{}, filter.SubdomainIDs...),
		DomainID:     strings.TrimSpace(filter.DomainID),
		DomainIDs:    append([]string{}, filter.DomainIDs...),
		Active:       filter.Active,
	}

	if strings.TrimSpace(filter.Key) != "" {
		key, err := normalizeCatalogTaxonomyKey(filter.Key, "subdomain key filter")
		if err != nil {
			return persistence.CatalogSubdomainListFilter{}, err
		}
		mapped.Key = key
	}

	keys, err := normalizeCatalogTaxonomyKeyList(filter.Keys, "subdomain keys filter")
	if err != nil {
		return persistence.CatalogSubdomainListFilter{}, err
	}
	mapped.Keys = keys

	return mapped, nil
}

func mapCatalogTaxonomyTagListFilter(
	filter CatalogTaxonomyTagListFilter,
) (persistence.CatalogTagListFilter, error) {
	mapped := persistence.CatalogTagListFilter{
		TagID:  strings.TrimSpace(filter.TagID),
		TagIDs: append([]string{}, filter.TagIDs...),
		Active: filter.Active,
	}

	if strings.TrimSpace(filter.Key) != "" {
		key, err := normalizeCatalogTaxonomyKey(filter.Key, "tag key filter")
		if err != nil {
			return persistence.CatalogTagListFilter{}, err
		}
		mapped.Key = key
	}

	keys, err := normalizeCatalogTaxonomyKeyList(filter.Keys, "tag keys filter")
	if err != nil {
		return persistence.CatalogTagListFilter{}, err
	}
	mapped.Keys = keys

	return mapped, nil
}

func normalizeCatalogTaxonomyDomainCreateInput(
	input CatalogTaxonomyDomainCreateInput,
	now time.Time,
) (persistence.CatalogDomainRow, error) {
	domainID, err := normalizeCatalogTaxonomyRequiredID(input.DomainID, "domain_id")
	if err != nil {
		return persistence.CatalogDomainRow{}, err
	}
	key, err := normalizeCatalogTaxonomyKey(input.Key, "domain key")
	if err != nil {
		return persistence.CatalogDomainRow{}, err
	}
	name, err := normalizeCatalogTaxonomyRequiredName(input.Name, "domain name")
	if err != nil {
		return persistence.CatalogDomainRow{}, err
	}

	createdAt := now.UTC()
	if input.CreatedAt != nil && !input.CreatedAt.IsZero() {
		createdAt = input.CreatedAt.UTC()
	}

	updatedAt := createdAt
	if input.UpdatedAt != nil && !input.UpdatedAt.IsZero() {
		updatedAt = input.UpdatedAt.UTC()
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}

	return persistence.CatalogDomainRow{
		DomainID:    domainID,
		Key:         key,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Active:      active,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func applyCatalogTaxonomyDomainUpdate(
	existing persistence.CatalogDomainRow,
	input CatalogTaxonomyDomainUpdateInput,
	now time.Time,
) (persistence.CatalogDomainRow, error) {
	updated := existing

	if input.Key != nil {
		key, err := normalizeCatalogTaxonomyKey(*input.Key, "domain key")
		if err != nil {
			return persistence.CatalogDomainRow{}, err
		}
		updated.Key = key
	}
	if input.Name != nil {
		name, err := normalizeCatalogTaxonomyRequiredName(*input.Name, "domain name")
		if err != nil {
			return persistence.CatalogDomainRow{}, err
		}
		updated.Name = name
	}
	if input.Description != nil {
		updated.Description = strings.TrimSpace(*input.Description)
	}
	if input.Active != nil {
		updated.Active = *input.Active
	}

	updated.UpdatedAt = now.UTC()
	if input.UpdatedAt != nil && !input.UpdatedAt.IsZero() {
		updated.UpdatedAt = input.UpdatedAt.UTC()
	}

	return updated, nil
}

func normalizeCatalogTaxonomySubdomainCreateInput(
	input CatalogTaxonomySubdomainCreateInput,
	now time.Time,
) (persistence.CatalogSubdomainRow, error) {
	subdomainID, err := normalizeCatalogTaxonomyRequiredID(input.SubdomainID, "subdomain_id")
	if err != nil {
		return persistence.CatalogSubdomainRow{}, err
	}
	domainID, err := normalizeCatalogTaxonomyRequiredID(input.DomainID, "subdomain domain_id")
	if err != nil {
		return persistence.CatalogSubdomainRow{}, err
	}
	key, err := normalizeCatalogTaxonomyKey(input.Key, "subdomain key")
	if err != nil {
		return persistence.CatalogSubdomainRow{}, err
	}
	name, err := normalizeCatalogTaxonomyRequiredName(input.Name, "subdomain name")
	if err != nil {
		return persistence.CatalogSubdomainRow{}, err
	}

	createdAt := now.UTC()
	if input.CreatedAt != nil && !input.CreatedAt.IsZero() {
		createdAt = input.CreatedAt.UTC()
	}
	updatedAt := createdAt
	if input.UpdatedAt != nil && !input.UpdatedAt.IsZero() {
		updatedAt = input.UpdatedAt.UTC()
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}

	return persistence.CatalogSubdomainRow{
		SubdomainID: subdomainID,
		DomainID:    domainID,
		Key:         key,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Active:      active,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func applyCatalogTaxonomySubdomainUpdate(
	existing persistence.CatalogSubdomainRow,
	input CatalogTaxonomySubdomainUpdateInput,
	now time.Time,
) (persistence.CatalogSubdomainRow, error) {
	updated := existing

	if input.DomainID != nil {
		domainID, err := normalizeCatalogTaxonomyRequiredID(*input.DomainID, "subdomain domain_id")
		if err != nil {
			return persistence.CatalogSubdomainRow{}, err
		}
		updated.DomainID = domainID
	}
	if input.Key != nil {
		key, err := normalizeCatalogTaxonomyKey(*input.Key, "subdomain key")
		if err != nil {
			return persistence.CatalogSubdomainRow{}, err
		}
		updated.Key = key
	}
	if input.Name != nil {
		name, err := normalizeCatalogTaxonomyRequiredName(*input.Name, "subdomain name")
		if err != nil {
			return persistence.CatalogSubdomainRow{}, err
		}
		updated.Name = name
	}
	if input.Description != nil {
		updated.Description = strings.TrimSpace(*input.Description)
	}
	if input.Active != nil {
		updated.Active = *input.Active
	}

	updated.UpdatedAt = now.UTC()
	if input.UpdatedAt != nil && !input.UpdatedAt.IsZero() {
		updated.UpdatedAt = input.UpdatedAt.UTC()
	}

	return updated, nil
}

func normalizeCatalogTaxonomyTagCreateInput(
	input CatalogTaxonomyTagCreateInput,
	now time.Time,
) (persistence.CatalogTagRow, error) {
	tagID, err := normalizeCatalogTaxonomyRequiredID(input.TagID, "tag_id")
	if err != nil {
		return persistence.CatalogTagRow{}, err
	}
	key, err := normalizeCatalogTaxonomyKey(input.Key, "tag key")
	if err != nil {
		return persistence.CatalogTagRow{}, err
	}
	name, err := normalizeCatalogTaxonomyRequiredName(input.Name, "tag name")
	if err != nil {
		return persistence.CatalogTagRow{}, err
	}

	createdAt := now.UTC()
	if input.CreatedAt != nil && !input.CreatedAt.IsZero() {
		createdAt = input.CreatedAt.UTC()
	}
	updatedAt := createdAt
	if input.UpdatedAt != nil && !input.UpdatedAt.IsZero() {
		updatedAt = input.UpdatedAt.UTC()
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}

	return persistence.CatalogTagRow{
		TagID:       tagID,
		Key:         key,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Color:       strings.TrimSpace(input.Color),
		Active:      active,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func applyCatalogTaxonomyTagUpdate(
	existing persistence.CatalogTagRow,
	input CatalogTaxonomyTagUpdateInput,
	now time.Time,
) (persistence.CatalogTagRow, error) {
	updated := existing

	if input.Key != nil {
		key, err := normalizeCatalogTaxonomyKey(*input.Key, "tag key")
		if err != nil {
			return persistence.CatalogTagRow{}, err
		}
		updated.Key = key
	}
	if input.Name != nil {
		name, err := normalizeCatalogTaxonomyRequiredName(*input.Name, "tag name")
		if err != nil {
			return persistence.CatalogTagRow{}, err
		}
		updated.Name = name
	}
	if input.Description != nil {
		updated.Description = strings.TrimSpace(*input.Description)
	}
	if input.Color != nil {
		updated.Color = strings.TrimSpace(*input.Color)
	}
	if input.Active != nil {
		updated.Active = *input.Active
	}

	updated.UpdatedAt = now.UTC()
	if input.UpdatedAt != nil && !input.UpdatedAt.IsZero() {
		updated.UpdatedAt = input.UpdatedAt.UTC()
	}

	return updated, nil
}

func normalizeCatalogTaxonomyRequiredID(value string, fieldName string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", &CatalogTaxonomyValidationError{Field: fieldName, Detail: "is required"}
	}
	return trimmed, nil
}

func normalizeCatalogTaxonomyRequiredName(value string, fieldName string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", &CatalogTaxonomyValidationError{Field: fieldName, Detail: "is required"}
	}
	return trimmed, nil
}

func normalizeCatalogTaxonomyKey(value string, fieldName string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "", &CatalogTaxonomyValidationError{Field: fieldName, Detail: "is required"}
	}

	var builder strings.Builder
	lastWasSeparator := false
	for _, candidate := range trimmed {
		switch {
		case unicode.IsLetter(candidate) || unicode.IsDigit(candidate):
			builder.WriteRune(candidate)
			lastWasSeparator = false
		case unicode.IsSpace(candidate) || candidate == '-' || candidate == '_':
			if builder.Len() == 0 || lastWasSeparator {
				continue
			}
			builder.WriteRune('-')
			lastWasSeparator = true
		default:
			return "", &CatalogTaxonomyValidationError{
				Field:  fieldName,
				Detail: "must only include letters, numbers, spaces, underscores, or hyphens",
			}
		}
	}

	normalized := strings.Trim(builder.String(), "-")
	if normalized == "" {
		return "", &CatalogTaxonomyValidationError{
			Field:  fieldName,
			Detail: "must include at least one letter or number",
		}
	}
	return normalized, nil
}

func normalizeCatalogTaxonomyKeyList(values []string, fieldName string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		key, err := normalizeCatalogTaxonomyKey(value, fieldName)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, key)
	}

	sort.Strings(normalized)
	return normalized, nil
}

func collectTaxonomyItemIDsFromSubdomainAssignments(
	assignments []persistence.CatalogItemTaxonomyAssignmentRow,
) []string {
	itemIDs := make([]string, 0, len(assignments))
	seen := make(map[string]struct{}, len(assignments))
	for _, assignment := range assignments {
		itemID := strings.TrimSpace(assignment.ItemID)
		if itemID == "" {
			continue
		}
		if _, exists := seen[itemID]; exists {
			continue
		}
		seen[itemID] = struct{}{}
		itemIDs = append(itemIDs, itemID)
	}
	sort.Strings(itemIDs)
	return itemIDs
}

func collectTaxonomyItemIDsFromTagAssignments(
	assignments []persistence.CatalogItemTagAssignmentRow,
) []string {
	itemIDs := make([]string, 0, len(assignments))
	seen := make(map[string]struct{}, len(assignments))
	for _, assignment := range assignments {
		itemID := strings.TrimSpace(assignment.ItemID)
		if itemID == "" {
			continue
		}
		if _, exists := seen[itemID]; exists {
			continue
		}
		seen[itemID] = struct{}{}
		itemIDs = append(itemIDs, itemID)
	}
	sort.Strings(itemIDs)
	return itemIDs
}

func newCatalogTaxonomyDuplicateKeyConflict(
	objectType CatalogTaxonomyObjectType,
	objectID string,
	key string,
	cause error,
) error {
	return &CatalogTaxonomyConflictError{
		ObjectType: objectType,
		ObjectID:   objectID,
		Reason:     CatalogTaxonomyConflictReasonDuplicateKey,
		Detail: fmt.Sprintf(
			"%s %q conflicts with existing key %q",
			objectType,
			objectID,
			key,
		),
		Cause: cause,
	}
}

func newCatalogTaxonomyInUseConflict(
	objectType CatalogTaxonomyObjectType,
	objectID string,
	itemIDs []string,
) error {
	return &CatalogTaxonomyConflictError{
		ObjectType:        objectType,
		ObjectID:          objectID,
		Reason:            CatalogTaxonomyConflictReasonInUse,
		Detail:            fmt.Sprintf("%s %q is assigned to catalog items", objectType, objectID),
		ReferencedItemIDs: append([]string{}, itemIDs...),
	}
}

func isCatalogTaxonomyUniqueConstraintError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}

func isCatalogTaxonomyForeignKeyConstraintError(err error) bool {
	return strings.Contains(strings.ToUpper(err.Error()), "FOREIGN KEY")
}

func catalogTaxonomyStringPointer(value string) *string {
	return &value
}
