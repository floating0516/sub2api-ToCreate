# AI consultation conversation history

The support page now offers a new conversation action and paginated server
history. The title comes from the first user message; no model call is needed.
Selecting a conversation restores its saved turns, evidence, draft and linked
ticket. Starting a conversation only changes the current selection and never
deletes saved data. It becomes a server session after its first message.

The existing per-account browser selection is preserved. A new device restores
the newest server session; explicitly starting an empty conversation remains
empty across refresh. Pending request navigation is disabled, unsent input
requires confirmation before switching, and failed switching preserves the
current conversation. Responses from an earlier signed-in account are ignored.

GET /api/v1/support/threads forwards pagination to the Agent's GET /threads.
The Agent uses its JWT subject for ownership. Titles are redacted, and list
responses contain no full turns, draft contents or private idempotency keys.

Base console: bff0f7aaf (tc1.53-rc.5). Agent base: 0acf8f2 (rc.20), retaining
the quality fixes developed in parallel. Production release is a separate step.
