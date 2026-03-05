package domain

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mudler/skillserver/pkg/persistence"
)

type catalogEffectiveSourceRepository interface {
	List(ctx context.Context, filter persistence.CatalogSourceListFilter) ([]persistence.CatalogSourceRow, error)
}

type catalogEffectiveOverlayRepository interface {
	List(ctx context.Context, filter persistence.CatalogMetadataOverlayListFilter) ([]persistence.CatalogMetadataOverlayRow, error)
}

type catalogEffectiveTaxonomyAssignmentRepository interface {
	List(
		ctx context.Context,
		filter persistence.CatalogItemTaxonomyAssignmentListFilter,
	) ([]persistence.CatalogItemTaxonomyAssignmentRow, error)
}

type catalogEffectiveTagAssignmentRepository interface {
	List(
		ctx context.Context,
		filter persistence.CatalogItemTagAssignmentListFilter,
	) ([]persistence.CatalogItemTagAssignmentRow, error)
}

type catalogEffectiveDomainRepository interface {
	List(ctx context.Context, filter persistence.CatalogDomainListFilter) ([]persistence.CatalogDomainRow, error)
}

type catalogEffectiveSubdomainRepository interface {
	List(ctx context.Context, filter persistence.CatalogSubdomainListFilter) ([]persistence.CatalogSubdomainRow, error)
}

type catalogEffectiveTagRepository interface {
	List(ctx context.Context, filter persistence.CatalogTagListFilter) ([]persistence.CatalogTagRow, error)
}

// CatalogTagMatchMode controls taxonomy tag matching behavior in effective list filters.
type CatalogTagMatchMode string

const (
	// CatalogTagMatchAny includes items that match at least one requested tag.
	CatalogTagMatchAny CatalogTagMatchMode = "any"
	// CatalogTagMatchAll includes items that match every requested tag.
	CatalogTagMatchAll CatalogTagMatchMode = "all"
)

// IsValid reports whether the requested tag match mode is supported.
func (m CatalogTagMatchMode) IsValid() bool {
	switch m {
	case CatalogTagMatchAny, CatalogTagMatchAll:
		return true
	default:
		return false
	}
}

// CatalogEffectiveListFilter constrains effective projection list queries.
type CatalogEffectiveListFilter struct {
	ItemID               string
	ItemIDs              []string
	Classifier           *CatalogClassifier
	SourceType           *persistence.CatalogSourceType
	SourceRepo           *string
	IncludeDeleted       bool
	PrimaryDomainID      string
	SecondaryDomainID    string
	DomainID             string
	PrimarySubdomainID   string
	SecondarySubdomainID string
	SubdomainID          string
	TagIDs               []string
	TagMatch             CatalogTagMatchMode
}

type catalogEffectiveTaxonomyFilter struct {
	PrimaryDomainID      string
	SecondaryDomainID    string
	DomainID             string
	PrimarySubdomainID   string
	SecondarySubdomainID string
	SubdomainID          string
	TagIDs               []string
	TagMatch             CatalogTagMatchMode
}

func (f catalogEffectiveTaxonomyFilter) hasConstraints() bool {
	return f.PrimaryDomainID != "" ||
		f.SecondaryDomainID != "" ||
		f.DomainID != "" ||
		f.PrimarySubdomainID != "" ||
		f.SecondarySubdomainID != "" ||
		f.SubdomainID != "" ||
		len(f.TagIDs) > 0
}

type catalogEffectiveTaxonomyReferences struct {
	domainByID    map[string]persistence.CatalogDomainRow
	subdomainByID map[string]persistence.CatalogSubdomainRow
	tagByID       map[string]persistence.CatalogTagRow
}

type catalogEffectiveTaxonomyProjection struct {
	PrimaryDomain      *CatalogTaxonomyReference
	PrimarySubdomain   *CatalogTaxonomyReference
	SecondaryDomain    *CatalogTaxonomyReference
	SecondarySubdomain *CatalogTaxonomyReference
	Tags               []CatalogTaxonomyReference
}

// CatalogEffectiveService projects source + overlay rows into effective catalog items.
type CatalogEffectiveService struct {
	sourceRepo             catalogEffectiveSourceRepository
	overlayRepo            catalogEffectiveOverlayRepository
	taxonomyAssignmentRepo catalogEffectiveTaxonomyAssignmentRepository
	tagAssignmentRepo      catalogEffectiveTagAssignmentRepository
	domainRepo             catalogEffectiveDomainRepository
	subdomainRepo          catalogEffectiveSubdomainRepository
	tagRepo                catalogEffectiveTagRepository
}

