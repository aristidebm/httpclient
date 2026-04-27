# Login Feature Implementation Tasks

## Overview
Implement `/login` command with support for:
- Basic authentication
- Token-based authentication
- OAuth authentication (Device Flow default, manual override)

All auth config is stored in the environment and persisted to disk.

---

## Phase 1: Model Changes

- [x] **Task 1.1**: Add `AuthConfig` struct to `internal/model/environment.go`
  - Fields: `Type` (string), `Username` (string), `Password` (string) for basic
  - Fields: `Token`, `TokenType`, `HeaderName` for token auth
  - Fields: `ClientID`, `ClientSecret`, `TokenURL`, `AccessToken`, `RefreshToken`, `ExpiresAt` for OAuth

- [x] **Task 1.2**: Add `Auth *AuthConfig` field to `Environment` struct

- [x] **Task 1.3**: Update `Environment.Clone()` to include Auth field

---

## Phase 2: Auth Execution Logic

- [x] **Task 2.1**: Rewrite `internal/executor/auth.go` to handle all auth types
  - Basic: Generate `Authorization: Basic base64(user:pass)`
  - Token: Set header (default: `Authorization: Bearer <token>`, custom header support)
  - OAuth: Use access token (assume already obtained)

- [x] **Task 2.2**: Update `ApplyAuth()` function signature if needed to handle new auth types

---

## Phase 3: Login Command - Basic Auth

- [x] **Task 3.1**: Create `internal/commands/login.go` with command structure
  - Register with `repl.Register(&loginCmd{})` in `init()`

- [x] **Task 3.2**: Implement `/login basic <username> [password]`
  - If password not provided, prompt for it (using readline)
  - Store credentials in `Environment.Auth`
  - Show confirmation message

- [x] **Task 3.3**: Implement `/login clear`
  - Remove auth config from current environment

- [x] **Task 3.4**: Implement `/login show`
  - Display current auth type and config (redact passwords/tokens)

---

## Phase 4: Login Command - Token Auth

- [ ] **Task 4.1**: Implement `/login token <token> [--header=X]`
  - Default header: `Authorization` with `Bearer` prefix
  - Custom header support via `--header=` flag
  - TokenType option (Bearer, Token, custom)
  - Store in `Environment.Auth`

---

## Phase 5: Login Command - OAuth

- [ ] **Task 5.1**: Implement OAuth config storage
  - Add subcommand: `/login oauth config <client-id> <token-url> [client-secret]`
  - Store in `Environment.Auth`

- [ ] **Task 5.2**: Implement Device Flow (default)
  - `/login oauth --flow=device`
  - Make request to device authorization endpoint
  - Display code and URL to user
  - Poll for token
  - Store access/refresh tokens

- [ ] **Task 5.3**: Implement Manual Flow
  - `/login oauth --flow=manual <access-token>`
  - Store provided token

---

## Phase 6: Integration & Testing

- [ ] **Task 6.1**: Test basic auth end-to-end
  - Set credentials via `/login basic`
  - Execute request and verify Authorization header

- [ ] **Task 6.2**: Test token auth end-to-end
  - Set token via `/login token`
  - Execute request and verify header

- [ ] **Task 6.3**: Test OAuth flow (if possible with test provider)

- [ ] **Task 6.4**: Verify auth persists across sessions (save/load)

---

## Progress Tracking

Current Phase: **Phase 4 - Login Command (Token Auth)**

Last completed: Phase 1, 2, 3 (All basic auth tasks done)

Next task: **Task 4.1** - Implement /login token
