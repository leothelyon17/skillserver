package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mudler/skillserver/pkg/domain"
)

var _ = Describe("Catalog Contracts and Classifier Rules", func() {
	Context("Catalog classifier contract", func() {
		It("should validate known classifier values", func() {
			Expect(domain.CatalogClassifierSkill.IsValid()).To(BeTrue())
			Expect(domain.CatalogClassifierPrompt.IsValid()).To(BeTrue())
			Expect(domain.CatalogClassifierRule.IsValid()).To(BeTrue())
			Expect(domain.CatalogClassifier("unknown").IsValid()).To(BeFalse())
		})

		It("should parse classifier input safely", func() {
			parsed, err := domain.ParseCatalogClassifier("  Prompt ")
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed).To(Equal(domain.CatalogClassifierPrompt))

			parsed, err = domain.ParseCatalogClassifier(" Rule ")
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed).To(Equal(domain.CatalogClassifierRule))

			_, err = domain.ParseCatalogClassifier("skills")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Classifier helper rules", func() {
		It("should always classify SKILL.md as skill", func() {
			classifier, ok := domain.ClassifyCatalogPath("SKILL.md", nil)
			Expect(ok).To(BeTrue())
			Expect(classifier).To(Equal(domain.CatalogClassifierSkill))

			classifier, ok = domain.ClassifyCatalogPath("imports/prompts/SKILL.md", []string{"prompts"})
			Expect(ok).To(BeTrue())
			Expect(classifier).To(Equal(domain.CatalogClassifierSkill))
		})

		It("should classify markdown files in allowed prompt directories as prompt", func() {
			testCases := []string{
				"prompts/system.md",
				"agents/coach.markdown",
				"imports/plugins/agent-teams/agents/coach.md",
				"imports/prompts/GLOBAL-SYSTEM.MD",
			}

			for _, candidate := range testCases {
				classifier, ok := domain.ClassifyCatalogPath(candidate, nil)
				Expect(ok).To(BeTrue(), "expected %s to be classifiable", candidate)
				Expect(classifier).To(Equal(domain.CatalogClassifierPrompt), "expected %s to classify as prompt", candidate)
				Expect(domain.IsPromptCatalogCandidate(candidate, nil)).To(BeTrue(), "expected %s to be a prompt candidate", candidate)
			}
		})

		It("should classify markdown files in allowed rule directories or filename allowlist as rule", func() {
			testCases := []string{
				"rules/project-guidelines.md",
				"rule/conventions.markdown",
				"imports/plugins/agent-teams/rules/project.md",
				"AGENTS.md",
				"imports/GEMINI.md",
			}

			for _, candidate := range testCases {
				classifier, ok := domain.ClassifyCatalogPath(candidate, nil)
				Expect(ok).To(BeTrue(), "expected %s to be classifiable", candidate)
				Expect(classifier).To(Equal(domain.CatalogClassifierRule), "expected %s to classify as rule", candidate)
				Expect(domain.IsRuleCatalogCandidate(candidate, nil, nil)).To(BeTrue(), "expected %s to be a rule candidate", candidate)
			}
		})

		It("should reject non-markdown prompt directory files", func() {
			classifier, ok := domain.ClassifyCatalogPath("prompts/system.txt", nil)
			Expect(ok).To(BeFalse())
			Expect(classifier).To(BeEmpty())
			Expect(domain.IsPromptCatalogCandidate("prompts/system.txt", nil)).To(BeFalse())
		})

		It("should reject look-alike path segments and extension mismatches", func() {
			invalidCases := []string{
				"agentic/system.md",
				"prompting/system.md",
				"prompts-v2/system.md",
				"imports/plugins/agent-teams/resources/coach.md",
			}

			for _, candidate := range invalidCases {
				classifier, ok := domain.ClassifyCatalogPath(candidate, nil)
				Expect(ok).To(BeFalse(), "expected %s not to be classifiable", candidate)
				Expect(classifier).To(BeEmpty())
				Expect(domain.IsPromptCatalogCandidate(candidate, nil)).To(BeFalse(), "expected %s not to be a prompt candidate", candidate)
			}
		})

		It("should reject non-markdown or unallowlisted rule files", func() {
			invalidCases := []string{
				"rules/project.txt",
				"governance/project.md",
				"docs/AGENTS.md.backup",
				"imports/plugins/agent-teams/references/project.md",
			}

			for _, candidate := range invalidCases {
				classifier, ok := domain.ClassifyCatalogPath(candidate, nil)
				Expect(ok).To(BeFalse(), "expected %s not to be classifiable", candidate)
				Expect(classifier).To(BeEmpty())
				Expect(domain.IsRuleCatalogCandidate(candidate, nil, nil)).To(BeFalse(), "expected %s not to be a rule candidate", candidate)
			}
		})

		It("should honor configurable prompt directory allowlist", func() {
			classifier, ok := domain.ClassifyCatalogPath("prompts/system.md", []string{"agents"})
			Expect(ok).To(BeFalse())
			Expect(classifier).To(BeEmpty())

			classifier, ok = domain.ClassifyCatalogPath("nested/agents/system.md", []string{"agents"})
			Expect(ok).To(BeTrue())
			Expect(classifier).To(Equal(domain.CatalogClassifierPrompt))
		})

		It("should honor configurable rule directory and filename allowlists", func() {
			classifier, ok := domain.ClassifyCatalogPathWithAllowlists(
				"rules/project.md",
				nil,
				[]string{"governance"},
				[]string{"CODEOWNERS.md"},
			)
			Expect(ok).To(BeFalse())
			Expect(classifier).To(BeEmpty())

			classifier, ok = domain.ClassifyCatalogPathWithAllowlists(
				"governance/project.md",
				nil,
				[]string{"governance"},
				[]string{"CODEOWNERS.md"},
			)
			Expect(ok).To(BeTrue())
			Expect(classifier).To(Equal(domain.CatalogClassifierRule))

			classifier, ok = domain.ClassifyCatalogPathWithAllowlists(
				"CODEOWNERS.md",
				nil,
				[]string{"governance"},
				[]string{"CODEOWNERS.md"},
			)
			Expect(ok).To(BeTrue())
			Expect(classifier).To(Equal(domain.CatalogClassifierRule))
		})

		It("should normalize allowlist entries and remove invalid values", func() {
			normalized := domain.NormalizePromptDirectoryAllowlist([]string{" Prompts ", "prompts", "agents", "nested/path", "", " /agent/ "})
			Expect(normalized).To(Equal([]string{"prompts", "agents", "agent"}))
		})

		It("should normalize rule allowlist entries and remove invalid values", func() {
			normalizedDirs := domain.NormalizeRuleDirectoryAllowlist([]string{" Rules ", "rules", "rule", "nested/path", "", " /governance/ "})
			Expect(normalizedDirs).To(Equal([]string{"rules", "rule", "governance"}))

			normalizedFiles := domain.NormalizeRuleFilenameAllowlist([]string{" AGENTS.md ", "agents.md", "CLAUDE.md", "configs/GEMINI.md", "", "RULES.MD"})
			Expect(normalizedFiles).To(Equal([]string{"agents.md", "claude.md", "rules.md"}))
		})

		It("should return a defensive copy for default allowlist", func() {
			defaults := domain.DefaultPromptDirectoryAllowlist()
			defaults[0] = "changed"

			freshDefaults := domain.DefaultPromptDirectoryAllowlist()
			Expect(freshDefaults[0]).To(Equal("agent"))
		})

		It("should return defensive copies for default rule allowlists", func() {
			dirs := domain.DefaultRuleDirectoryAllowlist()
			dirs[0] = "changed"

			freshDirs := domain.DefaultRuleDirectoryAllowlist()
			Expect(freshDirs[0]).To(Equal("rule"))

			files := domain.DefaultRuleFilenameAllowlist()
			files[0] = "changed.md"

			freshFiles := domain.DefaultRuleFilenameAllowlist()
			Expect(freshFiles[0]).To(Equal("agents.md"))
		})
	})

	Context("Deterministic key and ID helpers", func() {
		It("should build stable skill IDs across canonical-equivalent forms", func() {
			idA := domain.BuildSkillCatalogItemID("repo/skill-name")
			idB := domain.BuildSkillCatalogItemID("./repo\\skill-name/")
			Expect(idA).To(Equal(idB))
			Expect(idA).To(Equal("skill:repo/skill-name"))
		})

		It("should build stable prompt IDs across normalized path variants", func() {
			idA := domain.BuildPromptCatalogItemID("repo/skill-name", "imports/prompts/system.md")
			idB := domain.BuildPromptCatalogItemID("./repo\\skill-name", "./imports\\prompts\\system.md")
			Expect(idA).To(Equal(idB))
			Expect(idA).To(Equal("prompt:repo/skill-name:imports/prompts/system.md"))
		})

		It("should build canonical prompt keys usable for dedupe", func() {
			keyA := domain.CanonicalPromptCatalogKey("repo/skill-name", "imports/./prompts/system.md")
			keyB := domain.CanonicalPromptCatalogKey("repo\\skill-name", "imports/prompts/system.md")
			Expect(keyA).To(Equal("repo/skill-name:imports/prompts/system.md"))
			Expect(keyA).To(Equal(keyB))

			differentResourceKey := domain.CanonicalPromptCatalogKey("repo/skill-name", "imports/prompts/assistant.md")
			Expect(differentResourceKey).NotTo(Equal(keyA))
		})

		It("should build stable rule IDs across normalized path variants", func() {
			idA := domain.BuildRuleCatalogItemID("repo/skill-name", "rules/project.md")
			idB := domain.BuildRuleCatalogItemID("./repo\\skill-name", "./rules\\project.md")
			Expect(idA).To(Equal(idB))
			Expect(idA).To(Equal("rule:repo/skill-name:rules/project.md"))
		})
	})

	Context("Install metadata parsing and validation", func() {
		It("should parse valid materialize metadata from frontmatter", func() {
			content := `---
name: project-guidelines
description: Contributor rules
materialize:
  target_path: ./AGENTS.md
  conflict_policy: overwrite
---
# Rules
Follow these rules.
`
			metadata, err := domain.ParseCatalogInstallMetadata(content)
			Expect(err).NotTo(HaveOccurred())
			Expect(metadata).NotTo(BeNil())
			Expect(metadata.Materialize.TargetPath).To(Equal("AGENTS.md"))
			Expect(metadata.Materialize.ConflictPolicy).To(Equal(domain.CatalogMaterializeConflictPolicyOverwrite))
		})

		It("should keep existing behavior when frontmatter is malformed", func() {
			content := `---
name: [invalid
---
# Rules
`
			metadata, err := domain.ParseCatalogInstallMetadata(content)
			Expect(err).NotTo(HaveOccurred())
			Expect(metadata).To(BeNil())
		})

		It("should reject invalid target paths", func() {
			content := `---
materialize:
  target_path: ../AGENTS.md
---
# Rules
`
			metadata, err := domain.ParseCatalogInstallMetadata(content)
			Expect(err).To(HaveOccurred())
			Expect(metadata).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("target_path"))
		})

		It("should reject unsupported conflict policies", func() {
			content := `---
materialize:
  conflict_policy: replace
---
# Rules
`
			metadata, err := domain.ParseCatalogInstallMetadata(content)
			Expect(err).To(HaveOccurred())
			Expect(metadata).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("conflict_policy"))
		})

		It("should reject non-map materialize frontmatter values", func() {
			content := `---
materialize: AGENTS.md
---
# Rules
`
			metadata, err := domain.ParseCatalogInstallMetadata(content)
			Expect(err).To(HaveOccurred())
			Expect(metadata).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("materialize must be a map"))
		})
	})
})
