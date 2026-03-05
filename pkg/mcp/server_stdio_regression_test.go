package mcp

import (
	"context"
	"fmt"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mudler/skillserver/pkg/domain"
)

func TestMCPServer_StdioRegression(t *testing.T) {
	t.Run("registers legacy and catalog stdio tool set by default", func(t *testing.T) {
		server := NewServer(newFakeSkillManager())
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		tools, err := session.ListTools(context.Background(), nil)
		if err != nil {
			t.Fatalf("list tools failed: %v", err)
		}

		expectedTools := []string{
			"list_skills",
			"read_skill",
			"search_skills",
			"list_catalog",
			"search_catalog",
			"list_taxonomy_domains",
			"list_taxonomy_subdomains",
			"list_taxonomy_tags",
			"get_catalog_item_taxonomy",
			"list_skill_resources",
			"read_skill_resource",
			"get_skill_resource_info",
		}

		registered := make(map[string]struct{}, len(tools.Tools))
		for _, tool := range tools.Tools {
			registered[tool.Name] = struct{}{}
		}

		for _, expected := range expectedTools {
			if _, ok := registered[expected]; !ok {
				t.Fatalf("expected tool %q to be registered", expected)
			}
		}

		for _, writeTool := range taxonomyWriteToolNames() {
			if _, ok := registered[writeTool]; ok {
				t.Fatalf("expected write tool %q to be absent when write gate is disabled", writeTool)
			}
		}
	})

	t.Run("registers taxonomy write tools when enabled", func(t *testing.T) {
		server := NewServer(newFakeSkillManager(), ServerOptions{
			EnableTaxonomyWriteTools: true,
		})
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		tools, err := session.ListTools(context.Background(), nil)
		if err != nil {
			t.Fatalf("list tools failed: %v", err)
		}

		registered := make(map[string]struct{}, len(tools.Tools))
		for _, tool := range tools.Tools {
			registered[tool.Name] = struct{}{}
		}

		for _, writeTool := range taxonomyWriteToolNames() {
			if _, ok := registered[writeTool]; !ok {
				t.Fatalf("expected write tool %q to be registered", writeTool)
			}
		}
	})

	t.Run("invokes list and read tools end-to-end", func(t *testing.T) {
		manager := newFakeSkillManager()
		server := NewServer(manager)
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		listResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_skills",
		})
		if err != nil {
			t.Fatalf("list_skills call failed: %v", err)
		}
		if listResult.IsError {
			t.Fatalf("list_skills returned tool error")
		}

		listStructured, ok := listResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected list_skills structured content map, got %T", listResult.StructuredContent)
		}

		rawSkills, ok := listStructured["skills"].([]any)
		if !ok || len(rawSkills) == 0 {
			t.Fatalf("expected non-empty skills list, got %#v", listStructured["skills"])
		}

		firstSkill, ok := rawSkills[0].(map[string]any)
		if !ok {
			t.Fatalf("expected first skill object, got %T", rawSkills[0])
		}

		skillID, _ := firstSkill["id"].(string)
		if skillID != manager.skill.ID {
			t.Fatalf("expected skill id %q, got %q", manager.skill.ID, skillID)
		}

		readResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name:      "read_skill",
			Arguments: map[string]any{"id": manager.skill.ID},
		})
		if err != nil {
			t.Fatalf("read_skill call failed: %v", err)
		}
		if readResult.IsError {
			t.Fatalf("read_skill returned tool error")
		}

		readStructured, ok := readResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected read_skill structured content map, got %T", readResult.StructuredContent)
		}

		content, _ := readStructured["content"].(string)
		if content != manager.skill.Content {
			t.Fatalf("expected read content %q, got %q", manager.skill.Content, content)
		}
	})

	t.Run("invokes catalog tools end-to-end with classifier filtering", func(t *testing.T) {
		manager := newFakeSkillManager()
		server := NewServer(manager)
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		listResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
		})
		if err != nil {
			t.Fatalf("list_catalog call failed: %v", err)
		}
		if listResult.IsError {
			t.Fatalf("list_catalog returned tool error")
		}

		listStructured, ok := listResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected list_catalog structured content map, got %T", listResult.StructuredContent)
		}

		rawItems, ok := listStructured["items"].([]any)
		if !ok {
			t.Fatalf("expected items array, got %T", listStructured["items"])
		}
		if len(rawItems) != len(manager.catalogItems) {
			t.Fatalf("expected %d catalog items, got %d", len(manager.catalogItems), len(rawItems))
		}

		promptItem := findCatalogItemByClassifier(t, rawItems, string(domain.CatalogClassifierPrompt))
		if parentSkillID, _ := promptItem["parent_skill_id"].(string); parentSkillID != "sample-skill" {
			t.Fatalf("expected prompt parent_skill_id sample-skill, got %q", parentSkillID)
		}
		if resourcePath, _ := promptItem["resource_path"].(string); resourcePath != "imports/prompts/system.md" {
			t.Fatalf("expected prompt resource_path imports/prompts/system.md, got %q", resourcePath)
		}

		filteredResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"classifier": "Prompt",
			},
		})
		if err != nil {
			t.Fatalf("list_catalog with classifier call failed: %v", err)
		}
		if filteredResult.IsError {
			t.Fatalf("list_catalog with classifier returned tool error")
		}

		filteredStructured, ok := filteredResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected filtered list_catalog structured content map, got %T", filteredResult.StructuredContent)
		}

		filteredItems, ok := filteredStructured["items"].([]any)
		if !ok {
			t.Fatalf("expected filtered items array, got %T", filteredStructured["items"])
		}
		if len(filteredItems) != 1 {
			t.Fatalf("expected 1 filtered catalog item, got %d", len(filteredItems))
		}
		filteredItem, ok := filteredItems[0].(map[string]any)
		if !ok {
			t.Fatalf("expected filtered item object, got %T", filteredItems[0])
		}
		filteredClassifier, _ := filteredItem["classifier"].(string)
		if filteredClassifier != string(domain.CatalogClassifierPrompt) {
			t.Fatalf("expected filtered classifier %q, got %q", domain.CatalogClassifierPrompt, filteredClassifier)
		}

		searchResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "search_catalog",
			Arguments: map[string]any{
				"query":      "System Prompt",
				"classifier": "prompt",
			},
		})
		if err != nil {
			t.Fatalf("search_catalog call failed: %v", err)
		}
		if searchResult.IsError {
			t.Fatalf("search_catalog returned tool error")
		}

		searchStructured, ok := searchResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected search_catalog structured content map, got %T", searchResult.StructuredContent)
		}

		rawResults, ok := searchStructured["results"].([]any)
		if !ok {
			t.Fatalf("expected search results array, got %T", searchStructured["results"])
		}
		if len(rawResults) != 1 {
			t.Fatalf("expected 1 search result, got %d", len(rawResults))
		}

		searchPrompt, ok := rawResults[0].(map[string]any)
		if !ok {
			t.Fatalf("expected search result object, got %T", rawResults[0])
		}
		if classifier, _ := searchPrompt["classifier"].(string); classifier != string(domain.CatalogClassifierPrompt) {
			t.Fatalf("expected search result classifier %q, got %q", domain.CatalogClassifierPrompt, classifier)
		}
	})

	t.Run("invokes taxonomy read tools end-to-end", func(t *testing.T) {
		manager := newFakeSkillManager()
		server := NewServer(manager)
		server.SetCatalogTaxonomyRegistryService(newFakeCatalogTaxonomyRegistryService())
		server.SetCatalogTaxonomyAssignmentService(newFakeCatalogTaxonomyAssignmentService())
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		domainsResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_taxonomy_domains",
			Arguments: map[string]any{
				"active": true,
			},
		})
		if err != nil {
			t.Fatalf("list_taxonomy_domains call failed: %v", err)
		}
		if domainsResult.IsError {
			t.Fatalf("list_taxonomy_domains returned tool error")
		}

		domainsStructured, ok := domainsResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected list_taxonomy_domains structured content map, got %T", domainsResult.StructuredContent)
		}
		rawDomains, ok := domainsStructured["domains"].([]any)
		if !ok {
			t.Fatalf("expected domains array, got %T", domainsStructured["domains"])
		}
		if len(rawDomains) != 2 {
			t.Fatalf("expected 2 active taxonomy domains, got %d", len(rawDomains))
		}

		subdomainsResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_taxonomy_subdomains",
			Arguments: map[string]any{
				"domain_id": "domain-observability",
			},
		})
		if err != nil {
			t.Fatalf("list_taxonomy_subdomains call failed: %v", err)
		}
		if subdomainsResult.IsError {
			t.Fatalf("list_taxonomy_subdomains returned tool error")
		}

		subdomainsStructured, ok := subdomainsResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected list_taxonomy_subdomains structured content map, got %T", subdomainsResult.StructuredContent)
		}
		rawSubdomains, ok := subdomainsStructured["subdomains"].([]any)
		if !ok {
			t.Fatalf("expected subdomains array, got %T", subdomainsStructured["subdomains"])
		}
		if len(rawSubdomains) != 1 {
			t.Fatalf("expected 1 subdomain for domain-observability, got %d", len(rawSubdomains))
		}

		tagsResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_taxonomy_tags",
			Arguments: map[string]any{
				"tag_ids": []string{"tag-metrics"},
			},
		})
		if err != nil {
			t.Fatalf("list_taxonomy_tags call failed: %v", err)
		}
		if tagsResult.IsError {
			t.Fatalf("list_taxonomy_tags returned tool error")
		}

		tagsStructured, ok := tagsResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected list_taxonomy_tags structured content map, got %T", tagsResult.StructuredContent)
		}
		rawTags, ok := tagsStructured["tags"].([]any)
		if !ok {
			t.Fatalf("expected tags array, got %T", tagsStructured["tags"])
		}
		if len(rawTags) != 1 {
			t.Fatalf("expected 1 tag match for tag-metrics filter, got %d", len(rawTags))
		}

		taxonomyResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "get_catalog_item_taxonomy",
			Arguments: map[string]any{
				"item_id": "prompt:sample-skill:imports/prompts/system.md",
			},
		})
		if err != nil {
			t.Fatalf("get_catalog_item_taxonomy call failed: %v", err)
		}
		if taxonomyResult.IsError {
			t.Fatalf("get_catalog_item_taxonomy returned tool error")
		}

		taxonomyStructured, ok := taxonomyResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected get_catalog_item_taxonomy structured content map, got %T", taxonomyResult.StructuredContent)
		}
		primaryDomain, ok := taxonomyStructured["primary_domain"].(map[string]any)
		if !ok {
			t.Fatalf("expected primary_domain object, got %T", taxonomyStructured["primary_domain"])
		}
		primaryDomainID, _ := primaryDomain["id"].(string)
		if primaryDomainID != "domain-observability" {
			t.Fatalf("expected primary_domain.id domain-observability, got %q", primaryDomainID)
		}
		assignmentTags, ok := taxonomyStructured["tags"].([]any)
		if !ok {
			t.Fatalf("expected tags array in assignment view, got %T", taxonomyStructured["tags"])
		}
		if len(assignmentTags) != 2 {
			t.Fatalf("expected 2 tags in assignment view, got %d", len(assignmentTags))
		}
	})

	t.Run("applies taxonomy filters on catalog tools when effective catalog service is configured", func(t *testing.T) {
		manager := newFakeSkillManager()
		server := NewServer(manager)
		server.SetCatalogMetadataService(newFakeCatalogMetadataService(manager.catalogItems))
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		listFilteredResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"primary_domain_id": "domain-observability",
			},
		})
		if err != nil {
			t.Fatalf("list_catalog taxonomy-filtered call failed: %v", err)
		}
		if listFilteredResult.IsError {
			t.Fatalf("list_catalog taxonomy-filtered call returned tool error")
		}

		listStructured, ok := listFilteredResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected list_catalog taxonomy-filtered structured content map, got %T", listFilteredResult.StructuredContent)
		}
		filteredItems, ok := listStructured["items"].([]any)
		if !ok {
			t.Fatalf("expected filtered items array, got %T", listStructured["items"])
		}
		if len(filteredItems) != 1 {
			t.Fatalf("expected 1 item matching primary_domain_id filter, got %d", len(filteredItems))
		}

		secondaryDomainResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"secondary_domain_id": "domain-platform",
			},
		})
		if err != nil {
			t.Fatalf("list_catalog secondary-domain filtered call failed: %v", err)
		}
		if secondaryDomainResult.IsError {
			t.Fatalf("list_catalog secondary-domain filtered call returned tool error")
		}
		secondaryDomainStructured, ok := secondaryDomainResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf(
				"expected list_catalog secondary-domain structured content map, got %T",
				secondaryDomainResult.StructuredContent,
			)
		}
		secondaryDomainItems, ok := secondaryDomainStructured["items"].([]any)
		if !ok {
			t.Fatalf("expected secondary-domain filtered items array, got %T", secondaryDomainStructured["items"])
		}
		if len(secondaryDomainItems) != 1 {
			t.Fatalf("expected 1 item matching secondary_domain_id filter, got %d", len(secondaryDomainItems))
		}

		subdomainResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"subdomain_id": "subdomain-platform-api",
			},
		})
		if err != nil {
			t.Fatalf("list_catalog subdomain filtered call failed: %v", err)
		}
		if subdomainResult.IsError {
			t.Fatalf("list_catalog subdomain filtered call returned tool error")
		}
		subdomainStructured, ok := subdomainResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf(
				"expected list_catalog subdomain-filtered structured content map, got %T",
				subdomainResult.StructuredContent,
			)
		}
		subdomainItems, ok := subdomainStructured["items"].([]any)
		if !ok {
			t.Fatalf("expected subdomain filtered items array, got %T", subdomainStructured["items"])
		}
		if len(subdomainItems) != 2 {
			t.Fatalf("expected 2 items matching subdomain_id filter across primary/secondary, got %d", len(subdomainItems))
		}

		searchFilteredResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "search_catalog",
			Arguments: map[string]any{
				"query":     "system prompt",
				"tag_ids":   []string{"tag-backend", "tag-metrics"},
				"tag_match": "all",
			},
		})
		if err != nil {
			t.Fatalf("search_catalog taxonomy-filtered call failed: %v", err)
		}
		if searchFilteredResult.IsError {
			t.Fatalf("search_catalog taxonomy-filtered call returned tool error")
		}

		searchStructured, ok := searchFilteredResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected search_catalog taxonomy-filtered structured content map, got %T", searchFilteredResult.StructuredContent)
		}
		filteredResults, ok := searchStructured["results"].([]any)
		if !ok {
			t.Fatalf("expected filtered results array, got %T", searchStructured["results"])
		}
		if len(filteredResults) != 1 {
			t.Fatalf("expected 1 result matching tag_match=all filter, got %d", len(filteredResults))
		}

		listAnyResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"tag_ids": []string{"tag-backend", "tag-metrics"},
			},
		})
		if err != nil {
			t.Fatalf("list_catalog tag-match-any call failed: %v", err)
		}
		if listAnyResult.IsError {
			t.Fatalf("list_catalog tag-match-any call returned tool error")
		}
		listAnyStructured, ok := listAnyResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected list_catalog tag-match-any structured content map, got %T", listAnyResult.StructuredContent)
		}
		listAnyItems, ok := listAnyStructured["items"].([]any)
		if !ok {
			t.Fatalf("expected tag-match-any filtered items array, got %T", listAnyStructured["items"])
		}
		if len(listAnyItems) != 2 {
			t.Fatalf("expected 2 items matching tag_ids with implicit any semantics, got %d", len(listAnyItems))
		}
	})

	t.Run("returns tool errors for invalid catalog inputs", func(t *testing.T) {
		server := NewServer(newFakeSkillManager())
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		invalidListResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"classifier": "skills",
			},
		})
		if err != nil {
			t.Fatalf("list_catalog invalid classifier call failed: %v", err)
		}
		if !invalidListResult.IsError {
			t.Fatalf("expected list_catalog invalid classifier call to return tool error")
		}

		invalidSearchResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "search_catalog",
			Arguments: map[string]any{
				"query":      "sample",
				"classifier": "skills",
			},
		})
		if err != nil {
			t.Fatalf("search_catalog invalid classifier call failed: %v", err)
		}
		if !invalidSearchResult.IsError {
			t.Fatalf("expected search_catalog invalid classifier call to return tool error")
		}

		missingQueryResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "search_catalog",
			Arguments: map[string]any{
				"query": "   ",
			},
		})
		if err != nil {
			t.Fatalf("search_catalog missing query call failed: %v", err)
		}
		if !missingQueryResult.IsError {
			t.Fatalf("expected search_catalog missing query call to return tool error")
		}

		invalidTagMatchResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"tag_match": "every",
			},
		})
		if err != nil {
			t.Fatalf("list_catalog invalid tag_match call failed: %v", err)
		}
		if !invalidTagMatchResult.IsError {
			t.Fatalf("expected list_catalog invalid tag_match call to return tool error")
		}

		unavailableTaxonomyFilterResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"primary_domain_id": "domain-platform",
			},
		})
		if err != nil {
			t.Fatalf("list_catalog taxonomy-filter without metadata service call failed: %v", err)
		}
		if !unavailableTaxonomyFilterResult.IsError {
			t.Fatalf("expected list_catalog taxonomy-filter without metadata service to return tool error")
		}

		missingItemIDResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name:      "get_catalog_item_taxonomy",
			Arguments: map[string]any{"item_id": " "},
		})
		if err != nil {
			t.Fatalf("get_catalog_item_taxonomy missing item_id call failed: %v", err)
		}
		if !missingItemIDResult.IsError {
			t.Fatalf("expected get_catalog_item_taxonomy missing item_id to return tool error")
		}
	})

	t.Run("returns additive resource metadata without breaking legacy fields", func(t *testing.T) {
		manager := newFakeSkillManager()
		server := NewServer(manager)
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		listResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name:      "list_skill_resources",
			Arguments: map[string]any{"skill_id": manager.skill.ID},
		})
		if err != nil {
			t.Fatalf("list_skill_resources call failed: %v", err)
		}
		if listResult.IsError {
			t.Fatalf("list_skill_resources returned tool error")
		}

		listStructured, ok := listResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected list_skill_resources structured content map, got %T", listResult.StructuredContent)
		}

		rawResources, ok := listStructured["resources"].([]any)
		if !ok {
			t.Fatalf("expected resources array, got %T", listStructured["resources"])
		}
		if len(rawResources) != len(manager.resources) {
			t.Fatalf("expected %d resources, got %d", len(manager.resources), len(rawResources))
		}

		hasPromptType := false
		for idx, rawResource := range rawResources {
			resource, ok := rawResource.(map[string]any)
			if !ok {
				t.Fatalf("expected resource[%d] object, got %T", idx, rawResource)
			}

			for _, field := range []string{"type", "path", "name", "size", "mime_type", "readable"} {
				if _, exists := resource[field]; !exists {
					t.Fatalf("expected legacy field %q in resource[%d]: %#v", field, idx, resource)
				}
			}
			for _, field := range []string{"origin", "writable"} {
				if _, exists := resource[field]; !exists {
					t.Fatalf("expected additive field %q in resource[%d]: %#v", field, idx, resource)
				}
			}

			resourceType, _ := resource["type"].(string)
			if resourceType == string(domain.ResourceTypePrompt) {
				hasPromptType = true
			}
		}

		if !hasPromptType {
			t.Fatalf("expected list_skill_resources to include type=%q", domain.ResourceTypePrompt)
		}

		infoResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "get_skill_resource_info",
			Arguments: map[string]any{
				"skill_id":      manager.skill.ID,
				"resource_path": "imports/prompts/system.md",
			},
		})
		if err != nil {
			t.Fatalf("get_skill_resource_info call failed: %v", err)
		}
		if infoResult.IsError {
			t.Fatalf("get_skill_resource_info returned tool error")
		}

		infoStructured, ok := infoResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected get_skill_resource_info structured content map, got %T", infoResult.StructuredContent)
		}

		for _, field := range []string{"exists", "type", "path", "name", "size", "mime_type", "readable", "origin", "writable"} {
			if _, exists := infoStructured[field]; !exists {
				t.Fatalf("expected field %q in get_skill_resource_info output: %#v", field, infoStructured)
			}
		}

		exists, _ := infoStructured["exists"].(bool)
		if !exists {
			t.Fatalf("expected exists=true for known imported resource")
		}

		resourceType, _ := infoStructured["type"].(string)
		if resourceType != string(domain.ResourceTypePrompt) {
			t.Fatalf("expected type=%q, got %q", domain.ResourceTypePrompt, resourceType)
		}

		origin, _ := infoStructured["origin"].(string)
		if origin != string(domain.ResourceOriginImported) {
			t.Fatalf("expected origin=%q, got %q", domain.ResourceOriginImported, origin)
		}

		writable, ok := infoStructured["writable"].(bool)
		if !ok {
			t.Fatalf("expected writable to be bool, got %T", infoStructured["writable"])
		}
		if writable {
			t.Fatalf("expected writable=false for imported resource")
		}

		missingResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "get_skill_resource_info",
			Arguments: map[string]any{
				"skill_id":      manager.skill.ID,
				"resource_path": "imports/prompts/missing.md",
			},
		})
		if err != nil {
			t.Fatalf("get_skill_resource_info missing-resource call failed: %v", err)
		}
		if missingResult.IsError {
			t.Fatalf("get_skill_resource_info missing-resource call returned tool error")
		}

		missingStructured, ok := missingResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected missing-resource structured content map, got %T", missingResult.StructuredContent)
		}

		missingExists, _ := missingStructured["exists"].(bool)
		if missingExists {
			t.Fatalf("expected exists=false for missing resource")
		}
	})
}

