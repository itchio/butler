# OAuth Authorization Code Flow - API Implementation Guide

This document describes the itch.io API changes needed to support OAuth authorization code flow with PKCE for the desktop app.

## Overview

The desktop app (Electron) will authenticate users via OAuth using the authorization code flow with PKCE (Proof Key for Code Exchange). This is more secure than the implicit flow currently used by the CLI.

### Flow Diagram

```
Desktop App                     itch.io                          butlerd
     |                            |                                  |
     |  1. Generate code_verifier (random string)                    |
     |  2. Compute code_challenge = base64url(sha256(code_verifier)) |
     |                            |                                  |
     |  3. Open browser --------->|                                  |
     |     /user/oauth?           |                                  |
     |       response_type=code   |                                  |
     |       code_challenge=...   |                                  |
     |       redirect_uri=...     |                                  |
     |                            |                                  |
     |                     4. User logs in                           |
     |                            |                                  |
     |  5. Redirect <-------------|                                  |
     |     redirect_uri?code=ABC  |                                  |
     |                            |                                  |
     |  6. Profile.LoginWithOAuthCode(code, code_verifier) --------->|
     |                            |                                  |
     |                            |<------ 7. POST /oauth/token -----|
     |                            |           code=ABC               |
     |                            |           code_verifier=...      |
     |                            |                                  |
     |                            |------- 8. { key, cookie } ------>|
     |                            |                                  |
     |<-------------------------- 9. Profile + Cookie ---------------|
```

---

## 1. Authorization Endpoint Changes

### Endpoint: `GET /user/oauth`

The existing OAuth authorization endpoint needs to support new parameters for the authorization code flow.

### New/Modified Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `response_type` | string | Yes | `"code"` for authorization code flow (existing `"token"` for implicit flow) |
| `client_id` | string | Yes | `"butler"` |
| `redirect_uri` | string | Yes | Where to redirect after authorization (e.g., `itch://oauth-callback` or `http://127.0.0.1:PORT/callback`) |
| `code_challenge` | string | Yes* | Base64url-encoded SHA256 hash of the code verifier. *Required when `response_type=code` |
| `code_challenge_method` | string | Yes* | Must be `"S256"`. *Required when `response_type=code` |
| `scope` | string | No | Requested scope (e.g., `"wharf"`) |
| `state` | string | No | Opaque value for CSRF protection, returned unchanged in redirect |

### Behavior for `response_type=code`

1. Validate all required parameters are present
2. Validate `code_challenge_method` is `"S256"`
3. Display login/authorization UI to user
4. On user approval:
   - Generate a random authorization code (e.g., 32+ bytes, base64url-encoded)
   - Store the code with associated data:
     - `user_id`
     - `code_challenge`
     - `redirect_uri`
     - `client_id`
     - `scope`
     - `created_at` (for expiration)
   - Redirect to: `{redirect_uri}?code={authorization_code}&state={state}`

### Example Authorization URL

```
https://itch.io/user/oauth?
  response_type=code&
  client_id=butler&
  redirect_uri=itch://oauth-callback&
  code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&
  code_challenge_method=S256&
  state=xyz123
```

### Example Redirect (Success)

```
itch://oauth-callback?code=SplxlOBeZQQYbYS6WxSbIA&state=xyz123
```

### Example Redirect (Error)

```
itch://oauth-callback?error=access_denied&error_description=User%20denied%20access&state=xyz123
```

---

## 2. Token Endpoint (New)

### Endpoint: `POST /oauth/token`

Exchanges an authorization code for an API key.

### Request

**Content-Type:** `application/x-www-form-urlencoded`

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `grant_type` | string | Yes | Must be `"authorization_code"` |
| `code` | string | Yes | The authorization code received from the redirect |
| `code_verifier` | string | Yes | The original random string (before hashing) |
| `redirect_uri` | string | Yes | Must exactly match the `redirect_uri` used in the authorization request |
| `client_id` | string | Yes | Must match the `client_id` used in the authorization request |

### Example Request

```http
POST /oauth/token HTTP/1.1
Host: api.itch.io
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&
code=SplxlOBeZQQYbYS6WxSbIA&
code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk&
redirect_uri=itch://oauth-callback&
client_id=butler
```

### Response (Success)

**Status:** `200 OK`

**Content-Type:** `application/json`

```json
{
  "key": {
    "key": "abc123def456...",
    "id": 12345,
    "user_id": 67890,
    "created_at": "2024-01-15T10:30:00Z"
  },
  "cookie": {
    "itchio": "session_cookie_value",
    "itchio_token": "token_cookie_value"
  }
}
```

### Response (Error)

**Status:** `400 Bad Request` or `401 Unauthorized`

**Content-Type:** `application/json`

```json
{
  "error": "invalid_grant",
  "error_description": "Authorization code has expired"
}
```

### Standard OAuth Error Codes

| Error | Description |
|-------|-------------|
| `invalid_request` | Missing required parameter or malformed request |
| `invalid_client` | Unknown `client_id` |
| `invalid_grant` | Code is invalid, expired, or already used; or PKCE verification failed |
| `unauthorized_client` | Client not authorized for this grant type |

---

## 3. Server-Side Implementation Details

### PKCE Verification

The server must verify that the `code_verifier` matches the `code_challenge` that was stored during authorization:

