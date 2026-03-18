package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mudler/skillserver/pkg/domain"
)

// ServerOptions configures MCP server behavior.
type ServerOptions struct {
	EnableTaxonomyWriteTools               bool
	EnableMaterializationTools             bool
	AllowedMaterializationDestinationRoots []string
}

// CatalogMetadataReader exposes effective catalog item listing for MCP read tools.
type CatalogMetadataReader interface {
	List(ctx context.Context, filter domain.CatalogEffectiveListFilter) ([]domain.CatalogItem, error)
}

// CatalogTaxonomyAssignmentReader exposes catalog-item taxonomy assignment reads for MCP tools.
type CatalogTaxonomyAssignmentReader interface {
	Get(ctx context.Context, itemID string) (domain.CatalogItemTaxonomyAssignment, error)
}

// CatalogTaxonomyAssignmentWriter exposes catalog-item taxonomy assignment writes for MCP tools.
type CatalogTaxonomyAssignmentWriter interface {
	Patch(
		ctx context.Context,
		input domain.CatalogItemTaxonomyAssignmentPatchInput,
	) (domain.CatalogItemTaxonomyAssignment, error)
	PatchBatch(
		ctx context.Context,
		request domain.CatalogItemTaxonomyBatchPatchRequest,
	) (domain.CatalogItemTaxonomyBatchPatchResult, error)
}

// CatalogTaxonomyRegistryReader exposes taxonomy registry reads for MCP tools.
type CatalogTaxonomyRegistryReader interface {
	ListDomains(ctx context.Context, filter domain.CatalogTaxonomyDomainListFilter) ([]domain.CatalogTaxonomyDomain, error)
	ListSubdomains(ctx context.Context, filter domain.CatalogTaxonomySubdomainListFilter) ([]domain.CatalogTaxonomySubdomain, error)
	ListTags(ctx context.Context, filter domain.CatalogTaxonomyTagListFilter) ([]domain.CatalogTaxonomyTag, error)
}

// CatalogTaxonomyRegistryWriter exposes taxonomy registry writes for MCP tools.
type CatalogTaxonomyRegistryWriter interface {
	CreateDomain(
		ctx context.Context,
		input domain.CatalogTaxonomyDomainCreateInput,
	) (domain.CatalogTaxonomyDomain, error)
	UpdateDomain(
		ctx context.Context,
		input domain.CatalogTaxonomyDomainUpdateInput,
	) (domain.CatalogTaxonomyDomain, error)
	DeleteDomain(ctx context.Context, domainID string) error
	CreateSubdomain(
		ctx context.Context,
		input domain.CatalogTaxonomySubdomainCreateInput,
	) (domain.CatalogTaxonomySubdomain, error)
	UpdateSubdomain(
		ctx context.Context,
		input domain.CatalogTaxonomySubdomainUpdateInput,
	) (domain.CatalogTaxonomySubdomain, error)
	DeleteSubdomain(ctx context.Context, subdomainID string) error
	CreateTag(
		ctx context.Context,
		input domain.CatalogTaxonomyTagCreateInput,
	) (domain.CatalogTaxonomyTag, error)
	UpdateTag(
		ctx context.Context,
		input domain.CatalogTaxonomyTagUpdateInput,
	) (domain.CatalogTaxonomyTag, error)
	DeleteTag(ctx context.Context, tagID string) error
}

// CatalogTaxonomyUsageReader exposes taxonomy usage/preflight reads for MCP tools.
type CatalogTaxonomyUsageReader interface {
	GetDomainUsage(ctx context.Context, domainID string, previewLimit int) (domain.CatalogTaxonomyUsageSummary, error)
	GetSubdomainUsage(
		ctx context.Context,
		subdomainID string,
		previewLimit int,
	) (domain.CatalogTaxonomyUsageSummary, error)
	GetTagUsage(ctx context.Context, tagID string, previewLimit int) (domain.CatalogTaxonomyUsageSummary, error)
}

// CatalogRelationshipReader exposes relationship projection reads for MCP tools.
type CatalogRelationshipReader interface {
	Get(ctx context.Context, itemID string) (domain.CatalogRelationshipView, error)
}

// Server wraps the MCP server and provides access to the skill manager
type Server struct {
	mcpServer                              *mcp.Server
	skillManager                           domain.SkillManager
	catalogMetadata                        CatalogMetadataReader
	taxonomyAssign                         CatalogTaxonomyAssignmentReader
	taxonomyAssignWrite                    CatalogTaxonomyAssignmentWriter
	taxonomyRegistry                       CatalogTaxonomyRegistryReader
	taxonomyRegistryWrite                  CatalogTaxonomyRegistryWriter
	taxonomyUsage                          CatalogTaxonomyUsageReader
	relationships                          CatalogRelationshipReader
	enableTaxonomyWriteTools               bool
	enableMaterializationTools             bool
	allowedMaterializationDestinationRoots []string
	runWithTransport                       func(context.Context, mcp.Transport) error
}

