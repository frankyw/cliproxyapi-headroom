# Validation — v0.5.0

Tested on Linux amd64 with CLIProxyAPI v7.3.17 and a live Headroom service reporting v0.37.0.

- Go tests with the race detector: protocol preservation across Chat, Anthropic, Responses/Codex and Gemini; default-on user-message compression, disabled user-message compression, runtime Headroom option, numeric precision, malformed compression responses, CCR rejection, timeout and original-request fallback.
- Persistent counters: concurrent updates, outcome accounting, atomic snapshots, restart restore, bounded history and preservation of corrupt input files.
- Service integration: all five Headroom endpoints, caching, readiness errors and filtering of sensitive metadata.
- Isolated host: native library registration, streaming and non-streaming requests, compressed user text with system instructions unchanged, preserved tool ID and unique error, public resource statistics and authenticated management APIs, sidebar menu, static page and persistence across restart.
- Production smoke: GPT-5.5 returned HTTP 200 and correctly identified the synthetic error. A repetitive 37,577-byte tool result was reduced to 177 bytes; Headroom estimated 10,532 -> 61 tokens for the projected tool output. This is a synthetic test, not a typical savings claim.
- Manager UI: the Headroom Stats menu and public dashboard rendered in CPA Manager Plus; narrow-layout cards, model breakdown and history storage status were checked. Both manager ports served public plugin resources and authenticated management APIs.

Token estimates cover eligible text accepted by the plugin before upstream execution, not provider billing. User-message compression is lossy and answer quality was not evaluated on production documents. WebSocket traffic was not independently tested. Historical Headroom totals include other clients and are displayed separately from new CPA-only counters.
