package cliapp

import (
	"bytes"
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
}

func TestNewApp_HelpContainsCommands(t *testing.T) {
	t.Helper()
	t.Parallel()

	app := New()
	var stdout bytes.Buffer
	app.Writer = &stdout

	err := app.Run([]string{"kb", "--help"})
	require.NoError(t, err)

	out := stdout.String()
	require.Contains(t, out, "serve")
	require.Contains(t, out, "validate")
	require.Contains(t, out, "init")
	require.Contains(t, out, "dump-images")
	require.Contains(t, out, "expand-urls")
	require.Contains(t, out, "reindex-links")
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
	err := app.Run([]string{"kb", "serve"})
	require.ErrorIs(t, err, expectedErr)
	require.True(t, called)
}
