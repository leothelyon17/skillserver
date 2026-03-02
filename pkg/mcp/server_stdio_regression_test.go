package mcp

import (
	"context"
	"fmt"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mudler/skillserver/pkg/domain"
)

func TestMCPServer_StdioRegression(t *testing.T) {
	t.Run("registers legacy stdio tool set", func(t *testing.T) {
		server := NewServer(newFakeSkillManager())
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		tools, err := session.ListTools(context.Background(), nil)
		if err != nil {
			t.Fatalf("list tools failed: %v", err)
		}

		expectedTools := []string{
			"list_skills",
			"read_skill",
			"search_skills",
			"list_skill_resources",
			"read_skill_resource",
			"get_skill_resource_info",
		}

		registered := make(map[string]struct{}, len(tools.Tools))
		for _, tool := range tools.Tools {
			registered[tool.Name] = struct{}{}
		}

		for _, expected := range expectedTools {
			if _, ok := registered[expected]; !ok {
				t.Fatalf("expected tool %q to be registered", expected)
			}
		}
	})

	t.Run("invokes list and read tools end-to-end", func(t *testing.T) {
		manager := newFakeSkillManager()
		server := NewServer(manager)
		session, cleanup := connectMCPClientSession(t, server)
		defer cleanup()

		listResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name: "list_skills",
		})
		if err != nil {
			t.Fatalf("list_skills call failed: %v", err)
		}
		if listResult.IsError {
			t.Fatalf("list_skills returned tool error")
		}

		listStructured, ok := listResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected list_skills structured content map, got %T", listResult.StructuredContent)
		}

		rawSkills, ok := listStructured["skills"].([]any)
		if !ok || len(rawSkills) == 0 {
			t.Fatalf("expected non-empty skills list, got %#v", listStructured["skills"])
		}

		firstSkill, ok := rawSkills[0].(map[string]any)
		if !ok {
			t.Fatalf("expected first skill object, got %T", rawSkills[0])
		}

		skillID, _ := firstSkill["id"].(string)
		if skillID != manager.skill.ID {
			t.Fatalf("expected skill id %q, got %q", manager.skill.ID, skillID)
		}

		readResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
			Name:      "read_skill",
			Arguments: map[string]any{"id": manager.skill.ID},
		})
		if err != nil {
			t.Fatalf("read_skill call failed: %v", err)
		}
		if readResult.IsError {
			t.Fatalf("read_skill returned tool error")
		}

		readStructured, ok := readResult.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("expected read_skill structured content map, got %T", readResult.StructuredContent)
		}

		content, _ := readStructured["content"].(string)
		if content != manager.skill.Content {
			t.Fatalf("expected read content %q, got %q", manager.skill.Content, content)
		}
	})
}

func connectMCPClientSession(t *testing.T, server *Server) (*mcpsdk.ClientSession, func()) {
	t.Helper()

	ctx := context.Background()
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := server.mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server session: %v", err)
	}

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatalf("failed to connect client session: %v", err)
	}

	cleanup := func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	}
	return clientSession, cleanup
}

type fakeSkillManager struct {
	skill domain.Skill
}

func newFakeSkillManager() *fakeSkillManager {
	return &fakeSkillManager{
		skill: domain.Skill{
			ID:      "sample-skill",
			Name:    "sample-skill",
			Content: "# Sample Skill\n\nSample skill content.",
			Metadata: &domain.SkillMetadata{
				Name:        "sample-skill",
				Description: "Sample skill used for MCP regression tests",
			},
		},
	}
}

func (m *fakeSkillManager) ListSkills() ([]domain.Skill, error) {
	return []domain.Skill{m.skill}, nil
}

func (m *fakeSkillManager) ReadSkill(name string) (*domain.Skill, error) {
	if name != m.skill.ID {
		return nil, fmt.Errorf("skill not found: %s", name)
	}
	skill := m.skill
	return &skill, nil
}

func (m *fakeSkillManager) SearchSkills(query string) ([]domain.Skill, error) {
	if strings.Contains(m.skill.ID, query) ||
		strings.Contains(m.skill.Name, query) ||
		strings.Contains(m.skill.Content, query) {
		return []domain.Skill{m.skill}, nil
	}
	return []domain.Skill{}, nil
}

func (m *fakeSkillManager) RebuildIndex() error {
	return nil
}

func (m *fakeSkillManager) ListSkillResources(skillID string) ([]domain.SkillResource, error) {
	return []domain.SkillResource{}, nil
}

func (m *fakeSkillManager) ReadSkillResource(skillID, resourcePath string) (*domain.ResourceContent, error) {
	return nil, fmt.Errorf("resource not found: %s", resourcePath)
}

func (m *fakeSkillManager) GetSkillResourceInfo(skillID, resourcePath string) (*domain.SkillResource, error) {
	return nil, fmt.Errorf("resource not found: %s", resourcePath)
}
