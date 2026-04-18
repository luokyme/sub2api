package service

import (
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

const compatPromptCacheKeyPrefix = "compat_cc_"

func shouldAutoInjectPromptCacheKeyForCompat(model string) bool {
	switch normalizeCodexModel(strings.TrimSpace(model)) {
	case "gpt-5.4", "gpt-5.3-codex":
		return true
	default:
		return false
	}
}

func deriveCompatPromptCacheKey(req *apicompat.ChatCompletionsRequest, mappedModel string) string {
	if req == nil {
		return ""
	}

	normalizedModel := normalizeCodexModel(strings.TrimSpace(mappedModel))
	if normalizedModel == "" {
		normalizedModel = normalizeCodexModel(strings.TrimSpace(req.Model))
	}
	if normalizedModel == "" {
		normalizedModel = strings.TrimSpace(req.Model)
	}

	seedParts := []string{"model=" + normalizedModel}
	if req.ReasoningEffort != "" {
		seedParts = append(seedParts, "reasoning_effort="+strings.TrimSpace(req.ReasoningEffort))
	}
	if len(req.ToolChoice) > 0 {
		seedParts = append(seedParts, "tool_choice="+normalizeCompatSeedJSON(req.ToolChoice))
	}
	if len(req.Tools) > 0 {
		if raw, err := json.Marshal(req.Tools); err == nil {
			seedParts = append(seedParts, "tools="+normalizeCompatSeedJSON(raw))
		}
	}
	if len(req.Functions) > 0 {
		if raw, err := json.Marshal(req.Functions); err == nil {
			seedParts = append(seedParts, "functions="+normalizeCompatSeedJSON(raw))
		}
	}

	for _, msg := range req.Messages {
		switch strings.TrimSpace(msg.Role) {
		case "system":
			seedParts = append(seedParts, "system="+normalizeCompatSeedJSON(msg.Content))
		}
	}

	return compatPromptCacheKeyPrefix + hashSensitiveValueForLog(strings.Join(seedParts, "|"))
}

func deriveAnthropicCompatPromptCacheKey(req *apicompat.AnthropicRequest, mappedModel string) string {
	if req == nil {
		return ""
	}

	normalizedModel := normalizeCodexModel(strings.TrimSpace(mappedModel))
	if normalizedModel == "" {
		normalizedModel = normalizeCodexModel(strings.TrimSpace(req.Model))
	}
	if normalizedModel == "" {
		normalizedModel = strings.TrimSpace(req.Model)
	}

	seedParts := []string{"model=" + normalizedModel}
	if req.OutputConfig != nil && req.OutputConfig.Effort != "" {
		seedParts = append(seedParts, "reasoning_effort="+strings.TrimSpace(req.OutputConfig.Effort))
	}
	if len(req.ToolChoice) > 0 {
		seedParts = append(seedParts, "tool_choice="+normalizeCompatSeedJSON(req.ToolChoice))
	}
	if len(req.Tools) > 0 {
		if raw, err := json.Marshal(req.Tools); err == nil {
			seedParts = append(seedParts, "tools="+normalizeCompatSeedJSON(raw))
		}
	}
	if len(req.System) > 0 {
		seedParts = append(seedParts, "system="+normalizeCompatSeedJSON(stripAnthropicBillingHeaderRaw(req.System)))
	}
	return compatPromptCacheKeyPrefix + hashSensitiveValueForLog(strings.Join(seedParts, "|"))
}

func normalizeCompatSeedJSON(v json.RawMessage) string {
	if len(v) == 0 {
		return ""
	}
	var tmp any
	if err := json.Unmarshal(v, &tmp); err != nil {
		return string(v)
	}
	out, err := json.Marshal(tmp)
	if err != nil {
		return string(v)
	}
	return string(out)
}

func stripAnthropicBillingHeaderRaw(v json.RawMessage) json.RawMessage {
	if len(v) == 0 {
		return v
	}

	var blocks []map[string]any
	if err := json.Unmarshal(v, &blocks); err != nil {
		return v
	}

	filtered := make([]map[string]any, 0, len(blocks))
	for _, block := range blocks {
		text, _ := block["text"].(string)
		if strings.HasPrefix(strings.TrimSpace(text), "x-anthropic-billing-header:") {
			continue
		}
		filtered = append(filtered, block)
	}

	out, err := json.Marshal(filtered)
	if err != nil {
		return v
	}
	return out
}

func filterAnthropicCompatMessagesForKey(messages []apicompat.AnthropicMessage) []apicompat.AnthropicMessage {
	filtered := make([]apicompat.AnthropicMessage, 0, len(messages))
	for _, msg := range messages {
		content := filterAnthropicCompatContentForKey(msg.Content)
		if len(content) == 0 || string(content) == "[]" || string(content) == `""` {
			continue
		}
		filtered = append(filtered, apicompat.AnthropicMessage{
			Role:    msg.Role,
			Content: content,
		})
	}
	return filtered
}

func filterAnthropicCompatContentForKey(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}

	var blocks []map[string]any
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return raw
	}

	filtered := make([]map[string]any, 0, len(blocks))
	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		text, _ := block["text"].(string)
		trimmed := strings.TrimSpace(text)
		if blockType == "text" {
			if strings.HasPrefix(trimmed, "<system-reminder>") ||
				strings.HasPrefix(trimmed, "<local-command-caveat>") ||
				strings.HasPrefix(trimmed, "<command-name>/clear</command-name>") ||
				strings.HasPrefix(trimmed, "<local-command-stdout>") {
				continue
			}
		}
		filtered = append(filtered, block)
	}

	out, err := json.Marshal(filtered)
	if err != nil {
		return raw
	}
	return out
}
