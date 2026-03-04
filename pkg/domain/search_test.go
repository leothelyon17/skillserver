package domain_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mudler/skillserver/pkg/domain"
)

var _ = Describe("Searcher", func() {
	var (
		searcher *domain.Searcher
		tempDir  string
		err      error
	)

	BeforeEach(func() {
		tempDir, err = os.MkdirTemp("", "skillserver-search-test")
		Expect(err).NotTo(HaveOccurred())

		searcher, err = domain.NewSearcher(tempDir)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if searcher != nil {
			Expect(searcher.Close()).To(Succeed())
		}
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	seedMixedCatalog := func() {
		items := []domain.CatalogItem{
			{
				ID:          domain.BuildSkillCatalogItemID("docker"),
				Classifier:  domain.CatalogClassifierSkill,
				Name:        "docker",
				Description: "Container skills",
				Content:     "Kubernetes orchestration and containerization basics",
				ReadOnly:    false,
			},
			{
				ID:            domain.BuildPromptCatalogItemID("docker", "prompts/assistant.md"),
				Classifier:    domain.CatalogClassifierPrompt,
				Name:          "assistant.md",
				Description:   "Assistant prompt",
				Content:       "Kubernetes orchestration assistant prompt template",
				ParentSkillID: "docker",
				ResourcePath:  "prompts/assistant.md",
				ReadOnly:      true,
			},
		}

		Expect(searcher.IndexCatalogItems(items)).To(Succeed())
	}

	Context("Catalog index and classifier filtering", func() {
		It("should return mixed catalog docs when no classifier filter is provided", func() {
			seedMixedCatalog()

			results, err := searcher.SearchCatalog("orchestration", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))

			foundSkill := false
			foundPrompt := false
			for _, result := range results {
				switch result.Classifier {
				case domain.CatalogClassifierSkill:
					foundSkill = true
				case domain.CatalogClassifierPrompt:
					foundPrompt = true
				}
			}

			Expect(foundSkill).To(BeTrue())
			Expect(foundPrompt).To(BeTrue())
		})

		It("should return only skill docs when classifier is skill", func() {
			seedMixedCatalog()

			classifier := domain.CatalogClassifierSkill
			results, err := searcher.SearchCatalog("orchestration", &classifier)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Classifier).To(Equal(domain.CatalogClassifierSkill))
			Expect(results[0].ID).To(Equal(domain.BuildSkillCatalogItemID("docker")))
		})

		It("should return only prompt docs when classifier is prompt", func() {
			seedMixedCatalog()

			classifier := domain.CatalogClassifierPrompt
			results, err := searcher.SearchCatalog("orchestration", &classifier)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Classifier).To(Equal(domain.CatalogClassifierPrompt))
			Expect(results[0].ID).To(Equal(domain.BuildPromptCatalogItemID("docker", "prompts/assistant.md")))
		})

		It("should return empty results for empty query", func() {
			seedMixedCatalog()

			results, err := searcher.SearchCatalog("   ", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})

		It("should reject invalid classifier filters", func() {
			seedMixedCatalog()

			invalidClassifier := domain.CatalogClassifier("invalid")
			_, err := searcher.SearchCatalog("orchestration", &invalidClassifier)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Backward-compatible skill search behavior", func() {
		It("should keep Search wrapper skill-only with skill IDs", func() {
			seedMixedCatalog()

			results, err := searcher.Search("orchestration")
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].ID).To(Equal("docker"))
			Expect(results[0].Name).To(Equal("docker"))
		})

		It("should index and search legacy skill metadata fields", func() {
			skills := []domain.Skill{
				{
					Name:    "docker",
					ID:      "docker",
					Content: "Containerization basics",
					Metadata: &domain.SkillMetadata{
						Name:          "docker",
						Description:   "Docker guide",
						License:       "MIT",
						Compatibility: "linux",
					},
				},
				{
					Name:    "kubernetes",
					ID:      "kubernetes",
					Content: "Cluster scheduling",
					Metadata: &domain.SkillMetadata{
						Name:        "kubernetes",
						Description: "Kubernetes guide",
						License:     "Apache-2.0",
					},
				},
			}

			Expect(searcher.IndexSkills(skills)).To(Succeed())

			licenseResults, err := searcher.Search("MIT")
			Expect(err).NotTo(HaveOccurred())
			Expect(licenseResults).To(HaveLen(1))
			Expect(licenseResults[0].ID).To(Equal("docker"))

			compatibilityResults, err := searcher.Search("linux")
			Expect(err).NotTo(HaveOccurred())
			Expect(compatibilityResults).To(HaveLen(1))
			Expect(compatibilityResults[0].ID).To(Equal("docker"))
		})

		It("should return empty results for empty query in compatibility wrapper", func() {
			skills := []domain.Skill{
				{
					Name:    "docker",
					ID:      "docker",
					Content: "Containerization basics",
					Metadata: &domain.SkillMetadata{
						Name:        "docker",
						Description: "Docker guide",
					},
				},
			}
			Expect(searcher.IndexSkills(skills)).To(Succeed())

			results, err := searcher.Search("   ")
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})
	})

	Context("Catalog indexing validation", func() {
		It("should reject catalog items with invalid classifier", func() {
			items := []domain.CatalogItem{
				{
					ID:         "invalid-item",
					Classifier: domain.CatalogClassifier("unknown"),
					Name:       "invalid",
					Content:    "content",
				},
			}

			err := searcher.IndexCatalogItems(items)
			Expect(err).To(HaveOccurred())
		})
	})
})
