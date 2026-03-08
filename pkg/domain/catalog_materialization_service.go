package domain

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

var (
	// ErrCatalogMaterializationInvalidRequest indicates malformed materialization inputs.
	ErrCatalogMaterializationInvalidRequest = errors.New("catalog materialization request is invalid")
	// ErrCatalogMaterializationItemNotFound indicates a requested item could not be resolved.
	ErrCatalogMaterializationItemNotFound = errors.New("catalog materialization item not found")
	// ErrCatalogMaterializationUnsupportedClassifier indicates classifier-specific handling is missing.
	ErrCatalogMaterializationUnsupportedClassifier = errors.New("catalog materialization classifier is not supported")
	// ErrCatalogMaterializationDestinationOutsideAllowedRoots indicates destination-path boundary violations.
	ErrCatalogMaterializationDestinationOutsideAllowedRoots = errors.New(
		"catalog materialization destination is outside allowed roots",
	)
)

// CatalogMaterializationRequest describes one materialization plan/write request.
type CatalogMaterializationRequest struct {
	ItemIDs        []string                         `json:"item_ids"`
	DestinationDir string                           `json:"destination_dir"`
	ConflictPolicy CatalogMaterializeConflictPolicy `json:"conflict_policy,omitempty"`
	DryRun         bool                             `json:"dry_run,omitempty"`
}

// CatalogMaterializationItemStatus summarizes item-level outcomes.
type CatalogMaterializationItemStatus string

const (
	CatalogMaterializationItemStatusPlanned  CatalogMaterializationItemStatus = "planned"
	CatalogMaterializationItemStatusWritten  CatalogMaterializationItemStatus = "written"
	CatalogMaterializationItemStatusSkipped  CatalogMaterializationItemStatus = "skipped"
	CatalogMaterializationItemStatusConflict CatalogMaterializationItemStatus = "conflict"
)

// CatalogMaterializationAction describes one file-level action outcome.
type CatalogMaterializationAction string

const (
	CatalogMaterializationActionCreate    CatalogMaterializationAction = "create"
	CatalogMaterializationActionOverwrite CatalogMaterializationAction = "overwrite"
	CatalogMaterializationActionSkip      CatalogMaterializationAction = "skip"
	CatalogMaterializationActionConflict  CatalogMaterializationAction = "conflict"
)

// CatalogMaterializationFileResult captures one planned/executed file action.
type CatalogMaterializationFileResult struct {
	SourcePath     string                           `json:"source_path,omitempty"`
	TargetPath     string                           `json:"target_path"`
	ResolvedPath   string                           `json:"resolved_path"`
	Action         CatalogMaterializationAction     `json:"action"`
	ConflictPolicy CatalogMaterializeConflictPolicy `json:"conflict_policy"`
	Exists         bool                             `json:"exists"`
	Written        bool                             `json:"written"`
	Bytes          int                              `json:"bytes"`
	Error          string                           `json:"error,omitempty"`
}

// CatalogMaterializationItemResult captures one item-level plan/execution result.
type CatalogMaterializationItemResult struct {
	ItemID         string                             `json:"item_id"`
	Classifier     CatalogClassifier                  `json:"classifier"`
	SourceRef      string                             `json:"source_ref,omitempty"`
	TargetPath     string                             `json:"target_path,omitempty"`
	ConflictPolicy CatalogMaterializeConflictPolicy   `json:"conflict_policy"`
	Status         CatalogMaterializationItemStatus   `json:"status"`
	Files          []CatalogMaterializationFileResult `json:"files"`
}

// CatalogMaterializationResult captures one dry-run/write response payload.
type CatalogMaterializationResult struct {
	DryRun                 bool                               `json:"dry_run"`
	DestinationDir         string                             `json:"destination_dir"`
	ResolvedDestinationDir string                             `json:"resolved_destination_dir"`
	Items                  []CatalogMaterializationItemResult `json:"items"`
}

type catalogMaterializationCatalogReader interface {
	ListCatalogItems() ([]CatalogItem, error)
	ReadSkill(name string) (*Skill, error)
}

type catalogMaterializationFilePlan struct {
	sourcePath   string
	targetPath   string
	resolvedPath string
	action       CatalogMaterializationAction
	exists       bool
	written      bool
	mode         fs.FileMode
	content      []byte
	errMessage   string
}

