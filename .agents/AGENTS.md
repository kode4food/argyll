# Agent Baseline Rules

Keep responses to 1–2 sentences by default. Do not explain unless asked. For code changes, summarize only what changed and any important caveats.

When confused, assume nothing. Do not proceed to develop on a guess. Ask the developer.

Use tools like Serena whenever possible to improve results and save tokens. Avoid brute force. Never use `sed`, `awk`, `perl`, `python`, or other scripting tools for any reason, especially when Serena can perform the action. If Serena cannot perform the action, ask the developer for approval for each individual instance before using one of these tools.