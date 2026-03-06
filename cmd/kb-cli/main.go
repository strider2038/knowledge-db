package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := cobra.Command{
		Use:   "kb-cli",
		Short: "Консольная утилита для работы с базой знаний",
	}
	root.AddCommand(validateCmd())
	root.AddCommand(initCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