type catalogMaterializationItemPlan struct {
	itemID         string
	classifier     CatalogClassifier
	sourceRef      string
	targetPath     string
	conflictPolicy CatalogMaterializeConflictPolicy
	files          []catalogMaterializationFilePlan
}

// CatalogMaterializationService provides shared planning/writing for catalog items.
type CatalogMaterializationService struct {
	catalogReader             catalogMaterializationCatalogReader
	allowedDestinationRoots   []string
	canonicalAllowedDestRoots []string
}

// NewCatalogMaterializationService constructs a shared materialization planner/writer.
func NewCatalogMaterializationService(
	catalogReader catalogMaterializationCatalogReader,
	allowedDestinationRoots []string,
) (*CatalogMaterializationService, error) {
	if catalogReader == nil {
		return nil, fmt.Errorf("catalog materialization reader is required")
	}

	normalizedRoots, canonicalRoots, err := normalizeCatalogMaterializationAllowedRoots(allowedDestinationRoots)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCatalogMaterializationInvalidRequest, err)
	}
	if len(normalizedRoots) == 0 {
		return nil, fmt.Errorf(
			"%w: at least one allowed destination root is required",
			ErrCatalogMaterializationInvalidRequest,
		)
	}

	return &CatalogMaterializationService{
		catalogReader:             catalogReader,
		allowedDestinationRoots:   normalizedRoots,
		canonicalAllowedDestRoots: canonicalRoots,
	}, nil
}

// Plan resolves deterministic target-path/action results without writing files.
func (s *CatalogMaterializationService) Plan(
	ctx context.Context,
	request CatalogMaterializationRequest,
) (CatalogMaterializationResult, error) {
	request.DryRun = true
	return s.Materialize(ctx, request)
}

// Materialize plans catalog item target paths and optionally applies writes.
func (s *CatalogMaterializationService) Materialize(
	ctx context.Context,
	request CatalogMaterializationRequest,
) (CatalogMaterializationResult, error) {
	if s == nil {
		return CatalogMaterializationResult{}, fmt.Errorf("catalog materialization service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return CatalogMaterializationResult{}, ctx.Err()
	default:
	}

	normalizedPolicy, err := normalizeCatalogMaterializationConflictPolicy(request.ConflictPolicy)
	if err != nil {
		return CatalogMaterializationResult{}, err
	}

	destinationDir, canonicalDestinationDir, err := s.resolveDestinationDir(request.DestinationDir)
	if err != nil {
		return CatalogMaterializationResult{}, err
	}

	itemIDs, err := normalizeCatalogMaterializationItemIDs(request.ItemIDs)
	if err != nil {
		return CatalogMaterializationResult{}, err
	}

	catalogItems, err := s.catalogReader.ListCatalogItems()
	if err != nil {
		return CatalogMaterializationResult{}, fmt.Errorf("list catalog items: %w", err)
	}
	itemsByID := make(map[string]CatalogItem, len(catalogItems))
	for _, item := range catalogItems {
		itemsByID[item.ID] = item
	}

	plans := make([]catalogMaterializationItemPlan, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		select {
		case <-ctx.Done():
			return CatalogMaterializationResult{}, ctx.Err()
		default:
		}

		normalizedItemID, err := normalizeCatalogMaterializationItemID(itemID)
		if err != nil {
			return CatalogMaterializationResult{}, err
		}

		item, exists := itemsByID[normalizedItemID]
		if !exists {
			return CatalogMaterializationResult{}, fmt.Errorf(
				"%w: %q",
				ErrCatalogMaterializationItemNotFound,
				itemID,
			)
		}

		itemPlan, err := s.planCatalogMaterializationItem(
			item,
			destinationDir,
			normalizedPolicy,
		)
		if err != nil {
			return CatalogMaterializationResult{}, err
		}
		plans = append(plans, itemPlan)
	}

	result := CatalogMaterializationResult{
		DryRun:                 request.DryRun,
		DestinationDir:         destinationDir,
		ResolvedDestinationDir: canonicalDestinationDir,
		Items:                  buildCatalogMaterializationItemResults(plans, request.DryRun),
	}

	if request.DryRun {
		return result, nil
	}

	if err := s.executeCatalogMaterializationPlans(ctx, plans, result.Items); err != nil {
		return CatalogMaterializationResult{}, err
	}
	return result, nil
}

func normalizeCatalogMaterializationAllowedRoots(
	allowedRoots []string,
) ([]string, []string, error) {
	normalizedRoots := make([]string, 0, len(allowedRoots))
	canonicalRoots := make([]string, 0, len(allowedRoots))
	seenRoots := make(map[string]struct{}, len(allowedRoots))

	for _, root := range allowedRoots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		if !filepath.IsAbs(trimmed) {
			return nil, nil, fmt.Errorf("allowed destination root %q must be absolute", root)
		}

		absoluteRoot, err := filepath.Abs(filepath.Clean(trimmed))
		if err != nil {
			return nil, nil, fmt.Errorf("resolve allowed destination root %q: %w", root, err)
		}

		canonicalRoot, err := canonicalizePathForWrite(absoluteRoot)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve allowed destination root %q: %w", root, err)
		}
		if _, exists := seenRoots[canonicalRoot]; exists {
			continue
		}
		seenRoots[canonicalRoot] = struct{}{}

		normalizedRoots = append(normalizedRoots, absoluteRoot)
		canonicalRoots = append(canonicalRoots, canonicalRoot)
	}

	sort.Strings(normalizedRoots)
	sort.Strings(canonicalRoots)
	return normalizedRoots, canonicalRoots, nil
}

