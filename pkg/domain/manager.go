package domain

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SkillManager defines the interface for managing skills
type SkillManager interface {
	ListSkills() ([]Skill, error)
	ReadSkill(name string) (*Skill, error)
	SearchSkills(query string) ([]Skill, error)
	RebuildIndex() error

	// Resource management methods
	ListSkillResources(skillID string) ([]SkillResource, error)
	ReadSkillResource(skillID, resourcePath string) (*ResourceContent, error)
	GetSkillResourceInfo(skillID, resourcePath string) (*SkillResource, error)
}

// FileSystemManager implements SkillManager using the file system
type FileSystemManager struct {
	skillsDir             string
	searcher              *Searcher
	gitRepos              []string // List of git repo directory names (for read-only detection)
	enableImportDiscovery bool
}

// NewFileSystemManager creates a new FileSystemManager
func NewFileSystemManager(skillsDir string, gitRepos []string) (*FileSystemManager, error) {
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create skills directory: %w", err)
	}

	searcher, err := NewSearcher(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create searcher: %w", err)
	}

	manager := &FileSystemManager{
		skillsDir:             skillsDir,
		searcher:              searcher,
		gitRepos:              gitRepos,
		enableImportDiscovery: true,
	}

	// Initial index build
	if err := manager.RebuildIndex(); err != nil {
		return nil, fmt.Errorf("failed to build initial index: %w", err)
	}

	return manager, nil
}

// isGitRepoPath checks if a path is within a git repository directory
func (m *FileSystemManager) isGitRepoPath(path string) bool {
	relPath, err := filepath.Rel(m.skillsDir, path)
	if err != nil {
		return false
	}

	// Check if path starts with any git repo name
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) > 0 {
		for _, repoName := range m.gitRepos {
			if parts[0] == repoName {
				return true
			}
		}
	}
	return false
}

// findSkillDirs recursively finds all directories containing SKILL.md files
func (m *FileSystemManager) findSkillDirs(root string, basePath string) ([]string, error) {
	var skillDirs []string

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(root, entry.Name())

		// Check if this directory contains SKILL.md
		skillMdPath := filepath.Join(entryPath, "SKILL.md")
		if _, err := os.Stat(skillMdPath); err == nil {
			// Found a skill directory
			relPath, _ := filepath.Rel(basePath, entryPath)
			skillDirs = append(skillDirs, relPath)
		}

		// Recursively search subdirectories (for git repos)
		subDirs, err := m.findSkillDirs(entryPath, basePath)
		if err == nil {
			skillDirs = append(skillDirs, subDirs...)
		}
	}

	return skillDirs, nil
}

// ListSkills returns all skills (local and from git repos)
func (m *FileSystemManager) ListSkills() ([]Skill, error) {
	var skills []Skill

	// Find all directories containing SKILL.md
	skillDirs, err := m.findSkillDirs(m.skillsDir, m.skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find skill directories: %w", err)
	}

	for _, skillDir := range skillDirs {
		// Determine skill name and read-only status
		skillPath := filepath.Join(m.skillsDir, skillDir)

		// Check if this is from a git repo by checking if the path starts with a repo name
		relPath, err := filepath.Rel(m.skillsDir, skillPath)
		if err != nil {
			continue
		}
		parts := strings.Split(relPath, string(filepath.Separator))

		// Check if this skill is from a git repo (path has multiple parts and first part is a repo name)
		if len(parts) > 1 {
			repoName := parts[0]
			repoEnabled := false
			for _, enabledRepoName := range m.gitRepos {
				if enabledRepoName == repoName {
					repoEnabled = true
					break
				}
			}
			// Skip skills from disabled repos
			if !repoEnabled {
				continue
			}
		}

		isReadOnly := m.isGitRepoPath(skillPath)

		var skillName string
		if isReadOnly {
			// For git repo skills, use repoName/directoryName format
			if len(parts) >= 2 {
				// Extract repo name and skill directory name
				repoName := parts[0]
				skillDirName := parts[len(parts)-1]
				skillName = fmt.Sprintf("%s/%s", repoName, skillDirName)
			} else {
				skillName = skillDir
			}
		} else {
			// For local skills, use directory name
			skillName = filepath.Base(skillDir)
		}

		skill, err := m.readSkillFromPath(skillPath, skillName, isReadOnly)
		if err != nil {
			// Skip skills that can't be read
			continue
		}
		skills = append(skills, *skill)
	}

	return skills, nil
}

