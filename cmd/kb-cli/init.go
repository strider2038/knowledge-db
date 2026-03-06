package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const gitignoreContent = `**/.local/
**/.local/**

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
keywords: [example]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Пример узла для проверки структуры"
---

# Пример узла

Замените этот контент своим текстом.
`

const exampleNodeName = "sample-node"

func initCmd() *cobra.Command {
	var path string
	var example bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Инициализировать базу знаний (.gitignore, agent skills)",
		RunE: func(cmd *cobra.Command, args []string) error {
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
				examplePath := filepath.Join(basePath, "example", "topic", exampleNodeName)
				if err := os.MkdirAll(examplePath, 0o755); err != nil {
					return fmt.Errorf("create example node: %w", err)
				}
				nodeFile := filepath.Join(examplePath, exampleNodeName+".md")
				if err := os.WriteFile(nodeFile, []byte(exampleNodeContent), 0o644); err != nil {
					return fmt.Errorf("write example node: %w", err)
				}
				fmt.Printf("Создан пример узла: example/topic/%s/%s.md\n", exampleNodeName, exampleNodeName)
			}

			sourceSkill := findSourceSkill()
			if sourceSkill != "" {
				home := os.Getenv("HOME")
				if home == "" {
					return errors.New("HOME not set")
				}
				skillsDest := filepath.Join(home, ".cursor", "skills")
				destSkill := filepath.Join(skillsDest, "knowledge-db")
				_ = os.MkdirAll(destSkill, 0o755)
				skillPath := filepath.Join(sourceSkill, "SKILL.md")
				if _, err := os.Stat(skillPath); err == nil {
					data, _ := os.ReadFile(skillPath)
					replaced := strings.ReplaceAll(string(data), "{{DATA_PATH}}", basePath)
					destFile := filepath.Join(destSkill, "SKILL.md")
					if err := os.WriteFile(destFile, []byte(replaced), 0o644); err != nil {
						return fmt.Errorf("copy skill: %w", err)
					}
					fmt.Printf("Skill скопирован в %s\n", destSkill)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&path, "path", "p", "", "путь к базе знаний (по умолчанию текущая директория)")
	cmd.Flags().BoolVar(&example, "example", false, "создать пример узла (example/topic/example-node/)")

	return cmd
}

func findSourceSkill() string {
	cwd, _ := os.Getwd()
	exec, _ := os.Executable()
	execDir := filepath.Dir(exec)
	candidates := []string{
		filepath.Join(cwd, ".cursor", "skills", "knowledge-db"),
		filepath.Join(execDir, ".cursor", "skills", "knowledge-db"),
		filepath.Join(execDir, "..", ".cursor", "skills", "knowledge-db"),
		filepath.Join(execDir, "..", "..", ".cursor", "skills", "knowledge-db"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "SKILL.md")); err == nil {
			return c
		}
	}

	return ""
}
