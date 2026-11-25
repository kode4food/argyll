package worker

import _ "embed"

var (
	//go:embed prompts/chat.md
	chatPrompt string

	//go:embed prompts/chat-style.md
	chatStyle string

	//go:embed prompts/chat-nudge.md
	replyNudge string

	//go:embed prompts/step.md
	finishPrompt string

	//go:embed prompts/step-compact.md
	compactFinishPrompt string

	//go:embed prompts/step-nudge.md
	finishNudge string
)
