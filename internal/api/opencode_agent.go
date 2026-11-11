package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/muonsoft/errors"
	"github.com/strider2038/knowledge-db/internal/kb"
)

type openCodeRunner struct {
	opts NodeAgentOptions
}

func newOpenCodeRunner(opts NodeAgentOptions) *openCodeRunner {
	return &openCodeRunner{opts: opts}
}

// NewOpenCodeNodeNormalizer создаёт раннер нормализации OpenCode (для тестов и bootstrap).
func NewOpenCodeNodeNormalizer(opts NodeAgentOptions) nodeNormalizer {
	return newOpenCodeRunner(opts)
}

// NewOpenCodeNodeAgentEditor создаёт раннер agent-edit OpenCode (для тестов и bootstrap).
func NewOpenCodeNodeAgentEditor(opts NodeAgentOptions) NodeAgentEditor {
	return newOpenCodeRunner(opts)
}

func (r *openCodeRunner) Available() error {
	if _, err := exec.LookPath("opencode"); err != nil {
		return errJobOpencodeUnavailable
	}

	return nil
}

func (r *openCodeRunner) NormalizeNode(ctx context.Context, dataPath string, node *kb.Node, onLog func(stream, text string)) error {
	return r.run(ctx, dataPath, node.Path, buildNormalizationPrompt(node), onLog)
}

func (r *openCodeRunner) EditNode(ctx context.Context, dataPath string, node *kb.Node, instruction string, onLog func(stream, text string)) error {
	return r.run(ctx, dataPath, node.Path, buildAgentEditPrompt(node, instruction), onLog)
}

func (r *openCodeRunner) run(ctx context.Context, dataPath, nodeRelPath, prompt string, onLog func(stream, text string)) error {
	if err := r.Available(); err != nil {
		return err
	}

	jobRoot, err := prepareOpenCodeJob(dataPath, nodeRelPath, r.opts)
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(jobRoot) }()

	modelArg := openCodeCLIArg(r.opts)
	// OpenCode 2.2 не принимает --dir у run: каталог проекта — это PWD, затем cwd.
	cmd := exec.CommandContext(ctx, "opencode", openCodeRunArgs(modelArg, prompt)...)
	cmd.Dir = dataPath
	cmd.Env = buildOpenCodeChildEnv(jobRoot, r.opts.APIKey, dataPath)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return errors.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return errors.Errorf("start opencode: %w", err)
	}

	state := &openCodeStreamState{}
	readPipe := func(stream string, reader io.Reader, parseJSONL bool, wg *sync.WaitGroup) {
		defer wg.Done()
		s := bufio.NewScanner(reader)
		s.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
		for s.Scan() {
			text := s.Text()
			if !parseJSONL {
				if strings.TrimSpace(text) != "" {
					onLog(stream, text)
				}

				continue
			}
			for _, line := range parseOpenCodeLogEvents(text, state) {
				onLog(stream, line)
			}
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go readPipe("stdout", stdoutPipe, true, &wg)
	go readPipe("stderr", stderrPipe, false, &wg)
	waitErr := cmd.Wait()
	wg.Wait()

	if text := strings.TrimSpace(state.assistantText.String()); text != "" {
		onLog("stdout", text)
	}
	if state.runErr != nil {
		return state.runErr
	}
	if waitErr != nil {
		return errors.Errorf("run opencode: %w", waitErr)
	}

	return nil
}

type openCodeStreamState struct {
	assistantText strings.Builder
	runErr        error
}

func parseOpenCodeLogEvents(line string, state *openCodeStreamState) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return []string{line}
	}

	typeVal, _ := obj["type"].(string)
	switch typeVal {
	case "text":
		collectOpenCodeText(state, obj)

		return nil
	case "tool_use":
		tool := openCodeToolName(obj)
		if tool == "" {
			return []string{"tool_use"}
		}

		return []string{"tool:" + tool}
	case "error":
		msg := openCodeErrorMessage(obj)
		if msg == "" {
			msg = "opencode error"
		}
		state.runErr = errors.New(msg)

		return []string{msg}
	default:
		summary := typeVal + " " + compactKV(obj)
		if len(summary) > 200 {
			summary = summary[:200] + "…"
		}

		return []string{summary}
	}
}

func collectOpenCodeText(state *openCodeStreamState, obj map[string]any) {
	if part, ok := obj["part"].(map[string]any); ok {
		if text, ok := part["text"].(string); ok && text != "" {
			state.assistantText.WriteString(text)

			return
		}
	}
	if text, ok := obj["text"].(string); ok && text != "" {
		state.assistantText.WriteString(text)
	}
}

func openCodeToolName(obj map[string]any) string {
	if part, ok := obj["part"].(map[string]any); ok {
		if tool, ok := part["tool"].(string); ok {
			return tool
		}
	}
	if tool, ok := obj["tool"].(string); ok {
		return tool
	}

	return ""
}