func normalizeCatalogMaterializationConflictPolicy(
	raw CatalogMaterializeConflictPolicy,
) (CatalogMaterializeConflictPolicy, error) {
	if strings.TrimSpace(string(raw)) == "" {
		return CatalogMaterializeConflictPolicyError, nil
	}

	parsed, err := ParseCatalogMaterializeConflictPolicy(string(raw))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCatalogMaterializationInvalidRequest, err)
	}
	return parsed, nil
}

func normalizeCatalogMaterializationItemIDs(itemIDs []string) ([]string, error) {
	normalized := make([]string, 0, len(itemIDs))
	seen := make(map[string]struct{}, len(itemIDs))

	for _, itemID := range itemIDs {
		if strings.TrimSpace(itemID) == "" {
			continue
		}

		normalizedID, err := normalizeCatalogMaterializationItemID(itemID)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[normalizedID]; exists {
			continue
		}
		seen[normalizedID] = struct{}{}
		normalized = append(normalized, normalizedID)
	}

	if len(normalized) == 0 {
		return nil, fmt.Errorf(
			"%w: at least one item id is required",
			ErrCatalogMaterializationInvalidRequest,
		)
	}
	return normalized, nil
}

func normalizeCatalogMaterializationItemID(rawItemID string) (string, error) {
	itemID := strings.TrimSpace(rawItemID)
	if itemID == "" {
		return "", fmt.Errorf("%w: item id is required", ErrCatalogMaterializationInvalidRequest)
	}

	classifierToken, payload, hasClassifier := strings.Cut(itemID, ":")
	if !hasClassifier {
		skillKey := CanonicalSkillCatalogKey(itemID)
		if skillKey == "" {
			return "", fmt.Errorf("%w: item id %q is invalid", ErrCatalogMaterializationInvalidRequest, rawItemID)
		}
		return BuildSkillCatalogItemID(skillKey), nil
	}

	classifier, err := ParseCatalogClassifier(classifierToken)
	if err != nil {
		return "", fmt.Errorf(
			"%w: item id %q has an invalid classifier",
			ErrCatalogMaterializationInvalidRequest,
			rawItemID,
		)
	}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return "", fmt.Errorf("%w: item id %q has an empty payload", ErrCatalogMaterializationInvalidRequest, rawItemID)
	}

	switch classifier {
	case CatalogClassifierSkill:
		skillKey := CanonicalSkillCatalogKey(payload)
		if skillKey == "" {
			return "", fmt.Errorf("%w: item id %q is invalid", ErrCatalogMaterializationInvalidRequest, rawItemID)
		}
		return BuildSkillCatalogItemID(skillKey), nil
	case CatalogClassifierPrompt:
		skillID, resourcePath, err := parseCatalogResourceItemPayload(payload)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrCatalogMaterializationInvalidRequest, err)
		}
		return BuildPromptCatalogItemID(skillID, resourcePath), nil
	case CatalogClassifierRule:
		skillID, resourcePath, err := parseCatalogResourceItemPayload(payload)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrCatalogMaterializationInvalidRequest, err)
		}
		return BuildRuleCatalogItemID(skillID, resourcePath), nil
	default:
		return "", fmt.Errorf(
			"%w: classifier %q",
			ErrCatalogMaterializationUnsupportedClassifier,
			classifier,
		)
	}
}

