package domain

import (
	"reflect"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/persistence"
)

func TestCatalogMetadataService_Get_ExposesExplicitClassificationStateInEffectiveView(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	overlayRepo := newCatalogOverlayRepositoryForDomainTest(t, db)
	domainRepo := newCatalogDomainRepositoryForDomainTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)
	taxonomyAssignmentRepo := newCatalogItemTaxonomyAssignmentRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)

	itemID := BuildSkillCatalogItemID("metadata-item")
	syncedAt := time.Date(2026, time.March, 5, 6, 0, 0, 0, time.UTC)

	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "metadata-item",
		Description:      "metadata fixture item",
		Content:          "metadata fixture content",
		ContentHash:      buildCatalogContentHash("metadata fixture content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})
	if err := domainRepo.Create(ctx, persistence.CatalogDomainRow{
		DomainID: "domain-platform",
		Key:      "platform",
		Name:     "Platform",
		Active:   true,
	}); err != nil {
		t.Fatalf("expected create domain-platform to succeed, got %v", err)
	}
	if err := taxonomyAssignmentRepo.Upsert(ctx, persistence.CatalogItemTaxonomyAssignmentRow{
		ItemID:          itemID,
		PrimaryDomainID: stringPointer("domain-platform"),
		UpdatedAt:       syncedAt,
	}); err != nil {
		t.Fatalf("expected taxonomy assignment upsert to succeed, got %v", err)
	}

	effectiveService, err := NewCatalogEffectiveService(
		sourceRepo,
		overlayRepo,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
		domainRepo,
		subdomainRepo,
		tagRepo,
	)
	if err != nil {
		t.Fatalf("expected effective catalog service creation to succeed, got %v", err)
	}
	service, err := NewCatalogMetadataService(
		sourceRepo,
		overlayRepo,
		effectiveService,
		CatalogMetadataServiceOptions{},
	)
	if err != nil {
		t.Fatalf("expected metadata service creation to succeed, got %v", err)
	}

	view, err := service.Get(ctx, itemID)
	if err != nil {
		t.Fatalf("expected metadata get to succeed, got %v", err)
	}

	if !view.Effective.HasAssignment {
		t.Fatalf("expected effective has_assignment=true, got %+v", view.Effective)
	}
	if view.Effective.IsFullyClassified {
		t.Fatalf("expected effective is_fully_classified=false when tags are absent, got %+v", view.Effective)
	}
	expectedMissingFields := []string{
		CatalogClassificationMissingPrimarySubdomain,
		CatalogClassificationMissingSecondaryDomain,
		CatalogClassificationMissingSecondarySubdomain,
		CatalogClassificationMissingTags,
	}
	if !reflect.DeepEqual(view.Effective.MissingFields, expectedMissingFields) {
		t.Fatalf("expected effective missing_fields %+v, got %+v", expectedMissingFields, view.Effective.MissingFields)
	}
}

func TestCatalogMetadataService_Get_MapsRuleClassifierInSourceView(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	overlayRepo := newCatalogOverlayRepositoryForDomainTest(t, db)
	domainRepo := newCatalogDomainRepositoryForDomainTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)
	taxonomyAssignmentRepo := newCatalogItemTaxonomyAssignmentRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)

	itemID := BuildRuleCatalogItemID("repo-a/planner", "imports/rules/agents.md")
	repoName := "repo-a"
	parentSkillID := "repo-a/planner"
	resourcePath := "imports/rules/agents.md"
	syncedAt := time.Date(2026, time.March, 5, 6, 30, 0, 0, time.UTC)

	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       persistence.CatalogClassifierRule,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &repoName,
		ParentSkillID:    &parentSkillID,
		ResourcePath:     &resourcePath,
		Name:             "agents.md",
		Description:      "rule fixture item",
		Content:          "rule fixture content",
		ContentHash:      buildCatalogContentHash("rule fixture content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})

	effectiveService, err := NewCatalogEffectiveService(
		sourceRepo,
		overlayRepo,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
		domainRepo,
		subdomainRepo,
		tagRepo,
	)
	if err != nil {
		t.Fatalf("expected effective catalog service creation to succeed, got %v", err)
	}
	service, err := NewCatalogMetadataService(
		sourceRepo,
		overlayRepo,
		effectiveService,
		CatalogMetadataServiceOptions{},
	)
	if err != nil {
		t.Fatalf("expected metadata service creation to succeed, got %v", err)
	}

	view, err := service.Get(ctx, itemID)
	if err != nil {
		t.Fatalf("expected metadata get for rule item to succeed, got %v", err)
	}

	if view.Source.Classifier != CatalogClassifierRule {
		t.Fatalf("expected source classifier %q, got %q", CatalogClassifierRule, view.Source.Classifier)
	}
	expectedMissingFields := []string{
		CatalogClassificationMissingPrimaryDomain,
		CatalogClassificationMissingPrimarySubdomain,
		CatalogClassificationMissingSecondaryDomain,
		CatalogClassificationMissingSecondarySubdomain,
		CatalogClassificationMissingTags,
	}
	if !reflect.DeepEqual(view.Effective.MissingFields, expectedMissingFields) {
		t.Fatalf("expected rule effective missing_fields %+v, got %+v", expectedMissingFields, view.Effective.MissingFields)
	}
}