func openCodeErrorMessage(obj map[string]any) string {
	if errObj, ok := obj["error"].(map[string]any); ok {
		if data, ok := errObj["data"].(map[string]any); ok {
			if msg, ok := data["message"].(string); ok && msg != "" {
				return msg
			}
		}
		if msg, ok := errObj["message"].(string); ok && msg != "" {
			return msg
		}
		if name, ok := errObj["name"].(string); ok && name != "" {
			return name
		}
	}
	if msg, ok := obj["message"].(string); ok {
		return msg
	}
	if part, ok := obj["part"].(map[string]any); ok {
		if msg, ok := part["message"].(string); ok {
			return msg
		}
	}

	return ""
}

func resolveOpenCodeStateDir(override string) string {
	if strings.TrimSpace(override) != "" {
		return filepath.Clean(override)
	}
	if cache, err := os.UserCacheDir(); err == nil {
		return filepath.Join(cache, "knowledge-db", "opencode")
	}

	return filepath.Join(os.TempDir(), "knowledge-db-opencode")
}

func validateOpenCodeStateNotInDataPath(stateRoot, dataPath string) error {
	stateRoot = filepath.Clean(stateRoot)
	dataPath = filepath.Clean(dataPath)
	if stateRoot == dataPath {
		return errors.New("opencode state dir must not be inside the knowledge base data path")
	}
	rel, err := filepath.Rel(dataPath, stateRoot)
	if err == nil && (rel == "." || !strings.HasPrefix(rel, "..")) {
		return errors.New("opencode state dir must not be inside the knowledge base data path")
	}

	return nil
}

func prepareOpenCodeJob(dataPath, nodeRelPath string, opts NodeAgentOptions) (string, error) {
	base := resolveOpenCodeStateDir(opts.StateDir)
	if err := validateOpenCodeStateNotInDataPath(base, dataPath); err != nil {
		return "", errors.Errorf("opencode job: %w", err)
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", errors.Errorf("mkdir opencode state root: %w", err)
	}
	jobRoot, err := os.MkdirTemp(base, "job-*")
	if err != nil {
		return "", errors.Errorf("create opencode job dir: %w", err)
	}
	if err := writeOpenCodeConfig(jobRoot, nodeRelPath, opts); err != nil {
		_ = os.RemoveAll(jobRoot)

		return "", err
	}

	return jobRoot, nil
}

func openCodeJobConfigPath(jobRoot string) string {
	return filepath.Join(jobRoot, "config", "opencode", "opencode.json")
}

func buildOpenCodeChildEnv(stateRoot, apiKey, workDir string) []string {
	env := os.Environ()
	xdg := map[string]string{
		"XDG_CONFIG_HOME": filepath.Join(stateRoot, "config"),
		"XDG_DATA_HOME":   filepath.Join(stateRoot, "data"),
		"XDG_STATE_HOME":  filepath.Join(stateRoot, "state"),
		"XDG_CACHE_HOME":  filepath.Join(stateRoot, "cache"),
	}
	for k, v := range xdg {
		env = setEnvVar(env, k, v)
	}
	if strings.TrimSpace(apiKey) != "" {
		env = setEnvVar(env, "KB_NODE_AGENT_API_KEY", apiKey)
	}
	if strings.TrimSpace(workDir) != "" {
		env = setEnvVar(env, "PWD", workDir)
	}

	return env
}

func openCodeRunArgs(modelArg, prompt string) []string {
	return []string{
		"run", "--standalone", "--format", "json", "--archived",
		"--model", modelArg,
		prompt,
	}
}

func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	found := false
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			out = append(out, prefix+value)
			found = true

			continue
		}
		out = append(out, e)
	}
	if !found {
		out = append(out, prefix+value)
	}

	return out
}

func writeOpenCodeConfig(stateRoot, nodeRelPath string, opts NodeAgentOptions) error {
	cfgDir := filepath.Join(stateRoot, "config", "opencode")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return errors.Errorf("mkdir opencode config: %w", err)
	}
	payload := buildOpenCodeConfigJSON(nodeRelPath, opts)
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return errors.Errorf("marshal opencode config: %w", err)
	}

	return os.WriteFile(filepath.Join(cfgDir, "opencode.json"), data, 0o600)
}

func buildOpenCodeConfigJSON(nodeRelPath string, opts NodeAgentOptions) map[string]any {
	cfg := map[string]any{
		"$schema":     "https://opencode.ai/config.json",
		"snapshots":   false,
		"update":      "disable",
		"formatter":   false,
		"permissions": buildOpenCodePermissions(nodeRelPath),
	}
	if strings.TrimSpace(opts.APIURL) != "" {
		cfg["providers"] = map[string]any{
			"kb": map[string]any{
				"package": "@opencode/ai/providers/openai-compatible",
				"env":     []string{"KB_NODE_AGENT_API_KEY"},
				"settings": map[string]any{
					"baseURL": opts.APIURL,
				},
				"models": map[string]any{
					"default": map[string]any{
						"modelID": strings.TrimSpace(opts.Model),
					},
				},
			},
		}
	}

	return cfg
}

