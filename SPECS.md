# cdapi — Go Implementation Plan

> This document is a complete implementation guide for a coding agent.
> Work through phases sequentially. Each phase has acceptance criteria —
> do not move to the next phase until all criteria pass.

---

## Project Bootstrap

```
cdapi/
├── main.go
├── go.mod
├── go.sum
├── README.md
├── cmd/
│   └── root.go
└── internal/
    ├── model/
    ├── executor/
    ├── session/
    ├── environment/
    ├── input/
    ├── export/
    ├── commands/
    └── repl/
```

**`go.mod` dependencies:**

```
github.com/chzyer/readline          v1.5.1
github.com/itchyny/gojq             v0.12.16
github.com/getkin/kin-openapi/v3    v0.127.0
github.com/fatih/color              v1.17.0
github.com/spf13/cobra              v1.8.1
gopkg.in/yaml.v3                    v3.0.1
```

---

## Phase 1 — Data Model & Project Skeleton

**Goal:** Compilable project with all core types defined. No runtime behavior yet.

### 1.1 — `internal/model/request.go`

- [ ] Define `Request` struct:
  ```go
  type Request struct {
      ID          string
      Method      string
      URL         string
      Headers     map[string]string
      Params      map[string]string
      Body        []byte
      ContentType string
      Response    *Response
      ExecutedAt  time.Time
      Duration    time.Duration
      Note        string  // optional user label
  }
  ```
- [ ] Define `Response` struct:
  ```go
  type Response struct {
      StatusCode int
      Status     string
      Headers    map[string]string
      Body       []byte
      RawBody    []byte  // original, unmodified bytes for export fidelity
  }
  ```
- [ ] Add `func (r *Request) Clone() *Request` — deep copy, clears Response/ExecutedAt/Duration
- [ ] Add `func (r *Request) IsExecuted() bool` — returns `r.Response != nil`

### 1.2 — `internal/model/environment.go`

- [ ] Define `Environment` struct:
  ```go
  type Environment struct {
      Name    string
      BaseURL string            // trailing slash normalized on set
      Headers map[string]string // applied to every request in bound sessions
      Vars    map[string]any
  }
  ```
- [ ] Add `func (e *Environment) Clone() *Environment`
- [ ] Add `func (e *Environment) Resolve(key string) (any, bool)` — looks up Vars then Headers

### 1.3 — `internal/model/session.go`

- [ ] Define `Session` struct:
  ```go
  type Session struct {
      ID              string
      Name            string
      EnvName         string
      ParentID        string
      Requests        []*Request
      HeaderOverrides map[string]string
      VarOverrides    map[string]any
      CreatedAt       time.Time
  }
  ```
- [ ] Add `func (s *Session) NextRequestID() string` — returns `"r1"`, `"r2"`, etc.
- [ ] Add `func (s *Session) GetRequest(id string) (*Request, bool)`
- [ ] Add `func (s *Session) AddRequest(r *Request)` — assigns ID, appends
- [ ] Add `func (s *Session) RemoveRequest(id string) bool`

### 1.4 — `internal/model/tree.go`

- [ ] Define `SessionTree` struct:
  ```go
  type SessionTree struct {
      Sessions    map[string]*Session   // keyed by Session.ID
      CurrentID   string
      Environments map[string]*Environment
  }
  ```
- [ ] Add `func NewSessionTree() *SessionTree` — creates a default `local` environment and a `default` session bound to it
- [ ] Add `func (t *SessionTree) Current() *Session`
- [ ] Add `func (t *SessionTree) CurrentEnv() *Environment` — resolves from current session's EnvName
- [ ] Add `func (t *SessionTree) Children(sessionID string) []*Session`

### 1.5 — Variable resolution

- [ ] In `internal/model/vars.go`, implement `ResolveVars(template string, layers ...map[string]any) string`
  - Replaces `{key}` placeholders, iterating layers from highest to lowest priority
  - Priority order: request-level → session overrides → environment vars → shell globals
  - Returns original `{key}` untouched if not found in any layer
  - Returns an error list of unresolved keys

**Phase 1 acceptance criteria:**
- [ ] `go build ./...` succeeds with no errors
- [ ] All structs have unit tests covering Clone, GetRequest, NextRequestID, ResolveVars