func findCatalogItemByClassifier(t *testing.T, items []any, classifier string) map[string]any {
	t.Helper()

	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		value, _ := item["classifier"].(string)
		if value == classifier {
			return item
		}
	}

	t.Fatalf("expected catalog item with classifier %q", classifier)
	return nil
}

func taxonomyWriteToolNames() []string {
	return []string{
		"create_taxonomy_domain",
		"update_taxonomy_domain",
		"delete_taxonomy_domain",
		"create_taxonomy_subdomain",
		"update_taxonomy_subdomain",
		"delete_taxonomy_subdomain",
		"create_taxonomy_tag",
		"update_taxonomy_tag",
		"delete_taxonomy_tag",
		"patch_catalog_item_taxonomy",
	}
}

func connectMCPClientSession(t *testing.T, server *Server) (*mcpsdk.ClientSession, func()) {
	t.Helper()

	ctx := context.Background()
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := server.mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server session: %v", err)
	}

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatalf("failed to connect client session: %v", err)
	}

	cleanup := func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	}
	return clientSession, cleanup
}

type fakeSkillManager struct {
	skill                 domain.Skill
	catalogItems          []domain.CatalogItem
	resources             []domain.SkillResource
	resourceContentByPath map[string]domain.ResourceContent
	resourceInfoByPath    map[string]domain.SkillResource
}

