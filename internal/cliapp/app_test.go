package cliapp

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewApp_ContainsRequiredCommands(t *testing.T) {
	t.Helper()
	t.Parallel()

	app := New()
	require.Equal(t, "kb", app.Name)

	commands := make(map[string]struct{}, len(app.Commands))
	for _, command := range app.Commands {
		commands[command.Name] = struct{}{}
	}

	require.Contains(t, commands, "serve")
	require.Contains(t, commands, "validate")
	require.Contains(t, commands, "init")
	require.Contains(t, commands, "dump-images")
	require.Contains(t, commands, "expand-urls")
	require.Contains(t, commands, "reindex-links")
	require.Contains(t, commands, "rebuild-index")
	require.Contains(t, commands, "migrate-node-ids")
}

func TestNewApp_HelpContainsCommands(t *testing.T) {
	t.Helper()
	t.Parallel()

	app := New()
	var stdout bytes.Buffer
	app.Writer = &stdout

	err := app.Run(context.Background(), []string{"kb", "--help"})
	require.NoError(t, err)

	out := stdout.String()
	require.Contains(t, out, "serve")
	require.Contains(t, out, "validate")
	require.Contains(t, out, "init")
	require.Contains(t, out, "dump-images")
	require.Contains(t, out, "expand-urls")
	require.Contains(t, out, "reindex-links")
	require.Contains(t, out, "rebuild-index")
	require.Contains(t, out, "migrate-node-ids")
}

func TestServeCommand_UsesRunServe(t *testing.T) {
	t.Helper()
	t.Parallel()

	orig := runServe
	t.Cleanup(func() {
		runServe = orig
	})

	var called bool
	expectedErr := errors.New("boom")
	runServe = func() error {
		called = true

		return expectedErr
	}

	app := New()
	err := app.Run(context.Background(), []string{"kb", "serve"})
	require.ErrorIs(t, err, expectedErr)
	require.True(t, called)
}

func TestExecute_WhenServeFails_ExpectStderrAndExitCode(t *testing.T) {
	t.Helper()
	t.Parallel()

	orig := runServe
	t.Cleanup(func() {
		runServe = orig
	})
	runServe = func() error {
		return errors.New("invalid node agent configuration: node agent: OPENCODE_API_KEY is required for opencode-go models")
	}

	var stderr bytes.Buffer
	code := Execute(context.Background(), []string{"kb", "serve"}, &stderr)

	require.Equal(t, 1, code)
	require.Contains(t, stderr.String(), "OPENCODE_API_KEY is required")
}
