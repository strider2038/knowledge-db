package cliapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

type listNodesResponse struct {
	Nodes []struct {
		Path string `json:"path"`
	} `json:"nodes"`
	Total int `json:"total"`
}

type nodeResponse struct {
	Path     string         `json:"path"`
	Metadata map[string]any `json:"metadata"`
	Content  string         `json:"content"`
}

func reindexLinksCmd() *cli.Command {
	return &cli.Command{
		Name:  "reindex-links",
		Usage: "Одноразово обновить и переиндексировать link-узлы через refresh-description",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "base-url",
				Usage: "базовый URL kb serve",
				Value: "http://127.0.0.1:8080",
			},
			&cli.IntFlag{
				Name:  "page-size",
				Usage: "размер страницы для GET /api/nodes (1..200)",
				Value: 200,
			},
			&cli.BoolFlag{
				Name:  "all",
				Usage: "обновлять все link-узлы с source_url (по умолчанию только legacy)",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "только показать, какие узлы будут обработаны",
			},
			&cli.DurationFlag{
				Name:  "delay",
				Usage: "задержка между refresh-запросами (например 300ms)",
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "HTTP timeout на один запрос",
				Value: 60 * time.Second,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			baseURL := cmd.String("base-url")
			pageSize := cmd.Int("page-size")
			all := cmd.Bool("all")
			dryRun := cmd.Bool("dry-run")
			delay := cmd.Duration("delay")
			requestTimeout := cmd.Duration("timeout")
			if pageSize <= 0 || pageSize > 200 {
				return fmt.Errorf("invalid --page-size: expected 1..200, got %d", pageSize)
			}

			baseURL = strings.TrimRight(baseURL, "/")
			if _, err := url.ParseRequestURI(baseURL); err != nil {
				return fmt.Errorf("invalid --base-url: %w", err)
			}

			client := &http.Client{Timeout: requestTimeout}
			paths, err := loadAllLinkPaths(ctx, client, baseURL, pageSize)
			if err != nil {
				return err
			}
			fmt.Printf("Найдено link-узлов: %d\n", len(paths))

			var processed int
			var skipped int
			var failed int

			for _, nodePath := range paths {
				node, err := fetchNode(ctx, client, baseURL, nodePath)
				if err != nil {
					failed++
					fmt.Fprintf(os.Stderr, "[fail] %s: load node: %v\n", nodePath, err)

					continue
				}

				sourceURL, _ := node.Metadata["source_url"].(string)
				if strings.TrimSpace(sourceURL) == "" {
					skipped++
					fmt.Printf("[skip] %s: source_url is empty\n", nodePath)

					continue
				}

				if !all && !shouldRefreshLegacyLink(node) {
					skipped++
					fmt.Printf("[skip] %s: already profiled (use --all to force)\n", nodePath)

					continue
				}

				if dryRun {
					processed++
					fmt.Printf("[dry-run] %s\n", nodePath)

					continue
				}

				if err := refreshNodeDescription(ctx, client, baseURL, nodePath); err != nil {
					failed++
					fmt.Fprintf(os.Stderr, "[fail] %s: refresh: %v\n", nodePath, err)

					continue
				}

				processed++
				fmt.Printf("[ok] %s\n", nodePath)
				if delay > 0 {
					time.Sleep(delay)
				}
			}

			fmt.Printf("\nИтог: processed=%d skipped=%d failed=%d total=%d\n", processed, skipped, failed, len(paths))
			if failed > 0 {
				return fmt.Errorf("completed with %d failures", failed)
			}

			return nil
		},
	}
}

func loadAllLinkPaths(ctx context.Context, client *http.Client, baseURL string, pageSize int) ([]string, error) {
	var out []string
	offset := 0
	for {
		u := fmt.Sprintf("%s/api/nodes?recursive=true&type=link&limit=%d&offset=%d", baseURL, pageSize, offset)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list nodes: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read list nodes response: %w", readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close list nodes response: %w", closeErr)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("list nodes: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var payload listNodesResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("decode list nodes response: %w", err)
		}

		for _, n := range payload.Nodes {
			if strings.TrimSpace(n.Path) != "" {
				out = append(out, n.Path)
			}
		}

		offset += len(payload.Nodes)
		if len(payload.Nodes) == 0 || offset >= payload.Total {
			break
		}
	}

	return out, nil
}

func fetchNode(ctx context.Context, client *http.Client, baseURL, nodePath string) (*nodeResponse, error) {
	u, err := nodePathURL(baseURL, nodePath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read node response: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close node response: %w", closeErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get node: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var node nodeResponse
	if err := json.Unmarshal(body, &node); err != nil {
		return nil, fmt.Errorf("decode node response: %w", err)
	}

	return &node, nil
}

func refreshNodeDescription(ctx context.Context, client *http.Client, baseURL, nodePath string) error {
	u, err := nodePathURL(baseURL, nodePath)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u+"/refresh-description", nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post refresh-description: %w", err)
	}

	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		return fmt.Errorf("read refresh response: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close refresh response: %w", closeErr)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func nodePathURL(baseURL, nodePath string) (string, error) {
	parts := strings.Split(nodePath, "/")
	escaped := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(p))
	}
	if len(escaped) == 0 {
		return "", errors.New("empty node path")
	}

	return baseURL + path.Join("/api/nodes", strings.Join(escaped, "/")), nil
}

func shouldRefreshLegacyLink(node *nodeResponse) bool {
	if node == nil {
		return false
	}

	sourceKind, _ := node.Metadata["source_kind"].(string)
	contentProfile, _ := node.Metadata["content_profile"].(string)
	if strings.TrimSpace(sourceKind) == "" || strings.TrimSpace(contentProfile) == "" {
		return true
	}

	return false
}