func newFakeSkillManager() *fakeSkillManager {
	resources := []domain.SkillResource{
		{
			Type:     domain.ResourceTypeScript,
			Origin:   domain.ResourceOriginDirect,
			Path:     "scripts/hello.py",
			Name:     "hello.py",
			Size:     14,
			MimeType: "text/plain",
			Readable: true,
			Writable: true,
		},
		{
			Type:     domain.ResourceTypePrompt,
			Origin:   domain.ResourceOriginImported,
			Path:     "imports/prompts/system.md",
			Name:     "system.md",
			Size:     15,
			MimeType: "text/markdown; charset=utf-8",
			Readable: true,
			Writable: false,
		},
	}

	resourceContentByPath := map[string]domain.ResourceContent{
		"scripts/hello.py": {
			Content:  "print('hello')",
			Encoding: "utf-8",
			MimeType: "text/plain",
			Size:     14,
		},
		"imports/prompts/system.md": {
			Content:  "# System Prompt",
			Encoding: "utf-8",
			MimeType: "text/markdown; charset=utf-8",
			Size:     15,
		},
	}

	resourceInfoByPath := make(map[string]domain.SkillResource, len(resources))
	for _, resource := range resources {
		resourceInfoByPath[resource.Path] = resource
	}

	return &fakeSkillManager{
		skill: domain.Skill{
			ID:      "sample-skill",
			Name:    "sample-skill",
			Content: "# Sample Skill\n\nSample skill content.",
			Metadata: &domain.SkillMetadata{
				Name:        "sample-skill",
				Description: "Sample skill used for MCP regression tests",
			},
		},
		catalogItems: []domain.CatalogItem{
			{
				ID:          domain.BuildSkillCatalogItemID("sample-skill"),
				Classifier:  domain.CatalogClassifierSkill,
				Name:        "sample-skill",
				Description: "Sample skill used for MCP regression tests",
				Content:     "# Sample Skill\n\nSample skill content.",
				PrimaryDomain: &domain.CatalogTaxonomyReference{
					ID:   "domain-platform",
					Key:  "platform",
					Name: "Platform",
				},
				PrimarySubdomain: &domain.CatalogTaxonomyReference{
					ID:   "subdomain-platform-api",
					Key:  "api",
					Name: "API",
				},
				Tags: []domain.CatalogTaxonomyReference{
					{ID: "tag-backend", Key: "backend", Name: "Backend"},
				},
				ReadOnly: false,
			},
			{
				ID:            domain.BuildPromptCatalogItemID("sample-skill", "imports/prompts/system.md"),
				Classifier:    domain.CatalogClassifierPrompt,
				Name:          "system.md",
				Description:   "Sample skill used for MCP regression tests",
				Content:       "# System Prompt",
				ParentSkillID: "sample-skill",
				ResourcePath:  "imports/prompts/system.md",
				PrimaryDomain: &domain.CatalogTaxonomyReference{
					ID:   "domain-observability",
					Key:  "observability",
					Name: "Observability",
				},
				PrimarySubdomain: &domain.CatalogTaxonomyReference{
					ID:   "subdomain-observability-metrics",
					Key:  "metrics",
					Name: "Metrics",
				},
				SecondaryDomain: &domain.CatalogTaxonomyReference{
					ID:   "domain-platform",
					Key:  "platform",
					Name: "Platform",
				},
				SecondarySubdomain: &domain.CatalogTaxonomyReference{
					ID:   "subdomain-platform-api",
					Key:  "api",
					Name: "API",
				},
				Tags: []domain.CatalogTaxonomyReference{
					{ID: "tag-backend", Key: "backend", Name: "Backend"},
					{ID: "tag-metrics", Key: "metrics", Name: "Metrics"},
				},
				ReadOnly: true,
			},
		},
		resources:             resources,
		resourceContentByPath: resourceContentByPath,
		resourceInfoByPath:    resourceInfoByPath,
	}
}

