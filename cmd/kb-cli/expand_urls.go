package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func expandUrlsCmd() *cobra.Command {
	var path, file string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "expand-urls",
		Short: "Раскрыть редиректные ссылки и убрать UTM/трекинг-параметры в markdown",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return errors.New("--file is required")
			}

			absBase, mdPath, err := resolveMarkdownUnderBase(path, file)
			if err != nil {
				return err
			}
			_ = absBase

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			res, err := kb.WriteExpandURLsFile(ctx, mdPath, dryRun)
			if err != nil {
				return fmt.Errorf("expand-urls: %w", err)
			}

			for _, u := range res.FailedURLs {
				fmt.Fprintf(os.Stderr, "warning: не удалось нормализовать: %s\n", u)
			}

			switch {
			case dryRun:
				for _, p := range res.Pairs {
					fmt.Printf("%s -> %s\n", p.Old, p.New)
				}
				if len(res.Pairs) == 0 && len(res.FailedURLs) == 0 {
					fmt.Println("нет изменений")
				}
			case res.Changed:
				fmt.Printf("OK: записано замен: %d\n", res.Replacements)
			default:
				fmt.Println("OK: изменений нет")
			}

			if len(res.FailedURLs) > 0 {
				return fmt.Errorf("%d URL(s) не удалось нормализовать", len(res.FailedURLs))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "путь к базе знаний (по умолчанию текущая директория)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "путь к .md файлу (относительно --path или абсолютный)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "только показать пары старый→новый URL, не записывать файл")

	return cmd
}

func resolveMarkdownUnderBase(baseFlag, fileFlag string) (string, string, error) {
	basePath := baseFlag
	if basePath == "" {
		basePath = "."
	}
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", "", fmt.Errorf("path: %w", err)
	}
	info, err := os.Stat(absBase)
	if err != nil {
		return "", "", fmt.Errorf("path: %w", err)
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("path %s is not a directory", absBase)
	}

	var mdPath string
	if filepath.IsAbs(fileFlag) {
		mdPath = filepath.Clean(fileFlag)
		rel, relErr := filepath.Rel(absBase, mdPath)
		if relErr != nil || strings.HasPrefix(rel, "..") {
			return "", "", fmt.Errorf("file %s is not under base path %s", fileFlag, absBase)
		}
	} else {
		mdPath = filepath.Join(absBase, filepath.FromSlash(fileFlag))
	}

	if !strings.HasSuffix(mdPath, ".md") {
		return "", "", fmt.Errorf("file must be a .md file: %s", mdPath)
	}
	if _, statErr := os.Stat(mdPath); statErr != nil {
		return "", "", fmt.Errorf("file: %w", statErr)
	}

	return absBase, mdPath, nil
}
