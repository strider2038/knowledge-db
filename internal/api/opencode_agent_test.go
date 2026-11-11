package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildOpenCodeCommandSpec_OpencodeGo(t *testing.T) {
	t.Parallel()
	dataPath := t.TempDir()
	opts := NodeAgentOptions{
		Model: "opencode-go/kimi-k3",
	}
	spec := buildOpenCodeCommandSpec(dataPath, "topic/note.md", "prompt", opts)

	joined := strings.Join(spec.Argv, " ")
	require.Contains(t, joined, "--standalone")
	require.Contains(t, joined, "--format json")
	require.Contains(t, joined, "--archived")
	require.Contains(t, joined, "--model opencode-go/kimi-k3")
	require.NotContains(t, joined, "--dir")
	require.Contains(t, spec.Env, "PWD="+dataPath)

	for _, key := range []string{"XDG_CONFIG_HOME=", "XDG_DATA_HOME=", "XDG_STATE_HOME=", "XDG_CACHE_HOME="} {
		var val string
		for _, e := range spec.Env {
			if rest, ok := strings.CutPrefix(e, key); ok {
				val = rest

				break
			}
		}
		require.NotEmpty(t, val)
		require.NotEqual(t, dataPath, val)
		require.True(t, strings.HasPrefix(val, spec.StateRoot))
	}

	require.False(t, openCodeConfigHasProviderBaseURL(spec.Config))
	require.Equal(t, "deny", permissionEffectFor(spec.Config, "shell", "*"))
	require.Equal(t, "deny", permissionEffectFor(spec.Config, "edit", "*"))
	require.Equal(t, "allow", permissionEffectFor(spec.Config, "edit", "topic/note.md"))
}

func TestBuildOpenCodeCommandSpec_CustomURL(t *testing.T) {
	t.Parallel()
	dataPath := t.TempDir()
	opts := NodeAgentOptions{
		Model:  "gpt-4o-mini",
		APIURL: "https://llm.example.com/v1",
		APIKey: "sk-test",
	}
	spec := buildOpenCodeCommandSpec(dataPath, "a/b.md", "p", opts)
	require.Equal(t, "kb/default", spec.ModelArg)
	require.Contains(t, strings.Join(spec.Argv, " "), "--model kb/default")

	providers, ok := spec.Config["providers"].(map[string]any)
	require.True(t, ok)
	kbProv, ok := providers["kb"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "@opencode/ai/providers/openai-compatible", kbProv["package"])
	settings, ok := kbProv["settings"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "https://llm.example.com/v1", settings["baseURL"])
	models, ok := kbProv["models"].(map[string]any)
	require.True(t, ok)
	def, ok := models["default"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "gpt-4o-mini", def["modelID"])

	var apiKeyInEnv bool
	for _, e := range spec.Env {
		if rest, ok := strings.CutPrefix(e, "KB_NODE_AGENT_API_KEY="); ok {
			require.Equal(t, "sk-test", rest)
			apiKeyInEnv = true
		}
	}
	require.True(t, apiKeyInEnv)
}

func TestParseOpenCodeLogEvents(t *testing.T) {
	t.Parallel()
	state := &openCodeStreamState{}
	lines, err := ParseOpenCodeLogLineForTest(`{"type":"tool_use","part":{"tool":"read"}}`, state)
	require.NoError(t, err)
	require.Equal(t, []string{"tool:read"}, lines)

	state = &openCodeStreamState{}
	_, err = ParseOpenCodeLogLineForTest(`{"type":"text","part":{"text":"hello "}}`, state)
	require.NoError(t, err)
	_, err = ParseOpenCodeLogLineForTest(`{"type":"text","text":"world"}`, state)
	require.NoError(t, err)
	require.Equal(t, "hello world", strings.TrimSpace(state.assistantText.String()))

	state = &openCodeStreamState{}
	lines, err = ParseOpenCodeLogLineForTest(`{"type":"error","message":"boom"}`, state)
	require.Error(t, err)
	require.Equal(t, "boom", err.Error())
	require.Contains(t, lines, "boom")

	state = &openCodeStreamState{}
	_, err = ParseOpenCodeLogLineForTest(`{"type":"error","error":{"name":"APIError","data":{"message":"rate limit"}}}`, state)
	require.EqualError(t, err, "rate limit")
}

