package cliapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/strider2038/knowledge-db/internal/cliapp/embedskill"
	"github.com/urfave/cli/v3"
)

const gitignoreContent = `**/.local/
**/.local/**

# Embedding index (SQLite)
.kb/

# Obsidian
.obsidian/workspace.json
.obsidian/app.json
.obsidian/appearance.json
.obsidian/themes/
.obsidian/plugins/
.obsidian/snippets/
.trash/

# OS
.DS_Store
Thumbs.db
`

const exampleNodeContent = `---
id: "01900000-0000-7000-8000-000000000001"
keywords: [example]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: "Sample node"
annotation: "Example node to verify KB structure"
---

# Sample node

Replace this content with your own text.
`

const exampleNodeName = "sample-node"

func initCmd() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Инициализировать базу знаний (.gitignore, agent skill в .agents/skills/)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "путь к базе знаний (по умолчанию текущая директория)",
			},
			&cli.BoolFlag{
				Name:  "example",
				Usage: "создать пример узла (example/topic/sample-node.md)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			path := cmd.String("path")
			example := cmd.Bool("example")
			if path == "" {
				path = "."
			}
			basePath, err := absPath(path)
			if err != nil {
				return err
			}
			gitignorePath := filepath.Join(basePath, ".gitignore")
			if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o644); err != nil {
				return fmt.Errorf("write .gitignore: %w", err)
			}
			fmt.Println("Создан .gitignore")

			if example {
				exampleThemeDir := filepath.Join(basePath, "example", "topic")
				if err := os.MkdirAll(exampleThemeDir, 0o755); err != nil {
					return fmt.Errorf("create example dir: %w", err)
				}
				nodeFile := filepath.Join(exampleThemeDir, exampleNodeName+".md")
				if err := os.WriteFile(nodeFile, []byte(exampleNodeContent), 0o644); err != nil {
					return fmt.Errorf("write example node: %w", err)
				}
				fmt.Printf("Создан пример узла: example/topic/%s.md\n", exampleNodeName)
			}

			destFile, err := installKnowledgeDBSkill(basePath)
			if err != nil {
				return err
			}
			fmt.Printf("Skill установлен: %s\n", destFile)

			return nil
		},
	}
}

func installKnowledgeDBSkill(basePath string) (string, error) {
	if len(embedskill.KnowledgeDB) == 0 {
		return "", errors.New("embedded knowledge-db skill template is empty")
	}
	destDir := filepath.Join(basePath, ".agents", "skills", "knowledge-db")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create skill dir: %w", err)
	}
	replaced := strings.ReplaceAll(string(embedskill.KnowledgeDB), "{{DATA_PATH}}", basePath)
	destFile := filepath.Join(destDir, "SKILL.md")
	if err := os.WriteFile(destFile, []byte(replaced), 0o644); err != nil {
		return "", fmt.Errorf("write skill: %w", err)
	}

	return destFile, nil
}
