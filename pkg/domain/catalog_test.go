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
			Expect(domain.CatalogClassifier("unknown").IsValid()).To(BeFalse())
		})

		It("should parse classifier input safely", func() {
			parsed, err := domain.ParseCatalogClassifier("  Prompt ")
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed).To(Equal(domain.CatalogClassifierPrompt))

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

		It("should honor configurable prompt directory allowlist", func() {
			classifier, ok := domain.ClassifyCatalogPath("prompts/system.md", []string{"agents"})
			Expect(ok).To(BeFalse())
			Expect(classifier).To(BeEmpty())

			classifier, ok = domain.ClassifyCatalogPath("nested/agents/system.md", []string{"agents"})
			Expect(ok).To(BeTrue())
			Expect(classifier).To(Equal(domain.CatalogClassifierPrompt))
		})

		It("should normalize allowlist entries and remove invalid values", func() {
			normalized := domain.NormalizePromptDirectoryAllowlist([]string{" Prompts ", "prompts", "agents", "nested/path", "", " /agent/ "})
			Expect(normalized).To(Equal([]string{"prompts", "agents", "agent"}))
		})

		It("should return a defensive copy for default allowlist", func() {
			defaults := domain.DefaultPromptDirectoryAllowlist()
			defaults[0] = "changed"

			freshDefaults := domain.DefaultPromptDirectoryAllowlist()
			Expect(freshDefaults[0]).To(Equal("agent"))
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
	})
})
