package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func validateCmd() *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Проверить структуру базы знаний",
		RunE: func(cmd *cobra.Command, args []string) error {
			if path == "" {
				path = "."
			}
			path, err := absPath(path)
			if err != nil {
				return err
			}
			violations, err := kb.Validate(context.Background(), path)
			if err != nil {
				return fmt.Errorf("validate: %w", err)
			}
			if len(violations) > 0 {
				for _, v := range violations {
					fmt.Fprintf(os.Stderr, "  %s: %s\n", v.Path, v.Message)
				}
				os.Exit(1)
			}
			fmt.Println("OK: структура базы валидна")
			return nil
		},
	}
	cmd.Flags().StringVarP(&path, "path", "p", "", "путь к базе знаний (по умолчанию текущая директория)")
	return cmd
}