```python
import hashlib
import base64

def verify_pkce(code_verifier: str, stored_code_challenge: str) -> bool:
    # Compute SHA256 hash of the verifier
    digest = hashlib.sha256(code_verifier.encode('ascii')).digest()

    # Base64url encode (no padding)
    computed_challenge = base64.urlsafe_b64encode(digest).rstrip(b'=').decode('ascii')

    # Constant-time comparison
    return hmac.compare_digest(computed_challenge, stored_code_challenge)
```

### Authorization Code Storage Schema

```sql
CREATE TABLE oauth_authorization_codes (
    code VARCHAR(64) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    client_id VARCHAR(64) NOT NULL,
    redirect_uri TEXT NOT NULL,
    code_challenge VARCHAR(128) NOT NULL,
    scope VARCHAR(256),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    used_at TIMESTAMP NULL,

    INDEX idx_created_at (created_at)
);
```

### Token Endpoint Validation Steps

1. **Validate `grant_type`**
   - Must be `"authorization_code"`
   - Return `invalid_request` if missing or wrong

2. **Look up authorization code**
   - Find code in database
   - Return `invalid_grant` if not found

3. **Check code not already used**
   - If `used_at` is set, return `invalid_grant`
   - Mark code as used immediately (before other checks) to prevent race conditions

4. **Check code not expired**
   - Codes should expire after 10 minutes (OAuth 2.0 recommendation)
   - Return `invalid_grant` if expired

5. **Verify `client_id` matches**
   - Return `invalid_grant` if mismatch

6. **Verify `redirect_uri` matches exactly**
   - Return `invalid_grant` if mismatch

7. **Verify PKCE**
   - Compute `base64url(sha256(code_verifier))`
   - Compare with stored `code_challenge`
   - Return `invalid_grant` if mismatch

8. **Generate API key**
   - Create new API key for the user
   - Set appropriate scope/permissions

9. **Generate session cookies** (optional)
   - Create session cookies if you want to sync website login state

10. **Return response**
    - Return the API key and cookies as JSON

### Security Considerations

- **Single-use codes**: Authorization codes must be invalidated after first use
- **Short expiration**: Codes should expire within 10 minutes
- **Exact redirect_uri matching**: Don't allow partial matches or wildcards
- **PKCE is mandatory**: Always require `code_challenge` for authorization code flow
- **Constant-time comparison**: Use constant-time string comparison for codes and challenges
- **Rate limiting**: Rate limit the token endpoint to prevent brute force attacks

---

## 4. Testing

### Test Case 1: Successful Flow

1. Generate `code_verifier`: `dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk`
2. Compute `code_challenge`: `E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM`
3. Authorize and get code
4. Exchange code with correct verifier
5. Expect: 200 with API key

### Test Case 2: Wrong Verifier

1. Authorize with challenge A
2. Exchange with verifier for challenge B
3. Expect: 400 `invalid_grant`

### Test Case 3: Expired Code

1. Authorize and get code
2. Wait > 10 minutes
3. Exchange code
4. Expect: 400 `invalid_grant`

### Test Case 4: Reused Code

1. Authorize and get code
2. Exchange code successfully
3. Exchange same code again
4. Expect: 400 `invalid_grant`

### Test Case 5: Wrong Redirect URI

1. Authorize with `redirect_uri=A`
2. Exchange with `redirect_uri=B`
3. Expect: 400 `invalid_grant`

---

## 5. Reference: PKCE Code Generation (Client-Side)

For reference, here's how the desktop app generates PKCE values:

```javascript
// Generate code_verifier (43-128 characters, URL-safe)
function generateCodeVerifier() {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return base64UrlEncode(array);
}

// Generate code_challenge from verifier
async function generateCodeChallenge(verifier) {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const digest = await crypto.subtle.digest('SHA-256', data);
  return base64UrlEncode(new Uint8Array(digest));
}

// Base64url encoding (no padding)
function base64UrlEncode(buffer) {
  return btoa(String.fromCharCode(...buffer))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
}
```

---

## 6. butlerd Integration

The butlerd endpoint `Profile.LoginWithOAuthCode` will call `POST /oauth/token` with:

```go
// From go-itchio/endpoints_login.go
func (c *Client) ExchangeOAuthCode(ctx context.Context, params ExchangeOAuthCodeParams) (*ExchangeOAuthCodeResponse, error) {
    q := NewQuery(c, "/oauth/token")
    q.AddString("grant_type", "authorization_code")
    q.AddString("code", params.Code)
    q.AddString("code_verifier", params.CodeVerifier)
    q.AddString("redirect_uri", params.RedirectURI)
    q.AddString("client_id", params.ClientID)

    r := &ExchangeOAuthCodeResponse{}
    return r, q.Post(ctx, r)
}
```

Expected response structure:

```go
type ExchangeOAuthCodeParams struct {
    Code         string
    CodeVerifier string
    RedirectURI  string
    ClientID     string
}

type ExchangeOAuthCodeResponse struct {
    Key    *APIKey `json:"key"`
    Cookie Cookie  `json:"cookie"`
}

type APIKey struct {
    Key    string `json:"key"`
    ID     int64  `json:"id"`
    UserID int64  `json:"user_id,omitempty"`
}

type Cookie map[string]string
```
