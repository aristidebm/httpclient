# Fixes Tracker

## Issue 1: `/login` not showing in `/help`
**Problem**: The `/login` command is registered but doesn't appear in the help menu.

**Cause**: `internal/repl/dispatch.go` has a hardcoded list of commands in the `helpCmd.Run()` function instead of dynamically listing all registered commands.

**Fix**: Update `helpCmd.Run()` to iterate over the `commands` map and display all registered commands dynamically.

**Status**: [ ] Not started

---

## Issue 2: Remove header in `/vars` listing
**Problem**: `/vars` command shows a header line that should be removed (similar to `/logs` fix earlier).

**Cause**: `internal/commands/vars.go` prints a header before listing variables.

**Fix**: Remove the header and separator line from the `/vars` listing output.

**Status**: [ ] Not started

---

## Issue 3: BaseURL not working when set via `/vars set`
**Problem**: User set `baseURL` via `/vars set baseURL http://localhost:8000`, but requests still fail with "unsupported protocol scheme".

**Cause**: The executor (`internal/executor/client.go`) checks `env.BaseURL`, but `/vars set` stores to session `Vars`, not to the environment's `BaseURL` field. The user should use `/env set <env-name> url <base-url>`.

**Fix**: 
- Educate user to use `/env set <env-name> url <base-url>` for setting base URL
- OR make the executor also check for a `baseURL` variable as fallback

**Status**: [ ] Not started

---

## Issue 4: Curl export doesn't show auth information
**Problem**: When exporting to curl format with `/export curl`, the authentication headers (Basic, Token, etc.) from the environment are not included.

**Cause**: `internal/export/curl.go` only exports request-level headers, not the auth config from the environment.

**Fix**: Update the curl exporter to:
1. Check if environment has auth configured
2. Add appropriate auth headers to the curl output (e.g., `-u user:pass` for basic, `-H "Authorization: Bearer ..."` for token)

**Status**: [ ] Not started

---

## Issue 5: `/login basic` argument parsing
**Problem**: User typed `/login basic username:password` and it was parsed as username="username:password", password="" (empty).

**Expected**: The command should either:
- Accept `username:password` format and split on `:`
- Or show clear usage that password should be a separate argument

**Fix**: Update `/login basic` to support both formats:
- `/login basic username password` (current)
- `/login basic username:password` (new - split on `:`)

**Status**: [ ] Not started

---

## Progress

- [ ] Issue 1: `/login` in help
- [ ] Issue 2: Remove `/vars` header
- [ ] Issue 3: BaseURL documentation/education
- [ ] Issue 4: Curl export with auth
- [ ] Issue 5: `/login basic` parsing

---

## Testing After Fixes

1. `/help` should show `/login` command
2. `/vars` should show only data, no header
3. User should know to use `/env set <env> url <url>` for base URL
4. `/export curl` should include auth headers
5. `/login basic user:pass` should work correctly