func (m *fakeSkillManager) ListSkills() ([]domain.Skill, error) {
	return []domain.Skill{m.skill}, nil
}

func (m *fakeSkillManager) ReadSkill(name string) (*domain.Skill, error) {
	if name != m.skill.ID {
		return nil, fmt.Errorf("skill not found: %s", name)
	}
	skill := m.skill
	return &skill, nil
}

func (m *fakeSkillManager) SearchSkills(query string) ([]domain.Skill, error) {
	if strings.Contains(m.skill.ID, query) ||
		strings.Contains(m.skill.Name, query) ||
		strings.Contains(m.skill.Content, query) {
		return []domain.Skill{m.skill}, nil
	}
	return []domain.Skill{}, nil
}

func (m *fakeSkillManager) RebuildIndex() error {
	return nil
}

func (m *fakeSkillManager) ListCatalogItems() ([]domain.CatalogItem, error) {
	items := make([]domain.CatalogItem, len(m.catalogItems))
	copy(items, m.catalogItems)
	return items, nil
}

func (m *fakeSkillManager) SearchCatalogItems(query string, classifier *domain.CatalogClassifier) ([]domain.CatalogItem, error) {
	items := make([]domain.CatalogItem, 0, len(m.catalogItems))
	for _, item := range m.catalogItems {
		if classifier != nil && item.Classifier != *classifier {
			continue
		}
		if strings.Contains(item.Name, query) ||
			strings.Contains(item.Description, query) ||
			strings.Contains(item.Content, query) ||
			strings.Contains(item.ResourcePath, query) {
			items = append(items, item)
		}
	}
	return items, nil
}

