package web

import (
	"context"
	"embed"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/mudler/skillserver/pkg/domain"
	"github.com/mudler/skillserver/pkg/git"
)

//go:embed ui
var uiFiles embed.FS

const defaultMCPRoutePath = "/mcp"

// Server wraps the Echo server
type Server struct {
	echo                   *echo.Echo
	httpServer             *http.Server
	skillManager           domain.SkillManager
	fsManager              *domain.FileSystemManager
	catalogMetadataService *domain.CatalogMetadataService
	taxonomyAssignment     *domain.CatalogTaxonomyAssignmentService
	taxonomyRegistry       *domain.CatalogTaxonomyRegistryService
	gitRepos               []string
	gitSyncer              gitSyncer
	configManager          *git.ConfigManager
	manualRepoSyncHook     func(repo git.GitRepoConfig) error
}

type gitSyncer interface {
	GetRepos() []string
	AddRepo(repoURL string) error
	RemoveRepo(repoURL string) error
	UpdateRepos(repos []string) error
	SyncRepo(repoURL string) error
	GetSkillsDir() string
}

// NewServer creates a new web server.
func NewServer(
	skillManager domain.SkillManager,
	fsManager *domain.FileSystemManager,
	gitRepos []string,
	gitSyncer gitSyncer,
	configManager *git.ConfigManager,
	enableLogging bool,
	mcpHandler http.Handler,
	mcpPath string,
) *Server {
	e := echo.New()

	// Middleware
	// Only enable request logging if explicitly enabled (to avoid interfering with MCP stdio)
	if enableLogging {
		e.Use(middleware.RequestLogger())
	} else {
		e.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	e.Use(middleware.Recover())
	//e.Use(middleware.CORS())

	server := &Server{
		echo:          e,
		skillManager:  skillManager,
		fsManager:     fsManager,
		gitRepos:      gitRepos,
		gitSyncer:     gitSyncer,
		configManager: configManager,
	}

	// API routes
	api := e.Group("/api")
	api.GET("/catalog", server.listCatalog)
	api.GET("/catalog/search", server.searchCatalog)
	api.GET("/catalog/:id/metadata", server.getCatalogMetadata)
	api.PATCH("/catalog/:id/metadata", server.patchCatalogMetadata)
	api.GET("/catalog/:id/taxonomy", server.getCatalogItemTaxonomy)
	api.PATCH("/catalog/:id/taxonomy", server.patchCatalogItemTaxonomy)
	api.GET("/catalog/taxonomy/domains", server.listCatalogTaxonomyDomains)
	api.POST("/catalog/taxonomy/domains", server.createCatalogTaxonomyDomain)
	api.PATCH("/catalog/taxonomy/domains/:id", server.updateCatalogTaxonomyDomain)
	api.DELETE("/catalog/taxonomy/domains/:id", server.deleteCatalogTaxonomyDomain)
	api.GET("/catalog/taxonomy/subdomains", server.listCatalogTaxonomySubdomains)
	api.POST("/catalog/taxonomy/subdomains", server.createCatalogTaxonomySubdomain)
	api.PATCH("/catalog/taxonomy/subdomains/:id", server.updateCatalogTaxonomySubdomain)
	api.DELETE("/catalog/taxonomy/subdomains/:id", server.deleteCatalogTaxonomySubdomain)
	api.GET("/catalog/taxonomy/tags", server.listCatalogTaxonomyTags)
	api.POST("/catalog/taxonomy/tags", server.createCatalogTaxonomyTag)
	api.PATCH("/catalog/taxonomy/tags/:id", server.updateCatalogTaxonomyTag)
	api.DELETE("/catalog/taxonomy/tags/:id", server.deleteCatalogTaxonomyTag)
	api.GET("/skills", server.listSkills)
	api.GET("/skills/:name", server.getSkill)
	api.GET("/skills/by-id/:repo/:name", server.getSkill)
	api.POST("/skills", server.createSkill)
	api.PUT("/skills/:name", server.updateSkill)
	api.PUT("/skills/by-id/:repo/:name", server.updateSkill)
	api.DELETE("/skills/:name", server.deleteSkill)
	api.DELETE("/skills/by-id/:repo/:name", server.deleteSkill)
	api.GET("/skills/search", server.searchSkills)

	// Import/Export routes
	// Use wildcard for export to handle skill names with slashes (repoName/skillName)
	// Register before other /skills routes to ensure it matches first
	api.GET("/skills/export/*", server.exportSkill)
	api.POST("/skills/import", server.importSkill)

	// Resource management routes
	api.GET("/skills/:name/resources", server.listSkillResources)
	api.GET("/skills/by-id/:repo/:name/resources", server.listSkillResources)
	api.GET("/skills/:name/resources/*", server.getSkillResource)
	api.GET("/skills/by-id/:repo/:name/resources/*", server.getSkillResource)
	api.POST("/skills/:name/resources", server.createSkillResource)
	api.POST("/skills/by-id/:repo/:name/resources", server.createSkillResource)
	api.PUT("/skills/:name/resources/*", server.updateSkillResource)
	api.PUT("/skills/by-id/:repo/:name/resources/*", server.updateSkillResource)
	api.DELETE("/skills/:name/resources/*", server.deleteSkillResource)
	api.DELETE("/skills/by-id/:repo/:name/resources/*", server.deleteSkillResource)

	// Git repository management routes
	api.GET("/git-repos", server.listGitRepos)
	api.POST("/git-repos", server.addGitRepo)
	api.PUT("/git-repos/:id", server.updateGitRepo)
	api.DELETE("/git-repos/:id", server.deleteGitRepo)
	api.POST("/git-repos/:id/sync", server.syncGitRepo)
	api.POST("/git-repos/:id/toggle", server.toggleGitRepo)

	// Register MCP routes before the UI catch-all route so /mcp is not intercepted.
	if mcpHandler != nil {
		resolvedMCPPath := mcpPath
		if resolvedMCPPath == "" {
			resolvedMCPPath = defaultMCPRoutePath
		}

		wrappedMCPHandler := echo.WrapHandler(mcpHandler)
		e.GET(resolvedMCPPath, wrappedMCPHandler)
		e.POST(resolvedMCPPath, wrappedMCPHandler)
		e.DELETE(resolvedMCPPath, wrappedMCPHandler)
		e.OPTIONS(resolvedMCPPath, wrappedMCPHandler)
	}

	// Serve UI
	uiFS, err := fs.Sub(uiFiles, "ui")
	if err != nil {
		panic(err)
	}
	e.GET("/*", echo.WrapHandler(http.FileServer(http.FS(uiFS))))

	return server
}

// SetCatalogMetadataService configures metadata overlay handlers.
func (s *Server) SetCatalogMetadataService(service *domain.CatalogMetadataService) {
	s.catalogMetadataService = service
}

// SetCatalogTaxonomyAssignmentService configures catalog item taxonomy assignment handlers.
func (s *Server) SetCatalogTaxonomyAssignmentService(service *domain.CatalogTaxonomyAssignmentService) {
	s.taxonomyAssignment = service
}

// SetCatalogTaxonomyRegistryService configures taxonomy registry handlers.
func (s *Server) SetCatalogTaxonomyRegistryService(service *domain.CatalogTaxonomyRegistryService) {
	s.taxonomyRegistry = service
}

// SetManualGitRepoSyncHook configures post-sync behavior for POST /api/git-repos/:id/sync.
func (s *Server) SetManualGitRepoSyncHook(hook func(repo git.GitRepoConfig) error) {
	s.manualRepoSyncHook = hook
}

// Start starts the web server
func (s *Server) Start(addr string) error {
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.echo,
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	if s.httpServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}
