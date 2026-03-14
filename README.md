# HTTP Client

An HTTP client with an interactive REPL shell and one-shot mode for making HTTP requests.

![Demo](demo.gif)

## Features

- Interactive REPL shell for exploring APIs
- Support for all HTTP methods: GET, POST, PUT, DELETE, PATCH
- Import OpenAPI specifications, .http files, and curl commands
- Session-based request history management
- Environment and variable management
- Export to various formats (JSON, HAR, Bruno, cURL)

## Installation

```bash
go install ./...
```

## Usage

### Interactive Mode

Launch the shell without arguments:

```bash
httpclient
```

Load a specification file or URL to import:

```bash
httpclient path/to/spec.json
httpclient https://petstore.swagger.io/v2/swagger.json
```

### Command-Line Options

- `--session <name>` - Use or create a named session
- `-` - Read specification from stdin

### Shell Commands

| Command | Description |
|---------|-------------|
| `/get <endpoint>` | Execute GET request |
| `/post <endpoint>` | Execute POST request |
| `/put <endpoint>` | Execute PUT request |
| `/delete <endpoint>` | Execute DELETE request |
| `/patch <endpoint>` | Execute PATCH request |
| `/import <path>` | Import OpenAPI spec, .http file, or curl commands |
| `/export [format]` | Export session to various formats |
| `/logs` | List request history |
| `/logs show <id>` | Show request details |
| `/logs clear` | Clear request history |
| `/replay <id>` | Replay a previous request |
| `/vars` | List variables |
| `/vars set <key> <value>` | Set a variable |
| `/vars unset <key>` | Unset a variable |
| `/env` | Manage environments |
| `/session` | Manage sessions |
| `/help` | Show help |
| `/exit` | Exit and save |

### Shell Shortcuts

- `!command` - Execute shell command with variable substitution
- `{last}` - Path to last response body (temp file)
- `{data}` - Path to last response as JSON (temp file)
- `var = value` - Assign to shell variable
- `var = d` - Assign last response data to variable

## Request Options

All HTTP commands support flags:

- `-d, --data <body>` - Request body
- `-H, --header <header>` - Add header
- `-p, --param <key=value>` - Query parameter

## Examples

```bash
# Make a GET request
/get /api/users

# Make a POST request with body
/post /api/users -d '{"name":"John"}'

# Import an OpenAPI spec
httpclient https://petstore.swagger.io/v2/swagger.json

# Use a specific session
httpclient --session myapi

# Import from stdin
cat spec.json | httpclient -
```

## Project Structure

```
.
├── cmd/               # CLI commands
├── internal/
│   ├── commands/      # REPL commands
│   ├── executor/     # HTTP execution
│   ├── input/        # Import parsers
│   ├── model/        # Data models
│   ├── repl/         # Shell implementation
│   └── session/      # Session management
└── main.go           # Entry point
```

## License

MIT
