package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
			"read_catalog_item",
			"export_catalog_items",
			"list_taxonomy_domains",
			"list_taxonomy_subdomains",
			"list_taxonomy_tags",
			"get_catalog_item_taxonomy",
			"get_catalog_item_relationships",
			"get_taxonomy_domain_usage",
			"get_taxonomy_subdomain_usage",
			"get_taxonomy_tag_usage",
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
		for _, writeTool := range materializationWriteToolNames() {
			if _, ok := registered[writeTool]; ok {
				t.Fatalf("expected materialization tool %q to be absent when materialization gate is disabled", writeTool)
			}
		}
		for _, writeTool := range relationshipWriteToolNames() {
			if _, ok := registered[writeTool]; ok {
				t.Fatalf("expected relationship write tool %q to remain unavailable", writeTool)
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
		for _, writeTool := range materializationWriteToolNames() {
			if _, ok := registered[writeTool]; ok {
				t.Fatalf("expected materialization tool %q to remain gated when materialization capability is disabled", writeTool)
			}
		}
		for _, writeTool := range relationshipWriteToolNames() {
			if _, ok := registered[writeTool]; ok {
				t.Fatalf("expected relationship write tool %q to remain unavailable", writeTool)
			}
		}
	})

	t.Run("registers materialization write tools only when enabled", func(t *testing.T) {
		disabledServer := NewServer(newFakeSkillManager())
		disabledSession, disabledCleanup := connectMCPClientSession(t, disabledServer)
		defer disabledCleanup()

		disabledTools, err := disabledSession.ListTools(context.Background(), nil)
		if err != nil {
			t.Fatalf("list tools failed: %v", err)
		}
		disabledRegistered := make(map[string]struct{}, len(disabledTools.Tools))
		for _, tool := range disabledTools.Tools {
			disabledRegistered[tool.Name] = struct{}{}
		}
		for _, writeTool := range materializationWriteToolNames() {
			if _, ok := disabledRegistered[writeTool]; ok {
				t.Fatalf("expected materialization tool %q to be absent when gate is disabled", writeTool)
			}
		}

		enabledServer := NewServer(newFakeSkillManager(), ServerOptions{
			EnableMaterializationTools:             true,
			AllowedMaterializationDestinationRoots: []string{"/workspace"},
		})
		enabledSession, enabledCleanup := connectMCPClientSession(t, enabledServer)
		defer enabledCleanup()

		enabledTools, err := enabledSession.ListTools(context.Background(), nil)
		if err != nil {
			t.Fatalf("list tools failed: %v", err)
		}
		enabledRegistered := make(map[string]struct{}, len(enabledTools.Tools))
		for _, tool := range enabledTools.Tools {
			enabledRegistered[tool.Name] = struct{}{}
		}
		for _, writeTool := range materializationWriteToolNames() {
			if _, ok := enabledRegistered[writeTool]; !ok {
				t.Fatalf("expected materialization tool %q to be registered when gate is enabled", writeTool)
			}
		}
		for _, writeTool := range relationshipWriteToolNames() {
			if _, ok := enabledRegistered[writeTool]; ok {
				t.Fatalf("expected relationship write tool %q to remain unavailable", writeTool)
			}
		}
	})

	t.Run("materialization capability gate defaults to disabled and requires explicit enablement", func(t *testing.T) {
		disabledServer := NewServer(newFakeSkillManager())
		if disabledServer.MaterializationToolsEnabled() {
			t.Fatalf("expected materialization tools disabled by default")
		}
		if len(disabledServer.AllowedMaterializationDestinationRoots()) != 0 {
			t.Fatalf(
				"expected no default materialization destination roots, got %v",
				disabledServer.AllowedMaterializationDestinationRoots(),
			)
		}

		enabledServer := NewServer(newFakeSkillManager(), ServerOptions{
			EnableMaterializationTools:             true,
			AllowedMaterializationDestinationRoots: []string{"/workspace", "/projects"},
		})
		if !enabledServer.MaterializationToolsEnabled() {
			t.Fatalf("expected materialization tools enabled when explicitly configured")
		}
		roots := enabledServer.AllowedMaterializationDestinationRoots()
		if len(roots) != 2 || roots[0] != "/workspace" || roots[1] != "/projects" {
			t.Fatalf("expected configured materialization destination roots, got %v", roots)
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
		expectedSkillID := domain.BuildSkillCatalogItemID(manager.skill.ID)
		if skillID != expectedSkillID {
			t.Fatalf("expected skill id %q, got %q", expectedSkillID, skillID)
		}
		if name, _ := firstSkill["name"].(string); name != manager.skill.Name {
			t.Fatalf("expected populated skill name %q, got %q", manager.skill.Name, name)
		}

		readResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name:      "read_skill",
			Arguments: map[string]any{"id": skillID},
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

		searchResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name:      "search_skills",
			Arguments: map[string]any{"query": "sample"},
		})
		if err != nil {
			t.Fatalf("search_skills call failed: %v", err)
		}
		if searchResult.IsError {
			t.Fatalf("search_skills returned tool error")
		}

		searchStructured, ok := searchResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected search_skills structured content map, got %T", searchResult.StructuredContent)
		}
		searchResults, ok := searchStructured["results"].([]any)
		if !ok || len(searchResults) != 1 {
			t.Fatalf("expected one search_skills result, got %#v", searchStructured["results"])
		}
		firstSearchResult, ok := searchResults[0].(map[string]any)
		if !ok {
			t.Fatalf("expected search_skills result object, got %T", searchResults[0])
		}
		if id, _ := firstSearchResult["id"].(string); id != expectedSkillID {
			t.Fatalf("expected canonical search_skills id %q, got %q", expectedSkillID, id)
		}
		if name, _ := firstSearchResult["name"].(string); name != manager.skill.Name {
			t.Fatalf("expected populated search_skills name %q, got %q", manager.skill.Name, name)
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
		if hasMore, ok := listStructured["has_more"].(bool); !ok || hasMore {
			t.Fatalf("expected has_more=false for small default list, got %v", listStructured["has_more"])
		}

		promptItem := findCatalogItemByClassifier(t, rawItems, string(domain.CatalogClassifierPrompt))
		if parentSkillID, _ := promptItem["parent_skill_id"].(string); parentSkillID != "sample-skill" {
			t.Fatalf("expected prompt parent_skill_id sample-skill, got %q", parentSkillID)
		}
		if resourcePath, _ := promptItem["resource_path"].(string); resourcePath != "imports/prompts/system.md" {
			t.Fatalf("expected prompt resource_path imports/prompts/system.md, got %q", resourcePath)
		}
		if _, exists := promptItem["content"]; exists {
			t.Fatalf("did not expect content in metadata-first default response, got %+v", promptItem)
		}
		if hasAssignment, ok := promptItem["has_assignment"].(bool); !ok || !hasAssignment {
			t.Fatalf("expected explicit has_assignment=true on prompt item, got %v", promptItem["has_assignment"])
		}
		if _, ok := promptItem["missing_fields"].([]any); !ok {
			t.Fatalf("expected explicit missing_fields array on prompt item, got %T", promptItem["missing_fields"])
		}

		filteredResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"classifier":      "Prompt",
				"include_content": true,
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
		if _, exists := filteredItem["content"]; !exists {
			t.Fatalf("expected content when include_content=true, got %+v", filteredItem)
		}

		ruleFilteredResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"classifier": "rule",
			},
		})
		if err != nil {
			t.Fatalf("list_catalog with rule classifier call failed: %v", err)
		}
		if ruleFilteredResult.IsError {
			t.Fatalf("list_catalog with rule classifier returned tool error")
		}
		ruleStructured, ok := ruleFilteredResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected rule-filtered list_catalog structured content map, got %T", ruleFilteredResult.StructuredContent)
		}
		ruleItems, ok := ruleStructured["items"].([]any)
		if !ok {
			t.Fatalf("expected rule-filtered items array, got %T", ruleStructured["items"])
		}
		if len(ruleItems) != 1 {
			t.Fatalf("expected 1 rule-filtered item, got %d", len(ruleItems))
		}
		ruleItem, ok := ruleItems[0].(map[string]any)
		if !ok {
			t.Fatalf("expected rule-filtered item object, got %T", ruleItems[0])
		}
		if classifier, _ := ruleItem["classifier"].(string); classifier != string(domain.CatalogClassifierRule) {
			t.Fatalf("expected rule-filtered classifier %q, got %q", domain.CatalogClassifierRule, classifier)
		}
		if resourcePath, _ := ruleItem["resource_path"].(string); resourcePath != "imports/rules/agents.md" {
			t.Fatalf("expected rule resource_path imports/rules/agents.md, got %q", resourcePath)
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
		if _, exists := searchPrompt["content"]; exists {
			t.Fatalf("did not expect search result content without include_content=true, got %+v", searchPrompt)
		}
	})

	t.Run("invokes exact-id catalog read tool end-to-end", func(t *testing.T) {
		manager := newFakeSkillManager()
		server := NewServer(manager)
		server.SetCatalogMetadataService(newFakeCatalogMetadataService(manager.catalogItems))
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		skillResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "read_catalog_item",
			Arguments: map[string]any{
				"item_id": "sample-skill",
			},
		})
		if err != nil {
			t.Fatalf("read_catalog_item bare-skill call failed: %v", err)
		}
		if skillResult.IsError {
			t.Fatalf("read_catalog_item bare-skill call returned tool error: %s", toolResultErrorText(skillResult))
		}

		skillStructured, ok := skillResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected read_catalog_item skill structured content map, got %T", skillResult.StructuredContent)
		}
		if itemID, _ := skillStructured["id"].(string); itemID != domain.BuildSkillCatalogItemID("sample-skill") {
			t.Fatalf("expected canonical skill id %q, got %q", domain.BuildSkillCatalogItemID("sample-skill"), itemID)
		}
		if content, _ := skillStructured["content"].(string); !strings.Contains(content, "Sample skill content") {
			t.Fatalf("expected skill content in read_catalog_item output, got %q", content)
		}

		promptItemID := domain.BuildPromptCatalogItemID("sample-skill", "imports/prompts/system.md")
		promptResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "read_catalog_item",
			Arguments: map[string]any{
				"item_id": promptItemID,
			},
		})
		if err != nil {
			t.Fatalf("read_catalog_item prompt call failed: %v", err)
		}
		if promptResult.IsError {
			t.Fatalf("read_catalog_item prompt call returned tool error: %s", toolResultErrorText(promptResult))
		}

		promptStructured, ok := promptResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected read_catalog_item prompt structured content map, got %T", promptResult.StructuredContent)
		}
		if classifier, _ := promptStructured["classifier"].(string); classifier != string(domain.CatalogClassifierPrompt) {
			t.Fatalf("expected prompt classifier %q, got %q", domain.CatalogClassifierPrompt, classifier)
		}
		if resourcePath, _ := promptStructured["resource_path"].(string); resourcePath != "imports/prompts/system.md" {
			t.Fatalf("expected prompt resource_path imports/prompts/system.md, got %q", resourcePath)
		}
		if content, _ := promptStructured["content"].(string); content != "# System Prompt" {
			t.Fatalf("expected prompt content %q, got %q", "# System Prompt", content)
		}

		ruleItemID := domain.BuildRuleCatalogItemID("sample-skill", "imports/rules/agents.md")
		ruleResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "read_catalog_item",
			Arguments: map[string]any{
				"item_id": ruleItemID,
			},
		})
		if err != nil {
			t.Fatalf("read_catalog_item rule call failed: %v", err)
		}
		if ruleResult.IsError {
			t.Fatalf("read_catalog_item rule call returned tool error: %s", toolResultErrorText(ruleResult))
		}

		ruleStructured, ok := ruleResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected read_catalog_item rule structured content map, got %T", ruleResult.StructuredContent)
		}
		if classifier, _ := ruleStructured["classifier"].(string); classifier != string(domain.CatalogClassifierRule) {
			t.Fatalf("expected rule classifier %q, got %q", domain.CatalogClassifierRule, classifier)
		}
		if content, _ := ruleStructured["content"].(string); content != "# AGENTS\nFollow project rules." {
			t.Fatalf("expected rule content %q, got %q", "# AGENTS\nFollow project rules.", content)
		}
	})

	t.Run("invokes export and materialization tools with dry-run planning and explicit failures", func(t *testing.T) {
		manager := newFakeSkillManager()
		allowedRoot := t.TempDir()
		server := NewServer(manager, ServerOptions{
			EnableMaterializationTools:             true,
			AllowedMaterializationDestinationRoots: []string{allowedRoot},
		})
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		promptItemID := domain.BuildPromptCatalogItemID("sample-skill", "imports/prompts/system.md")
		ruleItemID := domain.BuildRuleCatalogItemID("sample-skill", "imports/rules/agents.md")

		exportDryRunResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "export_catalog_items",
			Arguments: map[string]any{
				"item_ids": []string{promptItemID, ruleItemID},
				"dry_run":  true,
			},
		})
		if err != nil {
			t.Fatalf("export_catalog_items dry-run call failed: %v", err)
		}
		if exportDryRunResult.IsError {
			t.Fatalf("export_catalog_items dry-run returned tool error: %s", toolResultErrorText(exportDryRunResult))
		}

		exportDryRunStructured, ok := exportDryRunResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected export dry-run structured content map, got %T", exportDryRunResult.StructuredContent)
		}
		if dryRun, _ := exportDryRunStructured["dry_run"].(bool); !dryRun {
			t.Fatalf("expected export dry_run=true, got %v", exportDryRunStructured["dry_run"])
		}
		manifest, ok := exportDryRunStructured["manifest"].(map[string]any)
		if !ok {
			t.Fatalf("expected export manifest object, got %T", exportDryRunStructured["manifest"])
		}
		manifestItems, ok := manifest["items"].([]any)
		if !ok || len(manifestItems) != 2 {
			t.Fatalf("expected 2 export manifest items, got %v", manifest["items"])
		}
		for _, rawManifestItem := range manifestItems {
			item, ok := rawManifestItem.(map[string]any)
			if !ok {
				t.Fatalf("expected export manifest item object, got %T", rawManifestItem)
			}
			if archiveRoot, _ := item["archive_root"].(string); strings.TrimSpace(archiveRoot) == "" {
				t.Fatalf("expected manifest archive_root to be populated, got %v", item)
			}
			if archiveRoot, _ := item["archive_root"].(string); strings.HasPrefix(archiveRoot, "prompts/") {
				t.Fatalf("expected flat archive_root to omit synthetic prompts/ wrapper, got %q", archiveRoot)
			}
		}
		if _, hasDownload := exportDryRunStructured["download"]; hasDownload {
			t.Fatalf("expected no download metadata on dry-run export response")
		}

		exportResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "export_catalog_items",
			Arguments: map[string]any{
				"item_ids": []string{promptItemID},
			},
		})
		if err != nil {
			t.Fatalf("export_catalog_items call failed: %v", err)
		}
		if exportResult.IsError {
			t.Fatalf("export_catalog_items returned tool error: %s", toolResultErrorText(exportResult))
		}

		exportStructured, ok := exportResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected export structured content map, got %T", exportResult.StructuredContent)
		}
		download, ok := exportStructured["download"].(map[string]any)
		if !ok {
			t.Fatalf("expected export download metadata object, got %T", exportStructured["download"])
		}
		if _, exists := download["archive_base64"]; exists {
			t.Fatalf("did not expect archive_base64 by default, got %+v", download["archive_base64"])
		}
		if contentType, _ := download["content_type"].(string); contentType != "application/gzip" {
			t.Fatalf("expected content_type application/gzip, got %q", contentType)
		}

		exportWithArchiveResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "export_catalog_items",
			Arguments: map[string]any{
				"item_ids":               []string{promptItemID},
				"archive_root_mode":      "materialized",
				"include_archive_base64": true,
			},
		})
		if err != nil {
			t.Fatalf("export_catalog_items include-archive call failed: %v", err)
		}
		if exportWithArchiveResult.IsError {
			t.Fatalf("export_catalog_items include-archive returned tool error: %s", toolResultErrorText(exportWithArchiveResult))
		}

		exportWithArchiveStructured, ok := exportWithArchiveResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected export include-archive structured content map, got %T", exportWithArchiveResult.StructuredContent)
		}
		manifest, ok = exportWithArchiveStructured["manifest"].(map[string]any)
		if !ok {
			t.Fatalf("expected export manifest object, got %T", exportWithArchiveStructured["manifest"])
		}
		manifestItems, ok = manifest["items"].([]any)
		if !ok || len(manifestItems) != 1 {
			t.Fatalf("expected one manifest item, got %v", manifest["items"])
		}
		manifestItem, ok := manifestItems[0].(map[string]any)
		if !ok {
			t.Fatalf("expected export manifest item object, got %T", manifestItems[0])
		}
		if archiveRoot, _ := manifestItem["archive_root"].(string); archiveRoot != "prompts/system.md" {
			t.Fatalf("expected materialized archive_root prompts/system.md, got %q", archiveRoot)
		}
		download, ok = exportWithArchiveStructured["download"].(map[string]any)
		if !ok {
			t.Fatalf("expected export include-archive download metadata object, got %T", exportWithArchiveStructured["download"])
		}
		if archiveBase64, _ := download["archive_base64"].(string); strings.TrimSpace(archiveBase64) == "" {
			t.Fatalf("expected archive_base64 when explicitly requested, got %v", download["archive_base64"])
		}

		dryRunDestination := filepath.Join(allowedRoot, "workspace")
		materializeDryRunResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "materialize_catalog_items",
			Arguments: map[string]any{
				"item_ids":        []string{promptItemID},
				"destination_dir": dryRunDestination,
				"conflict_policy": "overwrite",
				"dry_run":         true,
			},
		})
		if err != nil {
			t.Fatalf("materialize_catalog_items dry-run call failed: %v", err)
		}
		if materializeDryRunResult.IsError {
			t.Fatalf("materialize_catalog_items dry-run returned tool error: %s", toolResultErrorText(materializeDryRunResult))
		}

		materializeStructured, ok := materializeDryRunResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected materialize dry-run structured content map, got %T", materializeDryRunResult.StructuredContent)
		}
		items, ok := materializeStructured["items"].([]any)
		if !ok || len(items) != 1 {
			t.Fatalf("expected one materialization item result, got %v", materializeStructured["items"])
		}
		firstItem, ok := items[0].(map[string]any)
		if !ok {
			t.Fatalf("expected materialization item object, got %T", items[0])
		}
		files, ok := firstItem["files"].([]any)
		if !ok || len(files) != 1 {
			t.Fatalf("expected one materialization file result, got %v", firstItem["files"])
		}
		fileResult, ok := files[0].(map[string]any)
		if !ok {
			t.Fatalf("expected materialization file object, got %T", files[0])
		}
		resolvedPath, _ := fileResult["resolved_path"].(string)
		if strings.TrimSpace(resolvedPath) == "" {
			t.Fatalf("expected resolved_path to be populated, got %v", fileResult)
		}
		if _, statErr := os.Stat(resolvedPath); !os.IsNotExist(statErr) {
			t.Fatalf("expected no file write during dry-run, path=%q statErr=%v", resolvedPath, statErr)
		}

		invalidPolicyResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "materialize_catalog_items",
			Arguments: map[string]any{
				"item_ids":        []string{promptItemID},
				"destination_dir": dryRunDestination,
				"conflict_policy": "replace",
			},
		})
		if err != nil {
			t.Fatalf("materialize_catalog_items invalid conflict policy call failed: %v", err)
		}
		if !invalidPolicyResult.IsError {
			t.Fatalf("expected materialize_catalog_items invalid conflict policy to return tool error")
		}
		if !strings.Contains(strings.ToLower(toolResultErrorText(invalidPolicyResult)), "conflict policy") {
			t.Fatalf("expected conflict policy error, got %s", toolResultErrorText(invalidPolicyResult))
		}

		outsideDestination := t.TempDir()
		outsideRootResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "materialize_catalog_items",
			Arguments: map[string]any{
				"item_ids":        []string{promptItemID},
				"destination_dir": outsideDestination,
			},
		})
		if err != nil {
			t.Fatalf("materialize_catalog_items outside-root call failed: %v", err)
		}
		if !outsideRootResult.IsError {
			t.Fatalf("expected materialize_catalog_items outside-root call to return tool error")
		}
		if !strings.Contains(strings.ToLower(toolResultErrorText(outsideRootResult)), "outside allowed roots") {
			t.Fatalf("expected outside allowed roots error, got %s", toolResultErrorText(outsideRootResult))
		}

		relativeDestinationResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "materialize_catalog_items",
			Arguments: map[string]any{
				"item_ids":        []string{promptItemID},
				"destination_dir": "relative/path",
			},
		})
		if err != nil {
			t.Fatalf("materialize_catalog_items relative-path call failed: %v", err)
		}
		if !relativeDestinationResult.IsError {
			t.Fatalf("expected materialize_catalog_items relative-path call to return tool error")
		}
		if !strings.Contains(strings.ToLower(toolResultErrorText(relativeDestinationResult)), "destination_dir must be absolute") {
			t.Fatalf("expected absolute destination_dir error, got %s", toolResultErrorText(relativeDestinationResult))
		}

		missingItemResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "materialize_catalog_items",
			Arguments: map[string]any{
				"item_ids":        []string{"rule:sample-skill:imports/rules/missing.md"},
				"destination_dir": dryRunDestination,
				"dry_run":         true,
			},
		})
		if err != nil {
			t.Fatalf("materialize_catalog_items missing item call failed: %v", err)
		}
		if !missingItemResult.IsError {
			t.Fatalf("expected materialize_catalog_items missing item call to return tool error")
		}
		if !strings.Contains(strings.ToLower(toolResultErrorText(missingItemResult)), "item not found") {
			t.Fatalf("expected item-not-found error, got %s", toolResultErrorText(missingItemResult))
		}
	})

	t.Run("invokes taxonomy read tools end-to-end", func(t *testing.T) {
		manager := newFakeSkillManager()
		server := NewServer(manager)
		server.SetCatalogTaxonomyRegistryService(newFakeCatalogTaxonomyRegistryService())
		server.SetCatalogTaxonomyAssignmentService(newFakeCatalogTaxonomyAssignmentService())
		server.SetCatalogTaxonomyUsageService(newFakeCatalogTaxonomyUsageService())
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
		if hasAssignment, ok := taxonomyStructured["has_assignment"].(bool); !ok || !hasAssignment {
			t.Fatalf("expected explicit has_assignment=true, got %v", taxonomyStructured["has_assignment"])
		}
		if isFullyClassified, ok := taxonomyStructured["is_fully_classified"].(bool); !ok || !isFullyClassified {
			t.Fatalf(
				"expected explicit is_fully_classified=true, got %v",
				taxonomyStructured["is_fully_classified"],
			)
		}

		domainUsageResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "get_taxonomy_domain_usage",
			Arguments: map[string]any{
				"domain_id":     "domain-platform",
				"preview_limit": 1,
			},
		})
		if err != nil {
			t.Fatalf("get_taxonomy_domain_usage call failed: %v", err)
		}
		if domainUsageResult.IsError {
			t.Fatalf("get_taxonomy_domain_usage returned tool error")
		}
		domainUsageStructured, ok := domainUsageResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected get_taxonomy_domain_usage structured content map, got %T", domainUsageResult.StructuredContent)
		}
		if assignmentCount, ok := domainUsageStructured["assignment_count"].(float64); !ok || assignmentCount != 2 {
			t.Fatalf("expected assignment_count=2, got %v", domainUsageStructured["assignment_count"])
		}
		if previewItemIDs, ok := domainUsageStructured["preview_item_ids"].([]any); !ok || len(previewItemIDs) != 1 {
			t.Fatalf("expected one preview item id, got %+v", domainUsageStructured["preview_item_ids"])
		}

		subdomainUsageResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "get_taxonomy_subdomain_usage",
			Arguments: map[string]any{
				"subdomain_id":  "subdomain-platform-api",
				"preview_limit": 2,
			},
		})
		if err != nil {
			t.Fatalf("get_taxonomy_subdomain_usage call failed: %v", err)
		}
		if subdomainUsageResult.IsError {
			t.Fatalf("get_taxonomy_subdomain_usage returned tool error")
		}
		subdomainUsageStructured, ok := subdomainUsageResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf(
				"expected get_taxonomy_subdomain_usage structured content map, got %T",
				subdomainUsageResult.StructuredContent,
			)
		}
		if objectID, _ := subdomainUsageStructured["object_id"].(string); objectID != "subdomain-platform-api" {
			t.Fatalf("expected subdomain object_id %q, got %q", "subdomain-platform-api", objectID)
		}
		if previewItemIDs, ok := subdomainUsageStructured["preview_item_ids"].([]any); !ok || len(previewItemIDs) != 2 {
			t.Fatalf("expected two subdomain preview item ids, got %+v", subdomainUsageStructured["preview_item_ids"])
		}

		tagUsageResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "get_taxonomy_tag_usage",
			Arguments: map[string]any{
				"tag_id":        "tag-backend",
				"preview_limit": 1,
			},
		})
		if err != nil {
			t.Fatalf("get_taxonomy_tag_usage call failed: %v", err)
		}
		if tagUsageResult.IsError {
			t.Fatalf("get_taxonomy_tag_usage returned tool error")
		}
		tagUsageStructured, ok := tagUsageResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected get_taxonomy_tag_usage structured content map, got %T", tagUsageResult.StructuredContent)
		}
		if objectType, _ := tagUsageStructured["object_type"].(string); objectType != string(domain.CatalogTaxonomyObjectTag) {
			t.Fatalf("expected tag object_type %q, got %q", domain.CatalogTaxonomyObjectTag, objectType)
		}
		if previewItemIDs, ok := tagUsageStructured["preview_item_ids"].([]any); !ok || len(previewItemIDs) != 1 {
			t.Fatalf("expected one tag preview item id, got %+v", tagUsageStructured["preview_item_ids"])
		}
	})

	t.Run("invokes relationship read tool end-to-end for skill prompt and rule items", func(t *testing.T) {
		manager := newFakeSkillManager()
		server := NewServer(manager)
		server.SetCatalogRelationshipService(newFakeCatalogRelationshipService())
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		skillItemID := domain.BuildSkillCatalogItemID("sample-skill")
		promptItemID := domain.BuildPromptCatalogItemID("sample-skill", "imports/prompts/system.md")
		ruleItemID := domain.BuildRuleCatalogItemID("sample-skill", "imports/rules/agents.md")

		skillResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "get_catalog_item_relationships",
			Arguments: map[string]any{
				"item_id": skillItemID,
			},
		})
		if err != nil {
			t.Fatalf("get_catalog_item_relationships skill call failed: %v", err)
		}
		if skillResult.IsError {
			t.Fatalf("get_catalog_item_relationships skill call returned tool error: %s", toolResultErrorText(skillResult))
		}
		skillStructured, ok := skillResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf(
				"expected get_catalog_item_relationships skill structured content map, got %T",
				skillResult.StructuredContent,
			)
		}
		if itemID, _ := skillStructured["item_id"].(string); itemID != skillItemID {
			t.Fatalf("expected skill relationship item_id %q, got %q", skillItemID, itemID)
		}
		skillRelationships, ok := skillStructured["relationships"].(map[string]any)
		if !ok {
			t.Fatalf("expected relationships object for skill response, got %T", skillStructured["relationships"])
		}
		skillPrompt, ok := skillRelationships["prompt"].(map[string]any)
		if !ok {
			t.Fatalf("expected skill relationships.prompt object, got %T", skillRelationships["prompt"])
		}
		if id, _ := skillPrompt["id"].(string); id != promptItemID {
			t.Fatalf("expected skill relationships.prompt.id %q, got %q", promptItemID, id)
		}
		skillRules, ok := skillRelationships["rules"].([]any)
		if !ok {
			t.Fatalf("expected skill relationships.rules array, got %T", skillRelationships["rules"])
		}
		if len(skillRules) != 1 {
			t.Fatalf("expected 1 skill relationship rule, got %d", len(skillRules))
		}
		skillRule, ok := skillRules[0].(map[string]any)
		if !ok {
			t.Fatalf("expected skill relationship rule object, got %T", skillRules[0])
		}
		if id, _ := skillRule["id"].(string); id != ruleItemID {
			t.Fatalf("expected skill relationships.rules[0].id %q, got %q", ruleItemID, id)
		}
		skillReverseSkills, ok := skillRelationships["skills"].([]any)
		if !ok {
			t.Fatalf("expected skill relationships.skills array, got %T", skillRelationships["skills"])
		}
		if len(skillReverseSkills) != 0 {
			t.Fatalf("expected no reverse skills on skill view, got %+v", skillReverseSkills)
		}

		promptResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "get_catalog_item_relationships",
			Arguments: map[string]any{
				"item_id": promptItemID,
			},
		})
		if err != nil {
			t.Fatalf("get_catalog_item_relationships prompt call failed: %v", err)
		}
		if promptResult.IsError {
			t.Fatalf("get_catalog_item_relationships prompt call returned tool error: %s", toolResultErrorText(promptResult))
		}
		promptStructured, ok := promptResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf(
				"expected get_catalog_item_relationships prompt structured content map, got %T",
				promptResult.StructuredContent,
			)
		}
		promptRelationships, ok := promptStructured["relationships"].(map[string]any)
		if !ok {
			t.Fatalf("expected relationships object for prompt response, got %T", promptStructured["relationships"])
		}
		if promptRelationships["prompt"] != nil {
			t.Fatalf("expected prompt relationships.prompt to be null, got %+v", promptRelationships["prompt"])
		}
		promptRules, ok := promptRelationships["rules"].([]any)
		if !ok || len(promptRules) != 0 {
			t.Fatalf("expected prompt relationships.rules to be empty, got %+v", promptRelationships["rules"])
		}
		promptSkills, ok := promptRelationships["skills"].([]any)
		if !ok || len(promptSkills) != 1 {
			t.Fatalf("expected prompt relationships.skills to contain one reverse skill, got %+v", promptRelationships["skills"])
		}
		promptSkill, ok := promptSkills[0].(map[string]any)
		if !ok {
			t.Fatalf("expected prompt reverse skill object, got %T", promptSkills[0])
		}
		if id, _ := promptSkill["id"].(string); id != skillItemID {
			t.Fatalf("expected prompt reverse skill id %q, got %q", skillItemID, id)
		}

		ruleResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "get_catalog_item_relationships",
			Arguments: map[string]any{
				"item_id": ruleItemID,
			},
		})
		if err != nil {
			t.Fatalf("get_catalog_item_relationships rule call failed: %v", err)
		}
		if ruleResult.IsError {
			t.Fatalf("get_catalog_item_relationships rule call returned tool error: %s", toolResultErrorText(ruleResult))
		}
		ruleStructured, ok := ruleResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf(
				"expected get_catalog_item_relationships rule structured content map, got %T",
				ruleResult.StructuredContent,
			)
		}
		ruleRelationships, ok := ruleStructured["relationships"].(map[string]any)
		if !ok {
			t.Fatalf("expected relationships object for rule response, got %T", ruleStructured["relationships"])
		}
		if ruleRelationships["prompt"] != nil {
			t.Fatalf("expected rule relationships.prompt to be null, got %+v", ruleRelationships["prompt"])
		}
		ruleRules, ok := ruleRelationships["rules"].([]any)
		if !ok || len(ruleRules) != 0 {
			t.Fatalf("expected rule relationships.rules to be empty, got %+v", ruleRelationships["rules"])
		}
		ruleSkills, ok := ruleRelationships["skills"].([]any)
		if !ok || len(ruleSkills) != 1 {
			t.Fatalf("expected rule relationships.skills to contain one reverse skill, got %+v", ruleRelationships["skills"])
		}
		ruleSkill, ok := ruleSkills[0].(map[string]any)
		if !ok {
			t.Fatalf("expected rule reverse skill object, got %T", ruleSkills[0])
		}
		if id, _ := ruleSkill["id"].(string); id != skillItemID {
			t.Fatalf("expected rule reverse skill id %q, got %q", skillItemID, id)
		}
	})

	t.Run("invokes taxonomy write tools end-to-end with additive and batch contracts", func(t *testing.T) {
		manager := newFakeSkillManager()
		server := NewServer(manager, ServerOptions{EnableTaxonomyWriteTools: true})
		server.SetCatalogTaxonomyAssignmentService(newFakeCatalogTaxonomyAssignmentService())
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		singlePatchResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "patch_catalog_item_taxonomy",
			Arguments: map[string]any{
				"item_id":     "sample-skill",
				"add_tag_ids": []string{"tag-backend"},
			},
		})
		if err != nil {
			t.Fatalf("patch_catalog_item_taxonomy call failed: %v", err)
		}
		if singlePatchResult.IsError {
			t.Fatalf("patch_catalog_item_taxonomy returned tool error: %s", toolResultErrorText(singlePatchResult))
		}
		singlePatchStructured, ok := singlePatchResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected patch_catalog_item_taxonomy structured content map, got %T", singlePatchResult.StructuredContent)
		}
		tags, ok := singlePatchStructured["tags"].([]any)
		if !ok || len(tags) != 1 {
			t.Fatalf("expected one tag after additive single-item patch, got %+v", singlePatchStructured["tags"])
		}
		if itemID, _ := singlePatchStructured["item_id"].(string); itemID != domain.BuildSkillCatalogItemID("sample-skill") {
			t.Fatalf(
				"expected canonical item_id %q from bare single-item patch, got %q",
				domain.BuildSkillCatalogItemID("sample-skill"),
				itemID,
			)
		}

		getBareSkillResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "get_catalog_item_taxonomy",
			Arguments: map[string]any{
				"item_id": "sample-skill",
			},
		})
		if err != nil {
			t.Fatalf("get_catalog_item_taxonomy bare skill call failed: %v", err)
		}
		if getBareSkillResult.IsError {
			t.Fatalf("get_catalog_item_taxonomy bare skill returned tool error: %s", toolResultErrorText(getBareSkillResult))
		}
		getBareSkillStructured, ok := getBareSkillResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf(
				"expected get_catalog_item_taxonomy bare skill structured content map, got %T",
				getBareSkillResult.StructuredContent,
			)
		}
		if itemID, _ := getBareSkillStructured["item_id"].(string); itemID != domain.BuildSkillCatalogItemID("sample-skill") {
			t.Fatalf(
				"expected canonical item_id %q from bare taxonomy get, got %q",
				domain.BuildSkillCatalogItemID("sample-skill"),
				itemID,
			)
		}

		batchPatchResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "patch_catalog_items_taxonomy",
			Arguments: map[string]any{
				"dry_run": true,
				"items": []map[string]any{
					{
						"item_id":     "sample-skill",
						"add_tag_ids": []string{"tag-metrics"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("patch_catalog_items_taxonomy call failed: %v", err)
		}
		if batchPatchResult.IsError {
			t.Fatalf("patch_catalog_items_taxonomy returned tool error: %s", toolResultErrorText(batchPatchResult))
		}
		batchPatchStructured, ok := batchPatchResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected patch_catalog_items_taxonomy structured content map, got %T", batchPatchResult.StructuredContent)
		}
		if dryRun, ok := batchPatchStructured["dry_run"].(bool); !ok || !dryRun {
			t.Fatalf("expected dry_run=true in batch output, got %v", batchPatchStructured["dry_run"])
		}
		items, ok := batchPatchStructured["items"].([]any)
		if !ok || len(items) != 1 {
			t.Fatalf("expected one batch item result, got %+v", batchPatchStructured["items"])
		}
		item, ok := items[0].(map[string]any)
		if !ok {
			t.Fatalf("expected batch item result object, got %T", items[0])
		}
		if requestedItemID, _ := item["requested_item_id"].(string); requestedItemID != "sample-skill" {
			t.Fatalf("expected requested_item_id to preserve bare skill input, got %q", requestedItemID)
		}
		if itemID, _ := item["item_id"].(string); itemID != domain.BuildSkillCatalogItemID("sample-skill") {
			t.Fatalf(
				"expected canonical item_id %q in dry-run batch output, got %q",
				domain.BuildSkillCatalogItemID("sample-skill"),
				itemID,
			)
		}
		if status, _ := item["status"].(string); status != string(domain.CatalogItemTaxonomyBatchPatchStatusPlanned) {
			t.Fatalf("expected planned batch status, got %+v", item)
		}

		batchApplyResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "patch_catalog_items_taxonomy",
			Arguments: map[string]any{
				"items": []map[string]any{
					{
						"item_id":     "sample-skill",
						"add_tag_ids": []string{"tag-metrics"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("patch_catalog_items_taxonomy apply call failed: %v", err)
		}
		if batchApplyResult.IsError {
			t.Fatalf("patch_catalog_items_taxonomy apply returned tool error: %s", toolResultErrorText(batchApplyResult))
		}
		batchApplyStructured, ok := batchApplyResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf(
				"expected patch_catalog_items_taxonomy apply structured content map, got %T",
				batchApplyResult.StructuredContent,
			)
		}
		if dryRun, ok := batchApplyStructured["dry_run"].(bool); !ok || dryRun {
			t.Fatalf("expected dry_run=false in apply batch output, got %v", batchApplyStructured["dry_run"])
		}
		appliedItems, ok := batchApplyStructured["items"].([]any)
		if !ok || len(appliedItems) != 1 {
			t.Fatalf("expected one apply batch item result, got %+v", batchApplyStructured["items"])
		}
		appliedItem, ok := appliedItems[0].(map[string]any)
		if !ok {
			t.Fatalf("expected apply batch item result object, got %T", appliedItems[0])
		}
		if status, _ := appliedItem["status"].(string); status != string(domain.CatalogItemTaxonomyBatchPatchStatusUpdated) {
			t.Fatalf("expected updated apply batch status, got %+v", appliedItem)
		}
		assignment, ok := appliedItem["assignment"].(map[string]any)
		if !ok {
			t.Fatalf("expected assignment payload on apply batch item, got %+v", appliedItem["assignment"])
		}
		if tags, ok := assignment["tags"].([]any); !ok || len(tags) != 2 {
			t.Fatalf("expected apply batch to persist two tags, got %+v", assignment["tags"])
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

		paginatedResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"limit": 1,
			},
		})
		if err != nil {
			t.Fatalf("list_catalog paginated call failed: %v", err)
		}
		if paginatedResult.IsError {
			t.Fatalf("list_catalog paginated call returned tool error")
		}
		paginatedStructured, ok := paginatedResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected list_catalog paginated structured content map, got %T", paginatedResult.StructuredContent)
		}
		paginatedItems, ok := paginatedStructured["items"].([]any)
		if !ok || len(paginatedItems) != 1 {
			t.Fatalf("expected one paginated item, got %+v", paginatedStructured["items"])
		}
		if hasMore, ok := paginatedStructured["has_more"].(bool); !ok || !hasMore {
			t.Fatalf("expected has_more=true for first paginated page, got %v", paginatedStructured["has_more"])
		}
		nextCursor, ok := paginatedStructured["next_cursor"].(string)
		if !ok || strings.TrimSpace(nextCursor) == "" {
			t.Fatalf("expected next_cursor on paginated response, got %+v", paginatedStructured["next_cursor"])
		}

		unclassifiedResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"unclassified": true,
			},
		})
		if err != nil {
			t.Fatalf("list_catalog unclassified call failed: %v", err)
		}
		if unclassifiedResult.IsError {
			t.Fatalf("list_catalog unclassified call returned tool error")
		}
		unclassifiedStructured, ok := unclassifiedResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected list_catalog unclassified structured content map, got %T", unclassifiedResult.StructuredContent)
		}
		unclassifiedItems, ok := unclassifiedStructured["items"].([]any)
		if !ok || len(unclassifiedItems) != 1 {
			t.Fatalf("expected one unclassified item, got %+v", unclassifiedStructured["items"])
		}
		unclassifiedItem, ok := unclassifiedItems[0].(map[string]any)
		if !ok {
			t.Fatalf("expected unclassified item object, got %T", unclassifiedItems[0])
		}
		if classifier, _ := unclassifiedItem["classifier"].(string); classifier != string(domain.CatalogClassifierRule) {
			t.Fatalf("expected unclassified rule item, got %+v", unclassifiedItem)
		}

		missingPrimaryDomainResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"missing_primary_domain": true,
			},
		})
		if err != nil {
			t.Fatalf("list_catalog missing_primary_domain call failed: %v", err)
		}
		if missingPrimaryDomainResult.IsError {
			t.Fatalf("list_catalog missing_primary_domain call returned tool error")
		}
		missingPrimaryDomainStructured, ok := missingPrimaryDomainResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf(
				"expected list_catalog missing_primary_domain structured content map, got %T",
				missingPrimaryDomainResult.StructuredContent,
			)
		}
		missingPrimaryDomainItems, ok := missingPrimaryDomainStructured["items"].([]any)
		if !ok || len(missingPrimaryDomainItems) != 1 {
			t.Fatalf(
				"expected one missing_primary_domain item, got %+v",
				missingPrimaryDomainStructured["items"],
			)
		}

		missingTagsResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "search_catalog",
			Arguments: map[string]any{
				"query":        "agents",
				"missing_tags": true,
			},
		})
		if err != nil {
			t.Fatalf("search_catalog missing_tags call failed: %v", err)
		}
		if missingTagsResult.IsError {
			t.Fatalf("search_catalog missing_tags call returned tool error")
		}
		missingTagsStructured, ok := missingTagsResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected search_catalog missing_tags structured content map, got %T", missingTagsResult.StructuredContent)
		}
		missingTagsItems, ok := missingTagsStructured["results"].([]any)
		if !ok || len(missingTagsItems) != 1 {
			t.Fatalf("expected one missing_tags search result, got %+v", missingTagsStructured["results"])
		}
	})

	t.Run("returns tool errors for invalid catalog inputs", func(t *testing.T) {
		server := NewServer(newFakeSkillManager())
		server.SetCatalogTaxonomyUsageService(newFakeCatalogTaxonomyUsageService())
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

		missingReadItemIDResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name:      "read_catalog_item",
			Arguments: map[string]any{"item_id": " "},
		})
		if err != nil {
			t.Fatalf("read_catalog_item missing item_id call failed: %v", err)
		}
		if !missingReadItemIDResult.IsError {
			t.Fatalf("expected read_catalog_item missing item_id to return tool error")
		}

		unknownReadItemResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name:      "read_catalog_item",
			Arguments: map[string]any{"item_id": "rule:sample-skill:imports/rules/missing.md"},
		})
		if err != nil {
			t.Fatalf("read_catalog_item missing item call failed: %v", err)
		}
		if !unknownReadItemResult.IsError {
			t.Fatalf("expected read_catalog_item missing item to return tool error")
		}
		if !strings.Contains(strings.ToLower(toolResultErrorText(unknownReadItemResult)), "catalog item not found") {
			t.Fatalf(
				"expected read_catalog_item missing-item error to include catalog item not found, got %s",
				toolResultErrorText(unknownReadItemResult),
			)
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

		missingRelationshipServiceResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name:      "get_catalog_item_relationships",
			Arguments: map[string]any{"item_id": "skill:sample-skill"},
		})
		if err != nil {
			t.Fatalf("get_catalog_item_relationships missing service call failed: %v", err)
		}
		if !missingRelationshipServiceResult.IsError {
			t.Fatalf("expected get_catalog_item_relationships missing service call to return tool error")
		}

		serverWithRelationships := NewServer(newFakeSkillManager())
		serverWithRelationships.SetCatalogRelationshipService(newFakeCatalogRelationshipService())
		relationshipSession, relationshipCleanup := connectMCPClientSession(t, serverWithRelationships)
		defer relationshipCleanup()

		missingRelationshipItemIDResult, err := relationshipSession.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name:      "get_catalog_item_relationships",
			Arguments: map[string]any{"item_id": " "},
		})
		if err != nil {
			t.Fatalf("get_catalog_item_relationships missing item_id call failed: %v", err)
		}
		if !missingRelationshipItemIDResult.IsError {
			t.Fatalf("expected get_catalog_item_relationships missing item_id to return tool error")
		}

		unknownRelationshipItemResult, err := relationshipSession.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name:      "get_catalog_item_relationships",
			Arguments: map[string]any{"item_id": "skill:missing-skill"},
		})
		if err != nil {
			t.Fatalf("get_catalog_item_relationships missing item call failed: %v", err)
		}
		if !unknownRelationshipItemResult.IsError {
			t.Fatalf("expected get_catalog_item_relationships missing item to return tool error")
		}
		if !strings.Contains(strings.ToLower(toolResultErrorText(unknownRelationshipItemResult)), "catalog relationship item not found") {
			t.Fatalf(
				"expected relationship missing-item error to include catalog relationship item not found, got %s",
				toolResultErrorText(unknownRelationshipItemResult),
			)
		}

		invalidLimitResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_catalog",
			Arguments: map[string]any{
				"limit": 0,
			},
		})
		if err != nil {
			t.Fatalf("list_catalog invalid limit call failed: %v", err)
		}
		if !invalidLimitResult.IsError {
			t.Fatalf("expected list_catalog invalid limit to return tool error")
		}

		invalidPreviewLimitResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "get_taxonomy_domain_usage",
			Arguments: map[string]any{
				"domain_id":     "domain-platform",
				"preview_limit": 201,
			},
		})
		if err != nil {
			t.Fatalf("get_taxonomy_domain_usage invalid preview_limit call failed: %v", err)
		}
		if !invalidPreviewLimitResult.IsError {
			t.Fatalf("expected get_taxonomy_domain_usage invalid preview_limit to return tool error")
		}

		invalidSkillResourceResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_skill_resources",
			Arguments: map[string]any{
				"skill_id": "prompt:sample-skill:imports/prompts/system.md",
			},
		})
		if err != nil {
			t.Fatalf("list_skill_resources invalid skill_id call failed: %v", err)
		}
		if !invalidSkillResourceResult.IsError {
			t.Fatalf("expected list_skill_resources invalid skill_id to return tool error")
		}
	})

	t.Run("returns additive resource metadata without breaking legacy fields", func(t *testing.T) {
		manager := newFakeSkillManager()
		server := NewServer(manager)
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		canonicalSkillID := domain.BuildSkillCatalogItemID(manager.skill.ID)

		listResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name:      "list_skill_resources",
			Arguments: map[string]any{"skill_id": canonicalSkillID},
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
				"skill_id":      canonicalSkillID,
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
				"skill_id":      canonicalSkillID,
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

		readResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "read_skill_resource",
			Arguments: map[string]any{
				"skill_id":      canonicalSkillID,
				"resource_path": "imports/prompts/system.md",
			},
		})
		if err != nil {
			t.Fatalf("read_skill_resource call failed: %v", err)
		}
		if readResult.IsError {
			t.Fatalf("read_skill_resource returned tool error")
		}
		readStructured, ok := readResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected read_skill_resource structured content map, got %T", readResult.StructuredContent)
		}
		if content, _ := readStructured["content"].(string); content != "# System Prompt" {
			t.Fatalf("expected canonical skill resource read content, got %q", content)
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
		"patch_catalog_items_taxonomy",
	}
}