func (m *fakeSkillManager) ListSkillResources(skillID string) ([]domain.SkillResource, error) {
	if skillID != m.skill.ID {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}

	resources := make([]domain.SkillResource, len(m.resources))
	copy(resources, m.resources)
	return resources, nil
}

func (m *fakeSkillManager) ReadSkillResource(skillID, resourcePath string) (*domain.ResourceContent, error) {
	if skillID != m.skill.ID {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}

	content, ok := m.resourceContentByPath[resourcePath]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", resourcePath)
	}

	contentCopy := content
	return &contentCopy, nil
}

func (m *fakeSkillManager) GetSkillResourceInfo(skillID, resourcePath string) (*domain.SkillResource, error) {
	if skillID != m.skill.ID {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}

	resource, ok := m.resourceInfoByPath[resourcePath]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", resourcePath)
	}

	resourceCopy := resource
	return &resourceCopy, nil
}

type fakeCatalogMetadataService struct {
	items []domain.CatalogItem
}

func newFakeCatalogMetadataService(items []domain.CatalogItem) *fakeCatalogMetadataService {
	cloned := make([]domain.CatalogItem, len(items))
	copy(cloned, items)
	return &fakeCatalogMetadataService{items: cloned}
}

func (s *fakeCatalogMetadataService) List(
	ctx context.Context,
	filter domain.CatalogEffectiveListFilter,
) ([]domain.CatalogItem, error) {
	results := make([]domain.CatalogItem, 0, len(s.items))
	for _, item := range s.items {
		if filter.Classifier != nil && item.Classifier != *filter.Classifier {
			continue
		}
		if filter.PrimaryDomainID != "" && (item.PrimaryDomain == nil || item.PrimaryDomain.ID != filter.PrimaryDomainID) {
			continue
		}
		if filter.SecondaryDomainID != "" && (item.SecondaryDomain == nil || item.SecondaryDomain.ID != filter.SecondaryDomainID) {
			continue
		}
		if filter.SubdomainID != "" &&
			((item.PrimarySubdomain == nil || item.PrimarySubdomain.ID != filter.SubdomainID) &&
				(item.SecondarySubdomain == nil || item.SecondarySubdomain.ID != filter.SubdomainID)) {
			continue
		}
		if len(filter.TagIDs) > 0 && !catalogItemMatchesTagFilter(item, filter.TagIDs, filter.TagMatch) {
			continue
		}

		results = append(results, item)
	}

	return results, nil
}