---

## Phase 2 — Persistence Layer

**Goal:** Session tree and environments survive process restarts.

### 2.1 — `internal/session/store.go`

- [ ] Define storage paths:
  ```go
  const (
      ConfigDir        = "~/.cdapi"
      EnvironmentsFile = "~/.cdapi/environments.json"
      SessionsFile     = "~/.cdapi/sessions.json"
      ConfigFile       = "~/.cdapi/config.toml"
  )
  ```
- [ ] Expand `~` using `os.UserHomeDir()`, create `~/.cdapi/` if it does not exist
- [ ] `func SaveTree(tree *model.SessionTree) error` — marshals full tree to `sessions.json`
- [ ] `func LoadTree() (*model.SessionTree, error)` — unmarshals; on first run returns `NewSessionTree()`
- [ ] `func SaveEnvironments(envs map[string]*model.Environment) error`
- [ ] `func LoadEnvironments() (map[string]*model.Environment, error)`
- [ ] Response bodies are stored as base64 strings inside JSON — use a custom `json.Marshaler` on `Response.RawBody`
- [ ] If `sessions.json` is corrupt, log a warning and start fresh rather than crashing

### 2.2 — `internal/session/config.go`

- [ ] Define `Config` struct:
  ```go
  type Config struct {
      DefaultEnv    string
      DefaultEditor string  // fallback if $EDITOR/$VISUAL not set
      HistoryFile   string
  }
  ```
- [ ] `func LoadConfig() (*Config, error)` — reads `config.toml`, returns defaults if missing
- [ ] `func SaveConfig(c *Config) error`

**Phase 2 acceptance criteria:**
- [ ] Write a tree, kill the process, restart, tree is intact
- [ ] Corrupt `sessions.json` → clean start, no panic
- [ ] Unit tests for marshal/unmarshal round-trip of Request with binary body

---

## Phase 3 — HTTP Executor

**Goal:** Execute requests against real servers, populate Response on the Request.

### 3.1 — `internal/executor/client.go`

- [ ] Define `Client` struct wrapping `*http.Client` with configurable timeout
- [ ] `func NewClient(timeout time.Duration) *Client`
- [ ] `func (c *Client) Execute(req *model.Request, env *model.Environment) error`:
  - Merge env default headers + session header overrides + request headers (request wins)
  - Resolve `{var}` in URL and body using the full var layer stack
  - Set `req.ExecutedAt = time.Now()` before sending
  - Populate `req.Response` from the HTTP response
  - Store raw body bytes in `Response.RawBody`; attempt JSON parse into `Response.Body` for pretty-printing
  - Set `req.Duration`

### 3.2 — `internal/executor/auth.go`

- [ ] `func ApplyAuth(req *http.Request, env *model.Environment)` — checks `env.Headers` for `Authorization` and applies it
- [ ] Support token auth header format `TOKEN <value>` (matching existing Python tool behaviour)

### 3.3 — Error handling

- [ ] Network errors → return typed `ExecutorError{Kind: "network", Cause: err}`
- [ ] Timeout → return `ExecutorError{Kind: "timeout"}`
- [ ] Non-2xx responses are **not** errors — they are stored in `Response` normally

**Phase 3 acceptance criteria:**
- [ ] Integration test: execute a real GET against `https://httpbin.org/get`, verify status 200 and response body populated
- [ ] Test timeout: set 1ms timeout, verify `ExecutorError{Kind: "timeout"}` returned
- [ ] Test header merging: env header `X-Foo: env`, request header `X-Foo: req` → request value wins

---

## Phase 4 — REPL Shell

**Goal:** Interactive shell with prompt, command dispatch, history, and tab completion.

### 4.1 — `internal/repl/shell.go`

- [ ] Define `ShellContext`:
  ```go
  type ShellContext struct {
      Tree      *model.SessionTree
      Executor  *executor.Client
      OpenAPI   *openapi.Spec       // nil if not loaded
      Vars      map[string]any      // shell-global variables
      LastResp  *model.Response
      LastData  any                 // parsed JSON of last response, or raw string
      LastReqID string              // ID of last executed request in current session
  }
  ```
