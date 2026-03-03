package domain_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mudler/skillserver/pkg/domain"
)

var _ = Describe("Resource Imports", func() {
	var (
		tempDir string
		err     error
	)

	BeforeEach(func() {
		tempDir, err = os.MkdirTemp("", "skillserver-imports-test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Context("ParseImportCandidates", func() {
		It("should return empty results for empty markdown", func() {
			Expect(domain.ParseImportCandidates("")).To(BeEmpty())
		})

		It("should extract deterministic candidates from markdown links and include tokens", func() {
			markdown := `
# Skill
[Guide](references/guide.md)
[Prompt](./prompts/system.md "system")
[SameDir](same-dir.md)
@references/shared/context.md
@<references/trimmed.md>,
@/shared/policy.md
[WithFragment](references/guide.md#section)
[External](https://example.com/reference.md)
@owner
@references/shared/context.md
`

			firstResult := domain.ParseImportCandidates(markdown)
			secondResult := domain.ParseImportCandidates(markdown)

			Expect(firstResult).To(Equal([]string{
				"/shared/policy.md",
				"prompts/system.md",
				"references/guide.md",
				"references/shared/context.md",
				"references/trimmed.md",
				"same-dir.md",
			}))
			Expect(secondResult).To(Equal(firstResult))
		})

		It("should ignore malformed or non-file candidates", func() {
			markdown := `
[Broken](
[Anchor](#section)
[Web](https://example.com/readme.md)
mailto:test@example.com
@owner
@http://example.com/file.md
`

			candidates := domain.ParseImportCandidates(markdown)
			Expect(candidates).To(BeEmpty())
		})
	})

	Context("ResolveImportTarget", func() {
		It("should reject invalid import candidates early", func() {
			skillRoot := filepath.Join(tempDir, "skill")
			sourcePath := filepath.Join(skillRoot, "SKILL.md")
			Expect(os.MkdirAll(skillRoot, 0755)).To(Succeed())
			Expect(os.WriteFile(sourcePath, []byte("# Skill"), 0644)).To(Succeed())

			_, err := domain.ResolveImportTarget(sourcePath, "https://example.com/file.md", skillRoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid import candidate"))
		})

		It("should resolve local-skill imports within the skill root", func() {
			skillRoot := filepath.Join(tempDir, "local-skill")
			sourcePath := filepath.Join(skillRoot, "SKILL.md")
			targetPath := filepath.Join(skillRoot, "references", "guide.md")

			Expect(os.MkdirAll(filepath.Dir(targetPath), 0755)).To(Succeed())
			Expect(os.WriteFile(sourcePath, []byte("# Skill"), 0644)).To(Succeed())
			Expect(os.WriteFile(targetPath, []byte("# Guide"), 0644)).To(Succeed())

			resolved, err := domain.ResolveImportTarget(sourcePath, "references/guide.md", skillRoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved.Candidate).To(Equal("references/guide.md"))
			Expect(resolved.TargetPath).To(Equal(filepath.Clean(targetPath)))
			Expect(resolved.VirtualPath).To(Equal("imports/references/guide.md"))

			rootRelativeResolved, err := domain.ResolveImportTarget(sourcePath, "/references/guide.md", skillRoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(rootRelativeResolved.TargetPath).To(Equal(filepath.Clean(targetPath)))
		})

		It("should enforce git-repo root boundaries for nested skills", func() {
			repoRoot := filepath.Join(tempDir, "demo-repo")
			skillRoot := filepath.Join(repoRoot, "skills", "triage")
			sourcePath := filepath.Join(skillRoot, "SKILL.md")
			sharedPrompt := filepath.Join(repoRoot, "prompts", "shared.md")

			Expect(os.MkdirAll(filepath.Dir(sourcePath), 0755)).To(Succeed())
			Expect(os.MkdirAll(filepath.Dir(sharedPrompt), 0755)).To(Succeed())
			Expect(os.WriteFile(sourcePath, []byte("# Skill"), 0644)).To(Succeed())
			Expect(os.WriteFile(sharedPrompt, []byte("# Shared"), 0644)).To(Succeed())

			_, err := domain.ResolveImportTarget(sourcePath, "../../prompts/shared.md", skillRoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("escapes allowed root"))

			resolved, err := domain.ResolveImportTarget(sourcePath, "../../prompts/shared.md", repoRoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved.TargetPath).To(Equal(filepath.Clean(sharedPrompt)))
			Expect(resolved.VirtualPath).To(Equal("imports/prompts/shared.md"))
		})

		It("should reject traversal and non-file targets", func() {
			skillRoot := filepath.Join(tempDir, "skill")
			sourcePath := filepath.Join(skillRoot, "SKILL.md")
			outsidePath := filepath.Join(tempDir, "outside.md")
			directoryPath := filepath.Join(skillRoot, "references")

			Expect(os.MkdirAll(directoryPath, 0755)).To(Succeed())
			Expect(os.WriteFile(sourcePath, []byte("# Skill"), 0644)).To(Succeed())
			Expect(os.WriteFile(outsidePath, []byte("outside"), 0644)).To(Succeed())

			_, err := domain.ResolveImportTarget(sourcePath, "../outside.md", skillRoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("escapes allowed root"))

			_, err = domain.ResolveImportTarget(sourcePath, "references", skillRoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not a file"))
		})

		It("should reject source files outside the allowed root", func() {
			allowedRoot := filepath.Join(tempDir, "allowed")
			sourceRoot := filepath.Join(tempDir, "source")
			sourcePath := filepath.Join(sourceRoot, "SKILL.md")

			Expect(os.MkdirAll(allowedRoot, 0755)).To(Succeed())
			Expect(os.MkdirAll(sourceRoot, 0755)).To(Succeed())
			Expect(os.WriteFile(sourcePath, []byte("# Skill"), 0644)).To(Succeed())

			_, err := domain.ResolveImportTarget(sourcePath, "doc.md", allowedRoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("source path is outside allowed root"))
		})

		It("should return an error when import target does not exist", func() {
			skillRoot := filepath.Join(tempDir, "skill")
			sourcePath := filepath.Join(skillRoot, "SKILL.md")

			Expect(os.MkdirAll(skillRoot, 0755)).To(Succeed())
			Expect(os.WriteFile(sourcePath, []byte("# Skill"), 0644)).To(Succeed())

			_, err := domain.ResolveImportTarget(sourcePath, "missing.md", skillRoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to resolve import target"))
		})

		It("should reject symlink escapes outside the allowed root", func() {
			skillRoot := filepath.Join(tempDir, "skill")
			sourcePath := filepath.Join(skillRoot, "SKILL.md")
			referencesDir := filepath.Join(skillRoot, "references")
			outsideDir := filepath.Join(tempDir, "outside")
			outsidePath := filepath.Join(outsideDir, "secret.md")
			linkPath := filepath.Join(referencesDir, "escape.md")

			Expect(os.MkdirAll(referencesDir, 0755)).To(Succeed())
			Expect(os.MkdirAll(outsideDir, 0755)).To(Succeed())
			Expect(os.WriteFile(sourcePath, []byte("# Skill"), 0644)).To(Succeed())
			Expect(os.WriteFile(outsidePath, []byte("# Secret"), 0644)).To(Succeed())

			if err := os.Symlink(outsidePath, linkPath); err != nil {
				Skip("symlink creation is not supported in this environment")
			}

			_, err := domain.ResolveImportTarget(sourcePath, "references/escape.md", skillRoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("escapes allowed root"))
		})
	})

	Context("BuildImportedVirtualPath", func() {
		It("should generate deterministic imports paths and reject out-of-root targets", func() {
			allowedRoot := filepath.Join(tempDir, "repo")
			targetPath := filepath.Join(allowedRoot, "prompts", "sys.md")
			outsidePath := filepath.Join(tempDir, "outside.md")

			Expect(os.MkdirAll(filepath.Dir(targetPath), 0755)).To(Succeed())
			Expect(os.WriteFile(targetPath, []byte("# Prompt"), 0644)).To(Succeed())
			Expect(os.WriteFile(outsidePath, []byte("# Outside"), 0644)).To(Succeed())

			virtualPath, err := domain.BuildImportedVirtualPath(allowedRoot, targetPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(virtualPath).To(Equal("imports/prompts/sys.md"))

			_, err = domain.BuildImportedVirtualPath(allowedRoot, outsidePath)
			Expect(err).To(HaveOccurred())
		})

		It("should fail when root or target cannot be resolved", func() {
			allowedRoot := filepath.Join(tempDir, "missing-root")
			targetPath := filepath.Join(tempDir, "missing-target.md")

			_, err := domain.BuildImportedVirtualPath(allowedRoot, targetPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to resolve allowed root"))

			validRoot := filepath.Join(tempDir, "repo")
			validTarget := filepath.Join(validRoot, "prompts", "sys.md")
			Expect(os.MkdirAll(filepath.Dir(validTarget), 0755)).To(Succeed())
			Expect(os.WriteFile(validTarget, []byte("# Prompt"), 0644)).To(Succeed())

			_, err = domain.BuildImportedVirtualPath(validRoot, targetPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to resolve target path"))
		})
	})
})