// NewCatalogEffectiveService creates an effective catalog projection service.
func NewCatalogEffectiveService(
	sourceRepo catalogEffectiveSourceRepository,
	overlayRepo catalogEffectiveOverlayRepository,
	taxonomyAssignmentRepo catalogEffectiveTaxonomyAssignmentRepository,
	tagAssignmentRepo catalogEffectiveTagAssignmentRepository,
	domainRepo catalogEffectiveDomainRepository,
	subdomainRepo catalogEffectiveSubdomainRepository,
	tagRepo catalogEffectiveTagRepository,
) (*CatalogEffectiveService, error) {
	if sourceRepo == nil {
		return nil, fmt.Errorf("catalog effective source repository is required")
	}
	if overlayRepo == nil {
		return nil, fmt.Errorf("catalog effective overlay repository is required")
	}
	if taxonomyAssignmentRepo == nil {
		return nil, fmt.Errorf("catalog effective taxonomy assignment repository is required")
	}
	if tagAssignmentRepo == nil {
		return nil, fmt.Errorf("catalog effective tag assignment repository is required")
	}
	if domainRepo == nil {
		return nil, fmt.Errorf("catalog effective domain repository is required")
	}
	if subdomainRepo == nil {
		return nil, fmt.Errorf("catalog effective subdomain repository is required")
	}
	if tagRepo == nil {
		return nil, fmt.Errorf("catalog effective tag repository is required")
	}

	return &CatalogEffectiveService{
		sourceRepo:             sourceRepo,
		overlayRepo:            overlayRepo,
		taxonomyAssignmentRepo: taxonomyAssignmentRepo,
		tagAssignmentRepo:      tagAssignmentRepo,
		domainRepo:             domainRepo,
		subdomainRepo:          subdomainRepo,
		tagRepo:                tagRepo,
	}, nil
}