- [ ] Main loop using `readline.NewEx()`:
  - Read line
  - Strip leading whitespace
  - If starts with `!` → shell passthrough (see Phase 8)
  - If starts with `/` → strip `/`, dispatch to command registry
  - If contains `=` with valid identifier LHS → variable assignment (eval RHS)
  - Otherwise → evaluate as expression in shell context (print result)
- [ ] On `Ctrl-C` → cancel current operation, do not exit
- [ ] On `Ctrl-D` / `/exit` → save tree, save environments, exit cleanly

### 4.2 — `internal/repl/prompt.go`

- [ ] Build prompt string: `[cdapi : {user}@{env} : {session}] › `
- [ ] Color coding:
  - `cdapi` — dim grey
  - `{user}@{env}` — cyan
  - `{session}` — green
  - `›` — grey
- [ ] When no environment bound yet, show: `[cdapi] › ` in blue

### 4.3 — `internal/repl/dispatch.go`

- [ ] Define `Command` interface:
  ```go
  type Command interface {
      Name()     string
      Aliases()  []string
      Run(ctx *ShellContext, args []string) error
      Complete(ctx *ShellContext, partial string) []string
      Help()     string
  }
  ```
- [ ] `CommandRegistry` — map of name → Command, populated at startup
- [ ] `Register(cmd Command)` — also registers all aliases
- [ ] `Dispatch(ctx *ShellContext, line string) error` — tokenizes with shell-quote rules, looks up command, calls `Run`
- [ ] On unknown command → print `unknown command: /foo — type /help` (no exit)

### 4.4 — `internal/repl/complete.go`

- [ ] Implement `readline.AutoCompleter` backed by registered commands
- [ ] Command name completion: prefix-match on `/` prefix
- [ ] Argument completion: delegate to `Command.Complete()` of the matched command
- [ ] Route completion: if OpenAPI spec is loaded, complete endpoint paths
- [ ] Session name completion: for `/session switch`, `/session drop`, etc.
- [ ] Variable name completion: for `/jq`, `/update`, `/edit`, `/clip`
- [ ] Request ID completion: for `/replay`, `/watch`, `/session move`
- [ ] Environment name completion: for `/env` subcommands, `/session new`

### 4.5 — `internal/repl/render.go`

- [ ] `PrintResponse(resp *model.Response)`:
  - Status line with color: 2xx → green, 3xx → cyan, 4xx → yellow, 5xx → red
  - Pretty-print JSON body with 2-space indent
  - For binary responses: print content-type and size only
  - For text/non-JSON responses: print raw text
- [ ] `PrintRequest(req *model.Request)` — summary line: `[r3] POST /betonnages/ → 201 Created (142ms)`
- [ ] `PrintError(err error)` — red color, `Error: {message}`
- [ ] `PrintInfo(msg string)` — blue color
- [ ] `PrintSuccess(msg string)` — green color

**Phase 4 acceptance criteria:**
- [ ] Shell starts, shows prompt, accepts `/help`, `/exit`
- [ ] Unknown command shows helpful error, does not crash
- [ ] Tab completes `/` prefixed command names
- [ ] History persists across restarts (`~/.cdapi/history`)

---

## Phase 5 — HTTP Verb Commands

**Goal:** All HTTP verbs work as `/get`, `/post`, etc.

### 5.1 — `internal/commands/http.go`

Implement one struct per verb, all sharing a common `httpCmd` base:

```go
type httpCmd struct {
    method string
}
func (h *httpCmd) Name() string    { return strings.ToLower(h.method) }
func (h *httpCmd) Aliases() []string { return nil }
func (h *httpCmd) Run(ctx *ShellContext, args []string) error { ... }
```

- [ ] Register commands for: `get`, `post`, `put`, `delete`, `patch`, `head`, `options`, `trace`, `connect`
- [ ] Flag parsing within `Run` using `flag.NewFlagSet`:

  | Flag | Description |
  |------|-------------|
  | `-p key:val` or `-p key=expr` | Query params. `:` is literal. `=` evaluates against ShellContext |
  | `-d key:val` or `-d key=expr` | Request body fields |
  | `-d **varname=` | Entire body from variable (splat syntax) |
  | `-H key:val` | Request headers |
  | `-a json\|pdf\|csv\|…` | Accept header shortcut |
  | `--form` | Send body as `application/x-www-form-urlencoded` |
  | `--timeout N` | Per-request timeout in seconds |
  | `--save` | Auto-save binary response to file |
  | `-t` | Print execution time |
  | `--doc` | Show OpenAPI docs for this route instead of executing |
  | `--note "text"` | Attach a human label to this request in the session log |