func parseCatalogResourceItemPayload(payload string) (string, string, error) {
	separator := strings.LastIndex(payload, ":")
	if separator <= 0 || separator >= len(payload)-1 {
		return "", "", fmt.Errorf("catalog resource payload %q must be <skill-id>:<resource-path>", payload)
	}

	skillID := strings.TrimSpace(payload[:separator])
	resourcePath := strings.TrimSpace(payload[separator+1:])
	if skillID == "" || resourcePath == "" {
		return "", "", fmt.Errorf("catalog resource payload %q must be <skill-id>:<resource-path>", payload)
	}
	return skillID, resourcePath, nil
}

func (s *CatalogMaterializationService) resolveDestinationDir(raw string) (string, string, error) {
	destination := strings.TrimSpace(raw)
	if destination == "" {
		return "", "", fmt.Errorf(
			"%w: destination_dir is required",
			ErrCatalogMaterializationInvalidRequest,
		)
	}
	if !filepath.IsAbs(destination) {
		return "", "", fmt.Errorf(
			"%w: destination_dir must be absolute",
			ErrCatalogMaterializationInvalidRequest,
		)
	}

	absoluteDestination, err := filepath.Abs(filepath.Clean(destination))
	if err != nil {
		return "", "", fmt.Errorf(
			"%w: resolve destination_dir: %v",
			ErrCatalogMaterializationInvalidRequest,
			err,
		)
	}
	canonicalDestination, err := canonicalizePathForWrite(absoluteDestination)
	if err != nil {
		return "", "", fmt.Errorf(
			"%w: resolve destination_dir: %v",
			ErrCatalogMaterializationInvalidRequest,
			err,
		)
	}

	if !s.isAllowedDestinationPath(canonicalDestination) {
		return "", "", fmt.Errorf(
			"%w: %q",
			ErrCatalogMaterializationDestinationOutsideAllowedRoots,
			absoluteDestination,
		)
	}

	return absoluteDestination, canonicalDestination, nil
}

func (s *CatalogMaterializationService) isAllowedDestinationPath(canonicalPath string) bool {
	for _, canonicalRoot := range s.canonicalAllowedDestRoots {
		if canonicalPath == canonicalRoot || isWithinRoot(canonicalPath, canonicalRoot) {
			return true
		}
	}
	return false
}

func (s *CatalogMaterializationService) planCatalogMaterializationItem(
	item CatalogItem,
	destinationDir string,
	defaultConflictPolicy CatalogMaterializeConflictPolicy,
) (catalogMaterializationItemPlan, error) {
	switch item.Classifier {
	case CatalogClassifierSkill:
		return s.planSkillMaterializationItem(item, destinationDir, defaultConflictPolicy)
	case CatalogClassifierPrompt, CatalogClassifierRule:
		return s.planFileBackedMaterializationItem(item, destinationDir, defaultConflictPolicy)
	default:
		return catalogMaterializationItemPlan{}, fmt.Errorf(
			"%w: classifier %q for item %q",
			ErrCatalogMaterializationUnsupportedClassifier,
			item.Classifier,
			item.ID,
		)
	}
}