// NewServer creates a new MCP server for skills
func NewServer(skillManager domain.SkillManager, options ...ServerOptions) *Server {
	opts := ServerOptions{}
	if len(options) > 0 {
		opts = options[0]
	}

	impl := &mcp.Implementation{
		Name:    "skillserver",
		Version: "v1.0.0",
	}

	mcpServer := mcp.NewServer(impl, nil)
	server := &Server{
		mcpServer:                  mcpServer,
		skillManager:               skillManager,
		enableTaxonomyWriteTools:   opts.EnableTaxonomyWriteTools,
		enableMaterializationTools: opts.EnableMaterializationTools,
		allowedMaterializationDestinationRoots: append(
			[]string(nil),
			opts.AllowedMaterializationDestinationRoots...,
		),
	}

	registerReadTools(mcpServer, server)
	if server.enableMaterializationTools {
		registerMaterializationWriteTools(mcpServer, server)
	}
	if server.enableTaxonomyWriteTools {
		registerTaxonomyWriteTools(mcpServer, server)
	}

	server.runWithTransport = mcpServer.Run
	return server
}

func registerReadTools(mcpServer *mcp.Server, server *Server) {
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "list_skills",
		Description: "List all available skills",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListSkillsInput) (
		*mcp.CallToolResult,
		ListSkillsOutput,
		error,
	) {
		return listSkills(ctx, req, input, server.skillManager)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "read_skill",
		Description: "Read the full content of a skill by its ID (use the 'id' field returned by list_skills or search_skills)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ReadSkillInput) (
		*mcp.CallToolResult,
		ReadSkillOutput,
		error,
	) {
		return readSkill(ctx, req, input, server.skillManager)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "search_skills",
		Description: "Search for skills by query string",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchSkillsInput) (
		*mcp.CallToolResult,
		SearchSkillsOutput,
		error,
	) {
		return searchSkills(ctx, req, input, server.skillManager)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "list_catalog",
		Description: "List unified catalog items (skills, prompts, and rules) with optional classifier and taxonomy filters",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListCatalogInput) (
		*mcp.CallToolResult,
		ListCatalogOutput,
		error,
	) {
		return listCatalog(ctx, req, input, server.skillManager, server.catalogMetadata)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "search_catalog",
		Description: "Search unified catalog items (skills, prompts, and rules) by query with optional classifier and taxonomy filters",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchCatalogInput) (
		*mcp.CallToolResult,
		SearchCatalogOutput,
		error,
	) {
		return searchCatalog(ctx, req, input, server.skillManager, server.catalogMetadata)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "read_catalog_item",
		Description: "Read one unified catalog item by exact item_id, including full content",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ReadCatalogItemInput) (
		*mcp.CallToolResult,
		ReadCatalogItemOutput,
		error,
	) {
		return readCatalogItem(ctx, req, input, server.skillManager, server.catalogMetadata)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "export_catalog_items",
		Description: "Export catalog items into a tar.gz archive with optional dry-run planning output",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ExportCatalogItemsInput) (
		*mcp.CallToolResult,
		ExportCatalogItemsOutput,
		error,
	) {
		return exportCatalogItems(ctx, req, input, server.skillManager)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "list_taxonomy_domains",
		Description: "List catalog taxonomy domain objects with optional domain_id/domain_ids/key/keys/active filters",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListTaxonomyDomainsInput) (
		*mcp.CallToolResult,
		ListTaxonomyDomainsOutput,
		error,
	) {
		return listTaxonomyDomains(ctx, req, input, server.taxonomyRegistry)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "list_taxonomy_subdomains",
		Description: "List catalog taxonomy subdomain objects with optional subdomain/domain/key/active filters",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListTaxonomySubdomainsInput) (
		*mcp.CallToolResult,
		ListTaxonomySubdomainsOutput,
		error,
	) {
		return listTaxonomySubdomains(ctx, req, input, server.taxonomyRegistry)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "list_taxonomy_tags",
		Description: "List catalog taxonomy tag objects with optional tag_id/tag_ids/key/keys/active filters",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListTaxonomyTagsInput) (
		*mcp.CallToolResult,
		ListTaxonomyTagsOutput,
		error,
	) {
		return listTaxonomyTags(ctx, req, input, server.taxonomyRegistry)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "get_catalog_item_taxonomy",
		Description: "Get taxonomy assignment metadata for one catalog item by item_id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetCatalogItemTaxonomyInput) (
		*mcp.CallToolResult,
		GetCatalogItemTaxonomyOutput,
		error,
	) {
		return getCatalogItemTaxonomy(ctx, req, input, server.taxonomyAssign)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "get_catalog_item_relationships",
		Description: "Get relationship metadata for one catalog item by item_id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetCatalogItemRelationshipsInput) (
		*mcp.CallToolResult,
		GetCatalogItemRelationshipsOutput,
		error,
	) {
		return getCatalogItemRelationships(ctx, req, input, server.relationships)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "get_taxonomy_domain_usage",
		Description: "Get delete-preflight usage metadata for one taxonomy domain",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetTaxonomyDomainUsageInput) (
		*mcp.CallToolResult,
		GetTaxonomyDomainUsageOutput,
		error,
	) {
		return getTaxonomyDomainUsage(ctx, req, input, server.taxonomyUsage)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "get_taxonomy_subdomain_usage",
		Description: "Get delete-preflight usage metadata for one taxonomy subdomain",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetTaxonomySubdomainUsageInput) (
		*mcp.CallToolResult,
		GetTaxonomySubdomainUsageOutput,
		error,
	) {
		return getTaxonomySubdomainUsage(ctx, req, input, server.taxonomyUsage)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "get_taxonomy_tag_usage",
		Description: "Get delete-preflight usage metadata for one taxonomy tag",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetTaxonomyTagUsageInput) (
		*mcp.CallToolResult,
		GetTaxonomyTagUsageOutput,
		error,
	) {
		return getTaxonomyTagUsage(ctx, req, input, server.taxonomyUsage)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "list_skill_resources",
		Description: "List all resources in a skill, including scripts, references, prompts, assets, and imported resources under imports/ paths",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListSkillResourcesInput) (
		*mcp.CallToolResult,
		ListSkillResourcesOutput,
		error,
	) {
		return listSkillResources(ctx, req, input, server.skillManager)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "read_skill_resource",
		Description: "Read the content of a skill resource file (scripts, references, prompts, assets, or imported imports/... resources). Text files are returned as UTF-8, binary files as base64. Files larger than 1MB cannot be read via MCP.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ReadSkillResourceInput) (
		*mcp.CallToolResult,
		ReadSkillResourceOutput,
		error,
	) {
		return readSkillResource(ctx, req, input, server.skillManager)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "get_skill_resource_info",
		Description: "Get metadata about a specific skill resource (including imported imports/... resources) without reading its content",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetSkillResourceInfoInput) (
		*mcp.CallToolResult,
		GetSkillResourceInfoOutput,
		error,
	) {
		return getSkillResourceInfo(ctx, req, input, server.skillManager)
	})
}

