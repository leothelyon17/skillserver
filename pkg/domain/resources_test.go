package domain_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mudler/skillserver/pkg/domain"
)

var _ = Describe("Resource Management", func() {
	var (
		manager *domain.FileSystemManager
		tempDir string
		err     error
	)

	BeforeEach(func() {
		// Create a temp directory for each test
		tempDir, err = os.MkdirTemp("", "skillserver-resource-test")
		Expect(err).NotTo(HaveOccurred())

		// Initialize the manager
		manager, err = domain.NewFileSystemManager(tempDir, []string{})
		Expect(err).NotTo(HaveOccurred())

		// Create a test skill with SKILL.md
		skillDir := filepath.Join(tempDir, "test-skill")
		err = os.MkdirAll(skillDir, 0755)
		Expect(err).NotTo(HaveOccurred())

		skillMdContent := `---
name: test-skill
description: A test skill for resource management
---
# Test Skill
Content here.
`
		err = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMdContent), 0644)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Context("Listing Resources", func() {
		It("should return empty list when no resources exist", func() {
			resources, err := manager.ListSkillResources("test-skill")
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(BeEmpty())
		})

		It("should list scripts in scripts directory", func() {
			scriptsDir := filepath.Join(tempDir, "test-skill", "scripts")
			err := os.MkdirAll(scriptsDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			scriptContent := "#!/usr/bin/env python3\nprint('Hello')"
			err = os.WriteFile(filepath.Join(scriptsDir, "hello.py"), []byte(scriptContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			resources, err := manager.ListSkillResources("test-skill")
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(HaveLen(1))
			Expect(resources[0].Type).To(Equal(domain.ResourceTypeScript))
			Expect(resources[0].Origin).To(Equal(domain.ResourceOriginDirect))
			Expect(resources[0].Path).To(Equal("scripts/hello.py"))
			Expect(resources[0].Name).To(Equal("hello.py"))
			Expect(resources[0].Readable).To(BeTrue())
			Expect(resources[0].Writable).To(BeTrue())
		})

		It("should list references in references directory", func() {
			refsDir := filepath.Join(tempDir, "test-skill", "references")
			err := os.MkdirAll(refsDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			refContent := "# Reference\n\nSome reference content."
			err = os.WriteFile(filepath.Join(refsDir, "REFERENCE.md"), []byte(refContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			resources, err := manager.ListSkillResources("test-skill")
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(HaveLen(1))
			Expect(resources[0].Type).To(Equal(domain.ResourceTypeReference))
			Expect(resources[0].Origin).To(Equal(domain.ResourceOriginDirect))
			Expect(resources[0].Path).To(Equal("references/REFERENCE.md"))
			Expect(resources[0].Writable).To(BeTrue())
		})

		It("should list prompt resources from agents and prompts directories", func() {
			agentsDir := filepath.Join(tempDir, "test-skill", "agents")
			promptsDir := filepath.Join(tempDir, "test-skill", "prompts")
			Expect(os.MkdirAll(agentsDir, 0755)).To(Succeed())
			Expect(os.MkdirAll(promptsDir, 0755)).To(Succeed())

			Expect(os.WriteFile(filepath.Join(agentsDir, "coach.md"), []byte("# Coach"), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(promptsDir, "system.md"), []byte("# System"), 0644)).To(Succeed())

			resources, err := manager.ListSkillResources("test-skill")
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(HaveLen(2))
			Expect(resources[0].Path).To(Equal("agents/coach.md"))
			Expect(resources[0].Type).To(Equal(domain.ResourceTypePrompt))
			Expect(resources[0].Origin).To(Equal(domain.ResourceOriginDirect))
			Expect(resources[0].Writable).To(BeTrue())
			Expect(resources[1].Path).To(Equal("prompts/system.md"))
			Expect(resources[1].Type).To(Equal(domain.ResourceTypePrompt))
			Expect(resources[1].Origin).To(Equal(domain.ResourceOriginDirect))
			Expect(resources[1].Writable).To(BeTrue())
		})

		It("should list assets in assets directory", func() {
			assetsDir := filepath.Join(tempDir, "test-skill", "assets")
			err := os.MkdirAll(assetsDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			// Create a binary file (simulated)
			binaryContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
			err = os.WriteFile(filepath.Join(assetsDir, "image.png"), binaryContent, 0644)
			Expect(err).NotTo(HaveOccurred())

			resources, err := manager.ListSkillResources("test-skill")
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(HaveLen(1))
			Expect(resources[0].Type).To(Equal(domain.ResourceTypeAsset))
			Expect(resources[0].Origin).To(Equal(domain.ResourceOriginDirect))
			Expect(resources[0].Path).To(Equal("assets/image.png"))
			Expect(resources[0].Readable).To(BeFalse())
			Expect(resources[0].Writable).To(BeTrue())
		})

		It("should list resources from all directories", func() {
			// Create resources in all three directories
			scriptsDir := filepath.Join(tempDir, "test-skill", "scripts")
			refsDir := filepath.Join(tempDir, "test-skill", "references")
			assetsDir := filepath.Join(tempDir, "test-skill", "assets")

			os.MkdirAll(scriptsDir, 0755)
			os.MkdirAll(refsDir, 0755)
			os.MkdirAll(assetsDir, 0755)

			os.WriteFile(filepath.Join(scriptsDir, "script.py"), []byte("print('test')"), 0644)
			os.WriteFile(filepath.Join(refsDir, "ref.md"), []byte("# Ref"), 0644)
			os.WriteFile(filepath.Join(assetsDir, "asset.txt"), []byte("asset"), 0644)

			resources, err := manager.ListSkillResources("test-skill")
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(HaveLen(3))
		})

		It("should merge direct and imported resources with canonical dedupe and stable ordering", func() {
			skillMdContent := `---
name: test-skill
description: A test skill for import discovery
---
# Test Skill
[Direct Prompt](prompts/system.md)
[Shared Context](docs/shared.md)
@/docs/shared.md
`
			Expect(os.WriteFile(filepath.Join(tempDir, "test-skill", "SKILL.md"), []byte(skillMdContent), 0644)).To(Succeed())

			promptsDir := filepath.Join(tempDir, "test-skill", "prompts")
			Expect(os.MkdirAll(promptsDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(promptsDir, "system.md"), []byte("# System Prompt"), 0644)).To(Succeed())

			docsDir := filepath.Join(tempDir, "test-skill", "docs")
			Expect(os.MkdirAll(docsDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(docsDir, "shared.md"), []byte("# Shared Context"), 0644)).To(Succeed())

			firstResources, err := manager.ListSkillResources("test-skill")
			Expect(err).NotTo(HaveOccurred())
			secondResources, err := manager.ListSkillResources("test-skill")
			Expect(err).NotTo(HaveOccurred())

			Expect(firstResources).To(HaveLen(2))
			Expect(secondResources).To(HaveLen(2))
			Expect(firstResources).To(Equal(secondResources))

			Expect(firstResources[0].Path).To(Equal("imports/docs/shared.md"))
			Expect(firstResources[0].Origin).To(Equal(domain.ResourceOriginImported))
			Expect(firstResources[0].Type).To(Equal(domain.ResourceTypeReference))
			Expect(firstResources[0].Writable).To(BeFalse())

			Expect(firstResources[1].Path).To(Equal("prompts/system.md"))
			Expect(firstResources[1].Origin).To(Equal(domain.ResourceOriginDirect))
			Expect(firstResources[1].Type).To(Equal(domain.ResourceTypePrompt))
			Expect(firstResources[1].Writable).To(BeTrue())

			for _, resource := range firstResources {
				Expect(resource.Path).NotTo(Equal("imports/prompts/system.md"))
			}
		})

		It("should disable imported resource discovery when toggled off", func() {
			skillMdContent := `---
name: test-skill
description: A test skill for import discovery
---
# Test Skill
[Shared Context](references/shared.md)
`
			Expect(os.WriteFile(filepath.Join(tempDir, "test-skill", "SKILL.md"), []byte(skillMdContent), 0644)).To(Succeed())
			referencesDir := filepath.Join(tempDir, "test-skill", "references")
			Expect(os.MkdirAll(referencesDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(referencesDir, "shared.md"), []byte("# Shared"), 0644)).To(Succeed())

			manager.SetImportDiscoveryEnabled(false)
			resources, err := manager.ListSkillResources("test-skill")
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(HaveLen(1))
			Expect(resources[0].Path).To(Equal("references/shared.md"))
			Expect(resources[0].Origin).To(Equal(domain.ResourceOriginDirect))
		})
	})

	Context("Reading Resources", func() {
		BeforeEach(func() {
			scriptsDir := filepath.Join(tempDir, "test-skill", "scripts")
			os.MkdirAll(scriptsDir, 0755)
			os.WriteFile(filepath.Join(scriptsDir, "hello.py"), []byte("print('Hello, World!')"), 0644)
		})

		It("should read text resource as UTF-8", func() {
			content, err := manager.ReadSkillResource("test-skill", "scripts/hello.py")
			Expect(err).NotTo(HaveOccurred())
			Expect(content.Encoding).To(Equal("utf-8"))
			Expect(content.Content).To(Equal("print('Hello, World!')"))
			Expect(content.MimeType).To(ContainSubstring("python"))
		})

		It("should read binary resource as base64", func() {
			assetsDir := filepath.Join(tempDir, "test-skill", "assets")
			os.MkdirAll(assetsDir, 0755)
			// Create a file with null bytes to ensure it's detected as binary
			binaryData := make([]byte, 100)
			for i := range binaryData {
				if i%10 == 0 {
					binaryData[i] = 0 // Add null bytes
				} else {
					binaryData[i] = byte(i)
				}
			}
			os.WriteFile(filepath.Join(assetsDir, "test.bin"), binaryData, 0644)

			content, err := manager.ReadSkillResource("test-skill", "assets/test.bin")
			Expect(err).NotTo(HaveOccurred())
			Expect(content.Encoding).To(Equal("base64"))
			Expect(content.Content).NotTo(BeEmpty())
		})

		It("should return error for non-existent resource", func() {
			_, err := manager.ReadSkillResource("test-skill", "scripts/nonexistent.py")
			Expect(err).To(HaveOccurred())
		})

		It("should read imported virtual resources as UTF-8", func() {
			referencesDir := filepath.Join(tempDir, "test-skill", "references")
			Expect(os.MkdirAll(referencesDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(referencesDir, "shared.md"), []byte("# Shared"), 0644)).To(Succeed())

			content, err := manager.ReadSkillResource("test-skill", "imports/references/shared.md")
			Expect(err).NotTo(HaveOccurred())
			Expect(content.Encoding).To(Equal("utf-8"))
			Expect(content.Content).To(Equal("# Shared"))
		})

		It("should return error for missing imported virtual resources", func() {
			_, err := manager.ReadSkillResource("test-skill", "imports/references/missing.md")
			Expect(err).To(HaveOccurred())
		})

		It("should reject imported virtual resources that escape the allowed root", func() {
			referencesDir := filepath.Join(tempDir, "test-skill", "references")
			outsideDir := filepath.Join(tempDir, "outside")
			outsidePath := filepath.Join(outsideDir, "secret.md")
			linkPath := filepath.Join(referencesDir, "escape.md")

			Expect(os.MkdirAll(referencesDir, 0755)).To(Succeed())
			Expect(os.MkdirAll(outsideDir, 0755)).To(Succeed())
			Expect(os.WriteFile(outsidePath, []byte("# Secret"), 0644)).To(Succeed())

			if err := os.Symlink(outsidePath, linkPath); err != nil {
				Skip("symlink creation is not supported in this environment")
			}

			_, err := manager.ReadSkillResource("test-skill", "imports/references/escape.md")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("escapes allowed root"))
		})

		It("should reject imported virtual reads when import discovery is disabled", func() {
			manager.SetImportDiscoveryEnabled(false)
			_, err := manager.ReadSkillResource("test-skill", "imports/references/shared.md")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("import discovery is disabled"))
		})

		It("should return error for invalid path", func() {
			_, err := manager.ReadSkillResource("test-skill", "../invalid")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Getting Resource Info", func() {
		BeforeEach(func() {
			scriptsDir := filepath.Join(tempDir, "test-skill", "scripts")
			os.MkdirAll(scriptsDir, 0755)
			os.WriteFile(filepath.Join(scriptsDir, "script.py"), []byte("print('test')"), 0644)
		})

		It("should return resource metadata", func() {
			info, err := manager.GetSkillResourceInfo("test-skill", "scripts/script.py")
			Expect(err).NotTo(HaveOccurred())
			Expect(info).NotTo(BeNil())
			Expect(info.Type).To(Equal(domain.ResourceTypeScript))
			Expect(info.Origin).To(Equal(domain.ResourceOriginDirect))
			Expect(info.Path).To(Equal("scripts/script.py"))
			Expect(info.Name).To(Equal("script.py"))
			Expect(info.Size).To(BeNumerically(">", 0))
			Expect(info.Readable).To(BeTrue())
			Expect(info.Writable).To(BeTrue())
		})

		It("should return error for non-existent resource", func() {
			_, err := manager.GetSkillResourceInfo("test-skill", "scripts/nonexistent.py")
			Expect(err).To(HaveOccurred())
		})

		It("should return imported metadata for virtual imported paths", func() {
			referencesDir := filepath.Join(tempDir, "test-skill", "references")
			Expect(os.MkdirAll(referencesDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(referencesDir, "shared.md"), []byte("# Shared"), 0644)).To(Succeed())

			info, err := manager.GetSkillResourceInfo("test-skill", "imports/references/shared.md")
			Expect(err).NotTo(HaveOccurred())
			Expect(info).NotTo(BeNil())
			Expect(info.Path).To(Equal("imports/references/shared.md"))
			Expect(info.Origin).To(Equal(domain.ResourceOriginImported))
			Expect(info.Writable).To(BeFalse())
			Expect(info.Type).To(Equal(domain.ResourceTypeReference))
		})

		It("should return error for missing imported virtual resources in info lookups", func() {
			_, err := manager.GetSkillResourceInfo("test-skill", "imports/references/missing.md")
			Expect(err).To(HaveOccurred())
		})

		It("should reject escaped imported virtual resources in info lookups", func() {
			referencesDir := filepath.Join(tempDir, "test-skill", "references")
			outsideDir := filepath.Join(tempDir, "outside")
			outsidePath := filepath.Join(outsideDir, "secret.md")
			linkPath := filepath.Join(referencesDir, "escape.md")

			Expect(os.MkdirAll(referencesDir, 0755)).To(Succeed())
			Expect(os.MkdirAll(outsideDir, 0755)).To(Succeed())
			Expect(os.WriteFile(outsidePath, []byte("# Secret"), 0644)).To(Succeed())

			if err := os.Symlink(outsidePath, linkPath); err != nil {
				Skip("symlink creation is not supported in this environment")
			}

			_, err := manager.GetSkillResourceInfo("test-skill", "imports/references/escape.md")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("escapes allowed root"))
		})

		It("should reject imported metadata lookups when import discovery is disabled", func() {
			manager.SetImportDiscoveryEnabled(false)
			_, err := manager.GetSkillResourceInfo("test-skill", "imports/references/shared.md")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("import discovery is disabled"))
		})
	})

	Context("Path Validation", func() {
		It("should validate resource paths", func() {
			validPaths := []string{
				"scripts/test.py",
				"references/doc.md",
				"assets/image.png",
				"agents/assistant.md",
				"prompts/assistant.md",
			}

			for _, path := range validPaths {
				err := domain.ValidateResourcePath(path)
				Expect(err).NotTo(HaveOccurred(), "path %s should be valid", path)
			}
		})

		It("should reject invalid paths", func() {
			invalidPaths := []string{
				"../invalid",
				"/absolute/path",
				"invalid/path",
				"scripts/../../etc/passwd",
				"imports/shared.md",
			}

			for _, path := range invalidPaths {
				err := domain.ValidateResourcePath(path)
				Expect(err).To(HaveOccurred(), "path %s should be invalid", path)
			}
		})

		It("should validate readable paths including virtual imports", func() {
			validPaths := []string{
				"scripts/test.py",
				"prompts/assistant.md",
				"imports/shared/reference.md",
			}

			for _, path := range validPaths {
				err := domain.ValidateReadableResourcePath(path)
				Expect(err).NotTo(HaveOccurred(), "path %s should be valid for reads", path)
			}
		})

		It("should reject invalid readable paths", func() {
			invalidPaths := []string{
				"",
				"/absolute/path",
				"imports/../../etc/passwd",
				"../invalid",
				"unknown/path.md",
			}

			for _, path := range invalidPaths {
				err := domain.ValidateReadableResourcePath(path)
				Expect(err).To(HaveOccurred(), "path %s should be invalid for reads", path)
			}
		})
	})

	Context("Resource Type Inference", func() {
		It("should classify prompt resource prefixes", func() {
			Expect(domain.GetResourceType("agents/coach.md")).To(Equal(domain.ResourceTypePrompt))
			Expect(domain.GetResourceType("prompts/system.md")).To(Equal(domain.ResourceTypePrompt))
		})

		It("should preserve legacy resource type inference", func() {
			Expect(domain.GetResourceType("scripts/tool.py")).To(Equal(domain.ResourceTypeScript))
			Expect(domain.GetResourceType("references/readme.md")).To(Equal(domain.ResourceTypeReference))
			Expect(domain.GetResourceType("assets/logo.png")).To(Equal(domain.ResourceTypeAsset))
		})

		It("should classify imported prompt and non-prompt virtual paths", func() {
			Expect(domain.GetResourceType("imports/agents/coach.md")).To(Equal(domain.ResourceTypePrompt))
			Expect(domain.GetResourceType("imports/plugins/agent-teams/agents/coach.md")).To(Equal(domain.ResourceTypePrompt))
			Expect(domain.GetResourceType("imports/plugins/agent-orchestration/prompts/system.md")).To(Equal(domain.ResourceTypePrompt))
			Expect(domain.GetResourceType("imports/shared/context.md")).To(Equal(domain.ResourceTypeReference))
		})

		It("should detect virtual imported resource paths", func() {
			Expect(domain.IsImportedResourcePath("imports/shared/context.md")).To(BeTrue())
			Expect(domain.IsImportedResourcePath("references/context.md")).To(BeFalse())
		})
	})

	Context("Writability Metadata", func() {
		It("should mark git-repository resources as read-only", func() {
			gitRoot, err := os.MkdirTemp("", "skillserver-git-resource-test")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(gitRoot)

			repoName := "demo-repo"
			skillPath := filepath.Join(gitRoot, repoName, "git-skill")
			err = os.MkdirAll(filepath.Join(skillPath, "scripts"), 0755)
			Expect(err).NotTo(HaveOccurred())

			gitSkillContent := `---
name: git-skill
description: A git-backed skill
---
# Git Skill
`
			err = os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(gitSkillContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(filepath.Join(skillPath, "scripts", "tool.py"), []byte("print('ok')"), 0644)
			Expect(err).NotTo(HaveOccurred())

			gitManager, err := domain.NewFileSystemManager(gitRoot, []string{repoName})
			Expect(err).NotTo(HaveOccurred())

			resources, err := gitManager.ListSkillResources(repoName + "/git-skill")
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(HaveLen(1))
			Expect(resources[0].Writable).To(BeFalse())
			Expect(resources[0].Origin).To(Equal(domain.ResourceOriginDirect))

			info, err := gitManager.GetSkillResourceInfo(repoName+"/git-skill", "scripts/tool.py")
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Writable).To(BeFalse())
			Expect(info.Origin).To(Equal(domain.ResourceOriginDirect))
		})

		It("should discover imported resources within git repo boundaries as read-only", func() {
			repoName := "demo-repo"
			skillPath := filepath.Join(tempDir, repoName, "plugins", "planner")
			sharedDir := filepath.Join(tempDir, repoName, "shared")

			Expect(os.MkdirAll(skillPath, 0755)).To(Succeed())
			Expect(os.MkdirAll(sharedDir, 0755)).To(Succeed())

			skillMdContent := `---
name: planner
description: A git-backed skill with shared imports
---
# Planner
[Shared Context](../../shared/context.md)
`
			Expect(os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillMdContent), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(sharedDir, "context.md"), []byte("# Shared"), 0644)).To(Succeed())

			manager.UpdateGitRepos([]string{repoName})
			resources, err := manager.ListSkillResources(repoName + "/planner")
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(HaveLen(1))
			Expect(resources[0].Path).To(Equal("imports/shared/context.md"))
			Expect(resources[0].Origin).To(Equal(domain.ResourceOriginImported))
			Expect(resources[0].Writable).To(BeFalse())
		})

		It("should discover shared plugin agents and prompts as imported read-only prompts", func() {
			repoName := "demo-repo"
			skillPath := filepath.Join(tempDir, repoName, "plugins", "agent-teams", "skills", "planner")
			sharedAgentsDir := filepath.Join(tempDir, repoName, "plugins", "agent-teams", "agents")
			sharedPromptsDir := filepath.Join(tempDir, repoName, "prompts")

			Expect(os.MkdirAll(skillPath, 0755)).To(Succeed())
			Expect(os.MkdirAll(sharedAgentsDir, 0755)).To(Succeed())
			Expect(os.MkdirAll(sharedPromptsDir, 0755)).To(Succeed())

			skillMdContent := `---
name: planner
description: Plugin skill importing shared prompts
---
# Planner
[Team Coach](../../agents/team-coach.md)
[Global System](../../../../prompts/global-system.md)
`
			Expect(os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillMdContent), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(sharedAgentsDir, "team-coach.md"), []byte("# Team Coach"), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(sharedPromptsDir, "global-system.md"), []byte("# Global System"), 0644)).To(Succeed())

			manager.UpdateGitRepos([]string{repoName})
			resources, err := manager.ListSkillResources(repoName + "/planner")
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(HaveLen(2))

			expectedByPath := map[string]domain.ResourceType{
				"imports/plugins/agent-teams/agents/team-coach.md": domain.ResourceTypePrompt,
				"imports/prompts/global-system.md":                 domain.ResourceTypePrompt,
			}
			for _, resource := range resources {
				expectedType, exists := expectedByPath[resource.Path]
				Expect(exists).To(BeTrue(), "unexpected resource path %s", resource.Path)
				Expect(resource.Origin).To(Equal(domain.ResourceOriginImported))
				Expect(resource.Writable).To(BeFalse())
				Expect(resource.Type).To(Equal(expectedType))
			}
		})

		It("should discover shared plugin agents without explicit SKILL imports", func() {
			repoName := "demo-repo"
			skillPath := filepath.Join(tempDir, repoName, "plugins", "kubernetes-operations", "skills", "k8s-manifest-generator")
			sharedAgentsDir := filepath.Join(tempDir, repoName, "plugins", "kubernetes-operations", "agents")

			Expect(os.MkdirAll(skillPath, 0755)).To(Succeed())
			Expect(os.MkdirAll(sharedAgentsDir, 0755)).To(Succeed())

			skillMdContent := `---
name: k8s-manifest-generator
description: Plugin skill with sibling shared agents
---
# Kubernetes Manifest Generator
No explicit imports in this skill.
`
			Expect(os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillMdContent), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(sharedAgentsDir, "kubernetes-architect.md"), []byte("# Kubernetes Architect"), 0644)).To(Succeed())

			manager.UpdateGitRepos([]string{repoName})
			resources, err := manager.ListSkillResources(repoName + "/k8s-manifest-generator")
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(HaveLen(1))

			resource := resources[0]
			Expect(resource.Path).To(Equal("imports/plugins/kubernetes-operations/agents/kubernetes-architect.md"))
			Expect(resource.Type).To(Equal(domain.ResourceTypePrompt))
			Expect(resource.Origin).To(Equal(domain.ResourceOriginImported))
			Expect(resource.Writable).To(BeFalse())
		})
	})
})