func (s *CatalogMaterializationService) planSkillMaterializationItem(
	item CatalogItem,
	destinationDir string,
	defaultConflictPolicy CatalogMaterializeConflictPolicy,
) (catalogMaterializationItemPlan, error) {
	skillID := strings.TrimSpace(strings.TrimPrefix(item.ID, skillCatalogIDPrefix))
	if skillID == "" {
		return catalogMaterializationItemPlan{}, fmt.Errorf(
			"%w: skill item %q has an empty source ref",
			ErrCatalogMaterializationInvalidRequest,
			item.ID,
		)
	}

	skill, err := s.catalogReader.ReadSkill(skillID)
	if err != nil {
		return catalogMaterializationItemPlan{}, fmt.Errorf(
			"%w: %q",
			ErrCatalogMaterializationItemNotFound,
			item.ID,
		)
	}
	if skill == nil {
		return catalogMaterializationItemPlan{}, fmt.Errorf(
			"%w: %q",
			ErrCatalogMaterializationItemNotFound,
			item.ID,
		)
	}

	conflictPolicy := defaultConflictPolicy
	targetRoot := defaultSkillMaterializationTargetRoot(skill, skillID)
	metadata, err := ParseCatalogInstallMetadata(skill.Content)
	if err != nil {
		return catalogMaterializationItemPlan{}, fmt.Errorf(
			"%w: item %q install metadata is invalid: %v",
			ErrCatalogMaterializationInvalidRequest,
			item.ID,
			err,
		)
	}
	if metadata != nil {
		if strings.TrimSpace(metadata.Materialize.TargetPath) != "" {
			targetRoot = metadata.Materialize.TargetPath
		}
		if metadata.Materialize.ConflictPolicy.IsValid() {
			conflictPolicy = metadata.Materialize.ConflictPolicy
		}
	}

	canonicalSkillPath, err := canonicalizeExistingPath(skill.SourcePath)
	if err != nil {
		return catalogMaterializationItemPlan{}, fmt.Errorf(
			"resolve skill source path for item %q: %w",
			item.ID,
			err,
		)
	}

	filePlans := make([]catalogMaterializationFilePlan, 0, 8)
	if err := filepath.WalkDir(canonicalSkillPath, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		canonicalSourcePath, err := canonicalizeExistingPath(currentPath)
		if err != nil {
			return err
		}
		if !isWithinRoot(canonicalSourcePath, canonicalSkillPath) {
			return fmt.Errorf("skill source path escapes skill root")
		}

		relativeSourcePath, err := filepath.Rel(canonicalSkillPath, canonicalSourcePath)
		if err != nil {
			return err
		}
		normalizedRelativeSourcePath := filepath.ToSlash(filepath.Clean(relativeSourcePath))
		if normalizedRelativeSourcePath == "." || normalizedRelativeSourcePath == ".." ||
			strings.HasPrefix(normalizedRelativeSourcePath, "../") {
			return fmt.Errorf("invalid relative source path %q", relativeSourcePath)
		}

		targetPath, err := ValidateCatalogMaterializeTargetPath(path.Join(targetRoot, normalizedRelativeSourcePath))
		if err != nil {
			return fmt.Errorf("resolve target path for %q: %w", normalizedRelativeSourcePath, err)
		}
		resolvedTargetPath, exists, action, actionErrMessage, err := s.resolveMaterializationFileAction(
			destinationDir,
			targetPath,
			conflictPolicy,
		)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(canonicalSourcePath)
		if err != nil {
			return err
		}
		fileInfo, err := os.Stat(canonicalSourcePath)
		if err != nil {
			return err
		}

		filePlans = append(filePlans, catalogMaterializationFilePlan{
			sourcePath:   normalizedRelativeSourcePath,
			targetPath:   targetPath,
			resolvedPath: resolvedTargetPath,
			action:       action,
			exists:       exists,
			mode:         fileInfo.Mode().Perm(),
			content:      content,
			errMessage:   actionErrMessage,
		})
		return nil
	}); err != nil {
		return catalogMaterializationItemPlan{}, fmt.Errorf(
			"%w: resolve files for item %q: %v",
			ErrCatalogMaterializationInvalidRequest,
			item.ID,
			err,
		)
	}

	sort.Slice(filePlans, func(i, j int) bool { return filePlans[i].targetPath < filePlans[j].targetPath })
	return catalogMaterializationItemPlan{
		itemID:         item.ID,
		classifier:     CatalogClassifierSkill,
		sourceRef:      skillID,
		targetPath:     targetRoot,
		conflictPolicy: conflictPolicy,
		files:          filePlans,
	}, nil
}

func defaultSkillMaterializationTargetRoot(skill *Skill, fallbackSkillID string) string {
	if skill != nil {
		if baseName := filepath.Base(skill.SourcePath); baseName != "" && baseName != "." && baseName != string(filepath.Separator) {
			return path.Join("skills", baseName)
		}
		if skill.Metadata != nil {
			if metadataName := strings.TrimSpace(skill.Metadata.Name); metadataName != "" {
				return path.Join("skills", sanitizeCatalogMaterializationFileName(metadataName))
			}
		}
	}

	cleanSkillID := strings.TrimSpace(fallbackSkillID)
	if cleanSkillID == "" {
		return path.Join("skills", "skill")
	}
	return path.Join("skills", sanitizeCatalogMaterializationFileName(filepath.Base(filepath.FromSlash(cleanSkillID))))
}

