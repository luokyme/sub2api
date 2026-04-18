package service

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/stretchr/testify/require"
)

func mustRawJSON(t *testing.T, s string) json.RawMessage {
	t.Helper()
	return json.RawMessage(s)
}

func TestShouldAutoInjectPromptCacheKeyForCompat(t *testing.T) {
	require.True(t, shouldAutoInjectPromptCacheKeyForCompat("gpt-5.4"))
	require.True(t, shouldAutoInjectPromptCacheKeyForCompat("gpt-5.3"))
	require.True(t, shouldAutoInjectPromptCacheKeyForCompat("gpt-5.3-codex"))
	require.True(t, shouldAutoInjectPromptCacheKeyForCompat("gpt-5.3-codex-spark"))
	require.False(t, shouldAutoInjectPromptCacheKeyForCompat("gpt-4o"))
}

func TestDeriveCompatPromptCacheKey_StableAcrossLaterTurns(t *testing.T) {
	base := &apicompat.ChatCompletionsRequest{
		Model: "gpt-5.4",
		Messages: []apicompat.ChatMessage{
			{Role: "system", Content: mustRawJSON(t, `"You are helpful."`)},
			{Role: "user", Content: mustRawJSON(t, `"Hello"`)},
		},
	}
	extended := &apicompat.ChatCompletionsRequest{
		Model: "gpt-5.4",
		Messages: []apicompat.ChatMessage{
			{Role: "system", Content: mustRawJSON(t, `"You are helpful."`)},
			{Role: "user", Content: mustRawJSON(t, `"Hello"`)},
			{Role: "assistant", Content: mustRawJSON(t, `"Hi there!"`)},
			{Role: "user", Content: mustRawJSON(t, `"How are you?"`)},
		},
	}

	k1 := deriveCompatPromptCacheKey(base, "gpt-5.4")
	k2 := deriveCompatPromptCacheKey(extended, "gpt-5.4")
	require.Equal(t, k1, k2, "cache key should be stable across later turns")
	require.NotEmpty(t, k1)
}

func TestDeriveCompatPromptCacheKey_ReusesAcrossDifferentFirstUsers(t *testing.T) {
	req1 := &apicompat.ChatCompletionsRequest{
		Model: "gpt-5.4",
		Messages: []apicompat.ChatMessage{
			{Role: "system", Content: mustRawJSON(t, `"You are helpful."`)},
			{Role: "user", Content: mustRawJSON(t, `"Question A"`)},
		},
	}
	req2 := &apicompat.ChatCompletionsRequest{
		Model: "gpt-5.4",
		Messages: []apicompat.ChatMessage{
			{Role: "system", Content: mustRawJSON(t, `"You are helpful."`)},
			{Role: "user", Content: mustRawJSON(t, `"Question B"`)},
		},
	}

	k1 := deriveCompatPromptCacheKey(req1, "gpt-5.4")
	k2 := deriveCompatPromptCacheKey(req2, "gpt-5.4")
	require.Equal(t, k1, k2, "different first user messages should reuse the same stable-prefix key")
}

func TestDeriveCompatPromptCacheKey_DiffersAcrossSystems(t *testing.T) {
	req1 := &apicompat.ChatCompletionsRequest{
		Model: "gpt-5.4",
		Messages: []apicompat.ChatMessage{
			{Role: "system", Content: mustRawJSON(t, `"System A"`)},
			{Role: "user", Content: mustRawJSON(t, `"Question A"`)},
		},
	}
	req2 := &apicompat.ChatCompletionsRequest{
		Model: "gpt-5.4",
		Messages: []apicompat.ChatMessage{
			{Role: "system", Content: mustRawJSON(t, `"System B"`)},
			{Role: "user", Content: mustRawJSON(t, `"Question A"`)},
		},
	}

	k1 := deriveCompatPromptCacheKey(req1, "gpt-5.4")
	k2 := deriveCompatPromptCacheKey(req2, "gpt-5.4")
	require.NotEqual(t, k1, k2, "different stable prefixes should still yield different keys")
}

func TestDeriveCompatPromptCacheKey_UsesResolvedSparkFamily(t *testing.T) {
	req := &apicompat.ChatCompletionsRequest{
		Model: "gpt-5.3-codex-spark",
		Messages: []apicompat.ChatMessage{
			{Role: "user", Content: mustRawJSON(t, `"Question A"`)},
		},
	}

	k1 := deriveCompatPromptCacheKey(req, "gpt-5.3-codex-spark")
	k2 := deriveCompatPromptCacheKey(req, " openai/gpt-5.3-codex-spark ")
	require.NotEmpty(t, k1)
	require.Equal(t, k1, k2, "resolved spark family should derive a stable compat cache key")
}