// readSkillFromPath reads a skill from a directory path
func (m *FileSystemManager) readSkillFromPath(skillPath, skillName string, isReadOnly bool) (*Skill, error) {
	skillMdPath := filepath.Join(skillPath, "SKILL.md")
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	metadata, contentStr, err := ParseFrontmatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Validate that name in frontmatter matches directory name
	dirName := filepath.Base(skillPath)
	if metadata.Name != dirName {
		return nil, fmt.Errorf("skill name in frontmatter (%s) does not match directory name (%s)", metadata.Name, dirName)
	}

	return &Skill{
		Name:       skillName,
		ID:         skillName, // ID is the same as Name - the identifier to use when reading
		Content:    contentStr,
		Metadata:   metadata,
		SourcePath: skillPath,
		ReadOnly:   isReadOnly,
	}, nil
}

// findSkillDirByName recursively finds a skill directory by name within a base path
func (m *FileSystemManager) findSkillDirByName(basePath, targetName string) (string, error) {
	var foundPath string
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}
		if !info.IsDir() {
			return nil
		}
		// Check if this directory contains SKILL.md and matches the target name
		skillMdPath := filepath.Join(path, "SKILL.md")
		if _, err := os.Stat(skillMdPath); err == nil {
			dirName := filepath.Base(path)
			if dirName == targetName {
				foundPath = path
				return filepath.SkipAll // Found it, stop walking
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if foundPath == "" {
		return "", fmt.Errorf("skill directory not found: %s", targetName)
	}
	return foundPath, nil
}

// ReadSkill reads a skill by name (supports both local skills and git repo skills with repoName/skillName format)
func (m *FileSystemManager) ReadSkill(name string) (*Skill, error) {
	// Check if this is a git repo skill (format: repoName/skillName)
	if strings.Contains(name, "/") {
		parts := strings.Split(name, "/")
		if len(parts) == 2 {
			repoName := parts[0]
			skillDirName := parts[1]
			repoPath := filepath.Join(m.skillsDir, repoName)

			// Check if repo directory exists
			if _, err := os.Stat(repoPath); err != nil {
				return nil, fmt.Errorf("skill not found: %s", name)
			}

			// Recursively search for the skill directory within the repo
			skillPath, err := m.findSkillDirByName(repoPath, skillDirName)
			if err != nil {
				return nil, fmt.Errorf("skill not found: %s", name)
			}

			return m.readSkillFromPath(skillPath, name, true)
		}
	}

	// Local skill - look for directory with this name
	skillPath := filepath.Join(m.skillsDir, name)
	skillMdPath := filepath.Join(skillPath, "SKILL.md")

	if _, err := os.Stat(skillMdPath); err != nil {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	return m.readSkillFromPath(skillPath, name, false)
}

// SearchSkills searches for skills matching the query
func (m *FileSystemManager) SearchSkills(query string) ([]Skill, error) {
	results, err := m.searcher.Search(query)
	if err != nil {
		return nil, err
	}

	// Read full skill content for each result
	var skills []Skill
	for _, result := range results {
		skill, err := m.ReadSkill(result.Name)
		if err != nil {
			// Skip skills that can't be read
			continue
		}
		skills = append(skills, *skill)
	}

	return skills, nil
}

// RebuildIndex rebuilds the search index
func (m *FileSystemManager) RebuildIndex() error {
	skills, err := m.ListSkills()
	if err != nil {
		return err
	}

	return m.searcher.IndexSkills(skills)
}

// GetSkillsDir returns the skills directory path
func (m *FileSystemManager) GetSkillsDir() string {
	return m.skillsDir
}

// UpdateGitRepos updates the list of git repository names for read-only detection
func (m *FileSystemManager) UpdateGitRepos(gitRepoNames []string) {
	m.gitRepos = gitRepoNames
}

// SetImportDiscoveryEnabled toggles import discovery and virtual imports/... read support.
func (m *FileSystemManager) SetImportDiscoveryEnabled(enabled bool) {
	m.enableImportDiscovery = enabled
}

// getSkillPath returns the full path to a skill directory given its ID
func (m *FileSystemManager) getSkillPath(skillID string) (string, error) {
	// Check if this is a git repo skill (format: repoName/skillName)
	if strings.Contains(skillID, "/") {
		parts := strings.Split(skillID, "/")
		if len(parts) == 2 {
			repoName := parts[0]
			skillDirName := parts[1]
			repoPath := filepath.Join(m.skillsDir, repoName)

			// Recursively search for the skill directory within the repo
			skillPath, err := m.findSkillDirByName(repoPath, skillDirName)
			if err != nil {
				return "", fmt.Errorf("skill not found: %s", skillID)
			}
			return skillPath, nil
		}
	}

	// Local skill
	skillPath := filepath.Join(m.skillsDir, skillID)
	skillMdPath := filepath.Join(skillPath, "SKILL.md")
	if _, err := os.Stat(skillMdPath); err != nil {
		return "", fmt.Errorf("skill not found: %s", skillID)
	}
	return skillPath, nil
}

// ListSkillResources lists all resources in a skill's optional directories
func (m *FileSystemManager) ListSkillResources(skillID string) ([]SkillResource, error) {
	skillPath, err := m.getSkillPath(skillID)
	if err != nil {
		return nil, err
	}
	writable := !m.isGitRepoPath(skillPath)
	allowedRoot := m.getSkillAllowedRoot(skillPath)

	var resources []SkillResource
	resourceDirs := []string{"scripts", "references", "assets", "agents", "prompts"}

	for _, dir := range resourceDirs {
		dirPath := filepath.Join(skillPath, dir)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			// Directory doesn't exist, skip
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				// Recursively list subdirectories
				subResources, err := m.listResourcesInDir(skillPath, filepath.Join(dir, entry.Name()), writable)
				if err == nil {
					resources = append(resources, subResources...)
				}
				continue
			}

			resourcePath := filepath.Join(dir, entry.Name())
			fullPath := filepath.Join(skillPath, resourcePath)

			info, err := entry.Info()
			if err != nil {
				continue
			}

			// Read file to detect MIME type
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}

			mimeType := DetectMimeType(entry.Name(), content)
			readable := IsTextFile(mimeType)

			resources = append(resources, SkillResource{
				Type:     GetResourceType(resourcePath),
				Origin:   ResourceOriginDirect,
				Path:     filepath.ToSlash(resourcePath), // Use forward slashes for consistency
				Name:     entry.Name(),
				Size:     info.Size(),
				MimeType: mimeType,
				Readable: readable,
				Writable: writable,
				Modified: info.ModTime(),
			})
		}
	}

	if m.enableImportDiscovery {
		resources = append(resources, m.listImportedSkillResources(skillPath, allowedRoot)...)
	}
	resources = dedupeSkillResourcesByCanonicalTarget(resources, skillPath, allowedRoot)
	sortSkillResources(resources)

	return resources, nil
}