func buildOpenCodePermissions(nodeRelPath string) []map[string]string {
	nodeRelPath = strings.TrimSpace(nodeRelPath)
	perms := []map[string]string{
		{"action": "read", "resource": "*", "effect": "allow"},
		{"action": "glob", "resource": "*", "effect": "allow"},
		{"action": "grep", "resource": "*", "effect": "allow"},
		{"action": "edit", "resource": "*", "effect": "deny"},
	}
	if allow := openCodeEditAllowResource(nodeRelPath); allow != "" {
		perms = append(perms, map[string]string{
			"action": "edit", "resource": allow, "effect": "allow",
		})
	}
	denyAll := []string{"shell", "webfetch", "websearch", "subagent", "question", "external_directory"}
	for _, action := range denyAll {
		perms = append(perms, map[string]string{
			"action": action, "resource": "*", "effect": "deny",
		})
	}

	return perms
}

func openCodeEditAllowResource(nodeRelPath string) string {
	nodeRelPath = strings.TrimSpace(nodeRelPath)
	if nodeRelPath == "" {
		return ""
	}
	nodeRelPath = filepath.ToSlash(nodeRelPath)
	if strings.HasSuffix(strings.ToLower(nodeRelPath), ".md") {
		return nodeRelPath
	}

	return nodeRelPath + ".md"
}

func openCodeCLIArg(opts NodeAgentOptions) string {
	if strings.TrimSpace(opts.APIURL) != "" {
		return "kb/default"
	}

	return strings.TrimSpace(opts.Model)
}

// openCodeCommandSpec describes argv and env for tests (no process spawn).
type openCodeCommandSpec struct {
	Argv      []string
	Env       []string
	Config    map[string]any
	ModelArg  string
	StateRoot string
}

func buildOpenCodeCommandSpec(dataPath, nodeRelPath, prompt string, opts NodeAgentOptions) openCodeCommandSpec {
	base := resolveOpenCodeStateDir(opts.StateDir)
	jobRoot := filepath.Join(base, "job")
	modelArg := openCodeCLIArg(opts)
	cfg := buildOpenCodeConfigJSON(nodeRelPath, opts)

	return openCodeCommandSpec{
		Argv:      append([]string{"opencode"}, openCodeRunArgs(modelArg, prompt)...),
		Env:       buildOpenCodeChildEnv(jobRoot, opts.APIKey, dataPath),
		Config:    cfg,
		ModelArg:  modelArg,
		StateRoot: jobRoot,
	}
}

var _ nodeNormalizer = (*openCodeRunner)(nil)
var _ NodeAgentEditor = (*openCodeRunner)(nil)

// FormatOpenCodeConfigForTest returns marshaled config for assertions.
func FormatOpenCodeConfigForTest(nodeRelPath string, opts NodeAgentOptions) ([]byte, error) {
	data, err := json.Marshal(buildOpenCodeConfigJSON(nodeRelPath, opts))
	if err != nil {
		return nil, errors.Errorf("marshal: %w", err)
	}

	return data, nil
}

// ParseOpenCodeLogLineForTest exposes the JSONL parser for unit tests.
func ParseOpenCodeLogLineForTest(line string, state *openCodeStreamState) ([]string, error) {
	if state == nil {
		state = &openCodeStreamState{}
	}
	lines := parseOpenCodeLogEvents(line, state)
	if state.runErr != nil {
		return lines, state.runErr
	}

	return lines, nil
}

func openCodeConfigHasProviderBaseURL(cfg map[string]any) bool {
	providers, ok := cfg["providers"].(map[string]any)
	if !ok {
		return false
	}
	for _, pv := range providers {
		prov, ok := pv.(map[string]any)
		if !ok {
			continue
		}
		settings, ok := prov["settings"].(map[string]any)
		if !ok {
			continue
		}
		if _, ok := settings["baseURL"]; ok {
			return true
		}
	}

	return false
}

func permissionEffectFor(cfg map[string]any, action, resource string) string {
	raw, ok := cfg["permissions"].([]map[string]string)
	if !ok {
		if anyPerms, ok := cfg["permissions"].([]any); ok {
			for _, item := range anyPerms {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				if fmt.Sprint(m["action"]) == action && fmt.Sprint(m["resource"]) == resource {
					return fmt.Sprint(m["effect"])
				}
			}
		}

		return ""
	}
	for _, rule := range raw {
		if rule["action"] == action && rule["resource"] == resource {
			return rule["effect"]
		}
	}

	return ""
}