func registerMaterializationWriteTools(mcpServer *mcp.Server, server *Server) {
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "materialize_catalog_items",
		Description: "Materialize catalog items into an allowed destination directory. " +
			"Supports dry-run planning and write execution when capability-gated",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MaterializeCatalogItemsInput) (
		*mcp.CallToolResult,
		MaterializeCatalogItemsOutput,
		error,
	) {
		return materializeCatalogItems(
			ctx,
			req,
			input,
			server.skillManager,
			server.allowedMaterializationDestinationRoots,
		)
	})
}

func registerTaxonomyWriteTools(mcpServer *mcp.Server, server *Server) {
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "create_taxonomy_domain",
		Description: "Create one catalog taxonomy domain object",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateTaxonomyDomainInput) (
		*mcp.CallToolResult,
		CreateTaxonomyDomainOutput,
		error,
	) {
		return createTaxonomyDomain(ctx, req, input, server.taxonomyRegistryWrite)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "update_taxonomy_domain",
		Description: "Patch one catalog taxonomy domain object by domain_id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateTaxonomyDomainInput) (
		*mcp.CallToolResult,
		UpdateTaxonomyDomainOutput,
		error,
	) {
		return updateTaxonomyDomain(ctx, req, input, server.taxonomyRegistryWrite)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "delete_taxonomy_domain",
		Description: "Delete one catalog taxonomy domain object by domain_id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteTaxonomyDomainInput) (
		*mcp.CallToolResult,
		DeleteTaxonomyDomainOutput,
		error,
	) {
		return deleteTaxonomyDomain(ctx, req, input, server.taxonomyRegistryWrite)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "create_taxonomy_subdomain",
		Description: "Create one catalog taxonomy subdomain object",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateTaxonomySubdomainInput) (
		*mcp.CallToolResult,
		CreateTaxonomySubdomainOutput,
		error,
	) {
		return createTaxonomySubdomain(ctx, req, input, server.taxonomyRegistryWrite)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "update_taxonomy_subdomain",
		Description: "Patch one catalog taxonomy subdomain object by subdomain_id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateTaxonomySubdomainInput) (
		*mcp.CallToolResult,
		UpdateTaxonomySubdomainOutput,
		error,
	) {
		return updateTaxonomySubdomain(ctx, req, input, server.taxonomyRegistryWrite)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "delete_taxonomy_subdomain",
		Description: "Delete one catalog taxonomy subdomain object by subdomain_id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteTaxonomySubdomainInput) (
		*mcp.CallToolResult,
		DeleteTaxonomySubdomainOutput,
		error,
	) {
		return deleteTaxonomySubdomain(ctx, req, input, server.taxonomyRegistryWrite)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "create_taxonomy_tag",
		Description: "Create one catalog taxonomy tag object",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateTaxonomyTagInput) (
		*mcp.CallToolResult,
		CreateTaxonomyTagOutput,
		error,
	) {
		return createTaxonomyTag(ctx, req, input, server.taxonomyRegistryWrite)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "update_taxonomy_tag",
		Description: "Patch one catalog taxonomy tag object by tag_id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateTaxonomyTagInput) (
		*mcp.CallToolResult,
		UpdateTaxonomyTagOutput,
		error,
	) {
		return updateTaxonomyTag(ctx, req, input, server.taxonomyRegistryWrite)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "delete_taxonomy_tag",
		Description: "Delete one catalog taxonomy tag object by tag_id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteTaxonomyTagInput) (
		*mcp.CallToolResult,
		DeleteTaxonomyTagOutput,
		error,
	) {
		return deleteTaxonomyTag(ctx, req, input, server.taxonomyRegistryWrite)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "patch_catalog_item_taxonomy",
		Description: "Patch taxonomy assignment metadata for one catalog item by item_id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input PatchCatalogItemTaxonomyInput) (
		*mcp.CallToolResult,
		PatchCatalogItemTaxonomyOutput,
		error,
	) {
		return patchCatalogItemTaxonomy(ctx, req, input, server.taxonomyAssignWrite)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "patch_catalog_items_taxonomy",
		Description: "Patch taxonomy assignment metadata for multiple catalog items with optional dry-run planning",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input PatchCatalogItemsTaxonomyInput) (
		*mcp.CallToolResult,
		PatchCatalogItemsTaxonomyOutput,
		error,
	) {
		return patchCatalogItemsTaxonomy(ctx, req, input, server.taxonomyAssignWrite)
	})
}