func (m *FileSystemManager) getSkillAllowedRoot(skillPath string) string {
	if !m.isGitRepoPath(skillPath) {
		return skillPath
	}

	relPath, err := filepath.Rel(m.skillsDir, skillPath)
	if err != nil {
		return skillPath
	}

	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) == 0 || parts[0] == "" || parts[0] == "." || parts[0] == ".." {
		return skillPath
	}

	return filepath.Join(m.skillsDir, parts[0])
}

// listResourcesInDir recursively lists resources in a subdirectory
func (m *FileSystemManager) listResourcesInDir(skillPath, relPath string, writable bool) ([]SkillResource, error) {
	var resources []SkillResource
	fullPath := filepath.Join(skillPath, relPath)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Recursively list subdirectories
			subResources, err := m.listResourcesInDir(skillPath, filepath.Join(relPath, entry.Name()), writable)
			if err == nil {
				resources = append(resources, subResources...)
			}
			continue
		}

		resourcePath := filepath.Join(relPath, entry.Name())
		fullResourcePath := filepath.Join(skillPath, resourcePath)

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Read file to detect MIME type
		content, err := os.ReadFile(fullResourcePath)
		if err != nil {
			continue
		}

		mimeType := DetectMimeType(entry.Name(), content)
		readable := IsTextFile(mimeType)

		resources = append(resources, SkillResource{
			Type:     GetResourceType(resourcePath),
			Origin:   ResourceOriginDirect,
			Path:     filepath.ToSlash(resourcePath),
			Name:     entry.Name(),
			Size:     info.Size(),
			MimeType: mimeType,
			Readable: readable,
			Writable: writable,
			Modified: info.ModTime(),
		})
	}

	return resources, nil
}