func catalogItemMatchesTagFilter(
	item domain.CatalogItem,
	tagIDs []string,
	tagMatch domain.CatalogTagMatchMode,
) bool {
	if len(tagIDs) == 0 {
		return true
	}
	if len(item.Tags) == 0 {
		return false
	}

	tagSet := make(map[string]struct{}, len(item.Tags))
	for _, tag := range item.Tags {
		tagSet[tag.ID] = struct{}{}
	}

	if tagMatch == domain.CatalogTagMatchAll {
		for _, tagID := range tagIDs {
			if _, exists := tagSet[tagID]; !exists {
				return false
			}
		}
		return true
	}

	for _, tagID := range tagIDs {
		if _, exists := tagSet[tagID]; exists {
			return true
		}
	}
	return false
}

type fakeCatalogTaxonomyRegistryService struct {
	domains    []domain.CatalogTaxonomyDomain
	subdomains []domain.CatalogTaxonomySubdomain
	tags       []domain.CatalogTaxonomyTag
}

func newFakeCatalogTaxonomyRegistryService() *fakeCatalogTaxonomyRegistryService {
	return &fakeCatalogTaxonomyRegistryService{
		domains: []domain.CatalogTaxonomyDomain{
			{DomainID: "domain-platform", Key: "platform", Name: "Platform", Active: true},
			{DomainID: "domain-observability", Key: "observability", Name: "Observability", Active: true},
		},
		subdomains: []domain.CatalogTaxonomySubdomain{
			{
				SubdomainID: "subdomain-platform-api",
				DomainID:    "domain-platform",
				Key:         "api",
				Name:        "API",
				Active:      true,
			},
			{
				SubdomainID: "subdomain-observability-metrics",
				DomainID:    "domain-observability",
				Key:         "metrics",
				Name:        "Metrics",
				Active:      true,
			},
		},
		tags: []domain.CatalogTaxonomyTag{
			{TagID: "tag-backend", Key: "backend", Name: "Backend", Active: true},
			{TagID: "tag-metrics", Key: "metrics", Name: "Metrics", Active: true},
		},
	}
}

