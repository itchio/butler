# OAuth Authorization Code Flow - Desktop App Implementation Guide

This document describes what needs to be implemented in the itch.io desktop Electron app to support OAuth authorization code flow with PKCE.

## Overview

The desktop app is responsible for:
1. Generating PKCE credentials (code verifier and challenge)
2. Opening the browser to the itch.io authorization page
3. Capturing the authorization code (via protocol handler or manual entry)
4. Calling butlerd to exchange the code for an API key

---

## 1. PKCE Credential Generation

Before starting the OAuth flow, generate the PKCE credentials.

### Code Verifier

A cryptographically random string, 43-128 characters, using URL-safe characters.

```javascript
function generateCodeVerifier() {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return base64UrlEncode(array);
}

function base64UrlEncode(buffer) {
  return Buffer.from(buffer)
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
}
```

### Code Challenge

The SHA-256 hash of the code verifier, base64url-encoded.

```javascript
async function generateCodeChallenge(verifier) {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const digest = await crypto.subtle.digest('SHA-256', data);
  return base64UrlEncode(new Uint8Array(digest));
}
```

### Example Output

```
code_verifier:  dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk
code_challenge: E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM
```

---

## 2. Authorization URL Construction

Build the URL to open in the user's browser.

### Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| `response_type` | `code` | Request authorization code flow |
| `client_id` | Your app's client ID | e.g., `itch-desktop` |
| `redirect_uri` | `itch://oauth-callback` | Protocol handler URL (see fallback below) |
| `code_challenge` | Generated challenge | From step 1 |
| `code_challenge_method` | `S256` | Always SHA-256 |
| `scope` | `wharf` | Requested permissions |
| `state` | Random string | CSRF protection, verify on callback |

### Example

```javascript
function buildAuthorizationUrl({ codeChallenge, state, clientId }) {
  const params = new URLSearchParams({
    response_type: 'code',
    client_id: clientId,
    redirect_uri: 'itch://oauth-callback',
    code_challenge: codeChallenge,
    code_challenge_method: 'S256',
    scope: 'wharf',
    state: state,
  });

  return `https://itch.io/user/oauth?${params.toString()}`;
}
```

---

## 3. Protocol Handler Registration

Register `itch://` protocol handler to capture the OAuth callback.

### Electron Main Process

```javascript
const { app } = require('electron');

// Register as default protocol handler
if (process.defaultApp) {
  if (process.argv.length >= 2) {
    app.setAsDefaultProtocolClient('itch', process.execPath, [path.resolve(process.argv[1])]);
  }
} else {
  app.setAsDefaultProtocolClient('itch');
}

// Handle protocol URL on macOS
app.on('open-url', (event, url) => {
  event.preventDefault();
  handleOAuthCallback(url);
});

// Handle protocol URL on Windows/Linux (single instance)
const gotTheLock = app.requestSingleInstanceLock();
if (!gotTheLock) {
  app.quit();
} else {
  app.on('second-instance', (event, commandLine, workingDirectory) => {
    // Windows/Linux: protocol URL is in commandLine
    const url = commandLine.find(arg => arg.startsWith('itch://'));
    if (url) {
      handleOAuthCallback(url);
    }
  });
}
```

### Parsing the Callback URL

```javascript
function handleOAuthCallback(url) {
  // url: itch://oauth-callback?code=ABC123&state=xyz
  const parsed = new URL(url);

  if (parsed.hostname === 'oauth-callback') {
    const code = parsed.searchParams.get('code');
    const state = parsed.searchParams.get('state');
    const error = parsed.searchParams.get('error');

    if (error) {
      const errorDescription = parsed.searchParams.get('error_description');
      handleOAuthError(error, errorDescription);
      return;
    }

    if (code && state) {
      completeOAuthFlow(code, state);
    }
  }
}
```

---

## 4. Manual Code Entry Fallback

The `itch://` protocol may not work in development or certain environments. Provide a fallback UI for manual code entry.

### When to Show Manual Entry

- Protocol handler not registered (dev environment)
- User clicks "Having trouble?" link
- Timeout waiting for protocol callback

### Flow with Manual Entry

1. Show authorization URL to user (or open browser)
2. Display input field for authorization code
3. After user authorizes on itch.io, they see the code on a confirmation page
4. User copies and pastes code into the app
5. App proceeds with code exchange

### UI Implementation

```javascript
class OAuthLoginFlow {
  constructor({ clientId }) {
    this.clientId = clientId;
    this.codeVerifier = null;
    this.state = null;
    this.pendingResolve = null;
    this.pendingReject = null;
  }

  async start() {
    // Generate PKCE credentials
    this.codeVerifier = generateCodeVerifier();
    const codeChallenge = await generateCodeChallenge(this.codeVerifier);
    this.state = generateRandomState();

    // Build and open authorization URL
    const authUrl = buildAuthorizationUrl({
      codeChallenge,
      state: this.state,
      clientId: this.clientId,
    });

    // Open browser
    shell.openExternal(authUrl);

    // Show UI with:
    // - "Waiting for authorization..." message
    // - "Enter code manually" button/link
    // - Manual code input field (initially hidden)
    // - Cancel button

    return new Promise((resolve, reject) => {
      this.pendingResolve = resolve;
      this.pendingReject = reject;

      // Set timeout for showing manual entry option
      setTimeout(() => {
        this.showManualEntryOption();
      }, 10000); // Show after 10 seconds
    });
  }

  // Called when protocol handler receives callback
  handleProtocolCallback(code, state) {
    if (state !== this.state) {
      this.pendingReject(new Error('State mismatch - possible CSRF attack'));
      return;
    }
    this.pendingResolve({ code, codeVerifier: this.codeVerifier });
  }

  // Called when user submits manual code
  handleManualCode(code) {
    // Manual entry doesn't have state verification
    // (user is physically present, CSRF not a concern)
    this.pendingResolve({ code, codeVerifier: this.codeVerifier });
  }

  cancel() {
    this.pendingReject(new Error('User cancelled'));
  }
}
```