func (m *FileSystemManager) listImportedSkillResources(skillPath, allowedRoot string) []SkillResource {
	skillMarkdownPath := filepath.Join(skillPath, "SKILL.md")
	markdownBytes, err := os.ReadFile(skillMarkdownPath)
	if err != nil {
		return nil
	}

	markdown := string(markdownBytes)
	if _, content, err := ParseFrontmatter(markdown); err == nil {
		markdown = content
	}

	candidates := ParseImportCandidates(markdown)
	if len(candidates) == 0 {
		return nil
	}

	resources := make([]SkillResource, 0, len(candidates))
	for _, candidate := range candidates {
		resolvedImport, err := ResolveImportTarget(skillMarkdownPath, candidate, allowedRoot)
		if err != nil {
			continue
		}

		resource, err := buildSkillResource(resolvedImport.TargetPath, resolvedImport.VirtualPath, ResourceOriginImported, false)
		if err != nil {
			continue
		}
		resources = append(resources, *resource)
	}

	return resources
}

func buildSkillResource(fullPath, resourcePath string, origin ResourceOrigin, writable bool) (*SkillResource, error) {
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() {
		return nil, fmt.Errorf("resource path points to a directory, not a file")
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	normalizedPath := filepath.ToSlash(resourcePath)
	mimeType := DetectMimeType(filepath.Base(filepath.FromSlash(normalizedPath)), content)
	readable := IsTextFile(mimeType)

	return &SkillResource{
		Type:     GetResourceType(normalizedPath),
		Origin:   origin,
		Path:     normalizedPath,
		Name:     filepath.Base(filepath.FromSlash(normalizedPath)),
		Size:     fileInfo.Size(),
		MimeType: mimeType,
		Readable: readable,
		Writable: writable,
		Modified: fileInfo.ModTime(),
	}, nil
}

func dedupeSkillResourcesByCanonicalTarget(resources []SkillResource, skillPath, allowedRoot string) []SkillResource {
	if len(resources) == 0 {
		return resources
	}

	resourcesByCanonicalTarget := make(map[string]SkillResource, len(resources))
	for _, resource := range resources {
		canonicalTargetPath, err := canonicalResourceTargetPath(resource, skillPath, allowedRoot)
		if err != nil {
			canonicalTargetPath = string(resource.Origin) + ":" + filepath.ToSlash(resource.Path)
		}

		existingResource, exists := resourcesByCanonicalTarget[canonicalTargetPath]
		if !exists {
			resourcesByCanonicalTarget[canonicalTargetPath] = resource
			continue
		}

		if shouldReplaceResource(existingResource, resource) {
			resourcesByCanonicalTarget[canonicalTargetPath] = resource
		}
	}

	dedupedResources := make([]SkillResource, 0, len(resourcesByCanonicalTarget))
	for _, resource := range resourcesByCanonicalTarget {
		dedupedResources = append(dedupedResources, resource)
	}
	return dedupedResources
}

func canonicalResourceTargetPath(resource SkillResource, skillPath, allowedRoot string) (string, error) {
	targetPath, _, err := resolveSkillResourcePath(skillPath, allowedRoot, resource.Path)
	if err != nil {
		return "", err
	}
	return canonicalizeExistingPath(targetPath)
}

func resolveSkillResourcePath(skillPath, allowedRoot, resourcePath string) (string, ResourceOrigin, error) {
	normalizedPath := filepath.ToSlash(strings.TrimSpace(resourcePath))
	if IsImportedResourcePath(normalizedPath) {
		targetPath, err := resolveImportedVirtualResourcePath(normalizedPath, allowedRoot)
		if err != nil {
			return "", ResourceOriginImported, err
		}
		return targetPath, ResourceOriginImported, nil
	}

	targetPath := filepath.Join(skillPath, filepath.FromSlash(normalizedPath))
	return targetPath, ResourceOriginDirect, nil
}

func resolveImportedVirtualResourcePath(resourcePath, allowedRoot string) (string, error) {
	importedPath := filepath.ToSlash(strings.TrimSpace(resourcePath))
	if !strings.HasPrefix(importedPath, resourceDirImports) {
		return "", fmt.Errorf("imported resource is missing imports/ prefix")
	}

	relativePath := strings.TrimPrefix(importedPath, resourceDirImports)
	if relativePath == "" {
		return "", fmt.Errorf("resource path points to a directory, not a file")
	}

	canonicalAllowedRoot, err := canonicalizeExistingPath(allowedRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve allowed root: %w", err)
	}

	targetPath := filepath.Join(canonicalAllowedRoot, filepath.FromSlash(relativePath))
	canonicalTargetPath, err := canonicalizeExistingPath(targetPath)
	if err != nil {
		return "", fmt.Errorf("resource not found: %w", err)
	}

	if !isWithinRoot(canonicalTargetPath, canonicalAllowedRoot) {
		return "", fmt.Errorf("resource path escapes allowed root")
	}

	return canonicalTargetPath, nil
}

func shouldReplaceResource(existingResource, candidateResource SkillResource) bool {
	if existingResource.Origin != candidateResource.Origin {
		// Preserve direct resources as the primary virtual path when both map to the same target.
		return existingResource.Origin == ResourceOriginImported && candidateResource.Origin == ResourceOriginDirect
	}

	if existingResource.Path != candidateResource.Path {
		return candidateResource.Path < existingResource.Path
	}

	return candidateResource.Name < existingResource.Name
}

func sortSkillResources(resources []SkillResource) {
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Path != resources[j].Path {
			return resources[i].Path < resources[j].Path
		}
		if resources[i].Origin != resources[j].Origin {
			return resources[i].Origin < resources[j].Origin
		}
		return resources[i].Name < resources[j].Name
	})
}

