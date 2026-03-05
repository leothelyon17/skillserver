package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mudler/skillserver/pkg/domain"
)

// ServerOptions configures MCP server behavior.
type ServerOptions struct {
	EnableTaxonomyWriteTools bool
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

// Server wraps the MCP server and provides access to the skill manager
type Server struct {
	mcpServer                *mcp.Server
	skillManager             domain.SkillManager
	catalogMetadata          CatalogMetadataReader
	taxonomyAssign           CatalogTaxonomyAssignmentReader
	taxonomyAssignWrite      CatalogTaxonomyAssignmentWriter
	taxonomyRegistry         CatalogTaxonomyRegistryReader
	taxonomyRegistryWrite    CatalogTaxonomyRegistryWriter
	enableTaxonomyWriteTools bool
	runWithTransport         func(context.Context, mcp.Transport) error
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
		mcpServer:                mcpServer,
		skillManager:             skillManager,
		enableTaxonomyWriteTools: opts.EnableTaxonomyWriteTools,
	}

	registerReadTools(mcpServer, server)
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
		Description: "List unified catalog items (skills and prompts) with optional classifier and taxonomy filters",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListCatalogInput) (
		*mcp.CallToolResult,
		ListCatalogOutput,
		error,
	) {
		return listCatalog(ctx, req, input, server.skillManager, server.catalogMetadata)
	})

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "search_catalog",
		Description: "Search unified catalog items by query with optional classifier and taxonomy filters",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchCatalogInput) (
		*mcp.CallToolResult,
		SearchCatalogOutput,
		error,
	) {
		return searchCatalog(ctx, req, input, server.skillManager, server.catalogMetadata)
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

// RunWithTransport starts the MCP server with the given transport (e.g. in-memory for in-process embedding).
func (s *Server) RunWithTransport(ctx context.Context, transport mcp.Transport) error {
	if s.runWithTransport != nil {
		return s.runWithTransport(ctx, transport)
	}
	return s.mcpServer.Run(ctx, transport)
}