func TestCatalogMetadataService_Get_IncludesRelationshipProjectionWhenConfigured(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	overlayRepo := newCatalogOverlayRepositoryForDomainTest(t, db)
	domainRepo := newCatalogDomainRepositoryForDomainTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)
	taxonomyAssignmentRepo := newCatalogItemTaxonomyAssignmentRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)
	ruleRelationshipRepo := newCatalogSkillRuleRelationshipRepositoryForDomainTest(t, db)
	promptRelationshipRepo := newCatalogSkillPromptRelationshipRepositoryForDomainTest(t, db)

	skillItemID := BuildSkillCatalogItemID("repo-a/metadata-relationship")
	promptItemID := BuildPromptCatalogItemID("repo-a/metadata-relationship", "prompts/system.md")
	ruleItemID := BuildRuleCatalogItemID("repo-a/metadata-relationship", "rules/security.md")
	syncedAt := time.Date(2026, time.March, 7, 15, 0, 0, 0, time.UTC)

	for _, row := range []persistence.CatalogSourceRow{
		{
			ItemID:           skillItemID,
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeLocal,
			Name:             "metadata-relationship",
			Description:      "relationship metadata skill",
			Content:          "skill content",
			ContentHash:      buildCatalogContentHash("skill content"),
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           promptItemID,
			Classifier:       persistence.CatalogClassifierPrompt,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/metadata-relationship"),
			ResourcePath:     stringPointer("prompts/system.md"),
			Name:             "system.md",
			Description:      "metadata relationship prompt",
			Content:          "prompt content",
			ContentHash:      buildCatalogContentHash("prompt content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           ruleItemID,
			Classifier:       persistence.CatalogClassifierRule,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/metadata-relationship"),
			ResourcePath:     stringPointer("rules/security.md"),
			Name:             "security.md",
			Description:      "metadata relationship rule",
			Content:          "rule content",
			ContentHash:      buildCatalogContentHash("rule content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
	} {
		mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, row)
	}
	mustSetCatalogPromptRelationshipForDomainTest(
		t,
		ctx,
		promptRelationshipRepo,
		skillItemID,
		promptItemID,
		syncedAt,
	)
	mustReplaceCatalogRuleRelationshipsForDomainTest(
		t,
		ctx,
		ruleRelationshipRepo,
		skillItemID,
		[]string{ruleItemID},
		syncedAt,
	)

	effectiveService, err := NewCatalogEffectiveService(
		sourceRepo,
		overlayRepo,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
		domainRepo,
		subdomainRepo,
		tagRepo,
	)
	if err != nil {
		t.Fatalf("expected effective catalog service creation to succeed, got %v", err)
	}
	relationshipService, err := NewCatalogRelationshipService(
		sourceRepo,
		ruleRelationshipRepo,
		promptRelationshipRepo,
		CatalogRelationshipServiceOptions{},
	)
	if err != nil {
		t.Fatalf("expected relationship service creation to succeed, got %v", err)
	}
	service, err := NewCatalogMetadataService(
		sourceRepo,
		overlayRepo,
		effectiveService,
		CatalogMetadataServiceOptions{RelationshipService: relationshipService},
	)
	if err != nil {
		t.Fatalf("expected metadata service creation to succeed, got %v", err)
	}

	view, err := service.Get(ctx, skillItemID)
	if err != nil {
		t.Fatalf("expected metadata get to succeed, got %v", err)
	}

	if view.Relationships.Prompt == nil || view.Relationships.Prompt.ID != promptItemID {
		t.Fatalf("expected metadata relationship prompt %q, got %+v", promptItemID, view.Relationships.Prompt)
	}
	if len(view.Relationships.Rules) != 1 || view.Relationships.Rules[0].ID != ruleItemID {
		t.Fatalf("expected metadata relationship rules [%q], got %+v", ruleItemID, view.Relationships.Rules)
	}
	if len(view.Relationships.Skills) != 0 {
		t.Fatalf("expected skill metadata relationships to have no reverse skills, got %+v", view.Relationships.Skills)
	}
}
