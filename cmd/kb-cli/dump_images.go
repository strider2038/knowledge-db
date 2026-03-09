package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func dumpImagesCmd() *cobra.Command {
	var path, file string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "dump-images",
		Short: "Скачать удалённые изображения из статьи и заменить ссылки на локальные",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return errors.New("--file is required")
			}

			absBase, themePath, slug, err := resolveDumpPaths(path, file)
			if err != nil {
				return err
			}

			ctx := context.Background()
			fs := afero.NewOsFs()
			client := &http.Client{Timeout: 30 * time.Second}

			onDownload := func(url, targetPath string, size int64) {
				fmt.Fprintf(os.Stderr, "  %s -> %s (%d bytes)\n", url, targetPath, size)
			}
			modified, downloadErrors, dryRunResults, err := kb.RunDumpImages(ctx, fs, client, absBase, themePath, slug, dryRun, onDownload)
			if err != nil {
				return fmt.Errorf("dump-images: %w", err)
			}

			for _, de := range downloadErrors {
				fmt.Fprintf(os.Stderr, "  %s: %v\n", de.URL, de.Err)
			}

			if dryRun {
				for _, r := range dryRunResults {
					fmt.Printf("%s -> %s\n", r.URL, r.TargetPath)
				}
				if len(downloadErrors) > 0 {
					fmt.Fprintf(os.Stderr, "Warning: %d URL(s) could not be resolved\n", len(downloadErrors))
				}

				return nil
			}

			if modified {
				fmt.Println("OK: изображения загружены, ссылки заменены")
			} else if len(downloadErrors) == 0 {
				fmt.Println("OK: удалённых изображений не найдено")
			}

			if len(downloadErrors) > 0 {
				fmt.Fprintf(os.Stderr, "Warning: не удалось загрузить %d изображений\n", len(downloadErrors))

				return fmt.Errorf("%d images failed to download", len(downloadErrors))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "путь к базе знаний (по умолчанию текущая директория)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "путь к .md файлу статьи (относительно --path или абсолютный)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "только показать URL и целевые пути, не скачивать")

	return cmd
}

func resolveDumpPaths(baseFlag, fileFlag string) (string, string, string, error) {
	basePath := baseFlag
	if basePath == "" {
		basePath = "."
	}
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", "", "", fmt.Errorf("path: %w", err)
	}
	info, err := os.Stat(absBase)
	if err != nil {
		return "", "", "", fmt.Errorf("path: %w", err)
	}
	if !info.IsDir() {
		return "", "", "", fmt.Errorf("path %s is not a directory", absBase)
	}

	var mdPath string
	if filepath.IsAbs(fileFlag) {
		mdPath = filepath.Clean(fileFlag)
		rel, relErr := filepath.Rel(absBase, mdPath)
		if relErr != nil || strings.HasPrefix(rel, "..") {
			return "", "", "", fmt.Errorf("file %s is not under base path %s", fileFlag, absBase)
		}
	} else {
		mdPath = filepath.Join(absBase, filepath.FromSlash(fileFlag))
	}

	if !strings.HasSuffix(mdPath, ".md") {
		return "", "", "", fmt.Errorf("file must be a .md file: %s", mdPath)
	}
	if _, statErr := os.Stat(mdPath); statErr != nil {
		return "", "", "", fmt.Errorf("file: %w", statErr)
	}

	relStem, err := filepath.Rel(absBase, strings.TrimSuffix(mdPath, ".md"))
	if err != nil {
		return "", "", "", fmt.Errorf("resolve path: %w", err)
	}
	stemSlash := filepath.ToSlash(relStem)
	lastSlash := strings.LastIndex(stemSlash, "/")
	var themePath, slug string
	if lastSlash >= 0 {
		themePath = stemSlash[:lastSlash]
		slug = stemSlash[lastSlash+1:]
	} else {
		slug = stemSlash
	}

	return absBase, themePath, slug, nil
}