func materializationWriteToolNames() []string {
	return []string{
		"materialize_catalog_items",
	}
}

func relationshipWriteToolNames() []string {
	return []string{
		"patch_catalog_item_relationships",
	}
}

func toolResultErrorText(result *mcpsdk.CallToolResult) string {
	if result == nil {
		return ""
	}

	parts := make([]string, 0, len(result.Content))
	for _, content := range result.Content {
		switch typed := content.(type) {
		case *mcpsdk.TextContent:
			parts = append(parts, typed.Text)
		default:
			parts = append(parts, fmt.Sprintf("%v", typed))
		}
	}
	if len(parts) == 0 && result.StructuredContent != nil {
		parts = append(parts, fmt.Sprintf("%v", result.StructuredContent))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
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
				CatalogClassificationState: domain.DeriveCatalogClassificationState(
					&domain.CatalogTaxonomyReference{
						ID:   "domain-platform",
						Key:  "platform",
						Name: "Platform",
					},
					&domain.CatalogTaxonomyReference{
						ID:   "subdomain-platform-api",
						Key:  "api",
						Name: "API",
					},
					nil,
					nil,
					[]domain.CatalogTaxonomyReference{
						{ID: "tag-backend", Key: "backend", Name: "Backend"},
					},
				),
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
				CatalogClassificationState: domain.DeriveCatalogClassificationState(
					&domain.CatalogTaxonomyReference{
						ID:   "domain-observability",
						Key:  "observability",
						Name: "Observability",
					},
					&domain.CatalogTaxonomyReference{
						ID:   "subdomain-observability-metrics",
						Key:  "metrics",
						Name: "Metrics",
					},
					&domain.CatalogTaxonomyReference{
						ID:   "domain-platform",
						Key:  "platform",
						Name: "Platform",
					},
					&domain.CatalogTaxonomyReference{
						ID:   "subdomain-platform-api",
						Key:  "api",
						Name: "API",
					},
					[]domain.CatalogTaxonomyReference{
						{ID: "tag-backend", Key: "backend", Name: "Backend"},
						{ID: "tag-metrics", Key: "metrics", Name: "Metrics"},
					},
				),
			},
			{
				ID:                         domain.BuildRuleCatalogItemID("sample-skill", "imports/rules/agents.md"),
				Classifier:                 domain.CatalogClassifierRule,
				Name:                       "agents.md",
				Description:                "Repository agent guardrails",
				Content:                    "# AGENTS\nFollow project rules.",
				ParentSkillID:              "sample-skill",
				ResourcePath:               "imports/rules/agents.md",
				ReadOnly:                   true,
				CatalogClassificationState: domain.DeriveCatalogClassificationState(nil, nil, nil, nil, nil),
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
	itemIDSet := toStringSet(filter.ItemIDs)
	for _, item := range s.items {
		if filter.ItemID != "" && item.ID != filter.ItemID {
			continue
		}
		if len(itemIDSet) > 0 {
			if _, exists := itemIDSet[item.ID]; !exists {
				continue
			}
		}
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
			"skill:sample-skill": {
				ItemID: "skill:sample-skill",
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
				CatalogClassificationState: domain.DeriveCatalogClassificationState(
					&domain.CatalogTaxonomyReference{
						ID:   "domain-platform",
						Key:  "platform",
						Name: "Platform",
					},
					&domain.CatalogTaxonomyReference{
						ID:   "subdomain-platform-api",
						Key:  "api",
						Name: "API",
					},
					nil,
					nil,
					nil,
				),
			},
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
				CatalogClassificationState: domain.DeriveCatalogClassificationState(
					&domain.CatalogTaxonomyReference{
						ID:   "domain-observability",
						Key:  "observability",
						Name: "Observability",
					},
					&domain.CatalogTaxonomyReference{
						ID:   "subdomain-observability-metrics",
						Key:  "metrics",
						Name: "Metrics",
					},
					&domain.CatalogTaxonomyReference{
						ID:   "domain-platform",
						Key:  "platform",
						Name: "Platform",
					},
					&domain.CatalogTaxonomyReference{
						ID:   "subdomain-platform-api",
						Key:  "api",
						Name: "API",
					},
					[]domain.CatalogTaxonomyReference{
						{ID: "tag-backend", Key: "backend", Name: "Backend"},
						{ID: "tag-metrics", Key: "metrics", Name: "Metrics"},
					},
				),
			},
		},
	}
}