func (s *CatalogMaterializationService) planFileBackedMaterializationItem(
	item CatalogItem,
	destinationDir string,
	defaultConflictPolicy CatalogMaterializeConflictPolicy,
) (catalogMaterializationItemPlan, error) {
	conflictPolicy := defaultConflictPolicy
	targetPath, err := resolveCatalogMaterializationTargetPath(item)
	if err != nil {
		return catalogMaterializationItemPlan{}, fmt.Errorf(
			"%w: item %q target path is invalid: %v",
			ErrCatalogMaterializationInvalidRequest,
			item.ID,
			err,
		)
	}

	metadata, err := ParseCatalogInstallMetadata(item.Content)
	if err != nil {
		return catalogMaterializationItemPlan{}, fmt.Errorf(
			"%w: item %q install metadata is invalid: %v",
			ErrCatalogMaterializationInvalidRequest,
			item.ID,
			err,
		)
	}
	if metadata != nil {
		if strings.TrimSpace(metadata.Materialize.TargetPath) != "" {
			targetPath = metadata.Materialize.TargetPath
		}
		if metadata.Materialize.ConflictPolicy.IsValid() {
			conflictPolicy = metadata.Materialize.ConflictPolicy
		}
	}

	targetPath, err = ValidateCatalogMaterializeTargetPath(targetPath)
	if err != nil {
		return catalogMaterializationItemPlan{}, fmt.Errorf(
			"%w: item %q target path is invalid: %v",
			ErrCatalogMaterializationInvalidRequest,
			item.ID,
			err,
		)
	}

	resolvedTargetPath, exists, action, actionErrMessage, err := s.resolveMaterializationFileAction(
		destinationDir,
		targetPath,
		conflictPolicy,
	)
	if err != nil {
		return catalogMaterializationItemPlan{}, err
	}

	sourceRef := strings.TrimSpace(item.ParentSkillID)
	if resourcePath := strings.TrimSpace(item.ResourcePath); resourcePath != "" {
		if sourceRef != "" {
			sourceRef += ":"
		}
		sourceRef += resourcePath
	}

	filePlan := catalogMaterializationFilePlan{
		sourcePath:   strings.TrimSpace(item.ResourcePath),
		targetPath:   targetPath,
		resolvedPath: resolvedTargetPath,
		action:       action,
		exists:       exists,
		mode:         0644,
		content:      []byte(item.Content),
		errMessage:   actionErrMessage,
	}
	if filePlan.sourcePath == "" {
		filePlan.sourcePath = item.ID
	}

	return catalogMaterializationItemPlan{
		itemID:         item.ID,
		classifier:     item.Classifier,
		sourceRef:      sourceRef,
		targetPath:     targetPath,
		conflictPolicy: conflictPolicy,
		files:          []catalogMaterializationFilePlan{filePlan},
	}, nil
}

func resolveCatalogMaterializationTargetPath(item CatalogItem) (string, error) {
	switch item.Classifier {
	case CatalogClassifierPrompt:
		fileName := defaultCatalogMaterializationResourceFileName(item)
		return ValidateCatalogMaterializeTargetPath(path.Join("prompts", fileName))
	case CatalogClassifierRule:
		fileName := defaultCatalogMaterializationResourceFileName(item)
		if isKnownProjectRuleBaseName(fileName) {
			return ValidateCatalogMaterializeTargetPath(fileName)
		}
		return ValidateCatalogMaterializeTargetPath(path.Join("rules", fileName))
	default:
		return "", fmt.Errorf("classifier %q is unsupported for file-backed targets", item.Classifier)
	}
}

func defaultCatalogMaterializationResourceFileName(item CatalogItem) string {
	fileName := filepath.Base(filepath.FromSlash(strings.TrimSpace(item.ResourcePath)))
	fileName = sanitizeCatalogMaterializationFileName(fileName)
	if fileName == "" {
		fileName = sanitizeCatalogMaterializationFileName(strings.TrimSpace(item.Name))
	}
	if fileName == "" {
		fileName = "item.md"
	}
	if !strings.HasSuffix(strings.ToLower(fileName), ".md") &&
		!strings.HasSuffix(strings.ToLower(fileName), ".markdown") {
		fileName += ".md"
	}
	return fileName
}

