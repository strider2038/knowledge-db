package cliapp

import (
	"context"
	"fmt"
	"os"

	"github.com/muonsoft/errors"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"github.com/strider2038/knowledge-db/internal/kb"
)

func migrateNodeIDsCmd() *cli.Command {
	return &cli.Command{
		Name:  "migrate-node-ids",
		Usage: "Assign UUID v7 id to all nodes missing frontmatter id",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "path",
				Usage: "Knowledge base root path (default: KB_DATA_PATH or ./data)",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Print planned changes without writing files",
			},
		},
		Action: runMigrateNodeIDs,
	}
}

func runMigrateNodeIDs(ctx context.Context, cmd *cli.Command) error {
	path := cmd.String("path")
	if path == "" {
		path = os.Getenv("KB_DATA_PATH")
	}
	if path == "" {
		path = "."
	}
	basePath, err := absPath(path)
	if err != nil {
		return errors.Errorf("migrate-node-ids: %w", err)
	}
	dryRun := cmd.Bool("dry-run")
	store := kb.NewStore(afero.NewOsFs())

	allNodes, err := store.ListAllNodes(ctx, basePath)
	if err != nil {
		return errors.Errorf("migrate-node-ids: list nodes: %w", err)
	}

	seenIDs := make(map[string]string)
	var toAssign []string

	for _, n := range allNodes {
		node, getErr := store.GetNode(ctx, basePath, n.Path)
		if getErr != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", n.Path, getErr)

			continue
		}
		id := kb.NodeIDFromMetadata(node.Metadata)
		if id != "" {
			if !kb.ValidateNodeID(id) {
				return errors.Errorf("migrate-node-ids: invalid id in %s: %q", n.Path, id)
			}
			if prev, ok := seenIDs[id]; ok {
				return errors.Errorf("migrate-node-ids: duplicate id %s in %s and %s", id, prev, n.Path)
			}
			seenIDs[id] = n.Path

			continue
		}
		toAssign = append(toAssign, n.Path)
	}

	if len(toAssign) == 0 {
		fmt.Println("migrate-node-ids: all nodes already have id")

		return nil
	}

	fmt.Printf("migrate-node-ids: %d node(s) need id\n", len(toAssign))
	for _, path := range toAssign {
		fmt.Printf("  %s\n", path)
	}
	if dryRun {
		fmt.Println("dry-run: no files changed")

		return nil
	}

	for _, path := range toAssign {
		file, err := store.GetNodeFile(ctx, basePath, path)
		if err != nil {
			return errors.Errorf("migrate-node-ids: read %s: %w", path, err)
		}
		if err := kb.EnsureNodeID(file.Frontmatter); err != nil {
			return errors.Errorf("migrate-node-ids: assign id %s: %w", path, err)
		}
		id := kb.NodeIDFromMetadata(file.Frontmatter)
		if prev, ok := seenIDs[id]; ok {
			return errors.Errorf("migrate-node-ids: generated duplicate id %s for %s (already in %s)", id, path, prev)
		}
		seenIDs[id] = path
		if _, err := store.UpdateNode(ctx, basePath, path, kb.UpdateNodeParams{
			Frontmatter: file.Frontmatter,
			Content:     file.Content,
		}); err != nil {
			return errors.Errorf("migrate-node-ids: write %s: %w", path, err)
		}
	}

	fmt.Printf("migrate-node-ids: assigned id to %d node(s)\n", len(toAssign))

	return nil
}
