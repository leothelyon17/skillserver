package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/domain"
	"github.com/mudler/skillserver/pkg/persistence"
)

func TestCatalogRelationshipMetadataEndpoints_GetMetadataIncludesRelationshipsForSkillPromptAndRule(t *testing.T) {
	t.Parallel()

	server, sourceRepo := newCatalogMetadataFixtureServer(t)
	seedCatalogRelationshipSourceRows(t, sourceRepo, "demo-skill")

	skillItemID := domain.BuildSkillCatalogItemID("demo-skill")
	promptItemID := domain.BuildPromptCatalogItemID("demo-skill", "prompts/system.md")
	ruleItemID := domain.BuildRuleCatalogItemID("demo-skill", "rules/security.md")

	patchBody := `{"prompt_item_id":"` + promptItemID + `","rule_item_ids":["` + ruleItemID + `"],"updated_by":"gui"}`
	patchTarget := "/api/catalog/" + url.PathEscape(skillItemID) + "/relationships"
	req := httptest.NewRequest(http.MethodPatch, patchTarget, strings.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected patch status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	metadataTarget := "/api/catalog/" + url.PathEscape(skillItemID) + "/metadata"
	req = httptest.NewRequest(http.MethodGet, metadataTarget, nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected skill metadata status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	skillPayload := decodeJSONObject(t, rec.Body.Bytes())
	skillRelationships := decodeCatalogRelationshipsFromMetadataPayload(t, skillPayload)
	if skillRelationships["prompt"] == nil {
		t.Fatalf("expected skill relationships.prompt to be populated")
	}
	skillPrompt, ok := skillRelationships["prompt"].(map[string]any)
	if !ok {
		t.Fatalf("expected skill relationships.prompt object, got %T", skillRelationships["prompt"])
	}
	if id, _ := skillPrompt["id"].(string); id != promptItemID {
		t.Fatalf("expected skill relationship prompt id %q, got %q", promptItemID, id)
	}
	skillRules := decodeCatalogRelationshipArray(t, skillRelationships["rules"], "skill relationships.rules")
	if len(skillRules) != 1 {
		t.Fatalf("expected one skill rule relationship, got %+v", skillRules)
	}
	if id, _ := skillRules[0]["id"].(string); id != ruleItemID {
		t.Fatalf("expected skill rule relationship id %q, got %q", ruleItemID, id)
	}
	skillReverseSkills := decodeCatalogRelationshipArray(
		t,
		skillRelationships["skills"],
		"skill relationships.skills",
	)
	if len(skillReverseSkills) != 0 {
		t.Fatalf("expected skill relationships.skills to be empty, got %+v", skillReverseSkills)
	}

	queryMetadataTarget := "/api/catalog/metadata?item_id=" + url.QueryEscape(skillItemID)
	req = httptest.NewRequest(http.MethodGet, queryMetadataTarget, nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected query metadata status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	queryPayload := decodeJSONObject(t, rec.Body.Bytes())
	queryRelationships := decodeCatalogRelationshipsFromMetadataPayload(t, queryPayload)
	queryPrompt, ok := queryRelationships["prompt"].(map[string]any)
	if !ok {
		t.Fatalf("expected query metadata prompt object, got %T", queryRelationships["prompt"])
	}
	if id, _ := queryPrompt["id"].(string); id != promptItemID {
		t.Fatalf("expected query metadata prompt id %q, got %q", promptItemID, id)
	}

	promptMetadataTarget := "/api/catalog/" + url.PathEscape(promptItemID) + "/metadata"
	req = httptest.NewRequest(http.MethodGet, promptMetadataTarget, nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected prompt metadata status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	promptPayload := decodeJSONObject(t, rec.Body.Bytes())
	promptRelationships := decodeCatalogRelationshipsFromMetadataPayload(t, promptPayload)
	if promptRelationships["prompt"] != nil {
		t.Fatalf("expected prompt relationships.prompt to be null, got %+v", promptRelationships["prompt"])
	}
	promptRules := decodeCatalogRelationshipArray(t, promptRelationships["rules"], "prompt relationships.rules")
	if len(promptRules) != 0 {
		t.Fatalf("expected prompt relationships.rules to be empty, got %+v", promptRules)
	}
	promptSkills := decodeCatalogRelationshipArray(t, promptRelationships["skills"], "prompt relationships.skills")
	if len(promptSkills) != 1 {
		t.Fatalf("expected prompt reverse skills to include one skill, got %+v", promptSkills)
	}
	if id, _ := promptSkills[0]["id"].(string); id != skillItemID {
		t.Fatalf("expected prompt reverse skill id %q, got %q", skillItemID, id)
	}

	ruleMetadataTarget := "/api/catalog/" + url.PathEscape(ruleItemID) + "/metadata"
	req = httptest.NewRequest(http.MethodGet, ruleMetadataTarget, nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected rule metadata status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	rulePayload := decodeJSONObject(t, rec.Body.Bytes())
	ruleRelationships := decodeCatalogRelationshipsFromMetadataPayload(t, rulePayload)
	if ruleRelationships["prompt"] != nil {
		t.Fatalf("expected rule relationships.prompt to be null, got %+v", ruleRelationships["prompt"])
	}
	ruleRules := decodeCatalogRelationshipArray(t, ruleRelationships["rules"], "rule relationships.rules")
	if len(ruleRules) != 0 {
		t.Fatalf("expected rule relationships.rules to be empty, got %+v", ruleRules)
	}
	ruleSkills := decodeCatalogRelationshipArray(t, ruleRelationships["skills"], "rule relationships.skills")
	if len(ruleSkills) != 1 {
		t.Fatalf("expected rule reverse skills to include one skill, got %+v", ruleSkills)
	}
	if id, _ := ruleSkills[0]["id"].(string); id != skillItemID {
		t.Fatalf("expected rule reverse skill id %q, got %q", skillItemID, id)
	}
}

func TestCatalogRelationshipMetadataEndpoints_SoftDeletedRelationshipTargetsAreSuppressed(t *testing.T) {
	t.Parallel()

	server, sourceRepo := newCatalogMetadataFixtureServer(t)
	seedCatalogRelationshipSourceRows(t, sourceRepo, "demo-skill")

	skillItemID := domain.BuildSkillCatalogItemID("demo-skill")
	promptItemID := domain.BuildPromptCatalogItemID("demo-skill", "prompts/system.md")
	ruleItemID := domain.BuildRuleCatalogItemID("demo-skill", "rules/security.md")

	patchBody := `{"prompt_item_id":"` + promptItemID + `","rule_item_ids":["` + ruleItemID + `"],"updated_by":"gui"}`
	patchTarget := "/api/catalog/" + url.PathEscape(skillItemID) + "/relationships"
	req := httptest.NewRequest(http.MethodPatch, patchTarget, strings.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected relationship setup patch status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	ctx := context.Background()
	tombstoneAt := time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC)
	deleted, err := sourceRepo.SoftDeleteByItemID(ctx, promptItemID, tombstoneAt)
	if err != nil {
		t.Fatalf("expected prompt soft-delete to succeed, got %v", err)
	}
	if !deleted {
		t.Fatalf("expected prompt soft-delete to report deleted=true")
	}
	deleted, err = sourceRepo.SoftDeleteByItemID(ctx, ruleItemID, tombstoneAt)
	if err != nil {
		t.Fatalf("expected rule soft-delete to succeed, got %v", err)
	}
	if !deleted {
		t.Fatalf("expected rule soft-delete to report deleted=true")
	}

	metadataTarget := "/api/catalog/" + url.PathEscape(skillItemID) + "/metadata"
	req = httptest.NewRequest(http.MethodGet, metadataTarget, nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected skill metadata status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	skillPayload := decodeJSONObject(t, rec.Body.Bytes())
	skillRelationships := decodeCatalogRelationshipsFromMetadataPayload(t, skillPayload)
	if skillRelationships["prompt"] != nil {
		t.Fatalf("expected soft-deleted prompt endpoint to be suppressed, got %+v", skillRelationships["prompt"])
	}
	skillRules := decodeCatalogRelationshipArray(t, skillRelationships["rules"], "skill relationships.rules")
	if len(skillRules) != 0 {
		t.Fatalf("expected soft-deleted rule endpoint to be suppressed, got %+v", skillRules)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog/"+url.PathEscape(promptItemID)+"/metadata", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected soft-deleted prompt metadata status %d, got %d body=%q",
			http.StatusNotFound,
			rec.Code,
			rec.Body.String(),
		)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog/"+url.PathEscape(ruleItemID)+"/metadata", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected soft-deleted rule metadata status %d, got %d body=%q",
			http.StatusNotFound,
			rec.Code,
			rec.Body.String(),
		)
	}
}

func TestCatalogRelationshipMetadataEndpoints_PatchSkillSupportsPromptAndRuleReplacement(t *testing.T) {
	t.Parallel()

	server, sourceRepo := newCatalogMetadataFixtureServer(t)
	seedCatalogRelationshipSourceRows(t, sourceRepo, "demo-skill")
	skillItemID := domain.BuildSkillCatalogItemID("demo-skill")
	promptAItemID := domain.BuildPromptCatalogItemID("demo-skill", "prompts/system.md")
	promptBItemID := domain.BuildPromptCatalogItemID("demo-skill", "prompts/fallback.md")
	ruleAItemID := domain.BuildRuleCatalogItemID("demo-skill", "rules/security.md")
	ruleBItemID := domain.BuildRuleCatalogItemID("demo-skill", "rules/style.md")

	firstPatchBody := `{"prompt_item_id":"` + promptAItemID + `","rule_item_ids":["` + ruleAItemID + `"],"updated_by":"gui"}`
	firstPatchTarget := "/api/catalog/" + url.PathEscape(skillItemID) + "/relationships"
	req := httptest.NewRequest(http.MethodPatch, firstPatchTarget, strings.NewReader(firstPatchBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected first patch status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	secondPatchBody := `{"prompt_item_id":"` + promptBItemID + `","rule_item_ids":["` + ruleBItemID + `"],"updated_by":"gui"}`
	req = httptest.NewRequest(http.MethodPatch, firstPatchTarget, strings.NewReader(secondPatchBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected second patch status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	secondPatchPayload := decodeJSONObject(t, rec.Body.Bytes())
	relationships, ok := secondPatchPayload["relationships"].(map[string]any)
	if !ok {
		t.Fatalf("expected relationships object in patch response, got %T", secondPatchPayload["relationships"])
	}
	prompt, ok := relationships["prompt"].(map[string]any)
	if !ok {
		t.Fatalf("expected prompt relationship object in patch response, got %T", relationships["prompt"])
	}
	if id, _ := prompt["id"].(string); id != promptBItemID {
		t.Fatalf("expected replacement prompt id %q, got %q", promptBItemID, id)
	}
	rules := decodeCatalogRelationshipArray(t, relationships["rules"], "patch response relationships.rules")
	if len(rules) != 1 {
		t.Fatalf("expected one replacement rule relationship, got %+v", rules)
	}
	if id, _ := rules[0]["id"].(string); id != ruleBItemID {
		t.Fatalf("expected replacement rule id %q, got %q", ruleBItemID, id)
	}

	metadataTarget := "/api/catalog/" + url.PathEscape(skillItemID) + "/metadata"
	req = httptest.NewRequest(http.MethodGet, metadataTarget, nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected metadata status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	metadataPayload := decodeJSONObject(t, rec.Body.Bytes())
	metadataRelationships := decodeCatalogRelationshipsFromMetadataPayload(t, metadataPayload)
	metadataPrompt, ok := metadataRelationships["prompt"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata prompt object, got %T", metadataRelationships["prompt"])
	}
	if id, _ := metadataPrompt["id"].(string); id != promptBItemID {
		t.Fatalf("expected metadata prompt replacement id %q, got %q", promptBItemID, id)
	}
	metadataRules := decodeCatalogRelationshipArray(t, metadataRelationships["rules"], "metadata relationships.rules")
	if len(metadataRules) != 1 || metadataRules[0]["id"] != ruleBItemID {
		t.Fatalf("expected metadata rule replacement [%q], got %+v", ruleBItemID, metadataRules)
	}
}

func TestCatalogRelationshipMetadataEndpoints_PatchValidationAndAuthorityErrors(t *testing.T) {
	t.Parallel()

	server, sourceRepo := newCatalogMetadataFixtureServer(t)
	seedCatalogRelationshipSourceRows(t, sourceRepo, "demo-skill")
	skillItemID := domain.BuildSkillCatalogItemID("demo-skill")
	promptItemID := domain.BuildPromptCatalogItemID("demo-skill", "prompts/system.md")
	ruleItemID := domain.BuildRuleCatalogItemID("demo-skill", "rules/security.md")
	missingPromptItemID := domain.BuildPromptCatalogItemID("demo-skill", "prompts/missing.md")

	target := "/api/catalog/" + url.PathEscape(skillItemID) + "/relationships"

	req := httptest.NewRequest(
		http.MethodPatch,
		target,
		strings.NewReader(`{"prompt_item_id":"`+ruleItemID+`"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected classifier mismatch status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(
		http.MethodPatch,
		target,
		strings.NewReader(`{"rule_item_ids":["`+ruleItemID+`","`+ruleItemID+`"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected duplicate IDs status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(
		http.MethodPatch,
		target,
		strings.NewReader(`{"prompt_item_id":"`+missingPromptItemID+`"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected unknown item status %d, got %d body=%q", http.StatusNotFound, rec.Code, rec.Body.String())
	}

	nonSkillTarget := "/api/catalog/" + url.PathEscape(promptItemID) + "/relationships"
	req = httptest.NewRequest(
		http.MethodPatch,
		nonSkillTarget,
		strings.NewReader(`{"prompt_item_id":"`+promptItemID+`"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected non-skill write status %d, got %d body=%q", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

func TestCatalogRelationshipMetadataEndpoints_ListAndSearchRemainRelationshipLight(t *testing.T) {
	t.Parallel()

	server, sourceRepo := newCatalogMetadataFixtureServer(t)
	seedCatalogRelationshipSourceRows(t, sourceRepo, "demo-skill")

	skillItemID := domain.BuildSkillCatalogItemID("demo-skill")
	promptItemID := domain.BuildPromptCatalogItemID("demo-skill", "prompts/system.md")
	ruleItemID := domain.BuildRuleCatalogItemID("demo-skill", "rules/security.md")
	target := "/api/catalog/" + url.PathEscape(skillItemID) + "/relationships"
	req := httptest.NewRequest(
		http.MethodPatch,
		target,
		strings.NewReader(`{"prompt_item_id":"`+promptItemID+`","rule_item_ids":["`+ruleItemID+`"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected relationship setup patch status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected list status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	listItems := decodeJSONArray(t, rec.Body.Bytes())
	for _, item := range listItems {
		if _, exists := item["relationships"]; exists {
			t.Fatalf("expected list item payload to exclude relationships field, got %+v", item)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog/search?q=demo", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected search status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	searchItems := decodeJSONArray(t, rec.Body.Bytes())
	for _, item := range searchItems {
		if _, exists := item["relationships"]; exists {
			t.Fatalf("expected search item payload to exclude relationships field, got %+v", item)
		}
	}
}

func seedCatalogRelationshipSourceRows(
	t *testing.T,
	sourceRepo *persistence.CatalogSourceRepository,
	skillID string,
) {
	t.Helper()

	ctx := context.Background()
	syncedAt := time.Date(2026, time.March, 11, 12, 0, 0, 0, time.UTC)
	skillItemID := domain.BuildSkillCatalogItemID(skillID)
	parentSkillID := domain.CanonicalSkillCatalogKey(skillID)
	repoName := "repo-a"

	rows := []persistence.CatalogSourceRow{
		{
			ItemID:           domain.BuildPromptCatalogItemID(skillID, "prompts/system.md"),
			Classifier:       persistence.CatalogClassifierPrompt,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       &repoName,
			ParentSkillID:    catalogMetadataStringPtr(parentSkillID),
			ResourcePath:     catalogMetadataStringPtr("prompts/system.md"),
			Name:             "system.md",
			Description:      "system prompt",
			Content:          "prompt content",
			ContentHash:      "sha256:prompt-system",
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           domain.BuildPromptCatalogItemID(skillID, "prompts/fallback.md"),
			Classifier:       persistence.CatalogClassifierPrompt,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       &repoName,
			ParentSkillID:    catalogMetadataStringPtr(parentSkillID),
			ResourcePath:     catalogMetadataStringPtr("prompts/fallback.md"),
			Name:             "fallback.md",
			Description:      "fallback prompt",
			Content:          "fallback content",
			ContentHash:      "sha256:prompt-fallback",
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           domain.BuildRuleCatalogItemID(skillID, "rules/security.md"),
			Classifier:       persistence.CatalogClassifierRule,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       &repoName,
			ParentSkillID:    catalogMetadataStringPtr(parentSkillID),
			ResourcePath:     catalogMetadataStringPtr("rules/security.md"),
			Name:             "security.md",
			Description:      "security rule",
			Content:          "rule content",
			ContentHash:      "sha256:rule-security",
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           domain.BuildRuleCatalogItemID(skillID, "rules/style.md"),
			Classifier:       persistence.CatalogClassifierRule,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       &repoName,
			ParentSkillID:    catalogMetadataStringPtr(parentSkillID),
			ResourcePath:     catalogMetadataStringPtr("rules/style.md"),
			Name:             "style.md",
			Description:      "style rule",
			Content:          "rule style content",
			ContentHash:      "sha256:rule-style",
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
	}

	for _, row := range rows {
		if err := sourceRepo.Upsert(ctx, row); err != nil {
			t.Fatalf("expected relationship source upsert for %q to succeed, got %v", row.ItemID, err)
		}
	}

	if _, err := sourceRepo.GetByItemID(ctx, skillItemID); err != nil {
		t.Fatalf("expected skill source row %q to exist, got %v", skillItemID, err)
	}
}

func decodeCatalogRelationshipsFromMetadataPayload(
	t *testing.T,
	payload map[string]any,
) map[string]any {
	t.Helper()

	rawRelationships, exists := payload["relationships"]
	if !exists {
		t.Fatalf("expected metadata payload to include relationships field")
	}
	relationships, ok := rawRelationships.(map[string]any)
	if !ok {
		t.Fatalf("expected metadata relationships object, got %T", rawRelationships)
	}
	return relationships
}

func decodeCatalogRelationshipArray(
	t *testing.T,
	raw any,
	fieldName string,
) []map[string]any {
	t.Helper()

	rawItems, ok := raw.([]any)
	if !ok {
		t.Fatalf("expected %s array, got %T", fieldName, raw)
	}

	items := make([]map[string]any, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, ok := rawItem.(map[string]any)
		if !ok {
			t.Fatalf("expected %s entry object, got %T", fieldName, rawItem)
		}
		items = append(items, item)
	}

	return items
}