// List returns effective catalog items with deterministic ordering.
func (s *CatalogEffectiveService) List(ctx context.Context, filter CatalogEffectiveListFilter) ([]CatalogItem, error) {
	if s == nil {
		return nil, fmt.Errorf("catalog effective service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	sourceFilter, taxonomyFilter, err := mapEffectiveListFilter(filter)
	if err != nil {
		return nil, err
	}

	sourceRows, err := s.sourceRepo.List(ctx, sourceFilter)
	if err != nil {
		return nil, fmt.Errorf("list catalog source rows: %w", err)
	}
	if len(sourceRows) == 0 {
		return []CatalogItem{}, nil
	}

	itemIDs := make([]string, 0, len(sourceRows))
	for _, sourceRow := range sourceRows {
		itemIDs = append(itemIDs, sourceRow.ItemID)
	}
	sort.Strings(itemIDs)

	overlays, err := s.overlayRepo.List(ctx, persistence.CatalogMetadataOverlayListFilter{ItemIDs: itemIDs})
	if err != nil {
		return nil, fmt.Errorf("list catalog metadata overlays: %w", err)
	}

	overlayByItemID := make(map[string]persistence.CatalogMetadataOverlayRow, len(overlays))
	for _, overlay := range overlays {
		overlayByItemID[overlay.ItemID] = overlay
	}

	taxonomyAssignments, err := s.taxonomyAssignmentRepo.List(
		ctx,
		persistence.CatalogItemTaxonomyAssignmentListFilter{ItemIDs: itemIDs},
	)
	if err != nil {
		return nil, fmt.Errorf("list catalog taxonomy assignments: %w", err)
	}
	assignmentByItemID := make(map[string]persistence.CatalogItemTaxonomyAssignmentRow, len(taxonomyAssignments))
	for _, assignment := range taxonomyAssignments {
		assignmentByItemID[assignment.ItemID] = assignment
	}

	tagAssignments, err := s.tagAssignmentRepo.List(
		ctx,
		persistence.CatalogItemTagAssignmentListFilter{ItemIDs: itemIDs},
	)
	if err != nil {
		return nil, fmt.Errorf("list catalog tag assignments: %w", err)
	}

	tagIDsByItemID := make(map[string][]string, len(itemIDs))
	for _, tagAssignment := range tagAssignments {
		tagIDsByItemID[tagAssignment.ItemID] = append(tagIDsByItemID[tagAssignment.ItemID], tagAssignment.TagID)
	}

	taxonomyRefs, err := s.loadTaxonomyReferences(ctx, taxonomyAssignments, tagAssignments)
	if err != nil {
		return nil, err
	}

	effectiveItems := make([]CatalogItem, 0, len(sourceRows))
	for _, sourceRow := range sourceRows {
		assignment, hasAssignment := assignmentByItemID[sourceRow.ItemID]
		itemTagIDs := tagIDsByItemID[sourceRow.ItemID]
		if !effectiveTaxonomyFilterMatches(taxonomyFilter, hasAssignment, assignment, itemTagIDs) {
			continue
		}
		hasTagAssignments := len(itemTagIDs) > 0

		overlay, hasOverlay := overlayByItemID[sourceRow.ItemID]
		taxonomyProjection := buildCatalogEffectiveTaxonomyProjection(
			assignment,
			hasAssignment,
			itemTagIDs,
			taxonomyRefs,
		)

		item, mapErr := mapEffectiveCatalogItem(
			sourceRow,
			hasOverlay,
			overlay,
			hasTagAssignments,
			taxonomyProjection,
		)
		if mapErr != nil {
			return nil, mapErr
		}
		effectiveItems = append(effectiveItems, item)
	}

	return effectiveItems, nil
}

func mapEffectiveListFilter(
	filter CatalogEffectiveListFilter,
) (persistence.CatalogSourceListFilter, catalogEffectiveTaxonomyFilter, error) {
	mappedSourceFilter := persistence.CatalogSourceListFilter{
		ItemID:         filter.ItemID,
		ItemIDs:        filter.ItemIDs,
		SourceType:     filter.SourceType,
		SourceRepo:     filter.SourceRepo,
		IncludeDeleted: filter.IncludeDeleted,
	}

	if filter.Classifier != nil {
		if !filter.Classifier.IsValid() {
			return persistence.CatalogSourceListFilter{}, catalogEffectiveTaxonomyFilter{}, fmt.Errorf(
				"catalog classifier filter %q is invalid",
				*filter.Classifier,
			)
		}

		classifier, err := mapEffectiveClassifier(*filter.Classifier)
		if err != nil {
			return persistence.CatalogSourceListFilter{}, catalogEffectiveTaxonomyFilter{}, err
		}
		mappedSourceFilter.Classifier = &classifier
	}

	tagMatch, err := normalizeCatalogTagMatchMode(filter.TagMatch)
	if err != nil {
		return persistence.CatalogSourceListFilter{}, catalogEffectiveTaxonomyFilter{}, err
	}

	mappedTaxonomyFilter := catalogEffectiveTaxonomyFilter{
		PrimaryDomainID:      strings.TrimSpace(filter.PrimaryDomainID),
		SecondaryDomainID:    strings.TrimSpace(filter.SecondaryDomainID),
		DomainID:             strings.TrimSpace(filter.DomainID),
		PrimarySubdomainID:   strings.TrimSpace(filter.PrimarySubdomainID),
		SecondarySubdomainID: strings.TrimSpace(filter.SecondarySubdomainID),
		SubdomainID:          strings.TrimSpace(filter.SubdomainID),
		TagIDs:               normalizeCatalogOptionalIDList(filter.TagIDs),
		TagMatch:             tagMatch,
	}

	return mappedSourceFilter, mappedTaxonomyFilter, nil
}

func normalizeCatalogTagMatchMode(raw CatalogTagMatchMode) (CatalogTagMatchMode, error) {
	if strings.TrimSpace(string(raw)) == "" {
		return CatalogTagMatchAny, nil
	}

	normalized := CatalogTagMatchMode(strings.ToLower(strings.TrimSpace(string(raw))))
	if !normalized.IsValid() {
		return "", fmt.Errorf("catalog tag match mode %q is invalid", raw)
	}
	return normalized, nil
}

func normalizeCatalogOptionalIDList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	sort.Strings(normalized)
	return normalized
}

func (s *CatalogEffectiveService) loadTaxonomyReferences(
	ctx context.Context,
	assignments []persistence.CatalogItemTaxonomyAssignmentRow,
	tagAssignments []persistence.CatalogItemTagAssignmentRow,
) (catalogEffectiveTaxonomyReferences, error) {
	domainIDSet := map[string]struct{}{}
	subdomainIDSet := map[string]struct{}{}
	tagIDSet := map[string]struct{}{}

	for _, assignment := range assignments {
		if assignment.PrimaryDomainID != nil {
			domainIDSet[*assignment.PrimaryDomainID] = struct{}{}
		}
		if assignment.SecondaryDomainID != nil {
			domainIDSet[*assignment.SecondaryDomainID] = struct{}{}
		}
		if assignment.PrimarySubdomainID != nil {
			subdomainIDSet[*assignment.PrimarySubdomainID] = struct{}{}
		}
		if assignment.SecondarySubdomainID != nil {
			subdomainIDSet[*assignment.SecondarySubdomainID] = struct{}{}
		}
	}

	for _, tagAssignment := range tagAssignments {
		tagIDSet[tagAssignment.TagID] = struct{}{}
	}

	domainIDs := mapKeySetToSortedSlice(domainIDSet)
	subdomainIDs := mapKeySetToSortedSlice(subdomainIDSet)
	tagIDs := mapKeySetToSortedSlice(tagIDSet)

	var (
		domains    []persistence.CatalogDomainRow
		subdomains []persistence.CatalogSubdomainRow
		tags       []persistence.CatalogTagRow
		err        error
	)

	if len(domainIDs) > 0 {
		domains, err = s.domainRepo.List(ctx, persistence.CatalogDomainListFilter{DomainIDs: domainIDs})
		if err != nil {
			return catalogEffectiveTaxonomyReferences{}, fmt.Errorf("list catalog domains for effective projection: %w", err)
		}
	}

	if len(subdomainIDs) > 0 {
		subdomains, err = s.subdomainRepo.List(
			ctx,
			persistence.CatalogSubdomainListFilter{SubdomainIDs: subdomainIDs},
		)
		if err != nil {
			return catalogEffectiveTaxonomyReferences{}, fmt.Errorf(
				"list catalog subdomains for effective projection: %w",
				err,
			)
		}
	}

	if len(tagIDs) > 0 {
		tags, err = s.tagRepo.List(ctx, persistence.CatalogTagListFilter{TagIDs: tagIDs})
		if err != nil {
			return catalogEffectiveTaxonomyReferences{}, fmt.Errorf("list catalog tags for effective projection: %w", err)
		}
	}

	domainByID := make(map[string]persistence.CatalogDomainRow, len(domains))
	for _, domainRow := range domains {
		domainByID[domainRow.DomainID] = domainRow
	}

	subdomainByID := make(map[string]persistence.CatalogSubdomainRow, len(subdomains))
	for _, subdomainRow := range subdomains {
		subdomainByID[subdomainRow.SubdomainID] = subdomainRow
	}

	tagByID := make(map[string]persistence.CatalogTagRow, len(tags))
	for _, tagRow := range tags {
		tagByID[tagRow.TagID] = tagRow
	}

	return catalogEffectiveTaxonomyReferences{
		domainByID:    domainByID,
		subdomainByID: subdomainByID,
		tagByID:       tagByID,
	}, nil
}

func mapKeySetToSortedSlice(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}

	sorted := make([]string, 0, len(values))
	for value := range values {
		sorted = append(sorted, value)
	}
	sort.Strings(sorted)
	return sorted
}