func sanitizeCatalogMaterializationFileName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	trimmed = strings.Trim(trimmed, "/")
	trimmed = filepath.Base(filepath.FromSlash(trimmed))
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" || trimmed == "." || trimmed == ".." {
		return ""
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", string(filepath.Separator), "-")
	return replacer.Replace(trimmed)
}

func isKnownProjectRuleBaseName(fileName string) bool {
	normalized := strings.ToLower(strings.TrimSpace(fileName))
	if normalized == "" {
		return false
	}
	for _, known := range DefaultRuleFilenameAllowlist() {
		if normalized == strings.ToLower(strings.TrimSpace(known)) {
			return true
		}
	}
	return false
}

func (s *CatalogMaterializationService) resolveMaterializationFileAction(
	destinationDir string,
	relativeTargetPath string,
	conflictPolicy CatalogMaterializeConflictPolicy,
) (resolvedPath string, exists bool, action CatalogMaterializationAction, actionErrMessage string, err error) {
	targetPath := path.Join(relativeTargetPath)
	targetPath, err = ValidateCatalogMaterializeTargetPath(targetPath)
	if err != nil {
		return "", false, "", "", fmt.Errorf(
			"%w: target path %q is invalid: %v",
			ErrCatalogMaterializationInvalidRequest,
			relativeTargetPath,
			err,
		)
	}

	fullTargetPath := filepath.Join(destinationDir, filepath.FromSlash(targetPath))
	canonicalTargetPath, err := canonicalizePathForWrite(fullTargetPath)
	if err != nil {
		return "", false, "", "", fmt.Errorf(
			"%w: resolve target path %q: %v",
			ErrCatalogMaterializationInvalidRequest,
			targetPath,
			err,
		)
	}
	if !s.isAllowedDestinationPath(canonicalTargetPath) {
		return "", false, "", "", fmt.Errorf(
			"%w: target path %q",
			ErrCatalogMaterializationDestinationOutsideAllowedRoots,
			canonicalTargetPath,
		)
	}

	targetInfo, statErr := os.Stat(canonicalTargetPath)
	switch {
	case statErr == nil:
		exists = true
		if targetInfo.IsDir() {
			return canonicalTargetPath, true, CatalogMaterializationActionConflict,
				"target path points to a directory", nil
		}

		switch conflictPolicy {
		case CatalogMaterializeConflictPolicyOverwrite:
			return canonicalTargetPath, true, CatalogMaterializationActionOverwrite, "", nil
		case CatalogMaterializeConflictPolicySkip:
			return canonicalTargetPath, true, CatalogMaterializationActionSkip, "", nil
		default:
			return canonicalTargetPath, true, CatalogMaterializationActionConflict,
				"target file already exists", nil
		}
	case os.IsNotExist(statErr):
		return canonicalTargetPath, false, CatalogMaterializationActionCreate, "", nil
	default:
		return "", false, "", "", fmt.Errorf(
			"%w: stat target path %q: %v",
			ErrCatalogMaterializationInvalidRequest,
			targetPath,
			statErr,
		)
	}
}

func buildCatalogMaterializationItemResults(
	plans []catalogMaterializationItemPlan,
	dryRun bool,
) []CatalogMaterializationItemResult {
	results := make([]CatalogMaterializationItemResult, 0, len(plans))
	for _, plan := range plans {
		fileResults := make([]CatalogMaterializationFileResult, 0, len(plan.files))
		for _, filePlan := range plan.files {
			fileResults = append(fileResults, CatalogMaterializationFileResult{
				SourcePath:     filePlan.sourcePath,
				TargetPath:     filePlan.targetPath,
				ResolvedPath:   filePlan.resolvedPath,
				Action:         filePlan.action,
				ConflictPolicy: plan.conflictPolicy,
				Exists:         filePlan.exists,
				Written:        filePlan.written,
				Bytes:          len(filePlan.content),
				Error:          filePlan.errMessage,
			})
		}

		results = append(results, CatalogMaterializationItemResult{
			ItemID:         plan.itemID,
			Classifier:     plan.classifier,
			SourceRef:      plan.sourceRef,
			TargetPath:     plan.targetPath,
			ConflictPolicy: plan.conflictPolicy,
			Status:         deriveCatalogMaterializationItemStatus(fileResults, dryRun),
			Files:          fileResults,
		})
	}

	return results
}

