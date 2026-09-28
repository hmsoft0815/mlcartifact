<!-- mlc-dochub:begin — auto-managed, do not edit between these markers -->
## MLC Doc Hub — mlcartifact

Structured documentation lives in `.mlcai/`, maintained through the
`mlc-dochub` MCP server.

- **Project ID:** `mlcartifact` — mlcartifact
- **Source directory:** `/mnt/data2tb/mlcartifact` (read source files with native Read —
  the MCP tools only touch `.mlcai/`)

### Existing docs

Read any of these directly (native Read is fine) to gather context before you work:

- `.mlcai/INTEGRATION.md` — How this project fits into the larger system
- `.mlcai/TECH_STACK.md` — Stack & dependencies
- `.mlcai/API_CONTRACT.md` — API contract / endpoints
- `.mlcai/DECISION_LOG.md` — Architecture decisions
- `.mlcai/DETAIL_DOCS.md` — Project detail docs (links to docs/)
- `.mlcai/USER_DOCS.md` — End-user perspective
- `.mlcai/BACKLOG.md` — Index of open tickets (one file each: backlog/<ID>.md)
- `.mlcai/PRODUCT.md` — Product marketing page pointer (submodule meta)
- `.mlcai/WORKLOG.md` — Index of open work strands (one file each: worklog/<ID>.md)

### Product marketing page (separate concern)

Customer-facing marketing copy is **not** project documentation. It lives in
`./mlcprodweb/`, backed by `mlc@nas.local:/volume1/homes/mlc/repositories/product/mlcartifact.git`. Rendered live at https://mlcgo.eu/products/mlcartifact/.

### Tickets, worklog, release notes

`BACKLOG.md`, `WORKLOG.md` and `RELEASE_NOTES.md` are indexes the server
generates — never write them. Each entry is its own file, readable with
`get_doc` (or native Read): `backlog/<ID>.md`, `worklog/<ID>.md`,
`releases/<version>.md`. Tickets: `*_backlog_item` tools; close with
`update_backlog_item status=done` and the commit hash (`resolution=wontfix`
etc. when it was not fixed). A fix that still needs verifying:
`state=retest`; an accepted known issue: `state=known` (+ `user_facing=true`
if users notice it — it then belongs in the release notes). A missing or stale doc
is a doc ticket (`type=doc`, `doc_type=X.md`). At the end of a session write
**your** work strand: `update_worklog` with `entry_id` = the ticket you worked
on (or the `W-…` id you got back); it never touches another agent's strand.

### Working with `.mlcai/`

**Never write a `.mlcai/` file with a native editor.** Every create / update /
delete goes through the `mlc-dochub` MCP tools — they stamp the `## 📋 Meta`
footer, append to the activity log and guard against concurrent edits. Reading
with a native Read is fine and usually cheaper.

**If the tools are not available to you, read but do not write** — and point the
user at `task install-all` in the mlcintegration checkout (https://github.com/mlc911/mlcintegration).

The server states its full operating rules on connect (`author=`, `base_modified`,
which doc serves which purpose). Clients that drop server-level instructions —
Antigravity does, verified 29.08.2026 — get the same rules from the global
`~/.gemini/GEMINI.md`, section 6.
<!-- mlc-dochub:end -->
