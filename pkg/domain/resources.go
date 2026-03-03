package domain

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"
)

// ResourceType represents the type of resource directory
type ResourceType string

const (
	ResourceTypeScript    ResourceType = "script"
	ResourceTypeReference ResourceType = "reference"
	ResourceTypeAsset     ResourceType = "asset"
	ResourceTypePrompt    ResourceType = "prompt"
)

// ResourceOrigin represents how a resource is discovered.
type ResourceOrigin string

const (
	ResourceOriginDirect   ResourceOrigin = "direct"
	ResourceOriginImported ResourceOrigin = "imported"
)

// SkillResource represents a resource file in a skill
type SkillResource struct {
	Type     ResourceType
	Origin   ResourceOrigin // direct or imported
	Path     string         // Relative path from skill root (e.g., "scripts/script.py")
	Name     string         // Filename only
	Size     int64          // File size in bytes
	MimeType string         // MIME type
	Readable bool           // true if text file, false if binary
	Writable bool           // true if resource can be modified
	Modified time.Time      // Last modification time
}

// ResourceContent represents the content of a resource
type ResourceContent struct {
	Content  string // UTF-8 for text, base64 for binary
	Encoding string // "utf-8" or "base64"
	MimeType string
	Size     int64
}

const (
	resourceDirScripts    = "scripts/"
	resourceDirReferences = "references/"
	resourceDirAssets     = "assets/"
	resourceDirAgents     = "agents/"
	resourceDirPrompts    = "prompts/"
	resourceDirImports    = "imports/"
)

var writableResourcePrefixes = []string{
	resourceDirScripts,
	resourceDirReferences,
	resourceDirAssets,
	resourceDirAgents,
	resourceDirPrompts,
}

var readableResourcePrefixes = []string{
	resourceDirScripts,
	resourceDirReferences,
	resourceDirAssets,
	resourceDirAgents,
	resourceDirPrompts,
	resourceDirImports,
}

// ValidateResourcePath validates a resource path
func ValidateResourcePath(path string) error {
	return validateResourcePath(path, writableResourcePrefixes)
}

// ValidateReadableResourcePath validates a path that may include virtual imported resources.
func ValidateReadableResourcePath(path string) error {
	return validateResourcePath(path, readableResourcePrefixes)
}

// IsImportedResourcePath reports whether the path targets a virtual imported resource.
func IsImportedResourcePath(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	return strings.HasPrefix(path, resourceDirImports)
}

func validateResourcePath(path string, allowedPrefixes []string) error {
	path = filepath.ToSlash(strings.TrimSpace(path))

	if path == "" {
		return fmt.Errorf("resource path cannot be empty")
	}

	// Check for absolute paths
	if filepath.IsAbs(path) || strings.HasPrefix(path, "/") {
		return fmt.Errorf("resource path must be relative")
	}

	// Check for path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("resource path cannot contain '..'")
	}

	// Must start with one of the allowed directory prefixes
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return nil
		}
	}

	return fmt.Errorf("resource path must start with %s", strings.Join(allowedPrefixes, ", "))
}

// GetResourceType determines the resource type from a path
func GetResourceType(path string) ResourceType {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if strings.HasPrefix(path, resourceDirScripts) {
		return ResourceTypeScript
	}
	if strings.HasPrefix(path, resourceDirReferences) {
		return ResourceTypeReference
	}
	if strings.HasPrefix(path, resourceDirAgents) || strings.HasPrefix(path, resourceDirPrompts) {
		return ResourceTypePrompt
	}
	if strings.HasPrefix(path, resourceDirAssets) {
		return ResourceTypeAsset
	}
	if strings.HasPrefix(path, resourceDirImports) {
		importRelativePath := strings.TrimPrefix(path, resourceDirImports)
		normalizedImportPath := "/" + strings.TrimPrefix(importRelativePath, "/")
		if strings.Contains(normalizedImportPath, "/"+strings.TrimSuffix(resourceDirAgents, "/")+"/") ||
			strings.Contains(normalizedImportPath, "/"+strings.TrimSuffix(resourceDirPrompts, "/")+"/") {
			return ResourceTypePrompt
		}
		return ResourceTypeReference
	}
	return ResourceTypeAsset // Default fallback
}

// IsTextFile determines if a file is text-based based on MIME type
func IsTextFile(mimeType string) bool {
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	// Common text-based MIME types
	textMimeTypes := []string{
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-sh",
		"application/x-python",
		"application/x-yaml",
		"application/x-toml",
	}
	for _, t := range textMimeTypes {
		if mimeType == t {
			return true
		}
	}
	return false
}

// DetectMimeType detects MIME type from file extension and content
func DetectMimeType(filename string, content []byte) string {
	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		// Try to detect from content
		if len(content) > 0 {
			// Check for text file signatures
			if isTextContent(content) {
				// Default to text/plain for unknown text files
				return "text/plain"
			}
		}
		return "application/octet-stream"
	}
	return mimeType
}

// isTextContent checks if content appears to be text
func isTextContent(content []byte) bool {
	// Check first 512 bytes for null bytes
	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}
	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return false
		}
	}
	return true
}