func deriveCatalogMaterializationItemStatus(
	fileResults []CatalogMaterializationFileResult,
	dryRun bool,
) CatalogMaterializationItemStatus {
	if len(fileResults) == 0 {
		if dryRun {
			return CatalogMaterializationItemStatusPlanned
		}
		return CatalogMaterializationItemStatusSkipped
	}

	hasConflicts := false
	hasWrites := false
	hasCreateOrOverwrite := false
	for _, fileResult := range fileResults {
		if fileResult.Action == CatalogMaterializationActionConflict {
			hasConflicts = true
		}
		if fileResult.Action == CatalogMaterializationActionCreate ||
			fileResult.Action == CatalogMaterializationActionOverwrite {
			hasCreateOrOverwrite = true
		}
		if fileResult.Written {
			hasWrites = true
		}
	}

	if hasConflicts {
		return CatalogMaterializationItemStatusConflict
	}
	if dryRun || hasCreateOrOverwrite && !hasWrites {
		return CatalogMaterializationItemStatusPlanned
	}
	if hasWrites {
		return CatalogMaterializationItemStatusWritten
	}
	return CatalogMaterializationItemStatusSkipped
}

func (s *CatalogMaterializationService) executeCatalogMaterializationPlans(
	ctx context.Context,
	plans []catalogMaterializationItemPlan,
	results []CatalogMaterializationItemResult,
) error {
	for itemIndex := range plans {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		itemPlan := &plans[itemIndex]
		itemResult := &results[itemIndex]

		for fileIndex := range itemPlan.files {
			filePlan := &itemPlan.files[fileIndex]
			fileResult := &itemResult.Files[fileIndex]

			if filePlan.action != CatalogMaterializationActionCreate &&
				filePlan.action != CatalogMaterializationActionOverwrite {
				continue
			}

			resolvedTargetPath, err := canonicalizePathForWrite(filePlan.resolvedPath)
			if err != nil {
				return fmt.Errorf("resolve target write path %q: %w", filePlan.resolvedPath, err)
			}
			if !s.isAllowedDestinationPath(resolvedTargetPath) {
				return fmt.Errorf(
					"%w: target path %q",
					ErrCatalogMaterializationDestinationOutsideAllowedRoots,
					resolvedTargetPath,
				)
			}

			parentDir := filepath.Dir(resolvedTargetPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("create parent directory %q: %w", parentDir, err)
			}

			writeMode := fs.FileMode(0644)
			if filePlan.mode != 0 {
				writeMode = filePlan.mode
			}
			if err := os.WriteFile(resolvedTargetPath, filePlan.content, writeMode); err != nil {
				return fmt.Errorf("write target file %q: %w", resolvedTargetPath, err)
			}

			filePlan.written = true
			filePlan.resolvedPath = resolvedTargetPath
			fileResult.Written = true
			fileResult.ResolvedPath = resolvedTargetPath
		}

		itemResult.Status = deriveCatalogMaterializationItemStatus(itemResult.Files, false)
	}

	return nil
}

func canonicalizePathForWrite(targetPath string) (string, error) {
	if strings.TrimSpace(targetPath) == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	absolutePath, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		return "", err
	}
	absolutePath = filepath.Clean(absolutePath)

	existingPath := absolutePath
	missingSuffix := make([]string, 0, 8)
	for {
		_, statErr := os.Lstat(existingPath)
		if statErr == nil {
			canonicalExistingPath, err := canonicalizeExistingPath(existingPath)
			if err != nil {
				return "", err
			}

			resolvedPath := canonicalExistingPath
			for index := len(missingSuffix) - 1; index >= 0; index-- {
				resolvedPath = filepath.Join(resolvedPath, missingSuffix[index])
			}
			return filepath.Clean(resolvedPath), nil
		}
		if !os.IsNotExist(statErr) {
			return "", statErr
		}

		parentPath := filepath.Dir(existingPath)
		if parentPath == existingPath {
			return "", fmt.Errorf("failed to resolve existing ancestor for %q", targetPath)
		}
		missingSuffix = append(missingSuffix, filepath.Base(existingPath))
		existingPath = parentPath
	}
}
