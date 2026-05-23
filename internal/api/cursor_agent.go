package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/muonsoft/errors"
)

type cursorStreamState struct {
	assistantText strings.Builder
}

// RunCursorAgent запускает cursor-agent с промтом в dataPath и стримит логи через onLog.
func RunCursorAgent(ctx context.Context, dataPath, prompt string, onLog func(stream, text string)) error {
	if _, err := exec.LookPath("cursor-agent"); err != nil {
		return errors.Errorf("cursor-agent not found in PATH: %w", err)
	}

	cmd := exec.CommandContext(
		ctx,
		"cursor-agent",
		"--print",
		"--output-format", "stream-json",
		"--force",
		prompt,
	)
	cmd.Dir = dataPath

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return errors.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return errors.Errorf("start cursor-agent: %w", err)
	}
	state := &cursorStreamState{}
	readPipe := func(stream string, r io.Reader, wg *sync.WaitGroup) {
		defer wg.Done()
		s := bufio.NewScanner(r)
		s.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
		for s.Scan() {
			for _, line := range parseCursorLogEvents(s.Text(), state) {
				onLog(stream, line)
			}
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go readPipe("stdout", stdoutPipe, &wg)
	go readPipe("stderr", stderrPipe, &wg)
	err = cmd.Wait()
	wg.Wait()
	if text := strings.TrimSpace(state.assistantText.String()); text != "" {
		onLog("stdout", text)
	}
	if err != nil {
		return errors.Errorf("run cursor-agent: %w", err)
	}

	return nil
}

func parseCursorLogEvents(line string, state *cursorStreamState) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return []string{line}
	}
	typeVal, _ := obj["type"].(string)
	subtype, _ := obj["subtype"].(string)

	if thinking := parseThinkingEvent(typeVal, subtype); thinking != nil {
		return thinking
	}
	if typeVal == "assistant" {
		collectAssistantText(state, obj)

		return nil
	}
	if typeVal == "tool_call" {
		return []string{formatToolCallEvent(obj, subtype)}
	}
	if typeVal == "result" {
		if result, ok := obj["result"].(string); ok && result != "" {
			if strings.TrimSpace(result) != strings.TrimSpace(state.assistantText.String()) {
				return []string{result}
			}

			return nil
		}

		return []string{"result:" + subtype}
	}

	if subtype != "" {
		return []string{typeVal + ":" + subtype}
	}

	return []string{typeVal + " " + compactKV(obj)}
}

func parseThinkingEvent(typeVal, subtype string) []string {
	if typeVal != "thinking" {
		return nil
	}
	if subtype == "completed" {
		return []string{"thinking completed"}
	}

	return []string{}
}

func collectAssistantText(state *cursorStreamState, obj map[string]any) {
	msg, ok := obj["message"].(map[string]any)
	if !ok {
		return
	}
	content, ok := msg["content"].([]any)
	if !ok {
		return
	}
	for _, item := range content {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		text, ok := m["text"].(string)
		if ok && text != "" {
			state.assistantText.WriteString(text)
		}
	}
}

func formatToolCallEvent(obj map[string]any, subtype string) string {
	if toolCall, ok := obj["tool_call"].(map[string]any); ok {
		for toolName := range toolCall {
			if subtype != "" {
				return "tool:" + toolName + ":" + subtype
			}

			return "tool:" + toolName
		}
	}
	if subtype != "" {
		return "tool_call:" + subtype
	}

	return "tool_call"
}

func compactKV(obj map[string]any) string {
	parts := make([]string, 0, len(obj))
	for k, v := range obj {
		if k == "message" || k == "result" || k == "content" {
			continue
		}
		parts = append(parts, k+"="+valueToString(v))
	}

	return strings.Join(parts, " ")
}

func valueToString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		if x {
			return "true"
		}

		return "false"
	default:
		if vv := fmt.Sprintf("%T", v); strings.HasPrefix(vv, "[]") {
			return "[...]"
		}

		return "{...}"
	}
}
