# Fixes Tracker

## Issue 1: `/login` not showing in `/help`
**Problem**: The `/login` command is registered but doesn't appear in the help menu.

**Cause**: `internal/repl/dispatch.go` has a hardcoded list of commands in the `helpCmd.Run()` function instead of dynamically listing all registered commands.

**Fix**: Update `helpCmd.Run()` to iterate over the `commands` map and display all registered commands dynamically.

**Status**: [x] Fixed in commit 8e3c417

---

## Issue 2: Remove header in `/vars` listing
**Problem**: `/vars` command shows a header line that should be removed (similar to `/logs` fix earlier).

**Cause**: `internal/commands/vars.go` prints a header before listing variables.

**Fix**: Remove the header and separator line from the `/vars` listing output.

**Status**: [x] Fixed in commit 8e3c417

---

## Issue 3: BaseURL not working when set via `/vars set`
**Problem**: User set `baseURL` via `/vars set baseURL http://localhost:8000`, but requests still fail with "unsupported protocol scheme".

**Cause**: The executor (`internal/executor/client.go`) checks `env.BaseURL`, but `/vars set` stores to session `Vars`, not to the environment's `BaseURL` field.

**Resolution**: 
- Environment configuration (baseURL, headers) → Use `/env` commands
  - Set base URL: `/env set <env-name> url <base-url>`
  - Set headers: `/env set <env-name> <Header-Name> <value>`
- Variables (with scope) → Use `/vars` commands
  - Session scope (default): `/vars set name value`
  - Environment scope: `/vars set --env name value`
  - Shell scope: `/vars set --shell name value`

**Status**: [x] Resolved by design - documented correct usage

---

## Issue 4: Curl export doesn't show auth information
**Problem**: When exporting to curl format with `/export curl`, the authentication headers (Basic, Token, etc.) from the environment are not included.

**Cause**: `internal/export/curl.go` only exports request-level headers, not the auth config from the environment.

**Fix**: Update the curl exporter to:
1. Check if environment has auth configured
2. Add appropriate auth headers to the curl output (e.g., `-u user:pass` for basic, `-H "Authorization: Bearer ..."` for token)

**Status**: [x] Fixed in commit 8e3c417

---

## Issue 5: `/login basic` argument parsing
**Problem**: User typed `/login basic username:password` and it was parsed as username="username:password", password="" (empty).

**Expected**: The command should either:
- Accept `username:password` format and split on `:`
- Or show clear usage that password should be a separate argument

**Fix**: Update `/login basic` to support both formats:
- `/login basic username password` (current)
- `/login basic username:password` (new - split on `:`)

**Status**: [x] Fixed in commit 8e3c417

---

## Issue 6: `/env` help text not up to date
**Problem**: `/help` shows `/env` with outdated subcommands: "new, set, unset, list, show, copy"

**Cause**: The `Help()` function in `internal/commands/environment.go:15` has a hardcoded string that doesn't include all subcommands (like `set` for headers/vars, etc.)

**Fix**: Update the help string to accurately reflect available subcommands.

**Status**: [x] Fixed in commit 1069c17

---

## Issue 7: Add timestamp to `/vars` listing
**Problem**: When listing variables with `/vars`, there's no way to see when a variable was set/updated.

**Cause**: The `Variable` struct in `internal/model/vars.go` already has `Created` and `Updated` fields, but `/vars` listing doesn't display them.

**Fix**: Update `/vars` listing to show timestamp (use `Updated` field, or `Created` if not updated).

**Status**: [x] Fixed in commit 1069c17

---

## Issue 8: Add duration to `/logs` listing
**Problem**: The `/logs` command shows requests but doesn't display the duration (how long the request took).

**Cause**: `Request` struct has a `Duration` field (`internal/model/request.go:39`), but the `logsList()` function doesn't display it.

**Fix**: Add duration column to `/logs` output (format as ms or human-readable).

**Status**: [x] Fixed in commit 1069c17

---

## Progress

- [x] Issue 1: `/login` in help (commit 8e3c417)
- [x] Issue 2: Remove `/vars` header (commit 8e3c417)
- [x] Issue 3: BaseURL documentation/education (use `/env set <env> url <url>`)
- [x] Issue 4: Curl export with auth (commit 8e3c417)
- [x] Issue 5: `/login basic` parsing (commit 8e3c417)
- [x] Issue 6: `/env` help text update (commit 1069c17)
- [x] Issue 7: Timestamp in `/vars` listing (commit 1069c17)
- [x] Issue 8: Duration in `/logs` listing (commit 1069c17)

---

## Testing After Fixes

1. `/help` should show `/login` command
2. `/vars` should show only data, no header
3. User should know to use `/env set <env> url <url>` for base URL
4. `/export curl` should include auth headers
5. `/login basic user:pass` should work correctly
