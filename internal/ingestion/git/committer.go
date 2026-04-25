package git

import (
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"
)

// GitCommitter — интерфейс для git commit и sync.
type GitCommitter interface {
	CommitNode(ctx context.Context, nodePath, message string) error
	Sync(ctx context.Context) error
	Status(ctx context.Context) (*GitStatus, error)
	CommitAll(ctx context.Context, message string) error
	DiffStat(ctx context.Context) (string, error)
}

// GitStatus — результат проверки git status.
type GitStatus struct {
	HasChanges   bool
	ChangedFiles int
}

// ExecGitCommitter выполняет git-команды через exec.Command.
type ExecGitCommitter struct {
	repoPath string
}

// NewExecGitCommitter создаёт ExecGitCommitter.
func NewExecGitCommitter(repoPath string) *ExecGitCommitter {
	return &ExecGitCommitter{repoPath: repoPath}
}

// CommitNode выполняет git add + git commit + git push для указанного пути.
// Ошибка push логируется, но не возвращается — ingestion не должен падать из-за сетевых проблем.
func (g *ExecGitCommitter) CommitNode(ctx context.Context, nodePath, message string) error {
	if err := g.run(ctx, "add", nodePath); err != nil {
		return errors.Errorf("commit node: git add: %w", err)
	}
	if err := g.run(ctx, "commit", "-m", message); err != nil {
		return errors.Errorf("commit node: git commit: %w", err)
	}
	if err := g.run(ctx, "push"); err != nil {
		clog.Errorf(ctx, "commit node: git push failed (will retry on next sync): %w", err)
	}

	return nil
}

// Sync выполняет git fetch origin + git merge origin/main.
// При конфликтах логирует warning и не пытается автоматически разрешить.
func (g *ExecGitCommitter) Sync(ctx context.Context) error {
	if err := g.run(ctx, "fetch", "origin"); err != nil {
		return errors.Errorf("sync: git fetch: %w", err)
	}

	if err := g.run(ctx, "merge", "origin/main", "--no-edit"); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "CONFLICT") || strings.Contains(errMsg, "conflict") {
			clog.Errorf(ctx, "git sync: merge conflict detected, manual resolution required: %w", err)

			return nil
		}

		return errors.Errorf("sync: git merge: %w", err)
	}

	return nil
}

// Status возвращает информацию о незакоммиченных изменениях.
func (g *ExecGitCommitter) Status(ctx context.Context) (*GitStatus, error) {
	out, err := g.output(ctx, "status", "--porcelain")
	if err != nil {
		return nil, errors.Errorf("git status: %w", err)
	}
	lines := strings.TrimSpace(out)
	if lines == "" {
		return &GitStatus{HasChanges: false, ChangedFiles: 0}, nil
	}
	count := len(strings.Split(lines, "\n"))

	return &GitStatus{HasChanges: true, ChangedFiles: count}, nil
}

// CommitAll выполняет git add -A + git commit + git push с указанным сообщением.
func (g *ExecGitCommitter) CommitAll(ctx context.Context, message string) error {
	if err := g.run(ctx, "add", "-A"); err != nil {
		return errors.Errorf("commit all: git add: %w", err)
	}
	if err := g.run(ctx, "commit", "-m", message); err != nil {
		return errors.Errorf("commit all: git commit: %w", err)
	}
	if err := g.run(ctx, "push"); err != nil {
		clog.Errorf(ctx, "commit all: git push failed: %w", err)
	}

	return nil
}

// DiffStat возвращает результат git diff --stat.
func (g *ExecGitCommitter) DiffStat(ctx context.Context) (string, error) {
	out, err := g.output(ctx, "diff", "--stat")
	if err != nil {
		return "", errors.Errorf("git diff stat: %w", err)
	}

	return out, nil
}

func (g *ExecGitCommitter) run(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.repoPath

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return errors.Errorf("git %s: %w", strings.Join(args, " "), err,
			errors.String("output", out.String()),
		)
	}

	return nil
}

func (g *ExecGitCommitter) output(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.repoPath

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", errors.Errorf("git %s: %w", strings.Join(args, " "), err,
			errors.String("stderr", stderr.String()),
		)
	}

	return stdout.String(), nil
}