- [ ] Accept header shortcuts map (same as Python version): `json`, `xml`, `pdf`, `excel`, `xls`, `xlsx`, `csv`, `html`, `text`, `zip`
- [ ] Resolve endpoint URL: if starts with `/` use root of base URL; otherwise join with base URL
- [ ] `{var}` substitution in endpoint using `ResolveVars`
- [ ] After execution:
  - Add request to current session via `session.AddRequest`
  - Set `ctx.LastResp = req.Response`
  - Set `ctx.LastData` to parsed JSON (`map[string]any` or `[]any`) or raw string
  - Set `ctx.LastReqID = req.ID`
  - Call `render.PrintResponse`
- [ ] `Complete` returns OpenAPI routes filtered by prefix if spec is loaded

### 5.2 — `internal/commands/http_test.go`

- [ ] Test flag parsing: `-p`, `-d`, `-H`, `--form`, splat syntax
- [ ] Test URL resolution: relative path, absolute path, full URL
- [ ] Test var interpolation in URL and body
- [ ] Integration test against `httpbin.org`: GET, POST with JSON body, check response stored in session

**Phase 5 acceptance criteria:**
- [ ] `/get https://httpbin.org/get` executes and prints pretty JSON
- [ ] `/post https://httpbin.org/post -d name:alice age:30` sends JSON body
- [ ] Request is stored in current session with correct ID
- [ ] `ctx.LastData` holds parsed response body

---

## Phase 6 — Session & Environment Commands

### 6.1 — `/session` — `internal/commands/session.go`

- [ ] `/session new <name> <env>` — create new root session bound to named environment; switch to it; error if env does not exist
- [ ] `/session branch <name> [env]` — fork current session: new session gets same EnvName (or override), ParentID = current session ID, Requests list starts empty; switch to new session
- [ ] `/session switch <name>` — change `tree.CurrentID`; tab-completes session names
- [ ] `/session list` — print tree view showing hierarchy, env, request count, creation time:
  ```
  ● default          [local]   3 requests   2024-01-15 10:00
    └─ debug-run     [beta]    5 requests   2024-01-15 10:45  ← current
  ```
- [ ] `/session rename <old> <new>` — rename; update all ParentID references
- [ ] `/session drop <name>` — refuse if session has children; confirm prompt before deleting
- [ ] `/session move <req-id> <target-session-name>` — move a request from current session to target; re-sequence IDs in both sessions after move

### 6.2 — `/env` — `internal/commands/environment.go`

- [ ] `/env new <name> <base-url>` — create environment; normalize base URL (ensure trailing slash)
- [ ] `/env set <name> <key> <value>` — set a var or default header (keys starting with uppercase treated as headers, others as vars; or use a `--header` flag)
- [ ] `/env unset <name> <key>` — remove a key
- [ ] `/env list` — table view: name, base URL, header count, var count
- [ ] `/env show <name>` — full detail: URL, all headers, all vars
- [ ] `/env copy <src> <dst>` — deep clone; error if `dst` already exists

**Phase 6 acceptance criteria:**
- [ ] Create two environments (`beta`, `local`), create sessions bound to each, switch between them; prompt reflects correct env/session
- [ ] Branch a session; `/session list` shows correct parent-child tree
- [ ] Move a request between sessions; IDs re-sequence correctly
- [ ] Drop a session with children → refused with clear error

---

## Phase 7 — Input Loaders

**Goal:** `/load` accepts OpenAPI specs, `.http` files, and curl command lists.

### 7.1 — Format detection — `internal/input/detect.go`

- [ ] `func Detect(input []byte) (InputFormat, error)` implementing this priority:
  1. Try JSON parse → if root has `"openapi"` or `"swagger"` key → `FormatOpenAPI`
  2. Try YAML parse → same keys → `FormatOpenAPI`
  3. Lines starting with `curl ` → `FormatCurl`
  4. Lines matching `(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS) ` → `FormatHTTPFile`
  5. Return `FormatUnknown` error