func (s *fakeCatalogTaxonomyRegistryService) ListDomains(
	ctx context.Context,
	filter domain.CatalogTaxonomyDomainListFilter,
) ([]domain.CatalogTaxonomyDomain, error) {
	results := make([]domain.CatalogTaxonomyDomain, 0, len(s.domains))
	domainIDSet := toStringSet(filter.DomainIDs)
	keySet := toStringSet(filter.Keys)
	for _, domainRow := range s.domains {
		if filter.DomainID != "" && domainRow.DomainID != filter.DomainID {
			continue
		}
		if len(domainIDSet) > 0 {
			if _, exists := domainIDSet[domainRow.DomainID]; !exists {
				continue
			}
		}
		if filter.Key != "" && domainRow.Key != filter.Key {
			continue
		}
		if len(keySet) > 0 {
			if _, exists := keySet[domainRow.Key]; !exists {
				continue
			}
		}
		if filter.Active != nil && domainRow.Active != *filter.Active {
			continue
		}
		results = append(results, domainRow)
	}
	return results, nil
}

func (s *fakeCatalogTaxonomyRegistryService) ListSubdomains(
	ctx context.Context,
	filter domain.CatalogTaxonomySubdomainListFilter,
) ([]domain.CatalogTaxonomySubdomain, error) {
	results := make([]domain.CatalogTaxonomySubdomain, 0, len(s.subdomains))
	subdomainIDSet := toStringSet(filter.SubdomainIDs)
	domainIDSet := toStringSet(filter.DomainIDs)
	keySet := toStringSet(filter.Keys)
	for _, row := range s.subdomains {
		if filter.SubdomainID != "" && row.SubdomainID != filter.SubdomainID {
			continue
		}
		if len(subdomainIDSet) > 0 {
			if _, exists := subdomainIDSet[row.SubdomainID]; !exists {
				continue
			}
		}
		if filter.DomainID != "" && row.DomainID != filter.DomainID {
			continue
		}
		if len(domainIDSet) > 0 {
			if _, exists := domainIDSet[row.DomainID]; !exists {
				continue
			}
		}
		if filter.Key != "" && row.Key != filter.Key {
			continue
		}
		if len(keySet) > 0 {
			if _, exists := keySet[row.Key]; !exists {
				continue
			}
		}
		if filter.Active != nil && row.Active != *filter.Active {
			continue
		}
		results = append(results, row)
	}
	return results, nil
}

