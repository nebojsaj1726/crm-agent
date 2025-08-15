package utils

import (
	"context"
	"os"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

type mockLLM struct {
	response string
	err      error
}

func (m *mockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{Content: m.response},
		},
	}, nil
}

func (m *mockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestLoadMarkdownContent(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "testfile*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := "Hello, Markdown!"
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	got, err := LoadMarkdownContent(tmpfile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != content {
		t.Errorf("expected %q, got %q", content, got)
	}
}

func TestExtractKeys(t *testing.T) {
	m := map[string]any{"a": 1, "b": 2}
	keys := extractKeys(m)
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}

	expected := map[string]bool{"a": true, "b": true}
	for _, k := range keys {
		if !expected[k] {
			t.Errorf("unexpected key: %s", k)
		}
	}
}

func TestRunPrompt(t *testing.T) {
	mock := &mockLLM{response: "mock output"}
	input := map[string]any{"var1": "value"}
	out, err := RunPrompt(context.Background(), mock, "Template with {{.var1}}", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "mock output" {
		t.Errorf("expected 'mock output', got %q", out)
	}
}

func TestRunPromptAsync(t *testing.T) {
	mock := &mockLLM{response: "async output"}
	input := map[string]any{"foo": "bar"}
	ch := RunPromptAsync(context.Background(), mock, "Hi {{.foo}}", input)
	result := <-ch
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Response != "async output" {
		t.Errorf("expected 'async output', got %q", result.Response)
	}
}

func TestRunPrompt_Error(t *testing.T) {
	mock := &mockLLM{err: os.ErrNotExist}
	_, err := RunPrompt(context.Background(), mock, "any", map[string]any{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunPromptAsync_Error(t *testing.T) {
	mock := &mockLLM{err: os.ErrNotExist}
	ch := RunPromptAsync(context.Background(), mock, "any", map[string]any{})
	result := <-ch
	if result.Err == nil {
		t.Fatal("expected error, got nil")
	}
	if result.Response != "" {
		t.Errorf("expected empty response, got %q", result.Response)
	}
}

var _ llms.Model = (*mockLLM)(nil)
