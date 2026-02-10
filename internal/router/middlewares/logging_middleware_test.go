package middlewares

import (
	"testing"

	"github.com/nanami9426/imgo/internal/models"
)

func TestExtractTokenInfoJSONSetsCompletionID(t *testing.T) {
	body := []byte(`{"id":"chatcmpl-json-1","model":"qwen","usage":{"prompt_tokens":7,"completion_tokens":5,"total_tokens":12}}`)
	usage := &models.APIUsage{}

	extractTokenInfo(body, usage)

	if usage.CompletionID != "chatcmpl-json-1" {
		t.Fatalf("expected completion_id chatcmpl-json-1, got %q", usage.CompletionID)
	}
	if usage.Model != "qwen" {
		t.Fatalf("expected model qwen, got %q", usage.Model)
	}
	if usage.InputTokens != 7 || usage.OutputTokens != 5 || usage.TotalTokens != 12 {
		t.Fatalf("unexpected tokens: input=%d output=%d total=%d", usage.InputTokens, usage.OutputTokens, usage.TotalTokens)
	}
}

func TestExtractTokenInfoSSEBackfillsCompletionIDFromPreviousChunk(t *testing.T) {
	body := []byte(
		"data: {\"id\":\"chatcmpl-sse-1\",\"model\":\"qwen\",\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n" +
			"data: {\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5}}\n\n" +
			"data: [DONE]\n\n",
	)
	usage := &models.APIUsage{}

	extractTokenInfo(body, usage)

	if usage.CompletionID != "chatcmpl-sse-1" {
		t.Fatalf("expected completion_id chatcmpl-sse-1, got %q", usage.CompletionID)
	}
	if usage.TotalTokens != 5 {
		t.Fatalf("expected total_tokens 5, got %d", usage.TotalTokens)
	}
}
