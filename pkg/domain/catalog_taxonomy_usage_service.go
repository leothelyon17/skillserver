package domain

import (
	"context"
	"errors"
	"fmt"

	"github.com/mudler/skillserver/pkg/persistence"
)

type catalogTaxonomyUsageDomainRepository interface {
	GetByDomainID(ctx context.Context, domainID string) (persistence.CatalogDomainRow, error)
}

type catalogTaxonomyUsageSubdomainRepository interface {
	GetBySubdomainID(ctx context.Context, subdomainID string) (persistence.CatalogSubdomainRow, error)
}

type catalogTaxonomyUsageTagRepository interface {
	GetByTagID(ctx context.Context, tagID string) (persistence.CatalogTagRow, error)
}

type catalogTaxonomyUsageAssignmentRepository interface {
	GetUsageByDomainID(
		ctx context.Context,
		domainID string,
		previewLimit int,
	) (persistence.CatalogTaxonomyUsageQueryResult, error)
	GetUsageBySubdomainID(
		ctx context.Context,
		subdomainID string,
		previewLimit int,
	) (persistence.CatalogTaxonomyUsageQueryResult, error)
}

type catalogTaxonomyUsageTagAssignmentRepository interface {
	GetUsageByTagID(
		ctx context.Context,
		tagID string,
		previewLimit int,
	) (persistence.CatalogTaxonomyUsageQueryResult, error)
}

// CatalogTaxonomyUsageSummary describes one delete-preflight usage view.
type CatalogTaxonomyUsageSummary struct {
	ObjectType        CatalogTaxonomyObjectType      `json:"object_type"`
	ObjectID          string                         `json:"object_id"`
	AssignmentCount   int                            `json:"assignment_count"`
	DistinctItemCount int                            `json:"distinct_item_count"`
	PreviewItemIDs    []string                       `json:"preview_item_ids"`
	BlockingReason    *CatalogTaxonomyConflictReason `json:"blocking_reason,omitempty"`
}

// CatalogTaxonomyUsageService assembles reusable usage/preflight summaries.
type CatalogTaxonomyUsageService struct {
	domainRepo        catalogTaxonomyUsageDomainRepository
	subdomainRepo     catalogTaxonomyUsageSubdomainRepository
	tagRepo           catalogTaxonomyUsageTagRepository
	assignmentRepo    catalogTaxonomyUsageAssignmentRepository
	tagAssignmentRepo catalogTaxonomyUsageTagAssignmentRepository
}

// NewCatalogTaxonomyUsageService constructs a taxonomy usage/preflight service.
func NewCatalogTaxonomyUsageService(
	domainRepo catalogTaxonomyUsageDomainRepository,
	subdomainRepo catalogTaxonomyUsageSubdomainRepository,
	tagRepo catalogTaxonomyUsageTagRepository,
	assignmentRepo catalogTaxonomyUsageAssignmentRepository,
	tagAssignmentRepo catalogTaxonomyUsageTagAssignmentRepository,
) (*CatalogTaxonomyUsageService, error) {
	if domainRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy usage domain repository is required")
	}
	if subdomainRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy usage subdomain repository is required")
	}
	if tagRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy usage tag repository is required")
	}
	if assignmentRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy usage assignment repository is required")
	}
	if tagAssignmentRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy usage tag assignment repository is required")
	}

	return &CatalogTaxonomyUsageService{
		domainRepo:        domainRepo,
		subdomainRepo:     subdomainRepo,
		tagRepo:           tagRepo,
		assignmentRepo:    assignmentRepo,
		tagAssignmentRepo: tagAssignmentRepo,
	}, nil
}

