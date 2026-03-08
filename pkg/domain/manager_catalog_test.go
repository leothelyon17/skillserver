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
		Expect(manager.Close()).To(Succeed())
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

	It("should classify sibling plugin agents as prompt catalog items without explicit imports", func() {
		repoName := "demo-repo"
		skillPath := filepath.Join(tempDir, repoName, "plugins", "kubernetes-operations", "skills", "k8s-manifest-generator")
		sharedAgentsPath := filepath.Join(tempDir, repoName, "plugins", "kubernetes-operations", "agents")

		Expect(os.MkdirAll(skillPath, 0755)).To(Succeed())
		Expect(os.MkdirAll(sharedAgentsPath, 0755)).To(Succeed())

		skillMarkdown := `---
name: k8s-manifest-generator
description: Git kubernetes manifest skill
---
# Kubernetes Manifest Generator
No explicit shared prompt imports.
`
		Expect(os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillMarkdown), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(sharedAgentsPath, "kubernetes-architect.md"), []byte("# Kubernetes Architect\nShared specialist prompt."), 0644)).To(Succeed())

		manager.UpdateGitRepos([]string{repoName})

		catalogItems, err := manager.ListCatalogItems()
		Expect(err).NotTo(HaveOccurred())
		Expect(catalogItems).To(HaveLen(2))

		skillID := "demo-repo/k8s-manifest-generator"
		skillCatalogID := domain.BuildSkillCatalogItemID(skillID)
		promptPath := "imports/plugins/kubernetes-operations/agents/kubernetes-architect.md"
		promptCatalogID := domain.BuildPromptCatalogItemID(skillID, promptPath)

		byID := catalogItemsByID(catalogItems)
		Expect(byID).To(HaveKey(skillCatalogID))
		Expect(byID).To(HaveKey(promptCatalogID))

		promptItem := byID[promptCatalogID]
		Expect(promptItem.Classifier).To(Equal(domain.CatalogClassifierPrompt))
		Expect(promptItem.ParentSkillID).To(Equal(skillID))
		Expect(promptItem.ResourcePath).To(Equal(promptPath))
		Expect(promptItem.Content).To(ContainSubstring("Shared specialist prompt"))
		Expect(promptItem.ReadOnly).To(BeTrue())
	})

	It("should dedupe shared implicit prompts across multiple skills by prompt metadata", func() {
		repoName := "demo-repo"
		sharedAgentsPath := filepath.Join(tempDir, repoName, "plugins", "kubernetes-operations", "agents")
		skillPaths := []string{
			filepath.Join(tempDir, repoName, "plugins", "kubernetes-operations", "skills", "k8s-manifest-generator"),
			filepath.Join(tempDir, repoName, "plugins", "kubernetes-operations", "skills", "k8s-security-policies"),
		}

		Expect(os.MkdirAll(sharedAgentsPath, 0755)).To(Succeed())
		for _, skillPath := range skillPaths {
			Expect(os.MkdirAll(skillPath, 0755)).To(Succeed())
		}

		manifestSkillMarkdown := `---
name: k8s-manifest-generator
description: Kubernetes manifest generation skill
---
# Kubernetes Manifest Generator
No explicit shared prompt imports.
`
		securitySkillMarkdown := `---
name: k8s-security-policies
description: Kubernetes security policy skill
---
# Kubernetes Security Policies
No explicit shared prompt imports.
`
		Expect(os.WriteFile(filepath.Join(skillPaths[0], "SKILL.md"), []byte(manifestSkillMarkdown), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(skillPaths[1], "SKILL.md"), []byte(securitySkillMarkdown), 0644)).To(Succeed())

		sharedPromptMarkdown := `---
name: kubernetes-architect.md
description: Shared Kubernetes architecture guidance.
---
# Kubernetes Architect
Act as a Kubernetes architecture specialist.
`
		Expect(os.WriteFile(filepath.Join(sharedAgentsPath, "kubernetes-architect.md"), []byte(sharedPromptMarkdown), 0644)).To(Succeed())

		manager.UpdateGitRepos([]string{repoName})

		catalogItems, err := manager.ListCatalogItems()
		Expect(err).NotTo(HaveOccurred())

		var skillItems []domain.CatalogItem
		var promptItems []domain.CatalogItem
		for _, item := range catalogItems {
			switch item.Classifier {
			case domain.CatalogClassifierSkill:
				skillItems = append(skillItems, item)
			case domain.CatalogClassifierPrompt:
				promptItems = append(promptItems, item)
			}
		}

		Expect(skillItems).To(HaveLen(2))
		Expect(promptItems).To(HaveLen(1))

		promptItem := promptItems[0]
		Expect(promptItem.Name).To(Equal("kubernetes-architect.md"))
		Expect(promptItem.Description).To(Equal("Shared Kubernetes architecture guidance."))
		Expect(promptItem.ResourcePath).To(Equal("imports/plugins/kubernetes-operations/agents/kubernetes-architect.md"))
		Expect(promptItem.ParentSkillID).To(Equal("demo-repo/k8s-manifest-generator"))
		Expect(promptItem.Content).To(ContainSubstring("Kubernetes architecture specialist"))
		Expect(promptItem.ReadOnly).To(BeTrue())
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

	It("should discover direct and imported rules and support classifier filtering", func() {
		repoName := "demo-repo"
		skillPath := filepath.Join(tempDir, repoName, "plugins", "rule-teams", "skills", "planner")
		sharedRulesPath := filepath.Join(tempDir, repoName, "plugins", "rule-teams", "rules")
		repoRootRulePath := filepath.Join(tempDir, repoName, "AGENTS.md")
		outsideRulePath := filepath.Join(tempDir, "outside.md")

		Expect(os.MkdirAll(filepath.Join(skillPath, "rules"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(skillPath, "governance"), 0755)).To(Succeed())
		Expect(os.MkdirAll(sharedRulesPath, 0755)).To(Succeed())

		skillMarkdown := `---
name: planner
description: Planner skill
---
# Planner
[Local Rule](rules/local.md)
[Shared Rule](../../rules/team.md)
[Repo Root Rule](/AGENTS.md)
[False Positive](governance/guide.md)
[Escaped Rule](../../../../outside.md)
`
		Expect(os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillMarkdown), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(skillPath, "rules", "local.md"), []byte("# Local Rule\nLocal policy for planner workflows."), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(sharedRulesPath, "team.md"), []byte("# Team Rule\nTeam contributor guardrails."), 0644)).To(Succeed())
		Expect(os.WriteFile(repoRootRulePath, []byte("# AGENTS\nRepository-level contributor guardrails."), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(skillPath, "governance", "guide.md"), []byte("# Governance\nNot an allowlisted rule path."), 0644)).To(Succeed())
		Expect(os.WriteFile(outsideRulePath, []byte("# Outside\nShould never be imported."), 0644)).To(Succeed())

		manager.UpdateGitRepos([]string{repoName})

		catalogItems, err := manager.ListCatalogItems()
		Expect(err).NotTo(HaveOccurred())
		Expect(catalogItems).To(HaveLen(4))

		skillID := "demo-repo/planner"
		skillCatalogID := domain.BuildSkillCatalogItemID(skillID)
		localRuleID := domain.BuildRuleCatalogItemID(skillID, "rules/local.md")
		sharedRuleID := domain.BuildRuleCatalogItemID(skillID, "imports/plugins/rule-teams/rules/team.md")
		repoRootRuleID := domain.BuildRuleCatalogItemID(skillID, "imports/AGENTS.md")

		byID := catalogItemsByID(catalogItems)
		Expect(byID).To(HaveKey(skillCatalogID))
		Expect(byID).To(HaveKey(localRuleID))
		Expect(byID).To(HaveKey(sharedRuleID))
		Expect(byID).To(HaveKey(repoRootRuleID))
		Expect(byID).NotTo(HaveKey(domain.BuildRuleCatalogItemID(skillID, "governance/guide.md")))
		Expect(byID).NotTo(HaveKey(domain.BuildRuleCatalogItemID(skillID, "imports/outside.md")))

		localRule := byID[localRuleID]
		Expect(localRule.Classifier).To(Equal(domain.CatalogClassifierRule))
		Expect(localRule.ParentSkillID).To(Equal(skillID))
		Expect(localRule.ResourcePath).To(Equal("rules/local.md"))
		Expect(localRule.ReadOnly).To(BeTrue())

		repoRootRule := byID[repoRootRuleID]
		Expect(repoRootRule.Classifier).To(Equal(domain.CatalogClassifierRule))
		Expect(repoRootRule.ParentSkillID).To(Equal(skillID))
		Expect(repoRootRule.ResourcePath).To(Equal("imports/AGENTS.md"))
		Expect(repoRootRule.ReadOnly).To(BeTrue())

		Expect(manager.RebuildIndex()).To(Succeed())
		ruleClassifier := domain.CatalogClassifierRule
		ruleResults, err := manager.SearchCatalogItems("contributor guardrails", &ruleClassifier)
		Expect(err).NotTo(HaveOccurred())
		Expect(ruleResults).To(HaveLen(2))
		Expect(sortedIDs(ruleResults)).To(Equal(sortedIDs([]domain.CatalogItem{
			{ID: repoRootRuleID},
			{ID: sharedRuleID},
		})))
	})

	It("should discover shared repo rules as rule catalog items without explicit imports", func() {
		repoName := "demo-repo"
		skillPath := filepath.Join(tempDir, repoName, "skills", "planner")
		sharedRulesPath := filepath.Join(tempDir, repoName, "rules")

		Expect(os.MkdirAll(skillPath, 0755)).To(Succeed())
		Expect(os.MkdirAll(sharedRulesPath, 0755)).To(Succeed())

		skillMarkdown := `---
name: planner
description: Planner skill
---
# Planner
No explicit rule imports.
`
		Expect(os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillMarkdown), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(sharedRulesPath, "coding-standards.md"), []byte("# Coding Standards\nRepository policy for contributors."), 0644)).To(Succeed())

		manager.UpdateGitRepos([]string{repoName})

		catalogItems, err := manager.ListCatalogItems()
		Expect(err).NotTo(HaveOccurred())
		Expect(catalogItems).To(HaveLen(2))

		skillID := "demo-repo/planner"
		skillCatalogID := domain.BuildSkillCatalogItemID(skillID)
		importedRuleID := domain.BuildRuleCatalogItemID(skillID, "imports/rules/coding-standards.md")

		byID := catalogItemsByID(catalogItems)
		Expect(byID).To(HaveKey(skillCatalogID))
		Expect(byID).To(HaveKey(importedRuleID))

		importedRule := byID[importedRuleID]
		Expect(importedRule.Classifier).To(Equal(domain.CatalogClassifierRule))
		Expect(importedRule.ParentSkillID).To(Equal(skillID))
		Expect(importedRule.ResourcePath).To(Equal("imports/rules/coding-standards.md"))
		Expect(importedRule.ReadOnly).To(BeTrue())

		Expect(manager.RebuildIndex()).To(Succeed())
		ruleClassifier := domain.CatalogClassifierRule
		ruleResults, err := manager.SearchCatalogItems("repository policy", &ruleClassifier)
		Expect(err).NotTo(HaveOccurred())
		Expect(ruleResults).To(HaveLen(1))
		Expect(ruleResults[0].ID).To(Equal(importedRuleID))
	})

	It("should honor runtime rule catalog enablement and allowlists", func() {
		skillPath := filepath.Join(tempDir, "planner")
		Expect(os.MkdirAll(filepath.Join(skillPath, "rules"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(skillPath, "governance"), 0755)).To(Succeed())

		skillMarkdown := `---
name: planner
description: Planning skill
---
# Planner
Catalog runtime config test skill
`
		Expect(os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillMarkdown), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(skillPath, "rules", "default.md"), []byte("# Default Rule"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(skillPath, "governance", "guide.md"), []byte("# Governance Rule"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(skillPath, "AGENTS.md"), []byte("# Agents Rule"), 0644)).To(Succeed())

		manager.SetRuleCatalogEnabled(false)
		catalogItems, err := manager.ListCatalogItems()
		Expect(err).NotTo(HaveOccurred())
		Expect(catalogItems).To(HaveLen(1))
		Expect(catalogItems[0].Classifier).To(Equal(domain.CatalogClassifierSkill))

		manager.SetRuleCatalogEnabled(true)
		manager.SetRuleCatalogDirectoryAllowlist([]string{"governance"})
		manager.SetRuleCatalogFilenameAllowlist([]string{"RULES.md"})

		catalogItems, err = manager.ListCatalogItems()
		Expect(err).NotTo(HaveOccurred())
		Expect(catalogItems).To(HaveLen(2))

		byID := catalogItemsByID(catalogItems)
		Expect(byID).To(HaveKey(domain.BuildSkillCatalogItemID("planner")))
		Expect(byID).To(HaveKey(domain.BuildRuleCatalogItemID("planner", "governance/guide.md")))
		Expect(byID).NotTo(HaveKey(domain.BuildRuleCatalogItemID("planner", "rules/default.md")))
		Expect(byID).NotTo(HaveKey(domain.BuildRuleCatalogItemID("planner", "AGENTS.md")))
	})
})