### 7.2 — OpenAPI loader — `internal/input/openapi.go`

- [ ] `func LoadOpenAPI(data []byte) (*Spec, error)` using `kin-openapi`
- [ ] `Spec` struct:
  ```go
  type Spec struct {
      Title   string
      Version string
      Routes  []Route
      raw     *openapi3.T
  }
  type Route struct {
      Method      string
      Path        string      // e.g. "/betonnages/{id}/"
      Summary     string
      Tags        []string
      Parameters  []Parameter
      RequestBody *SchemaRef
  }
  ```
- [ ] `func (s *Spec) RoutesForMethod(method string) []string` — returns path list for tab completion
- [ ] `func (s *Spec) DocFor(method, path string) string` — returns formatted documentation string for `--doc` flag

### 7.3 — HTTP file parser — `internal/input/httpfile.go`

Parse the subset of the [RFC 7230 `.http` format](https://www.jetbrains.com/help/idea/exploring-http-syntax.html) used by JetBrains/VS Code:

- [ ] Split on `###` separators
- [ ] For each block:
  - First non-comment line: `METHOD URL` (and optional `HTTP/1.1`)
  - Following `Key: Value` lines: headers (until blank line)
  - Remaining content: body
  - Comments starting with `#` or `//` ignored
  - `@variable = value` declarations → session vars
- [ ] Return `[]model.Request`

### 7.4 — curl parser — `internal/input/curl.go`

- [ ] Parse a curl command string (single or multiline with `\`) into a `model.Request`
- [ ] Support flags: `-X`/`--request`, `-H`/`--header`, `-d`/`--data`/`--data-raw`/`--data-binary`, `-u`/`--user`, `--url`, `-G`/`--get`, `-F`/`--form`, `--json`
- [ ] If multiple curl commands (one per line or `###`-separated), return `[]model.Request`

### 7.5 — `/load` command — `internal/commands/load.go`

- [ ] `/load <filepath-or-url>` — reads from file or fetches URL
- [ ] Auto-detect format using `input.Detect`
- [ ] OpenAPI: store spec in `ctx.OpenAPI`; print route count summary
- [ ] HTTP file / curl list: append parsed requests (un-executed) to current session; print count added
- [ ] Tab-complete: local file paths

**Phase 7 acceptance criteria:**
- [ ] Load a real OpenAPI spec from URL; `/list` shows routes; tab completion works on routes
- [ ] Load a `.http` file; requests appear in session (un-executed)
- [ ] Parse `curl -X POST https://example.com/api -H 'Authorization: TOKEN abc' -d '{"name":"test"}'` correctly

---

## Phase 8 — Export Commands

### 8.1 — Export interface — `internal/export/exporter.go`

```go
type Exporter interface {
    Format() string
    Export(session *model.Session, env *model.Environment) ([]byte, error)
}
```

- [ ] `func Get(format string) (Exporter, error)` — registry lookup; supported: `json`, `curl`, `har`, `http`, `bruno`

### 8.2 — JSON exporter — `internal/export/json.go`

- [ ] Full session dump including environment block and all requests+responses
- [ ] Response bodies base64-encoded if binary, plain if UTF-8 JSON
- [ ] Output is a valid JSON object importable by other tools

### 8.3 — curl exporter — `internal/export/curl.go`

- [ ] One `curl` invocation per request that has been executed
- [ ] Skip un-executed requests
- [ ] Inline auth headers from environment
- [ ] Expand base URL into full URL
- [ ] Add `###` separator with request ID and note between commands

### 8.4 — HAR exporter — `internal/export/har.go`

- [ ] Produce valid [HAR 1.2](http://www.softwareishard.com/blog/har-12-spec/) JSON
- [ ] `log.creator.name` = `"cdapi"`
- [ ] Populate `timings.send`, `timings.wait`, `timings.receive` from `req.Duration` (distribute evenly if breakdown not available)
- [ ] `startedDateTime` from `req.ExecutedAt` in RFC 3339 format
- [ ] Include response `content.mimeType` and `content.text`

### 8.5 — HTTP file exporter — `internal/export/httpfile.go`

- [ ] One block per request separated by `###`
- [ ] Block format:
  ```
  ### r1 — GET /betonnages/
  GET {{baseUrl}}/betonnages/
  Authorization: TOKEN {{token}}
  
  ```
- [ ] Variable declarations at top of file: `@baseUrl = {env.BaseURL}`
- [ ] Only request data — no responses in output

### 8.6 — Bruno exporter — `internal/export/bruno.go`

- [ ] Output is a directory, not a single file. Structure:
  ```
  {session-name}/
  ├── bruno.json          collection metadata
  ├── environments/
  │   └── {env-name}.bru  environment file
  └── {r1-note-or-id}.bru
      {r2-note-or-id}.bru
      …
  ```
- [ ] Each `.bru` file format:
  ```
  meta {
    name: r1 — GET /betonnages/
    type: http
    seq: 1
  }
  
  http {
    method: GET
    url: {{baseUrl}}/betonnages/
  }
  
  headers {
    Authorization: TOKEN {{token}}
  }
  
  body:json {
    {}
  }
  ```
- [ ] Environment file:
  ```
  vars {
    baseUrl: https://beta.concretedispatch.eu/api/
  }
  ```

### 8.7 — `/export` command — `internal/commands/export.go`

- [ ] `/export [format] [session-name] [--out path]`
  - `format` defaults to `json`
  - `session-name` defaults to current session
  - `--out` defaults to stdout; for Bruno (directory output) `--out` is required or uses `./out/{session-name}`
- [ ] Tab-complete: format names, then session names

**Phase 8 acceptance criteria:**
- [ ] Export current session to each of the 5 formats; validate output is well-formed
- [ ] HAR output passes [HAR validator](https://har-validator.github.io/har-validator/)
- [ ] Bruno output can be imported into Bruno without errors
- [ ] HTTP file output can be re-loaded via `/load` and round-trips cleanly

---

## Phase 9 — Utility Commands

### 9.1 — `/jq` — `internal/commands/jq.go`

- [ ] `/jq <pattern>` — run pattern against `ctx.LastData` using `gojq`
- [ ] `-f` flag → return only first result
- [ ] Store result back in `ctx.LastData` and `ctx.LastResp` (as updated last output)
- [ ] Pretty-print result as JSON; fallback to `%v` if not serialisable
- [ ] Tab-complete: no completion (pattern is freeform)

### 9.2 — `/update` — `internal/commands/update.go`

- [ ] `/update [var-name]` — open `ctx.LastData` (or named variable) in `$VISUAL` / `$EDITOR` / `nano` / `vi`
- [ ] Write content to a temp file (`/tmp/cdapi_*.json`)
- [ ] Wait for editor to exit
- [ ] If mtime unchanged → print "No changes" and return
- [ ] Try to parse edited content as JSON; if valid → store as `map[string]any`; otherwise store as raw string
- [ ] Update `ctx.LastData` (if no var name) or `ctx.Vars[varName]`
- [ ] Tab-complete: variable names

### 9.3 — `/edit` — `internal/commands/edit.go`

- [ ] `/edit <var-name>` — same editor flow as `/update` but targets a named variable
- [ ] If variable does not exist → start with `{}` template
- [ ] After editing: store in `ctx.Vars[varName]`
- [ ] Tab-complete: existing variable names

### 9.4 — `/clip` — `internal/commands/clip.go`

- [ ] `/clip [var-name|req-id]` — copy to system clipboard
  - No argument → copy `ctx.LastData`
  - Variable name → copy `ctx.Vars[name]`
  - Request ID (e.g. `r3`) → copy that request's response body from current session
- [ ] Clipboard backend detection order: `pbcopy` (macOS), `wl-copy` (Wayland), `xclip -selection clipboard`, `xsel --clipboard --input`
- [ ] Print character count on success; print actionable error if no backend found
- [ ] Tab-complete: variable names + request IDs from current session

### 9.5 — `/replay` — `internal/commands/replay.go`

- [ ] `/replay [req-id|all]`
  - `req-id` — clone the specified request, re-execute, append as a new request with next ID
  - `all` — replay every request in current session in order; each replayed request gets a new ID
- [ ] Cloned requests preserve method, URL, headers, body, params but clear Response/ExecutedAt/Duration
- [ ] Print each result as it executes

### 9.6 — `/watch` — `internal/commands/watch.go`

- [ ] `/watch [req-id] [interval]`
  - `req-id` defaults to `ctx.LastReqID`
  - `interval` defaults to `2` seconds
- [ ] Loop: re-execute the request, print timestamped response, sleep interval
- [ ] `Ctrl-C` stops the loop gracefully
- [ ] Each iteration creates a new request in the session log (same as `/replay` behavior)

### 9.7 — `/list` — `internal/commands/list.go`

- [ ] `/list [filter]` — list routes from loaded OpenAPI spec
- [ ] Groups by tag (same as Python `do_routes`)
- [ ] Paged output using the system pager (`$PAGER` → `less` → `more`)
- [ ] Filter matches against path, summary, or tag (case-insensitive substring)
- [ ] If no spec loaded → print actionable hint: `No spec loaded. Use /load <url-or-file>`

### 9.8 — `/save` — `internal/commands/save.go`

- [ ] `/save [filename]` — save binary response body of last request to disk
- [ ] If no filename → extract from `Content-Disposition` header; fallback to `cdapi_{timestamp}{ext}`
- [ ] Print saved path on success

### 9.9 — `/info` — `internal/commands/info.go`

- [ ] Print current shell state:
  ```
  SESSION  : debug-run
  ENV      : beta  (https://beta.concretedispatch.eu/api/)
  USER     : test-courbevoie
  OPENAPI  : Spec loaded (243 routes)
  LAST REQ : r5  PATCH /betonnages/42/  →  200 OK (88ms)
  VARS     : payload, page_size (2 defined)
  ```

### 9.10 — Shell passthrough — `internal/repl/shell.go`

- [ ] Lines starting with `!` (after stripping `/` prefix is not present) → run remainder as shell command via `os/exec` with `sh -c`
- [ ] `{var}` substitution applied to the command string from `ctx.Vars`
- [ ] Inherit stdin/stdout/stderr so interactive commands work

### 9.11 — `/help` — `internal/commands/help.go`

- [ ] `/help` → print module docstring (full command list with one-line descriptions)
- [ ] `/help <command>` → print that command's full Help() string

**Phase 9 acceptance criteria:**
- [ ] `/jq .results[0].id` against a paginated response returns correct value
- [ ] `/replay r2` creates `r3` with same params; `/replay all` creates N new requests
- [ ] `/watch 2` polls every 2 seconds; stops cleanly on Ctrl-C
- [ ] `/clip` works on macOS and Linux (test both backends if available)
- [ ] `/save` correctly names file from `Content-Disposition`

---

## Phase 10 — CLI One-shot Mode

**Goal:** `cdapi /get <url> -u user@env` works without entering the shell, matching the Python CLI mode.

### 10.1 — `cmd/root.go`

- [ ] Use `cobra` root command
- [ ] Default (no args) → start interactive shell
- [ ] Single arg that does not start with `-` → start shell and run `/session new default <arg>` (login flow)
- [ ] Flags for one-shot mode:
  ```
  --method / -m / -X    HTTP method (default GET)
  --user / -u           user identifier
  --env                 environment name or base URL
  --token               explicit auth token
  -p, -d, -H, -a        same as shell flags
  --timeout             request timeout
  ```
- [ ] One-shot flow: build environment from flags, execute single request, print result, exit
- [ ] Binary responses auto-saved in one-shot mode (same as Python CLI)

**Phase 10 acceptance criteria:**
- [ ] `cdapi /get https://httpbin.org/get` prints JSON response and exits
- [ ] `cdapi /post https://httpbin.org/post -d name:test` sends body and exits
- [ ] Exit code 0 on 2xx, 1 on error or non-2xx

---

## Phase 11 — Polish & Testing

### 11.1 — Tab completion audit

- [ ] All commands have `Complete()` returning useful suggestions
- [ ] Multi-level completion for `/session`, `/env` subcommands
- [ ] Request ID completion respects current session
- [ ] File path completion for `/load` and `/save`

### 11.2 — Variable assignment

- [ ] `foo = /get https://httpbin.org/get` → execute request, assign response body to `foo`
- [ ] `foo = 42` → assign literal integer
- [ ] `foo = d` → assign `ctx.LastData` to `foo`
- [ ] `{foo}` works in subsequent endpoint strings and body values

### 11.3 — Config file

- [ ] `~/.cdapi/config.toml` supports:
  ```toml
  default_env    = "beta"
  default_editor = "nano"
  history_file   = "~/.cdapi/history"
  pager          = "less"
  ```
- [ ] Environment pre-population: if a TOML `[environments]` block is present, load them on first run

### 11.4 — Robustness

- [ ] All commands recover from panics and print an error rather than crashing the shell
- [ ] All file I/O errors print actionable messages, not stack traces
- [ ] Long responses are paged automatically if `len(body) > 4096` and stdout is a TTY
- [ ] Binary response detection: if `Content-Type` is not text or JSON, print metadata only and suggest `/save`

### 11.5 — Test coverage targets

- [ ] `internal/model` — 90% coverage
- [ ] `internal/executor` — 80% coverage (integration tests against httpbin.org)
- [ ] `internal/input` — 90% coverage (parsers)
- [ ] `internal/export` — 85% coverage (output validation)
- [ ] `internal/commands` — 70% coverage (unit tests with mock executor)

---

## Appendix A — Command Reference (Complete)

```
HTTP Verbs
  /get    <endpoint> [flags]
  /post   <endpoint> [flags]
  /put    <endpoint> [flags]
  /delete <endpoint> [flags]
  /patch  <endpoint> [flags]
  /head   <endpoint> [flags]
  /options <endpoint> [flags]
  /trace  <endpoint> [flags]
  /connect <endpoint> [flags]

  Shared flags:
    -p key:val / key=expr     Query params
    -d key:val / key=expr     Body fields
    -d **var=                 Splat variable as entire body
    -H key:val                Request header
    -a json|pdf|csv|…         Accept shortcut
    --form                    Send as form-encoded
    --timeout N               Timeout in seconds
    --save                    Auto-save binary response
    -t                        Print execution time
    --doc                     Show OpenAPI docs, do not execute
    --note "text"             Label this request in session log

Session
  /session new    <name> <env>
  /session branch <name> [env]
  /session switch <name>
  /session list
  /session rename <old> <new>
  /session drop   <name>
  /session move   <req-id> <target-session>

Environment
  /env new    <name> <base-url>
  /env set    <name> <key> <value>
  /env unset  <name> <key>
  /env list
  /env show   <name>
  /env copy   <src> <dst>

Import / Export
  /load   <filepath-or-url>
  /export [format] [session] [--out path]
          formats: json | curl | har | http | bruno

Utilities
  /jq     <pattern> [-f]
  /update [var-name]
  /edit   <var-name>
  /clip   [var-name|req-id]
  /save   [filename]
  /replay [req-id|all]
  /watch  [req-id] [interval]
  /list   [filter]
  /info
  /help   [command]

Shell passthrough
  ! <shell command>
```

---

## Appendix B — Key Design Decisions for the Agent

1. **Request IDs are session-scoped and never reused.** When a request is moved out, the gap in IDs is left; the target session appends with its own next ID. This avoids renumbering existing references.

2. **Sessions are immutable logs.** `/replay` and `/watch` always create *new* request entries. The original request is never mutated after execution. This makes the session log an accurate audit trail.

3. **Variable interpolation is lazy.** `{var}` placeholders are resolved at execution time, not at assignment time. This means you can define `base_id = 42` and have multiple queued requests reference `{base_id}` before any of them run.

4. **`/load` is non-destructive.** Loading a file appends to the current session. It never replaces or clears existing requests. Loading an OpenAPI spec only updates `ctx.OpenAPI` — it does not add requests.

5. **Environment is immutable on a session after creation.** To switch environments, branch the session. This prevents accidental cross-environment pollution in the session log.

6. **Export always uses the stored raw bytes.** `Response.RawBody` is what gets exported to HAR and HTTP file formats. `Response.Body` (parsed JSON) is only for display. This ensures export fidelity for binary and non-JSON responses.