// Run starts the MCP server with stdio transport
func (s *Server) Run(ctx context.Context) error {
	return s.RunWithTransport(ctx, &mcp.StdioTransport{})
}

// SetCatalogMetadataService configures effective catalog item reads for taxonomy-aware MCP filters.
func (s *Server) SetCatalogMetadataService(service CatalogMetadataReader) {
	s.catalogMetadata = service
}

// SetCatalogTaxonomyAssignmentService configures item taxonomy assignment reads for MCP tools.
func (s *Server) SetCatalogTaxonomyAssignmentService(service CatalogTaxonomyAssignmentReader) {
	s.taxonomyAssign = service
	if writer, ok := service.(CatalogTaxonomyAssignmentWriter); ok {
		s.taxonomyAssignWrite = writer
		return
	}
	s.taxonomyAssignWrite = nil
}

// SetCatalogTaxonomyRegistryService configures taxonomy registry reads for MCP tools.
func (s *Server) SetCatalogTaxonomyRegistryService(service CatalogTaxonomyRegistryReader) {
	s.taxonomyRegistry = service
	if writer, ok := service.(CatalogTaxonomyRegistryWriter); ok {
		s.taxonomyRegistryWrite = writer
		return
	}
	s.taxonomyRegistryWrite = nil
}

// SetCatalogTaxonomyUsageService configures taxonomy usage/preflight reads for MCP tools.
func (s *Server) SetCatalogTaxonomyUsageService(service CatalogTaxonomyUsageReader) {
	s.taxonomyUsage = service
}

// SetCatalogRelationshipService configures relationship projection reads for MCP tools.
func (s *Server) SetCatalogRelationshipService(service CatalogRelationshipReader) {
	s.relationships = service
}

// MaterializationToolsEnabled reports whether runtime config enabled write-capable materialization tools.
func (s *Server) MaterializationToolsEnabled() bool {
	return s.enableMaterializationTools
}

// AllowedMaterializationDestinationRoots returns the configured destination-root allowlist for materialization.
func (s *Server) AllowedMaterializationDestinationRoots() []string {
	return append([]string(nil), s.allowedMaterializationDestinationRoots...)
}

// RunWithTransport starts the MCP server with the given transport (e.g. in-memory for in-process embedding).
func (s *Server) RunWithTransport(ctx context.Context, transport mcp.Transport) error {
	if s.runWithTransport != nil {
		return s.runWithTransport(ctx, transport)
	}
	return s.mcpServer.Run(ctx, transport)
}
