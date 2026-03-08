package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/mudler/skillserver/pkg/domain"
	"github.com/mudler/skillserver/pkg/git"
	"github.com/mudler/skillserver/pkg/mcp"
	"github.com/mudler/skillserver/pkg/web"
)

// getEnvOrDefault returns the environment variable value or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvOrEmpty returns the environment variable value or empty string
func getEnvOrEmpty(key string) string {
	return os.Getenv(key)
}

// getEnvBool returns the environment variable as a boolean, or default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// setupLogger configures logging based on the enable flag
// When disabled, all logs go to io.Discard to avoid interfering with stdio MCP protocol
func setupLogger(enable bool) *log.Logger {
	var writer io.Writer
	if enable {
		writer = os.Stderr // Use stderr for logs when enabled (doesn't interfere with MCP stdio)
	} else {
		writer = io.Discard // Discard all logs when disabled
	}
	return log.New(writer, "", log.LstdFlags)
}

func main() {
	// Get default values from environment variables
	defaultDir := getEnvOrDefault("SKILLSERVER_DIR", getEnvOrDefault("SKILLS_DIR", "./skills"))
	defaultPort := getEnvOrDefault("SKILLSERVER_PORT", getEnvOrDefault("PORT", "8080"))
	defaultGitRepos := getEnvOrEmpty("SKILLSERVER_GIT_REPOS")
	if defaultGitRepos == "" {
		defaultGitRepos = getEnvOrEmpty("GIT_REPOS")
	}
	// Logging defaults to false (disabled) to avoid interfering with MCP stdio
	defaultEnableLogging := getEnvBool("SKILLSERVER_ENABLE_LOGGING", false)
	defaultEnableImportDiscovery := getEnvBool("SKILLSERVER_ENABLE_IMPORT_DISCOVERY", true)

	// Parse command line flags (flags override environment variables)
	skillsDir := flag.String("dir", defaultDir, "Directory to store skills (env: SKILLSERVER_DIR or SKILLS_DIR)")
	port := flag.String("port", defaultPort, "Port for the web server (env: SKILLSERVER_PORT or PORT)")
	gitReposFlag := flag.String("git-repos", defaultGitRepos, "Comma-separated list of Git repository URLs to sync (env: SKILLSERVER_GIT_REPOS or GIT_REPOS)")
	enableLogging := flag.Bool("enable-logging", defaultEnableLogging, "Enable logging to stderr (env: SKILLSERVER_ENABLE_LOGGING). Default: false (disabled to avoid interfering with MCP stdio)")
	enableImportDiscovery := flag.Bool("enable-import-discovery", defaultEnableImportDiscovery, "Enable imported resource discovery and imports/... virtual resources (env: SKILLSERVER_ENABLE_IMPORT_DISCOVERY)")
	mcpFlagValues := registerMCPRuntimeFlags(flag.CommandLine)
	catalogFlagValues := registerCatalogRuntimeFlags(flag.CommandLine)
	persistenceFlagValues := registerPersistenceRuntimeFlags(flag.CommandLine)
	gitCredentialFlagValues := registerGitCredentialRuntimeFlags(flag.CommandLine)
	flag.Parse()

	// Parse and validate MCP runtime config (flags > env > defaults).
	// Runtime wiring will consume this in later work packages.
	mcpRuntimeConfig, err := parseMCPRuntimeConfig(flag.CommandLine, mcpFlagValues, os.LookupEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid MCP runtime configuration: %v\n", err)
		os.Exit(2)
	}

	// Parse and validate prompt catalog runtime config (flags > env > defaults).
	catalogRuntimeConfig, err := parseCatalogRuntimeConfig(flag.CommandLine, catalogFlagValues, os.LookupEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid catalog runtime configuration: %v\n", err)
		os.Exit(2)
	}

	// Parse and validate persistence runtime config (flags > env > defaults).
	persistenceRuntimeConfig, err := parsePersistenceRuntimeConfig(flag.CommandLine, persistenceFlagValues, os.LookupEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid persistence runtime configuration: %v\n", err)
		os.Exit(2)
	}
	if err := validatePersistenceStartupConfig(persistenceRuntimeConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid persistence runtime configuration: %v\n", err)
		os.Exit(2)
	}

	gitCredentialRuntimeConfig, err := parseGitCredentialRuntimeConfig(
		flag.CommandLine,
		gitCredentialFlagValues,
		os.LookupEnv,
		os.ReadFile,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid git credential runtime configuration: %v\n", err)
		os.Exit(2)
	}
	if err := validateGitCredentialStartupConfig(gitCredentialRuntimeConfig, persistenceRuntimeConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid git credential runtime configuration: %v\n", err)
		os.Exit(2)
	}
	gitStoredCredentialsEnabled := gitStoredCredentialCapabilityEnabled(
		gitCredentialRuntimeConfig,
		persistenceRuntimeConfig,
	)

	// Setup logger based on flag
	logger := setupLogger(*enableLogging)
	log.SetOutput(logger.Writer())
	log.SetFlags(logger.Flags())

	// Get final values (flags take precedence over env vars)
	finalDir := *skillsDir
	finalPort := *port
	finalGitRepos := *gitReposFlag

	// Initialize config manager
	configManager := git.NewConfigManager(finalDir)

	// Load git repos from config file or use command line/env repos.
	var gitRepoConfigs []git.GitRepoConfig
	configRepos, err := configManager.LoadConfig()
	if err != nil && *enableLogging {
		log.Printf("Warning: Failed to load git repo config: %v", err)
	}

	// If config file has repos, use them; otherwise use command line/env repos
	if len(configRepos) > 0 {
		for _, repo := range configRepos {
			if repo.Enabled {
				gitRepoConfigs = append(gitRepoConfigs, repo)
			}
		}
	} else {
		// Parse git repos from command line/env
		if finalGitRepos != "" {
			parsedRepos := strings.Split(finalGitRepos, ",")
			seenCanonicalURLs := make(map[string]struct{}, len(parsedRepos))
			for _, rawRepoURL := range parsedRepos {
				trimmedRepoURL := strings.TrimSpace(rawRepoURL)
				if trimmedRepoURL == "" {
					continue
				}

				canonicalRepoURL, canonicalizeErr := git.CanonicalizeRepoURL(trimmedRepoURL)
				if canonicalizeErr != nil {
					fmt.Fprintf(os.Stderr, "Invalid git repository URL %q: %v\n", trimmedRepoURL, canonicalizeErr)
					os.Exit(2)
				}
				if _, seen := seenCanonicalURLs[canonicalRepoURL]; seen {
					continue
				}
				seenCanonicalURLs[canonicalRepoURL] = struct{}{}
				gitRepoConfigs = append(gitRepoConfigs, git.GitRepoConfig{
					ID:      git.GenerateID(canonicalRepoURL),
					URL:     canonicalRepoURL,
					Name:    git.ResolveCheckoutName(canonicalRepoURL),
					Enabled: true,
				})
			}

			// Save to config file if we have repos from command line/env
			if len(gitRepoConfigs) > 0 {
				if err := configManager.SaveConfig(gitRepoConfigs); err != nil && *enableLogging {
					log.Printf("Warning: Failed to save git repo config: %v", err)
				}
			}
		}
	}

	// Extract git repo checkout names for read-only detection.
	var gitRepoNames []string
	for _, repoConfig := range gitRepoConfigs {
		repoName := git.ResolveRepoCheckoutName(repoConfig)
		if repoName != "" {
			gitRepoNames = append(gitRepoNames, repoName)
		}
	}

	// Initialize skill manager
	skillManager, err := domain.NewFileSystemManager(finalDir, gitRepoNames)
	if err != nil {
		log.Fatalf("Failed to initialize skill manager: %v", err)
	}
	fsManager := skillManager
	skillManager.SetImportDiscoveryEnabled(*enableImportDiscovery)
	skillManager.SetPromptCatalogEnabled(catalogRuntimeConfig.EnablePrompts)
	skillManager.SetPromptCatalogDirectoryAllowlist(catalogRuntimeConfig.PromptDirectoryAllowlist)
	if err := skillManager.RebuildIndex(); err != nil {
		log.Fatalf("Failed to apply runtime catalog configuration: %v", err)
	}

	persistenceRuntime, err := bootstrapCatalogPersistenceRuntime(
		context.Background(),
		persistenceRuntimeConfig,
		fsManager,
		logger,
	)
	if err != nil {
		log.Fatalf("Failed to initialize persistence runtime: %v", err)
	}
	if persistenceRuntime != nil {
		defer func() {
			if closeErr := persistenceRuntime.Close(); closeErr != nil && *enableLogging {
				log.Printf("Warning: Failed to close persistence runtime: %v", closeErr)
			}
		}()
	}

	catalogOnUpdate := func() error {
		if persistenceRuntime == nil {
			return skillManager.RebuildIndex()
		}
		return persistenceRuntime.coordinator.FullSyncAndRebuild(context.Background())
	}

	if *enableLogging {
		log.Printf(
			"Resolved catalog runtime options: enable_prompts=%t prompt_dirs=%s",
			catalogRuntimeConfig.EnablePrompts,
			strings.Join(catalogRuntimeConfig.PromptDirectoryAllowlist, ","),
		)
		if persistenceRuntimeConfig.Enabled {
			log.Printf(
				"Resolved persistence runtime options: enabled=%t dir=%s db_path=%s",
				persistenceRuntimeConfig.Enabled,
				persistenceRuntimeConfig.Dir,
				persistenceRuntimeConfig.DBPath,
			)
		} else {
			log.Printf("Resolved persistence runtime options: enabled=false")
		}
		log.Printf(
			"Resolved git credential runtime options: stored_credentials_enabled=%t master_key_source=%s",
			gitStoredCredentialsEnabled,
			gitCredentialRuntimeConfig.MasterKeySource,
		)
	}

	storedCredentialProvider, gitCredentialStore, err := newStoredGitCredentialProvider(
		gitCredentialRuntimeConfig,
		persistenceRuntime,
	)
	if err != nil {
		log.Fatalf("Failed to initialize stored git credential provider: %v", err)
	}

	// Initialize Git syncer.
	var gitSyncer *git.GitSyncer
	gitSyncer = git.NewGitSyncer(finalDir, gitRepoConfigs, catalogOnUpdate)
	if storedCredentialProvider != nil {
		gitSyncer.SetStoredCredentialProvider(storedCredentialProvider)
	}
	// Configure git syncer output based on logging flag
	if *enableLogging {
		gitSyncer.SetProgressWriter(os.Stderr) // Use stderr for git progress
		gitSyncer.SetLogger(os.Stderr)         // Use stderr for log messages
	}
	if err := gitSyncer.Start(); err != nil {
		if persistenceRuntime != nil {
			log.Fatalf("Failed to start Git syncer with persistence synchronization: %v", err)
		}
		log.Printf("Warning: Failed to start Git syncer: %v", err)
	} else if *enableLogging {
		log.Println("Git syncer started")
	}

	// Create MCP server and optional HTTP transport handler.
	mcpServer := mcp.NewServer(skillManager, mcp.ServerOptions{
		EnableTaxonomyWriteTools: mcpRuntimeConfig.EnableWrites,
	})

	var mcpHandler http.Handler
	mcpPath := ""
	if requiresMCPHTTP(mcpRuntimeConfig.Transport) {
		mcpPath = mcpRuntimeConfig.HTTPPath
		mcpHandler = mcpServer.NewStreamableHTTPHandler(mcp.StreamableHTTPConfig{
			SessionTimeout:     mcpRuntimeConfig.SessionTimeout,
			Stateless:          mcpRuntimeConfig.Stateless,
			EnableEventStore:   mcpRuntimeConfig.EnableEventStore,
			EventStoreMaxBytes: mcpRuntimeConfig.EventStoreMaxBytes,
		})
	}

	gitRepoURLs := make([]string, len(gitRepoConfigs))
	for i, repo := range gitRepoConfigs {
		gitRepoURLs[i] = repo.URL
	}

	webServer := web.NewServer(
		skillManager,
		fsManager,
		gitRepoURLs,
		gitSyncer,
		configManager,
		*enableLogging,
		mcpHandler,
		mcpPath,
	)
	webServer.SetGitRuntimeCapabilities(web.GitRuntimeCapabilities{
		StoredCredentialsEnabled: gitStoredCredentialsEnabled,
	})
	webServer.SetGitCredentialStore(gitCredentialStore)
	if persistenceRuntime != nil {
		metadataService, metadataErr := domain.NewCatalogMetadataService(
			persistenceRuntime.sourceRepo,
			persistenceRuntime.overlayRepo,
			persistenceRuntime.coordinator.effectiveService,
			domain.CatalogMetadataServiceOptions{},
		)
		if metadataErr != nil {
			log.Fatalf("Failed to initialize catalog metadata service: %v", metadataErr)
		}
		webServer.SetCatalogMetadataService(metadataService)
		webServer.SetCatalogTaxonomyAssignmentService(persistenceRuntime.taxonomyAssignment)
		webServer.SetCatalogTaxonomyRegistryService(persistenceRuntime.taxonomyRegistryService)
		mcpServer.SetCatalogMetadataService(metadataService)
		mcpServer.SetCatalogTaxonomyAssignmentService(persistenceRuntime.taxonomyAssignment)
		mcpServer.SetCatalogTaxonomyRegistryService(persistenceRuntime.taxonomyRegistryService)

		webServer.SetManualGitRepoSyncHook(func(repo git.GitRepoConfig) error {
			repoName := git.ResolveRepoCheckoutName(repo)
			return persistenceRuntime.coordinator.RepoSyncAndRebuild(context.Background(), repoName)
		})
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	if err := runRuntime(context.Background(), runtimeDependencies{
		logger:        logger,
		enableLogging: *enableLogging,
		mcpConfig:     mcpRuntimeConfig,
		port:          finalPort,
		webServer:     webServer,
		mcpServer:     mcpServer,
		gitSyncer:     gitSyncer,
		signalChan:    sigChan,
	}); err != nil && *enableLogging {
		log.Printf("Runtime error: %v", err)
	}
}
