package service

import "strings"

func stripAnthropicBillingHeaderInstructions(instructions string) string {
	trimmed := strings.TrimSpace(instructions)
	if trimmed == "" {
		return ""
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 {
		return trimmed
	}
	if !strings.HasPrefix(strings.TrimSpace(lines[0]), "x-anthropic-billing-header:") {
		return trimmed
	}

	remaining := strings.TrimLeft(strings.Join(lines[1:], "\n"), "\n")
	return strings.TrimSpace(remaining)
}