func (s *fakeCatalogTaxonomyAssignmentService) Get(
	ctx context.Context,
	itemID string,
) (domain.CatalogItemTaxonomyAssignment, error) {
	itemID = normalizeFakeCatalogTaxonomyAssignmentItemID(itemID)
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

func (s *fakeCatalogTaxonomyAssignmentService) Patch(
	ctx context.Context,
	input domain.CatalogItemTaxonomyAssignmentPatchInput,
) (domain.CatalogItemTaxonomyAssignment, error) {
	itemID := normalizeFakeCatalogTaxonomyAssignmentItemID(input.ItemID)
	if itemID == "" {
		return domain.CatalogItemTaxonomyAssignment{}, fmt.Errorf(
			"%w: field=item_id detail=is required",
			domain.ErrCatalogTaxonomyValidation,
		)
	}

	current, exists := s.byItemID[itemID]
	if !exists {
		current = domain.CatalogItemTaxonomyAssignment{ItemID: itemID}
	}
	if input.TagIDs != nil {
		current.Tags = buildFakeTaxonomyReferences(*input.TagIDs)
	}
	if input.AddTagIDs != nil {
		seen := make(map[string]struct{}, len(current.Tags))
		for _, tag := range current.Tags {
			seen[tag.ID] = struct{}{}
		}
		for _, tagID := range *input.AddTagIDs {
			if _, exists := seen[tagID]; exists {
				continue
			}
			seen[tagID] = struct{}{}
			current.Tags = append(current.Tags, fakeTaxonomyReference(tagID))
		}
	}
	if input.RemoveTagIDs != nil {
		removeSet := toStringSet(*input.RemoveTagIDs)
		filtered := make([]domain.CatalogTaxonomyReference, 0, len(current.Tags))
		for _, tag := range current.Tags {
			if _, remove := removeSet[tag.ID]; remove {
				continue
			}
			filtered = append(filtered, tag)
		}
		current.Tags = filtered
	}
	if input.ClearTags != nil && *input.ClearTags {
		current.Tags = []domain.CatalogTaxonomyReference{}
	}
	current.CatalogClassificationState = domain.DeriveCatalogClassificationState(
		current.PrimaryDomain,
		current.PrimarySubdomain,
		current.SecondaryDomain,
		current.SecondarySubdomain,
		current.Tags,
	)
	s.byItemID[itemID] = current
	return current, nil
}

func (s *fakeCatalogTaxonomyAssignmentService) PatchBatch(
	ctx context.Context,
	request domain.CatalogItemTaxonomyBatchPatchRequest,
) (domain.CatalogItemTaxonomyBatchPatchResult, error) {
	result := domain.CatalogItemTaxonomyBatchPatchResult{
		DryRun: request.DryRun,
		Items:  make([]domain.CatalogItemTaxonomyBatchPatchItemResult, 0, len(request.Items)),
	}
	for _, item := range request.Items {
		normalizedItemID := normalizeFakeCatalogTaxonomyAssignmentItemID(item.ItemID)
		status := domain.CatalogItemTaxonomyBatchPatchStatusUpdated
		if request.DryRun {
			status = domain.CatalogItemTaxonomyBatchPatchStatusPlanned
		}
		entry := domain.CatalogItemTaxonomyBatchPatchItemResult{
			RequestedItemID: item.ItemID,
			ItemID:          normalizedItemID,
			Status:          status,
		}
		if !request.DryRun {
			assignment, err := s.Patch(ctx, item)
			if err != nil {
				return domain.CatalogItemTaxonomyBatchPatchResult{}, err
			}
			entry.Assignment = &assignment
		}
		result.Items = append(result.Items, entry)
	}
	return result, nil
}

func normalizeFakeCatalogTaxonomyAssignmentItemID(itemID string) string {
	normalized := strings.TrimSpace(itemID)
	if normalized == "" {
		return ""
	}
	reference, err := domain.NormalizeCatalogItemReference(normalized)
	if err != nil {
		return normalized
	}
	return reference.ItemID
}

type fakeCatalogRelationshipService struct {
	byItemID map[string]domain.CatalogRelationshipView
}

func newFakeCatalogRelationshipService() *fakeCatalogRelationshipService {
	skillItemID := domain.BuildSkillCatalogItemID("sample-skill")
	promptItemID := domain.BuildPromptCatalogItemID("sample-skill", "imports/prompts/system.md")
	ruleItemID := domain.BuildRuleCatalogItemID("sample-skill", "imports/rules/agents.md")

	parentSkillID := "sample-skill"
	promptResourcePath := "imports/prompts/system.md"
	ruleResourcePath := "imports/rules/agents.md"

	return &fakeCatalogRelationshipService{
		byItemID: map[string]domain.CatalogRelationshipView{
			skillItemID: {
				ItemID: skillItemID,
				Relationships: domain.CatalogRelationshipSet{
					Prompt: &domain.CatalogRelationshipItem{
						ID:            promptItemID,
						Classifier:    domain.CatalogClassifierPrompt,
						Name:          "system.md",
						ParentSkillID: &parentSkillID,
						ResourcePath:  &promptResourcePath,
					},
					Rules: []domain.CatalogRelationshipItem{
						{
							ID:            ruleItemID,
							Classifier:    domain.CatalogClassifierRule,
							Name:          "agents.md",
							ParentSkillID: &parentSkillID,
							ResourcePath:  &ruleResourcePath,
						},
					},
					Skills: []domain.CatalogRelationshipItem{},
				},
			},
			promptItemID: {
				ItemID: promptItemID,
				Relationships: domain.CatalogRelationshipSet{
					Prompt: nil,
					Rules:  []domain.CatalogRelationshipItem{},
					Skills: []domain.CatalogRelationshipItem{
						{
							ID:         skillItemID,
							Classifier: domain.CatalogClassifierSkill,
							Name:       "sample-skill",
						},
					},
				},
			},
			ruleItemID: {
				ItemID: ruleItemID,
				Relationships: domain.CatalogRelationshipSet{
					Prompt: nil,
					Rules:  []domain.CatalogRelationshipItem{},
					Skills: []domain.CatalogRelationshipItem{
						{
							ID:         skillItemID,
							Classifier: domain.CatalogClassifierSkill,
							Name:       "sample-skill",
						},
					},
				},
			},
		},
	}
}

func (s *fakeCatalogRelationshipService) Get(
	ctx context.Context,
	itemID string,
) (domain.CatalogRelationshipView, error) {
	normalized := strings.TrimSpace(itemID)
	reference, err := domain.NormalizeCatalogItemReference(normalized)
	if err != nil {
		return domain.CatalogRelationshipView{}, fmt.Errorf(
			"%w: field=item_id detail=is invalid",
			domain.ErrCatalogRelationshipValidation,
		)
	}

	view, ok := s.byItemID[reference.ItemID]
	if !ok {
		return domain.CatalogRelationshipView{}, fmt.Errorf(
			"%w: item_id=%q",
			domain.ErrCatalogRelationshipItemNotFound,
			reference.ItemID,
		)
	}

	return view, nil
}

type fakeCatalogTaxonomyUsageService struct{}

func newFakeCatalogTaxonomyUsageService() *fakeCatalogTaxonomyUsageService {
	return &fakeCatalogTaxonomyUsageService{}
}

func (s *fakeCatalogTaxonomyUsageService) GetDomainUsage(
	ctx context.Context,
	domainID string,
	previewLimit int,
) (domain.CatalogTaxonomyUsageSummary, error) {
	return fakeCatalogTaxonomyUsageSummary(domain.CatalogTaxonomyObjectDomain, domainID, previewLimit), nil
}

func (s *fakeCatalogTaxonomyUsageService) GetSubdomainUsage(
	ctx context.Context,
	subdomainID string,
	previewLimit int,
) (domain.CatalogTaxonomyUsageSummary, error) {
	return fakeCatalogTaxonomyUsageSummary(domain.CatalogTaxonomyObjectSubdomain, subdomainID, previewLimit), nil
}

func (s *fakeCatalogTaxonomyUsageService) GetTagUsage(
	ctx context.Context,
	tagID string,
	previewLimit int,
) (domain.CatalogTaxonomyUsageSummary, error) {
	return fakeCatalogTaxonomyUsageSummary(domain.CatalogTaxonomyObjectTag, tagID, previewLimit), nil
}

func fakeCatalogTaxonomyUsageSummary(
	objectType domain.CatalogTaxonomyObjectType,
	objectID string,
	previewLimit int,
) domain.CatalogTaxonomyUsageSummary {
	previewIDs := []string{"skill:sample-skill", "prompt:sample-skill:imports/prompts/system.md"}
	if previewLimit >= 0 && previewLimit < len(previewIDs) {
		previewIDs = previewIDs[:previewLimit]
	}
	reason := domain.CatalogTaxonomyConflictReasonInUse
	return domain.CatalogTaxonomyUsageSummary{
		ObjectType:        objectType,
		ObjectID:          objectID,
		AssignmentCount:   2,
		DistinctItemCount: 2,
		PreviewItemIDs:    previewIDs,
		BlockingReason:    &reason,
	}
}

func buildFakeTaxonomyReferences(tagIDs []string) []domain.CatalogTaxonomyReference {
	references := make([]domain.CatalogTaxonomyReference, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		references = append(references, fakeTaxonomyReference(tagID))
	}
	return references
}

func fakeTaxonomyReference(tagID string) domain.CatalogTaxonomyReference {
	switch strings.TrimSpace(tagID) {
	case "tag-backend":
		return domain.CatalogTaxonomyReference{ID: "tag-backend", Key: "backend", Name: "Backend"}
	case "tag-metrics":
		return domain.CatalogTaxonomyReference{ID: "tag-metrics", Key: "metrics", Name: "Metrics"}
	default:
		return domain.CatalogTaxonomyReference{ID: tagID, Key: tagID, Name: tagID}
	}
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