func effectiveTaxonomyFilterMatches(
	filter catalogEffectiveTaxonomyFilter,
	hasAssignment bool,
	assignment persistence.CatalogItemTaxonomyAssignmentRow,
	tagIDs []string,
) bool {
	if filter.PrimaryDomainID != "" {
		if !hasAssignment || !catalogOptionalIDEquals(assignment.PrimaryDomainID, filter.PrimaryDomainID) {
			return false
		}
	}
	if filter.SecondaryDomainID != "" {
		if !hasAssignment || !catalogOptionalIDEquals(assignment.SecondaryDomainID, filter.SecondaryDomainID) {
			return false
		}
	}
	if filter.DomainID != "" {
		if !hasAssignment ||
			(!catalogOptionalIDEquals(assignment.PrimaryDomainID, filter.DomainID) &&
				!catalogOptionalIDEquals(assignment.SecondaryDomainID, filter.DomainID)) {
			return false
		}
	}
	if filter.PrimarySubdomainID != "" {
		if !hasAssignment || !catalogOptionalIDEquals(assignment.PrimarySubdomainID, filter.PrimarySubdomainID) {
			return false
		}
	}
	if filter.SecondarySubdomainID != "" {
		if !hasAssignment || !catalogOptionalIDEquals(assignment.SecondarySubdomainID, filter.SecondarySubdomainID) {
			return false
		}
	}
	if filter.SubdomainID != "" {
		if !hasAssignment ||
			(!catalogOptionalIDEquals(assignment.PrimarySubdomainID, filter.SubdomainID) &&
				!catalogOptionalIDEquals(assignment.SecondarySubdomainID, filter.SubdomainID)) {
			return false
		}
	}

	if len(filter.TagIDs) == 0 {
		return true
	}
	if len(tagIDs) == 0 {
		return false
	}

	tagSet := make(map[string]struct{}, len(tagIDs))
	for _, tagID := range tagIDs {
		tagSet[tagID] = struct{}{}
	}

	switch filter.TagMatch {
	case CatalogTagMatchAll:
		for _, wantedTagID := range filter.TagIDs {
			if _, exists := tagSet[wantedTagID]; !exists {
				return false
			}
		}
		return true
	case CatalogTagMatchAny:
		for _, wantedTagID := range filter.TagIDs {
			if _, exists := tagSet[wantedTagID]; exists {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func catalogOptionalIDEquals(candidate *string, expected string) bool {
	if candidate == nil {
		return false
	}
	return strings.TrimSpace(*candidate) == expected
}

func buildCatalogEffectiveTaxonomyProjection(
	assignment persistence.CatalogItemTaxonomyAssignmentRow,
	hasAssignment bool,
	itemTagIDs []string,
	refs catalogEffectiveTaxonomyReferences,
) catalogEffectiveTaxonomyProjection {
	projection := catalogEffectiveTaxonomyProjection{
		Tags: make([]CatalogTaxonomyReference, 0, len(itemTagIDs)),
	}

	if hasAssignment {
		if assignment.PrimaryDomainID != nil {
			if domainRow, exists := refs.domainByID[*assignment.PrimaryDomainID]; exists {
				projection.PrimaryDomain = mapCatalogDomainReference(domainRow)
			}
		}
		if assignment.PrimarySubdomainID != nil {
			if subdomainRow, exists := refs.subdomainByID[*assignment.PrimarySubdomainID]; exists {
				projection.PrimarySubdomain = mapCatalogSubdomainReference(subdomainRow)
			}
		}
		if assignment.SecondaryDomainID != nil {
			if domainRow, exists := refs.domainByID[*assignment.SecondaryDomainID]; exists {
				projection.SecondaryDomain = mapCatalogDomainReference(domainRow)
			}
		}
		if assignment.SecondarySubdomainID != nil {
			if subdomainRow, exists := refs.subdomainByID[*assignment.SecondarySubdomainID]; exists {
				projection.SecondarySubdomain = mapCatalogSubdomainReference(subdomainRow)
			}
		}
	}

	for _, tagID := range itemTagIDs {
		tagRow, exists := refs.tagByID[tagID]
		if !exists {
			continue
		}
		projection.Tags = append(projection.Tags, *mapCatalogTagReference(tagRow))
	}

	return projection
}

func mapCatalogDomainReference(row persistence.CatalogDomainRow) *CatalogTaxonomyReference {
	return &CatalogTaxonomyReference{
		ID:   row.DomainID,
		Key:  row.Key,
		Name: row.Name,
	}
}

func mapCatalogSubdomainReference(row persistence.CatalogSubdomainRow) *CatalogTaxonomyReference {
	return &CatalogTaxonomyReference{
		ID:   row.SubdomainID,
		Key:  row.Key,
		Name: row.Name,
	}
}

func mapCatalogTagReference(row persistence.CatalogTagRow) *CatalogTaxonomyReference {
	return &CatalogTaxonomyReference{
		ID:   row.TagID,
		Key:  row.Key,
		Name: row.Name,
	}
}

func mapEffectiveCatalogItem(
	source persistence.CatalogSourceRow,
	hasOverlay bool,
	overlay persistence.CatalogMetadataOverlayRow,
	hasTagAssignments bool,
	taxonomyProjection catalogEffectiveTaxonomyProjection,
) (CatalogItem, error) {
	classifier, err := mapEffectiveDomainClassifier(source.Classifier)
	if err != nil {
		return CatalogItem{}, fmt.Errorf("map effective catalog item %q classifier: %w", source.ItemID, err)
	}

	name := strings.TrimSpace(source.Name)
	description := source.Description
	customMetadata := map[string]any{}
	overlayLabels := []string{}

	if hasOverlay {
		if override := resolveOverlayText(overlay.DisplayNameOverride); override != nil {
			name = *override
		}
		if override := resolveOverlayText(overlay.DescriptionOverride); override != nil {
			description = *override
		}
		customMetadata = copyCustomMetadata(overlay.CustomMetadata)
		overlayLabels = append(overlayLabels, overlay.Labels...)
	}
	labels := resolveEffectiveCatalogLabels(hasOverlay, overlayLabels, hasTagAssignments, taxonomyProjection.Tags)

	contentWritable := source.SourceType != persistence.CatalogSourceTypeGit
	metadataWritable := true

	item := CatalogItem{
		ID:                 source.ItemID,
		Classifier:         classifier,
		Name:               name,
		Description:        description,
		Content:            source.Content,
		PrimaryDomain:      taxonomyProjection.PrimaryDomain,
		PrimarySubdomain:   taxonomyProjection.PrimarySubdomain,
		SecondaryDomain:    taxonomyProjection.SecondaryDomain,
		SecondarySubdomain: taxonomyProjection.SecondarySubdomain,
		Tags:               append([]CatalogTaxonomyReference{}, taxonomyProjection.Tags...),
		ContentWritable:    contentWritable,
		MetadataWritable:   metadataWritable,
		CustomMetadata:     customMetadata,
		Labels:             labels,
		ReadOnly:           !contentWritable,
	}

	if source.ParentSkillID != nil {
		item.ParentSkillID = *source.ParentSkillID
	}
	if source.ResourcePath != nil {
		item.ResourcePath = *source.ResourcePath
	}

	return item, nil
}

func resolveEffectiveCatalogLabels(
	hasOverlay bool,
	overlayLabels []string,
	hasTagAssignments bool,
	tags []CatalogTaxonomyReference,
) []string {
	if hasTagAssignments {
		labels := make([]string, 0, len(tags))
		seen := make(map[string]struct{}, len(tags))
		for _, tag := range tags {
			label := strings.TrimSpace(tag.Name)
			if label == "" {
				label = strings.TrimSpace(tag.Key)
			}
			if label == "" {
				label = strings.TrimSpace(tag.ID)
			}
			if label == "" {
				continue
			}

			lookupKey := strings.ToLower(label)
			if _, exists := seen[lookupKey]; exists {
				continue
			}
			seen[lookupKey] = struct{}{}
			labels = append(labels, label)
		}
		return labels
	}

	if hasOverlay {
		return append([]string{}, overlayLabels...)
	}

	return []string{}
}

func mapEffectiveClassifier(classifier CatalogClassifier) (persistence.CatalogClassifier, error) {
	switch classifier {
	case CatalogClassifierSkill:
		return persistence.CatalogClassifierSkill, nil
	case CatalogClassifierPrompt:
		return persistence.CatalogClassifierPrompt, nil
	default:
		return "", fmt.Errorf("catalog classifier %q is invalid", classifier)
	}
}

func mapEffectiveDomainClassifier(classifier persistence.CatalogClassifier) (CatalogClassifier, error) {
	switch classifier {
	case persistence.CatalogClassifierSkill:
		return CatalogClassifierSkill, nil
	case persistence.CatalogClassifierPrompt:
		return CatalogClassifierPrompt, nil
	default:
		return "", fmt.Errorf("catalog classifier %q is invalid", classifier)
	}
}

func resolveOverlayText(override *string) *string {
	if override == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*override)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func copyCustomMetadata(customMetadata map[string]any) map[string]any {
	if len(customMetadata) == 0 {
		return map[string]any{}
	}
	copied := make(map[string]any, len(customMetadata))
	for key, value := range customMetadata {
		copied[key] = value
	}
	return copied
}
