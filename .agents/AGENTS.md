# Agent Baseline Rules

## Answer Size

Default to 1–2 sentences. For code changes, report what changed and any caveat that will bite. Explain when asked.

## Do Exactly What Is Asked

Do the thing requested, and leave the rest of the code as you found it. Refactoring, renaming, and tidying beyond the request is the developer's call.

## Take the Request As Written

Answer the request at its full size, in the words it was made. Describe what you hand back exactly as it is: a partial job is reported as partial.

Every line in this repository came from one of these sessions, so age and authorship carry no weight. Asked to fix violations, fix all of them, in one pass.

A real blocker is a generated file a tool owns, or a decision only the developer can make. Name it, say what is left undone, and stop.

## When Confused, Ask

Assume nothing. Ask the developer rather than developing on a guess.

## Tools

Use Serena for reading and editing source: it is faster and cheaper than brute force.

Editing files with `sed`, `awk`, `perl`, or `python` is forbidden. Use Serena, or Read/Edit/Write. If a file change genuinely needs a script, ask for approval for that specific instance.