### Manual Entry UI Mockup

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│   Sign in to itch.io                                    │
│                                                         │
│   A browser window should have opened.                  │
│   Please sign in and authorize the app.                 │
│                                                         │
│   ┌─────────────────────────────────────────────────┐   │
│   │  Waiting for authorization...                   │   │
│   └─────────────────────────────────────────────────┘   │
│                                                         │
│   ─────────────── Having trouble? ───────────────────   │
│                                                         │
│   Enter the code manually:                              │
│   ┌─────────────────────────────────────────────────┐   │
│   │                                                 │   │
│   └─────────────────────────────────────────────────┘   │
│                                                         │
│              [ Cancel ]        [ Submit Code ]          │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## 5. Exchange Code via butlerd

Once you have the authorization code, call butlerd to exchange it for an API key.

### butlerd Request

**Method:** `Profile.LoginWithOAuthCode`

**Params:**
```json
{
  "code": "SplxlOBeZQQYbYS6WxSbIA",
  "codeVerifier": "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
  "redirectUri": "itch://oauth-callback",
  "clientId": "itch-desktop"
}
```

### butlerd Response

```json
{
  "profile": {
    "id": 12345,
    "lastConnected": "2024-01-15T10:30:00Z",
    "user": {
      "id": 12345,
      "username": "leafo",
      "displayName": "Leaf",
      "coverUrl": "https://..."
    }
  },
  "cookie": {
    "itchio": "session_value",
    "itchio_token": "token_value"
  }
}
```

### Implementation

```javascript
async function exchangeCodeForProfile({ code, codeVerifier, clientId }) {
  const result = await butlerd.call('Profile.LoginWithOAuthCode', {
    code: code,
    codeVerifier: codeVerifier,
    redirectUri: 'itch://oauth-callback',
    clientId: clientId,
  });

  return {
    profile: result.profile,
    cookie: result.cookie,
  };
}
```

---

## 6. Complete Flow Example

```javascript
class AuthService {
  constructor(butlerd, clientId) {
    this.butlerd = butlerd;
    this.clientId = clientId;
    this.currentFlow = null;
  }

  async login() {
    try {
      // Start OAuth flow
      this.currentFlow = new OAuthLoginFlow({ clientId: this.clientId });
      const { code, codeVerifier } = await this.currentFlow.start();

      // Exchange code for profile via butlerd
      const result = await this.butlerd.call('Profile.LoginWithOAuthCode', {
        code,
        codeVerifier,
        redirectUri: 'itch://oauth-callback',
        clientId: this.clientId,
      });

      // Store profile and cookies
      await this.saveProfile(result.profile);
      await this.setCookies(result.cookie);

      return result.profile;

    } catch (error) {
      if (error.message === 'User cancelled') {
        return null;
      }
      throw error;
    } finally {
      this.currentFlow = null;
    }
  }

  // Called by protocol handler
  handleOAuthCallback(url) {
    if (this.currentFlow) {
      const parsed = new URL(url);
      const code = parsed.searchParams.get('code');
      const state = parsed.searchParams.get('state');

      if (code) {
        this.currentFlow.handleProtocolCallback(code, state);
      }
    }
  }

  // Called by manual entry UI
  submitManualCode(code) {
    if (this.currentFlow) {
      this.currentFlow.handleManualCode(code);
    }
  }
}
```

---

## 7. Error Handling

### OAuth Errors (from itch.io redirect)

| Error | Description | User Action |
|-------|-------------|-------------|
| `access_denied` | User denied authorization | Show "Authorization cancelled" |
| `invalid_request` | Malformed request | Show generic error, log details |
| `server_error` | itch.io server error | Show "Try again later" |

### butlerd Errors (from code exchange)

| Error | Description | User Action |
|-------|-------------|-------------|
| `invalid_grant` | Code expired/invalid | Restart OAuth flow |
| Network error | Can't reach API | Check connection, retry |

### Example Error Handling

```javascript
async function login() {
  try {
    const profile = await authService.login();
    if (profile) {
      showSuccess(`Logged in as ${profile.user.displayName}`);
    }
  } catch (error) {
    if (error.code === 'invalid_grant') {
      showError('Authorization expired. Please try again.');
    } else if (error.message.includes('network')) {
      showError('Connection error. Please check your internet connection.');
    } else {
      showError('Login failed. Please try again.');
      console.error('OAuth error:', error);
    }
  }
}
```

---

## 8. Security Considerations

1. **Store code_verifier securely** - Keep in memory only, never persist to disk
2. **Validate state parameter** - Prevents CSRF attacks (skip for manual entry)
3. **Use cryptographically secure random** - `crypto.getRandomValues()` or `crypto.randomBytes()`
4. **Clear sensitive data** - Clear code_verifier after exchange completes or fails
5. **HTTPS only** - Authorization URL must use HTTPS

---

## 9. Testing Checklist

- [ ] Protocol handler works on macOS
- [ ] Protocol handler works on Windows
- [ ] Protocol handler works on Linux
- [ ] Manual code entry works
- [ ] State validation rejects mismatched state
- [ ] Expired code shows appropriate error
- [ ] Cancel button works during flow
- [ ] Multiple rapid login attempts don't conflict
- [ ] Flow works in dev environment (manual entry fallback)