// GetDomainUsage returns one domain delete-preflight usage summary.
func (s *CatalogTaxonomyUsageService) GetDomainUsage(
	ctx context.Context,
	domainID string,
	previewLimit int,
) (CatalogTaxonomyUsageSummary, error) {
	if s == nil {
		return CatalogTaxonomyUsageSummary{}, fmt.Errorf("catalog taxonomy usage service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if _, err := s.domainRepo.GetByDomainID(ctx, domainID); err != nil {
		if errors.Is(err, persistence.ErrCatalogDomainNotFound) {
			return CatalogTaxonomyUsageSummary{}, fmt.Errorf("%w: domain_id=%q", ErrCatalogTaxonomyDomainNotFound, domainID)
		}
		return CatalogTaxonomyUsageSummary{}, fmt.Errorf("get catalog taxonomy domain %q: %w", domainID, err)
	}

	usage, err := s.assignmentRepo.GetUsageByDomainID(ctx, domainID, previewLimit)
	if err != nil {
		return CatalogTaxonomyUsageSummary{}, fmt.Errorf("get catalog taxonomy domain usage %q: %w", domainID, err)
	}

	return buildCatalogTaxonomyUsageSummary(CatalogTaxonomyObjectDomain, domainID, usage), nil
}

// GetSubdomainUsage returns one subdomain delete-preflight usage summary.
func (s *CatalogTaxonomyUsageService) GetSubdomainUsage(
	ctx context.Context,
	subdomainID string,
	previewLimit int,
) (CatalogTaxonomyUsageSummary, error) {
	if s == nil {
		return CatalogTaxonomyUsageSummary{}, fmt.Errorf("catalog taxonomy usage service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if _, err := s.subdomainRepo.GetBySubdomainID(ctx, subdomainID); err != nil {
		if errors.Is(err, persistence.ErrCatalogSubdomainNotFound) {
			return CatalogTaxonomyUsageSummary{}, fmt.Errorf("%w: subdomain_id=%q", ErrCatalogTaxonomySubdomainNotFound, subdomainID)
		}
		return CatalogTaxonomyUsageSummary{}, fmt.Errorf("get catalog taxonomy subdomain %q: %w", subdomainID, err)
	}

	usage, err := s.assignmentRepo.GetUsageBySubdomainID(ctx, subdomainID, previewLimit)
	if err != nil {
		return CatalogTaxonomyUsageSummary{}, fmt.Errorf("get catalog taxonomy subdomain usage %q: %w", subdomainID, err)
	}

	return buildCatalogTaxonomyUsageSummary(CatalogTaxonomyObjectSubdomain, subdomainID, usage), nil
}

// GetTagUsage returns one tag delete-preflight usage summary.
func (s *CatalogTaxonomyUsageService) GetTagUsage(
	ctx context.Context,
	tagID string,
	previewLimit int,
) (CatalogTaxonomyUsageSummary, error) {
	if s == nil {
		return CatalogTaxonomyUsageSummary{}, fmt.Errorf("catalog taxonomy usage service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if _, err := s.tagRepo.GetByTagID(ctx, tagID); err != nil {
		if errors.Is(err, persistence.ErrCatalogTagNotFound) {
			return CatalogTaxonomyUsageSummary{}, fmt.Errorf("%w: tag_id=%q", ErrCatalogTaxonomyTagNotFound, tagID)
		}
		return CatalogTaxonomyUsageSummary{}, fmt.Errorf("get catalog taxonomy tag %q: %w", tagID, err)
	}

	usage, err := s.tagAssignmentRepo.GetUsageByTagID(ctx, tagID, previewLimit)
	if err != nil {
		return CatalogTaxonomyUsageSummary{}, fmt.Errorf("get catalog taxonomy tag usage %q: %w", tagID, err)
	}

	return buildCatalogTaxonomyUsageSummary(CatalogTaxonomyObjectTag, tagID, usage), nil
}

func buildCatalogTaxonomyUsageSummary(
	objectType CatalogTaxonomyObjectType,
	objectID string,
	usage persistence.CatalogTaxonomyUsageQueryResult,
) CatalogTaxonomyUsageSummary {
	var blockingReason *CatalogTaxonomyConflictReason
	if usage.AssignmentCount > 0 {
		reason := CatalogTaxonomyConflictReasonInUse
		blockingReason = &reason
	}

	return CatalogTaxonomyUsageSummary{
		ObjectType:        objectType,
		ObjectID:          objectID,
		AssignmentCount:   usage.AssignmentCount,
		DistinctItemCount: usage.DistinctItemCount,
		PreviewItemIDs:    append([]string{}, usage.PreviewItemIDs...),
		BlockingReason:    blockingReason,
	}
}