func (s *fakeCatalogTaxonomyRegistryService) ListTags(
	ctx context.Context,
	filter domain.CatalogTaxonomyTagListFilter,
) ([]domain.CatalogTaxonomyTag, error) {
	results := make([]domain.CatalogTaxonomyTag, 0, len(s.tags))
	tagIDSet := toStringSet(filter.TagIDs)
	keySet := toStringSet(filter.Keys)
	for _, row := range s.tags {
		if filter.TagID != "" && row.TagID != filter.TagID {
			continue
		}
		if len(tagIDSet) > 0 {
			if _, exists := tagIDSet[row.TagID]; !exists {
				continue
			}
		}
		if filter.Key != "" && row.Key != filter.Key {
			continue
		}
		if len(keySet) > 0 {
			if _, exists := keySet[row.Key]; !exists {
				continue
			}
		}
		if filter.Active != nil && row.Active != *filter.Active {
			continue
		}
		results = append(results, row)
	}
	return results, nil
}

type fakeCatalogTaxonomyAssignmentService struct {
	byItemID map[string]domain.CatalogItemTaxonomyAssignment
}

func newFakeCatalogTaxonomyAssignmentService() *fakeCatalogTaxonomyAssignmentService {
	return &fakeCatalogTaxonomyAssignmentService{
		byItemID: map[string]domain.CatalogItemTaxonomyAssignment{
			"prompt:sample-skill:imports/prompts/system.md": {
				ItemID: "prompt:sample-skill:imports/prompts/system.md",
				PrimaryDomain: &domain.CatalogTaxonomyReference{
					ID:   "domain-observability",
					Key:  "observability",
					Name: "Observability",
				},
				PrimarySubdomain: &domain.CatalogTaxonomyReference{
					ID:   "subdomain-observability-metrics",
					Key:  "metrics",
					Name: "Metrics",
				},
				SecondaryDomain: &domain.CatalogTaxonomyReference{
					ID:   "domain-platform",
					Key:  "platform",
					Name: "Platform",
				},
				SecondarySubdomain: &domain.CatalogTaxonomyReference{
					ID:   "subdomain-platform-api",
					Key:  "api",
					Name: "API",
				},
				Tags: []domain.CatalogTaxonomyReference{
					{ID: "tag-backend", Key: "backend", Name: "Backend"},
					{ID: "tag-metrics", Key: "metrics", Name: "Metrics"},
				},
			},
		},
	}
}

func (s *fakeCatalogTaxonomyAssignmentService) Get(
	ctx context.Context,
	itemID string,
) (domain.CatalogItemTaxonomyAssignment, error) {
	assignment, ok := s.byItemID[itemID]
	if !ok {
		return domain.CatalogItemTaxonomyAssignment{}, fmt.Errorf(
			"%w: item_id=%q",
			domain.ErrCatalogTaxonomyAssignmentItemNotFound,
			itemID,
		)
	}
	return assignment, nil
}

func toStringSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	return set
}