// ReadSkillResource reads the content of a skill resource file
func (m *FileSystemManager) ReadSkillResource(skillID, resourcePath string) (*ResourceContent, error) {
	normalizedPath := filepath.ToSlash(strings.TrimSpace(resourcePath))
	if IsImportedResourcePath(normalizedPath) && !m.enableImportDiscovery {
		return nil, fmt.Errorf("import discovery is disabled")
	}

	// Validate path
	if err := ValidateReadableResourcePath(resourcePath); err != nil {
		return nil, err
	}

	skillPath, err := m.getSkillPath(skillID)
	if err != nil {
		return nil, err
	}

	allowedRoot := m.getSkillAllowedRoot(skillPath)
	fullPath, _, err := resolveSkillResourcePath(skillPath, allowedRoot, resourcePath)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource: %w", err)
	}

	mimeType := DetectMimeType(filepath.Base(filepath.FromSlash(normalizedPath)), content)
	readable := IsTextFile(mimeType)

	var encoding string
	var contentStr string

	if readable {
		// Try to decode as UTF-8
		contentStr = string(content)
		encoding = "utf-8"
	} else {
		// Encode as base64 for binary files
		contentStr = base64.StdEncoding.EncodeToString(content)
		encoding = "base64"
	}

	return &ResourceContent{
		Content:  contentStr,
		Encoding: encoding,
		MimeType: mimeType,
		Size:     int64(len(content)),
	}, nil
}

// GetSkillResourceInfo gets metadata about a specific resource without reading content
func (m *FileSystemManager) GetSkillResourceInfo(skillID, resourcePath string) (*SkillResource, error) {
	normalizedPath := filepath.ToSlash(strings.TrimSpace(resourcePath))
	if IsImportedResourcePath(normalizedPath) && !m.enableImportDiscovery {
		return nil, fmt.Errorf("import discovery is disabled")
	}

	// Validate path
	if err := ValidateReadableResourcePath(resourcePath); err != nil {
		return nil, err
	}

	skillPath, err := m.getSkillPath(skillID)
	if err != nil {
		return nil, err
	}

	allowedRoot := m.getSkillAllowedRoot(skillPath)
	fullPath, origin, err := resolveSkillResourcePath(skillPath, allowedRoot, resourcePath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("resource not found: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("resource path points to a directory, not a file")
	}

	// Read a small portion to detect MIME type
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open resource: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	mimeType := DetectMimeType(filepath.Base(filepath.FromSlash(normalizedPath)), buffer[:n])
	readable := IsTextFile(mimeType)
	writable := !m.isGitRepoPath(skillPath) && origin == ResourceOriginDirect

	return &SkillResource{
		Type:     GetResourceType(normalizedPath),
		Origin:   origin,
		Path:     normalizedPath,
		Name:     filepath.Base(filepath.FromSlash(normalizedPath)),
		Size:     info.Size(),
		MimeType: mimeType,
		Readable: readable,
		Writable: writable,
		Modified: info.ModTime(),
	}, nil
}