func TestDeriveAnthropicCompatPromptCacheKey_StableAcrossLaterTurns(t *testing.T) {
	base := &apicompat.AnthropicRequest{
		Model:  "claude-opus-4-6",
		System: mustRawJSON(t, `[{"type":"text","text":"x-anthropic-billing-header: test"},{"type":"text","text":"You are helpful."}]`),
		Messages: []apicompat.AnthropicMessage{
			{Role: "user", Content: mustRawJSON(t, `[{"type":"text","text":"Hello"}]`)},
		},
	}
	extended := &apicompat.AnthropicRequest{
		Model:  "claude-opus-4-6",
		System: mustRawJSON(t, `[{"type":"text","text":"x-anthropic-billing-header: another"},{"type":"text","text":"You are helpful."}]`),
		Messages: []apicompat.AnthropicMessage{
			{Role: "user", Content: mustRawJSON(t, `[{"type":"text","text":"Hello"}]`)},
			{Role: "assistant", Content: mustRawJSON(t, `[{"type":"text","text":"Hi there!"}]`)},
			{Role: "user", Content: mustRawJSON(t, `[{"type":"text","text":"How are you?"}]`)},
		},
	}

	k1 := deriveAnthropicCompatPromptCacheKey(base, "gpt-5.4")
	k2 := deriveAnthropicCompatPromptCacheKey(extended, "gpt-5.4")
	require.Equal(t, k1, k2, "anthropic compat cache key should stay stable across later turns")
	require.NotEmpty(t, k1)
}

func TestDeriveAnthropicCompatPromptCacheKey_ReusesAcrossDifferentFirstUsers(t *testing.T) {
	req1 := &apicompat.AnthropicRequest{
		Model:  "claude-opus-4-6",
		System: mustRawJSON(t, `[{"type":"text","text":"You are helpful."}]`),
		Messages: []apicompat.AnthropicMessage{
			{Role: "user", Content: mustRawJSON(t, `[{"type":"text","text":"Question A"}]`)},
		},
	}
	req2 := &apicompat.AnthropicRequest{
		Model:  "claude-opus-4-6",
		System: mustRawJSON(t, `[{"type":"text","text":"You are helpful."}]`),
		Messages: []apicompat.AnthropicMessage{
			{Role: "user", Content: mustRawJSON(t, `[{"type":"text","text":"Question B"}]`)},
		},
	}

	k1 := deriveAnthropicCompatPromptCacheKey(req1, "gpt-5.4")
	k2 := deriveAnthropicCompatPromptCacheKey(req2, "gpt-5.4")
	require.Equal(t, k1, k2, "different first user messages should reuse the same stable-prefix key")
}

func TestDeriveAnthropicCompatPromptCacheKey_DiffersAcrossSystems(t *testing.T) {
	req1 := &apicompat.AnthropicRequest{
		Model:  "claude-opus-4-6",
		System: mustRawJSON(t, `[{"type":"text","text":"System A"}]`),
		Messages: []apicompat.AnthropicMessage{
			{Role: "user", Content: mustRawJSON(t, `[{"type":"text","text":"Question A"}]`)},
		},
	}
	req2 := &apicompat.AnthropicRequest{
		Model:  "claude-opus-4-6",
		System: mustRawJSON(t, `[{"type":"text","text":"System B"}]`),
		Messages: []apicompat.AnthropicMessage{
			{Role: "user", Content: mustRawJSON(t, `[{"type":"text","text":"Question A"}]`)},
		},
	}

	k1 := deriveAnthropicCompatPromptCacheKey(req1, "gpt-5.4")
	k2 := deriveAnthropicCompatPromptCacheKey(req2, "gpt-5.4")
	require.NotEqual(t, k1, k2, "different stable prefixes should still yield different anthropic compat keys")
}

func TestDeriveAnthropicCompatPromptCacheKey_IgnoresReminderNoise(t *testing.T) {
	req1 := &apicompat.AnthropicRequest{
		Model:  "claude-opus-4-6",
		System: mustRawJSON(t, `[{"type":"text","text":"You are helpful."}]`),
		Messages: []apicompat.AnthropicMessage{
			{Role: "user", Content: mustRawJSON(t, `[{"type":"text","text":"<system-reminder>\nIgnore me\n</system-reminder>"},{"type":"text","text":"Hello"}]`)},
		},
	}
	req2 := &apicompat.AnthropicRequest{
		Model:  "claude-opus-4-6",
		System: mustRawJSON(t, `[{"type":"text","text":"You are helpful."}]`),
		Messages: []apicompat.AnthropicMessage{
			{Role: "user", Content: mustRawJSON(t, `[{"type":"text","text":"Hello"}]`)},
		},
	}

	k1 := deriveAnthropicCompatPromptCacheKey(req1, "gpt-5.4")
	k2 := deriveAnthropicCompatPromptCacheKey(req2, "gpt-5.4")
	require.Equal(t, k1, k2)
}
