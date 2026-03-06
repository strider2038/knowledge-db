package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const gitignoreContent = `**/.local/
**/.local/**
`

func initCmd() *cobra.Command {
	var path string
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

			sourceSkill := findSourceSkill()
			if sourceSkill != "" {
				home := os.Getenv("HOME")
				if home == "" {
					return fmt.Errorf("HOME not set")
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
