package domain_test

import (
	"os"
	"path/filepath"
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mudler/skillserver/pkg/domain"
)

var _ = Describe("Catalog Builder Integration", func() {
	var (
		manager *domain.FileSystemManager
		tempDir string
		err     error
	)

	BeforeEach(func() {
		tempDir, err = os.MkdirTemp("", "skillserver-catalog-manager-test")
		Expect(err).NotTo(HaveOccurred())

		manager, err = domain.NewFileSystemManager(tempDir, []string{})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	catalogItemsByID := func(items []domain.CatalogItem) map[string]domain.CatalogItem {
		result := make(map[string]domain.CatalogItem, len(items))
		for _, item := range items {
			result[item.ID] = item
		}
		return result
	}

	sortedIDs := func(items []domain.CatalogItem) []string {
		ids := make([]string, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		sort.Strings(ids)
		return ids
	}

	It("should emit deterministic mixed catalog output with prompt metadata and dedupe", func() {
		skillPath := filepath.Join(tempDir, "planner")
		Expect(os.MkdirAll(filepath.Join(skillPath, "prompts"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(skillPath, "agents"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(skillPath, "references"), 0755)).To(Succeed())

		skillMarkdown := `---
name: planner
description: Planning skill
---
# Planner
Git planner catalog skill
[System Prompt](prompts/system.md)
@/prompts/system.md
[Coach Prompt](agents/coach.md)
[General Context](references/guide.md)
`
		Expect(os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillMarkdown), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(skillPath, "prompts", "system.md"), []byte("# System Prompt\nApply deterministic guardrails."), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(skillPath, "agents", "coach.md"), []byte("# Coach Prompt\nCoaching template."), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(skillPath, "references", "guide.md"), []byte("# Guide\nReference material."), 0644)).To(Succeed())

		firstCatalog, err := manager.ListCatalogItems()
		Expect(err).NotTo(HaveOccurred())

		secondCatalog, err := manager.ListCatalogItems()
		Expect(err).NotTo(HaveOccurred())
		Expect(secondCatalog).To(Equal(firstCatalog))

		Expect(firstCatalog).To(HaveLen(3))
		byID := catalogItemsByID(firstCatalog)

		skillID := domain.BuildSkillCatalogItemID("planner")
		systemPromptID := domain.BuildPromptCatalogItemID("planner", "prompts/system.md")
		coachPromptID := domain.BuildPromptCatalogItemID("planner", "agents/coach.md")

		Expect(byID).To(HaveKey(skillID))
		Expect(byID).To(HaveKey(systemPromptID))
		Expect(byID).To(HaveKey(coachPromptID))
		Expect(byID).NotTo(HaveKey(domain.BuildPromptCatalogItemID("planner", "imports/prompts/system.md")))

		skillItem := byID[skillID]
		Expect(skillItem.Classifier).To(Equal(domain.CatalogClassifierSkill))
		Expect(skillItem.Name).To(Equal("planner"))

		systemPrompt := byID[systemPromptID]
		Expect(systemPrompt.Classifier).To(Equal(domain.CatalogClassifierPrompt))
		Expect(systemPrompt.ParentSkillID).To(Equal("planner"))
		Expect(systemPrompt.ResourcePath).To(Equal("prompts/system.md"))
		Expect(systemPrompt.Content).To(ContainSubstring("deterministic guardrails"))

		coachPrompt := byID[coachPromptID]
		Expect(coachPrompt.ParentSkillID).To(Equal("planner"))
		Expect(coachPrompt.ResourcePath).To(Equal("agents/coach.md"))

		Expect(manager.RebuildIndex()).To(Succeed())
		promptClassifier := domain.CatalogClassifierPrompt
		firstSearch, err := manager.SearchCatalogItems("guardrails", &promptClassifier)
		Expect(err).NotTo(HaveOccurred())
		Expect(firstSearch).To(HaveLen(1))
		Expect(firstSearch[0].ID).To(Equal(systemPromptID))

		Expect(manager.RebuildIndex()).To(Succeed())
		secondSearch, err := manager.SearchCatalogItems("guardrails", &promptClassifier)
		Expect(err).NotTo(HaveOccurred())
		Expect(sortedIDs(firstSearch)).To(Equal(sortedIDs(secondSearch)))
	})

	It("should include imported git prompt resources and keep skill search compatibility", func() {
		repoName := "demo-repo"
		skillPath := filepath.Join(tempDir, repoName, "plugins", "agent-teams", "skills", "planner")
		sharedAgentsPath := filepath.Join(tempDir, repoName, "plugins", "agent-teams", "agents")
		sharedPromptsPath := filepath.Join(tempDir, repoName, "prompts")

		Expect(os.MkdirAll(skillPath, 0755)).To(Succeed())
		Expect(os.MkdirAll(sharedAgentsPath, 0755)).To(Succeed())
		Expect(os.MkdirAll(sharedPromptsPath, 0755)).To(Succeed())

		skillMarkdown := `---
name: planner
description: Git planner skill
---
# Planner
Git planner catalog skill
[Team Coach](../../agents/team-coach.md)
[Global System](../../../../prompts/global-system.md)
`
		Expect(os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillMarkdown), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(sharedAgentsPath, "team-coach.md"), []byte("# Team Coach\nImported coaching prompt."), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(sharedPromptsPath, "global-system.md"), []byte("# Global System\nGlobal arbitration guardrails."), 0644)).To(Succeed())

		manager.UpdateGitRepos([]string{repoName})

		catalogItems, err := manager.ListCatalogItems()
		Expect(err).NotTo(HaveOccurred())
		Expect(catalogItems).To(HaveLen(3))

		skillID := "demo-repo/planner"
		skillCatalogID := domain.BuildSkillCatalogItemID(skillID)
		teamCoachPromptID := domain.BuildPromptCatalogItemID(skillID, "imports/plugins/agent-teams/agents/team-coach.md")
		globalPromptID := domain.BuildPromptCatalogItemID(skillID, "imports/prompts/global-system.md")

		byID := catalogItemsByID(catalogItems)
		Expect(byID).To(HaveKey(skillCatalogID))
		Expect(byID).To(HaveKey(teamCoachPromptID))
		Expect(byID).To(HaveKey(globalPromptID))

		skillItem := byID[skillCatalogID]
		Expect(skillItem.ReadOnly).To(BeTrue())
		Expect(skillItem.Classifier).To(Equal(domain.CatalogClassifierSkill))

		teamCoachItem := byID[teamCoachPromptID]
		Expect(teamCoachItem.ReadOnly).To(BeTrue())
		Expect(teamCoachItem.ParentSkillID).To(Equal(skillID))
		Expect(teamCoachItem.ResourcePath).To(Equal("imports/plugins/agent-teams/agents/team-coach.md"))

		globalPromptItem := byID[globalPromptID]
		Expect(globalPromptItem.ReadOnly).To(BeTrue())
		Expect(globalPromptItem.ParentSkillID).To(Equal(skillID))
		Expect(globalPromptItem.ResourcePath).To(Equal("imports/prompts/global-system.md"))

		Expect(manager.RebuildIndex()).To(Succeed())
		promptClassifier := domain.CatalogClassifierPrompt
		promptResults, err := manager.SearchCatalogItems("arbitration guardrails", &promptClassifier)
		Expect(err).NotTo(HaveOccurred())
		Expect(promptResults).To(HaveLen(1))
		Expect(promptResults[0].ID).To(Equal(globalPromptID))

		skillResults, err := manager.SearchSkills("Git planner catalog")
		Expect(err).NotTo(HaveOccurred())
		Expect(skillResults).To(HaveLen(1))
		Expect(skillResults[0].ID).To(Equal(skillID))
	})

	It("should honor runtime prompt catalog enablement and directory allowlist", func() {
		skillPath := filepath.Join(tempDir, "planner")
		Expect(os.MkdirAll(filepath.Join(skillPath, "prompts"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(skillPath, "agents"), 0755)).To(Succeed())

		skillMarkdown := `---
name: planner
description: Planning skill
---
# Planner
Catalog runtime config test skill
`
		Expect(os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillMarkdown), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(skillPath, "prompts", "system.md"), []byte("# System Prompt"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(skillPath, "agents", "coach.md"), []byte("# Coach Prompt"), 0644)).To(Succeed())

		manager.SetPromptCatalogEnabled(false)
		manager.SetPromptCatalogDirectoryAllowlist([]string{"agents"})

		catalogItems, err := manager.ListCatalogItems()
		Expect(err).NotTo(HaveOccurred())
		Expect(catalogItems).To(HaveLen(1))
		Expect(catalogItems[0].Classifier).To(Equal(domain.CatalogClassifierSkill))

		manager.SetPromptCatalogEnabled(true)
		manager.SetPromptCatalogDirectoryAllowlist([]string{"agents"})

		catalogItems, err = manager.ListCatalogItems()
		Expect(err).NotTo(HaveOccurred())
		Expect(catalogItems).To(HaveLen(2))

		byID := catalogItemsByID(catalogItems)
		Expect(byID).To(HaveKey(domain.BuildSkillCatalogItemID("planner")))
		Expect(byID).To(HaveKey(domain.BuildPromptCatalogItemID("planner", "agents/coach.md")))
		Expect(byID).NotTo(HaveKey(domain.BuildPromptCatalogItemID("planner", "prompts/system.md")))
	})
})
