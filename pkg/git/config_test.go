package git_test

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mudler/skillserver/pkg/git"
)

var _ = Describe("ConfigManager", func() {
	var (
		configManager *git.ConfigManager
		tempDir       string
		err           error
	)

	BeforeEach(func() {
		tempDir, err = os.MkdirTemp("", "skillserver-config-test")
		Expect(err).NotTo(HaveOccurred())

		configManager = git.NewConfigManager(tempDir)
	})

	AfterEach(func() {
		_ = os.RemoveAll(tempDir)
	})

	Context("LoadConfig", func() {
		It("should return empty list when config file doesn't exist", func() {
			repos, err := configManager.LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(repos).To(BeEmpty())
		})

		It("should normalize legacy URL-only config entries", func() {
			configPath := filepath.Join(tempDir, ".git-repos.json")
			configContent := `[
  {
    "id": "legacy-repo-id",
    "url": "https://GITHUB.com/acme/repo-one.git",
    "name": "repo-one",
    "enabled": true
  }
]`
			err := os.WriteFile(configPath, []byte(configContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			repos, err := configManager.LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(repos).To(HaveLen(1))
			Expect(repos[0].URL).To(Equal("https://github.com/acme/repo-one.git"))
			Expect(repos[0].ID).To(Equal(git.GenerateID("https://github.com/acme/repo-one.git")))
			Expect(repos[0].Name).To(Equal("repo-one"))
			Expect(repos[0].Enabled).To(BeTrue())
			Expect(repos[0].Auth.Mode).To(Equal(git.GitRepoAuthModeNone))
		})

		It("should load disabled repos and preserve enabled state", func() {
			configPath := filepath.Join(tempDir, ".git-repos.json")
			configContent := `[
  {
    "id": "legacy-enabled",
    "url": "https://github.com/acme/enabled.git",
    "name": "enabled",
    "enabled": true
  },
  {
    "id": "legacy-disabled",
    "url": "https://github.com/acme/disabled.git",
    "name": "disabled",
    "enabled": false
  }
]`
			err := os.WriteFile(configPath, []byte(configContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			repos, err := configManager.LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(repos).To(HaveLen(2))
			Expect(repos[0].Enabled).To(BeTrue())
			Expect(repos[1].Enabled).To(BeFalse())
			Expect(repos[0].ID).To(Equal(git.GenerateID(repos[0].URL)))
			Expect(repos[1].ID).To(Equal(git.GenerateID(repos[1].URL)))
			Expect(repos[0].Auth.Mode).To(Equal(git.GitRepoAuthModeNone))
			Expect(repos[1].Auth.Mode).To(Equal(git.GitRepoAuthModeNone))
		})
	})

	Context("SaveConfig", func() {
		It("should save normalized repos to config file", func() {
			repos := []git.GitRepoConfig{
				{
					ID:      "legacy-repo-id",
					URL:     "https://github.com:443/acme/repo1.git",
					Name:    "repo1",
					Enabled: true,
				},
				{
					ID:      "legacy-repo-id-two",
					URL:     "ssh://git@GITHUB.com:22/acme/repo2.git",
					Name:    "repo2",
					Enabled: false,
				},
			}

			err := configManager.SaveConfig(repos)
			Expect(err).NotTo(HaveOccurred())

			// Verify file was created
			configPath := filepath.Join(tempDir, ".git-repos.json")
			Expect(configPath).To(BeAnExistingFile())

			// Verify persisted schema includes auth metadata and normalized URL/id.
			fileBytes, err := os.ReadFile(configPath)
			Expect(err).NotTo(HaveOccurred())
			var persisted []map[string]any
			err = json.Unmarshal(fileBytes, &persisted)
			Expect(err).NotTo(HaveOccurred())
			Expect(persisted).To(HaveLen(2))
			Expect(persisted[0]).To(HaveKey("auth"))
			Expect(persisted[1]).To(HaveKey("auth"))

			// Load and verify migration invariants.
			loadedRepos, err := configManager.LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(loadedRepos).To(HaveLen(2))
			Expect(loadedRepos[0].URL).To(Equal("https://github.com/acme/repo1.git"))
			Expect(loadedRepos[0].ID).To(Equal(git.GenerateID("https://github.com/acme/repo1.git")))
			Expect(loadedRepos[0].Auth.Mode).To(Equal(git.GitRepoAuthModeNone))
			Expect(loadedRepos[1].URL).To(Equal("ssh://git@github.com/acme/repo2.git"))
			Expect(loadedRepos[1].ID).To(Equal(git.GenerateID("ssh://git@github.com/acme/repo2.git")))
			Expect(loadedRepos[1].Auth.Mode).To(Equal(git.GitRepoAuthModeNone))
			Expect(loadedRepos[0].Enabled).To(BeTrue())
			Expect(loadedRepos[1].Enabled).To(BeFalse())
		})

		It("should create directory if it doesn't exist", func() {
			nestedDir := filepath.Join(tempDir, "nested", "path")
			configManager = git.NewConfigManager(nestedDir)

			repos := []git.GitRepoConfig{
				{
					ID:      "repo1",
					URL:     "https://github.com/user/repo1.git",
					Name:    "repo1",
					Enabled: true,
				},
			}

			err := configManager.SaveConfig(repos)
			Expect(err).NotTo(HaveOccurred())

			configPath := filepath.Join(nestedDir, ".git-repos.json")
			Expect(configPath).To(BeAnExistingFile())
		})

		It("should reject URLs containing userinfo before persistence", func() {
			repos := []git.GitRepoConfig{
				{
					ID:      "repo-with-userinfo",
					URL:     "https://token@github.com/acme/private.git",
					Name:    "private",
					Enabled: true,
				},
			}

			err := configManager.SaveConfig(repos)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must not include userinfo"))
		})
	})

	Context("CanonicalizeRepoURL", func() {
		It("should canonicalize HTTPS URLs with default port", func() {
			canonical, err := git.CanonicalizeRepoURL("https://GitHub.com:443/acme/repo.git")
			Expect(err).NotTo(HaveOccurred())
			Expect(canonical).To(Equal("https://github.com/acme/repo.git"))
		})

		It("should canonicalize nested HTTPS paths", func() {
			canonical, err := git.CanonicalizeRepoURL("https://github.com/acme/team/repo.git")
			Expect(err).NotTo(HaveOccurred())
			Expect(canonical).To(Equal("https://github.com/acme/team/repo.git"))
		})

		It("should canonicalize SSH URLs", func() {
			canonical, err := git.CanonicalizeRepoURL("ssh://git@GitHub.com:22/acme/repo.git")
			Expect(err).NotTo(HaveOccurred())
			Expect(canonical).To(Equal("ssh://git@github.com/acme/repo.git"))
		})

		It("should canonicalize SCP-like SSH URLs", func() {
			canonical, err := git.CanonicalizeRepoURL("git@github.com:acme/repo.git")
			Expect(err).NotTo(HaveOccurred())
			Expect(canonical).To(Equal("ssh://git@github.com/acme/repo.git"))
		})

		It("should reject userinfo in HTTPS URLs", func() {
			_, err := git.CanonicalizeRepoURL("https://user:token@github.com/acme/repo.git")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must not include userinfo"))
		})
	})

	Context("ExtractRepoName", func() {
		It("should extract repo name from canonical-equivalent HTTPS forms", func() {
			name := git.ExtractRepoName("https://GitHub.com:443/acme/repo.git")
			Expect(name).To(Equal("repo"))
		})

		It("should extract repo name from canonical-equivalent SSH forms", func() {
			name := git.ExtractRepoName("git@github.com:acme/repo.git")
			Expect(name).To(Equal("repo"))
		})
	})

	Context("GenerateID", func() {
		It("should generate consistent IDs", func() {
			id1 := git.GenerateID("https://github.com/user/repo.git")
			id2 := git.GenerateID("https://github.com/user/repo.git")
			Expect(id1).To(Equal(id2))
		})

		It("should generate stable IDs across canonical-equivalent URLs", func() {
			id1 := git.GenerateID("https://github.com/acme/repo.git")
			id2 := git.GenerateID("https://github.com:443/acme/repo.git")
			Expect(id1).To(Equal(id2))
		})

		It("should generate stable IDs across canonical-equivalent SSH URLs", func() {
			id1 := git.GenerateID("git@github.com:acme/repo.git")
			id2 := git.GenerateID("ssh://git@github.com/acme/repo.git")
			Expect(id1).To(Equal(id2))
		})

		It("should generate different IDs for repos with colliding trailing names", func() {
			id1 := git.GenerateID("https://github.com/acme/repo.git")
			id2 := git.GenerateID("https://github.com/other-team/repo.git")
			Expect(id1).NotTo(Equal(id2))
		})
	})
})
