package mcp

import (
	"context"
	"net/http"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/strider2038/knowledge-db/internal/debugdata"
)

type debugStore interface {
	ListIssues(ctx context.Context, limit int) ([]debugdata.Issue, error)
	ReadIssue(ctx context.Context, issueID string) (debugdata.Issue, error)
	ReadLastTelegramRaw(ctx context.Context, limit int) ([]debugdata.TelegramRawRecord, error)
}

type DebugHandler struct {
	apiKey  string
	handler http.Handler
}

func NewDebugHandler(apiKey string, store debugStore) http.Handler {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "knowledge-db-debug-mcp",
		Version: "1.0.0",
	}, nil)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "debug_list_issues",
		Description: "List recent debug issues",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input struct {
		Limit int `json:"limit,omitempty"`
	}) (*sdkmcp.CallToolResult, any, error) {
		issues, err := store.ListIssues(ctx, input.Limit)
		if err != nil {
			return nil, nil, err
		}
		type item struct {
			ID        string `json:"id"`
			Status    string `json:"status"`
			Title     string `json:"title"`
			Page      string `json:"page"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		}
		out := make([]item, 0, len(issues))
		for _, issue := range issues {
			out = append(out, item{
				ID:        issue.ID,
				Status:    issue.Status,
				Title:     issue.Title,
				Page:      issue.Page,
				CreatedAt: issue.CreatedAt.Format(time.RFC3339),
				UpdatedAt: issue.UpdatedAt.Format(time.RFC3339),
			})
		}

		return &sdkmcp.CallToolResult{}, map[string]any{"issues": out, "total": len(out)}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "debug_get_issue",
		Description: "Read full debug issue by id",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input struct {
		ID string `json:"id"`
	}) (*sdkmcp.CallToolResult, any, error) {
		issue, err := store.ReadIssue(ctx, strings.TrimSpace(input.ID))
		if err != nil {
			return nil, nil, err
		}

		return &sdkmcp.CallToolResult{}, map[string]any{
			"id":          issue.ID,
			"status":      issue.Status,
			"title":       issue.Title,
			"description": issue.Description,
			"page":        issue.Page,
			"created_at":  issue.CreatedAt.Format(time.RFC3339),
			"updated_at":  issue.UpdatedAt.Format(time.RFC3339),
			"context":     issue.Context,
			"body":        issue.Body,
		}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "debug_get_telegram_raw",
		Description: "Read last telegram raw records",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input struct {
		Limit int `json:"limit,omitempty"`
	}) (*sdkmcp.CallToolResult, any, error) {
		records, err := store.ReadLastTelegramRaw(ctx, input.Limit)
		if err != nil {
			return nil, nil, err
		}

		return &sdkmcp.CallToolResult{}, map[string]any{"records": records, "total": len(records)}, nil
	})

	transport := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server {
		return server
	}, &sdkmcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})

	return &DebugHandler{
		apiKey:  strings.TrimSpace(apiKey),
		handler: transport,
	}
}

func (h *DebugHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok || token != h.apiKey {
		w.Header().Set("WWW-Authenticate", `Bearer realm="kb-debug-mcp"`)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}
	h.handler.ServeHTTP(w, r)
}