func TestFormatOpenCodeConfig_PermissionsJSON(t *testing.T) {
	t.Parallel()
	raw, err := FormatOpenCodeConfigForTest("topic/x.md", NodeAgentOptions{Model: "opencode-go/m"})
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, json.Unmarshal(raw, &cfg))
	perms, ok := cfg["permissions"].([]any)
	require.True(t, ok)
	foundShellDeny := false
	foundEditAllow := false
	for _, p := range perms {
		m, ok := p.(map[string]any)
		require.True(t, ok)
		if m["action"] == "shell" && m["effect"] == "deny" {
			foundShellDeny = true
		}
		if m["action"] == "edit" && m["resource"] == "topic/x.md" && m["effect"] == "allow" {
			foundEditAllow = true
		}
	}
	require.True(t, foundShellDeny)
	require.True(t, foundEditAllow)
}

func TestResolveOpenCodeStateDir_Override(t *testing.T) {
	t.Parallel()
	dir := resolveOpenCodeStateDir(filepath.Join(t.TempDir(), "custom"))
	require.Contains(t, dir, "custom")
}

func TestOpenCodeEditAllowResource_NodeStem(t *testing.T) {
	t.Parallel()
	cfg := buildOpenCodeConfigJSON("topic/note", NodeAgentOptions{Model: "opencode-go/m"})
	require.Equal(t, "allow", permissionEffectFor(cfg, "edit", "topic/note.md"))
	require.Empty(t, permissionEffectFor(cfg, "edit", "topic/note"))
	require.Equal(t, "allow", permissionEffectFor(buildOpenCodeConfigJSON("topic/note.md", NodeAgentOptions{Model: "m"}), "edit", "topic/note.md"))
	require.Empty(t, permissionEffectFor(buildOpenCodeConfigJSON("topic/note.md", NodeAgentOptions{Model: "m"}), "edit", "topic/note.md.md"))
}

func TestPrepareOpenCodeJob_IsolatedConfigs(t *testing.T) {
	t.Parallel()
	dataPath := t.TempDir()
	stateBase := filepath.Join(t.TempDir(), "opencode-state")
	opts := NodeAgentOptions{
		Model:    "opencode-go/m",
		StateDir: stateBase,
	}
	jobA, err := prepareOpenCodeJob(dataPath, "topic/a", opts)
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(jobA) }()
	jobB, err := prepareOpenCodeJob(dataPath, "topic/b", opts)
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(jobB) }()

	require.NotEqual(t, jobA, jobB)
	cfgPathA := openCodeJobConfigPath(jobA)
	cfgPathB := openCodeJobConfigPath(jobB)
	require.FileExists(t, cfgPathA)
	require.FileExists(t, cfgPathB)
	require.NotEqual(t, cfgPathA, cfgPathB)

	rawA, err := os.ReadFile(cfgPathA)
	require.NoError(t, err)
	rawB, err := os.ReadFile(cfgPathB)
	require.NoError(t, err)
	require.NotEqual(t, string(rawA), string(rawB))
	require.Contains(t, string(rawA), `"resource": "topic/a.md"`)
	require.Contains(t, string(rawB), `"resource": "topic/b.md"`)
}

func TestPrepareOpenCodeJob_StateInsideDataPath(t *testing.T) {
	t.Parallel()
	dataPath := t.TempDir()
	opts := NodeAgentOptions{
		Model:    "opencode-go/m",
		StateDir: filepath.Join(dataPath, ".opencode"),
	}
	_, err := prepareOpenCodeJob(dataPath, "x", opts)
	require.Error(t, err)
	opts.StateDir = dataPath
	_, err = prepareOpenCodeJob(dataPath, "x", opts)
	require.Error(t, err)
}
