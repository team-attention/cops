# Implementation Plan: Web Authentication Redirect Flow

## Overview

This plan addresses the web authentication redirect flow issue where unauthenticated users visiting `/auth/device?code=XXX` must be automatically redirected to Google OAuth login and then returned to the device approval page. The implementation will create an authentication context, login page, OAuth callback handler, and route guards to ensure seamless user experience.

## Package Changes

No package changes required. All necessary packages are already installed:
- `@tanstack/react-router` - For routing and navigation
- `@connectrpc/connect-query` - For gRPC API calls
- `@tanstack/react-query` - For state management

## Implementation Steps

### Step 1: Create Authentication Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts`: Understand how tokens are stored and read from localStorage
- `/Users/jayce/team-attention/cops/web/src/feature/auth/hook/use-approve-device.ts`: Example of auth feature hook pattern
- `.agent/rules/react/react-web-src.md`: Shared hook placement rules

#### `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts` (NEW)

**Description**:
Create a shared authentication hook that checks token presence in localStorage and provides login/logout utilities. This hook will be used across the application to check authentication state.

```ts
// Constants for token storage
const ACCESS_TOKEN_KEY = 'cops_access_token';
const REFRESH_TOKEN_KEY = 'cops_refresh_token';
const TOKEN_EXPIRES_AT_KEY = 'cops_token_expires_at';

// useAuth provides authentication state and management functions.
// Returns authentication status and token management utilities.
export const useAuth = () => {
  // Implementation outline:
  // 1. Read access token from localStorage using ACCESS_TOKEN_KEY
  // 2. Determine isAuthenticated by checking if token exists and has length > 0
  // 3. Define logout function:
  //    a. Remove ACCESS_TOKEN_KEY from localStorage
  //    b. Remove REFRESH_TOKEN_KEY from localStorage
  //    c. Remove TOKEN_EXPIRES_AT_KEY from localStorage
  // 4. Define storeTokens function:
  //    a. Accept TokenPair parameter (from auth_pb.ts)
  //    b. Store accessToken in localStorage with ACCESS_TOKEN_KEY
  //    c. Store refreshToken in localStorage with REFRESH_TOKEN_KEY
  //    d. Store expiresAt in localStorage with TOKEN_EXPIRES_AT_KEY
  // 5. Return object with: { isAuthenticated, logout, storeTokens }
}
```

**Test Scenarios**:

| Scenario | Token State | Expected isAuthenticated | Notes |
|:---------|:-----------|:------------------------|:------|
| User has valid token | Token exists in localStorage | `true` | Normal authenticated state |
| User has no token | No token in localStorage | `false` | User needs to login |
| User has empty token | Empty string in localStorage | `false` | Invalid token state |
| After logout called | Token removed from localStorage | `false` | State updates after logout |
| After storeTokens called | Tokens stored in localStorage | `true` | State updates after login |

---

### Step 2: Create Google OAuth Login Page

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx`: Example of auth route pattern and layout
- `/Users/jayce/team-attention/cops/web/src/route/index.tsx`: Example of beforeLoad redirect pattern
- `.agent/rules/react/react-web-src.md`: Route structure and naming conventions

#### `/Users/jayce/team-attention/cops/web/src/route/auth/login.tsx` (NEW)

**Description**:
Create login page route that validates redirect parameter and initiates Google OAuth flow. This page builds the Google OAuth URL and redirects the browser.

```tsx
// Constants
const GOOGLE_OAUTH_CLIENT_ID = import.meta.env.VITE_GOOGLE_OAUTH_CLIENT_ID;
const GOOGLE_OAUTH_REDIRECT_URI = import.meta.env.VITE_GOOGLE_OAUTH_REDIRECT_URI; // e.g., "http://localhost:5173/auth/callback"
const GOOGLE_OAUTH_SCOPES = 'openid email profile';
const GOOGLE_OAUTH_AUTHORIZE_URL = 'https://accounts.google.com/o/oauth2/v2/auth';

// LoginSearchParams defines search params for login route
interface LoginSearchParams {
  redirect?: string;
}

// Route configuration
export const Route = createFileRoute('/auth/login')({
  component: LoginPage,
  validateSearch: (search: Record<string, unknown>): LoginSearchParams => {
    // Implementation outline:
    // 1. Extract redirect parameter from search
    // 2. Validate that redirect is a string or undefined
    // 3. Return typed LoginSearchParams object
  },
});

// LoginPage component redirects to Google OAuth
function LoginPage() {
  const search = useSearch({ from: '/auth/login' });
  const navigate = useNavigate();

  useEffect(() => {
    // Implementation outline:
    // 1. Check if user is already authenticated using useAuth hook
    // 2. If authenticated and redirect param exists:
    //    a. Navigate to redirect URL
    //    b. Return early
    // 3. If authenticated but no redirect:
    //    a. Navigate to '/dashboard'
    //    b. Return early
    // 4. Store redirect URL in sessionStorage with key 'cops_oauth_redirect'
    // 5. Build Google OAuth URL with parameters:
    //    a. response_type=code
    //    b. client_id=GOOGLE_OAUTH_CLIENT_ID
    //    c. redirect_uri=GOOGLE_OAUTH_REDIRECT_URI
    //    d. scope=GOOGLE_OAUTH_SCOPES
    //    e. access_type=offline
    //    f. prompt=consent
    // 6. Redirect browser to Google OAuth URL using window.location.href
  }, [search.redirect, navigate]);

  // Implementation outline for render:
  // 1. Return loading card with:
  //    a. Shield icon from lucide-react
  //    b. "Redirecting to Google..." message
  //    c. Loader2 spinner icon
  // 2. Use same Card/CardHeader/CardContent structure as device.tsx
  // 3. Apply same styling (bg-zinc-950, border-zinc-800, text colors)
}
```

**Test Scenarios**:

| Scenario | Auth State | Redirect Param | Expected Behavior |
|:---------|:-----------|:--------------|:-----------------|
| Not authenticated, no redirect | `false` | `undefined` | Redirect to Google OAuth, no stored redirect |
| Not authenticated, with redirect | `false` | `/auth/device?code=ABC` | Redirect to Google OAuth, store redirect in sessionStorage |
| Already authenticated, no redirect | `true` | `undefined` | Navigate to `/dashboard` immediately |
| Already authenticated, with redirect | `true` | `/auth/device?code=ABC` | Navigate to redirect URL immediately |

---

### Step 3: Create OAuth Callback Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx`: Example of route validateSearch pattern
- `/Users/jayce/team-attention/cops/web/src/feature/auth/hook/use-approve-device.ts`: Example of mutation hook pattern
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/auth/v1/auth_pb.ts`: GoogleAuthReq and GoogleAuthRes types
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/auth/v1/auth-AuthService_connectquery.ts`: googleAuth RPC method

#### `/Users/jayce/team-attention/cops/web/src/feature/auth/hook/use-google-auth.ts` (NEW)

**Description**:
Create mutation hook for Google OAuth token exchange. This hook calls the API to exchange authorization code for JWT tokens.

```ts
// useGoogleAuth provides mutation for exchanging Google auth code for tokens.
// Returns a TanStack Query mutation object.
export const useGoogleAuth = () => {
  // Implementation outline:
  // 1. Import useMutation from '@connectrpc/connect-query'
  // 2. Import googleAuth from '@/gen/grpcstub/auth/v1/auth-AuthService_connectquery'
  // 3. Import transport from '@/shared/service/connect-transport'
  // 4. Return useMutation(googleAuth, { transport })
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Notes |
|:---------|:------|:---------------|:------|
| Valid auth code | `{ authorizationCode: "valid_code", redirectUri: "..." }` | `{ tokens: { accessToken, refreshToken, expiresAt } }` | Successful token exchange |
| Invalid auth code | `{ authorizationCode: "invalid", redirectUri: "..." }` | Error with Code.Unauthenticated | API rejects invalid code |
| Missing redirect URI | `{ authorizationCode: "code", redirectUri: "" }` | Error with Code.InvalidArgument | Validation fails |

---

#### `/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx` (NEW)

**Description**:
Create OAuth callback route that receives Google authorization code, exchanges it for tokens via API, and redirects to the original destination.

```tsx
// CallbackSearchParams defines search params for OAuth callback
interface CallbackSearchParams {
  code?: string;
  error?: string;
}

// Route configuration
export const Route = createFileRoute('/auth/callback')({
  component: CallbackPage,
  validateSearch: (search: Record<string, unknown>): CallbackSearchParams => {
    // Implementation outline:
    // 1. Extract code parameter (Google authorization code)
    // 2. Extract error parameter (OAuth error if any)
    // 3. Validate both are strings or undefined
    // 4. Return typed CallbackSearchParams
  },
});

// CallbackState represents the callback processing state
interface CallbackPending {
  status: 'pending';
}

interface CallbackSuccess {
  status: 'success';
}

interface CallbackError {
  status: 'error';
  message: string;
}

type CallbackState = CallbackPending | CallbackSuccess | CallbackError;

// CallbackPage processes OAuth callback and exchanges code for tokens
function CallbackPage() {
  const search = useSearch({ from: '/auth/callback' });
  const navigate = useNavigate();
  const { storeTokens } = useAuth();
  const mutation = useGoogleAuth();
  const [state, setState] = useState<CallbackState>({ status: 'pending' });

  useEffect(() => {
    // Implementation outline:
    // 1. If search.error exists:
    //    a. setState to error with message from search.error
    //    b. Return early
    // 2. If search.code does not exist:
    //    a. setState to error with message "No authorization code received"
    //    b. Return early
    // 3. Retrieve stored redirect URL from sessionStorage key 'cops_oauth_redirect'
    // 4. Build redirect URI (VITE_GOOGLE_OAUTH_REDIRECT_URI env var)
    // 5. Call mutation.mutateAsync with:
    //    a. authorizationCode: search.code
    //    b. redirectUri: built redirect URI
    // 6. On success:
    //    a. Extract tokens from response
    //    b. Call storeTokens(tokens)
    //    c. Remove 'cops_oauth_redirect' from sessionStorage
    //    d. setState to success
    //    e. Navigate to stored redirect URL or '/dashboard' as fallback
    // 7. On error:
    //    a. setState to error with error.message
    //    b. Keep on current page to show error
  }, [search.code, search.error, mutation, storeTokens, navigate]);

  // Implementation outline for render:
  // 1. If state.status === 'pending':
  //    a. Show loading card with Loader2 spinner
  //    b. Message: "Completing sign in..."
  // 2. If state.status === 'success':
  //    a. Show success card with CheckCircle icon
  //    b. Message: "Signed in successfully! Redirecting..."
  // 3. If state.status === 'error':
  //    a. Show error card with XCircle icon
  //    b. Display state.message
  //    c. Show "Try Again" button that navigates to '/auth/login'
  // 4. Use Card/CardHeader/CardContent from shadcn/ui
  // 5. Apply same styling as device.tsx (bg-zinc-950, borders, colors)
}
```

**Test Scenarios**:

| Scenario | Code Param | Error Param | Expected State | Expected Navigation |
|:---------|:-----------|:-----------|:--------------|:-------------------|
| Successful OAuth | `"valid_code"` | `undefined` | `success` | Redirect to stored URL or `/dashboard` |
| OAuth error | `undefined` | `"access_denied"` | `error` | Stay on callback page, show error |
| No code received | `undefined` | `undefined` | `error` | Stay on callback page, show error |
| API token exchange fails | `"code"` | `undefined` | `error` | Stay on callback page, show API error |

---

### Step 4: Add Authentication Guard to Device Route

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx`: Current device route implementation
- `/Users/jayce/team-attention/cops/web/src/route/index.tsx`: Example of beforeLoad redirect

#### `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx` (MODIFY)

**Description**:
Add beforeLoad guard to check authentication state. If user is not authenticated, redirect to login page with return URL preserved.

**Changes**:

1. Add beforeLoad function to Route configuration:

```tsx
export const Route = createFileRoute('/auth/device')({
  beforeLoad: ({ location, search }) => {
    // Implementation outline:
    // 1. Read access token from localStorage using key 'cops_access_token'
    // 2. If token does not exist or has length === 0:
    //    a. Build redirect URL: location.pathname + location.search (full URL with code param)
    //    b. Throw redirect to '/auth/login' with search param: { redirect: redirectUrl }
    // 3. If token exists, continue to component (no return/throw needed)
  },
  component: DeviceApprovalPage,
  validateSearch: (search: Record<string, unknown>): DeviceSearchParams => {
    return {
      code: typeof search.code === 'string' ? search.code : undefined,
    };
  },
});
```

2. No changes needed to DeviceApprovalPage component or DeviceApproval component.

**Test Scenarios**:

| Scenario | Token State | Code Param | Expected Behavior |
|:---------|:-----------|:-----------|:-----------------|
| Not authenticated | No token | `"ABC123"` | Redirect to `/auth/login?redirect=/auth/device?code=ABC123` |
| Authenticated | Valid token | `"ABC123"` | Show device approval page |
| Not authenticated | No token | `undefined` | Redirect to `/auth/login?redirect=/auth/device` |
| Authenticated | Valid token | `undefined` | Show "No Device Code" error (existing behavior) |

---

### Step 5: Update Device Approval Error Handling

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/feature/auth/component/device-approval.tsx`: Current error handling

#### `/Users/jayce/team-attention/cops/web/src/feature/auth/component/device-approval.tsx` (MODIFY)

**Description**:
Remove UNAUTHORIZED error case from error messages since route guard prevents unauthenticated access. The UNAUTHORIZED error should theoretically not occur anymore because beforeLoad ensures authentication.

**Changes**:

1. Remove UNAUTHORIZED from DeviceApprovalErrorCode type:

```ts
type DeviceApprovalErrorCode =
  | 'NOT_FOUND'
  | 'EXPIRED'
  | 'ALREADY_APPROVED'
  | 'UNKNOWN';
```

2. Remove UNAUTHORIZED case from error handling in handleApprove:

```ts
const handleApprove = async () => {
  try {
    await mutation.mutateAsync({ userCode });
    setState({
      status: 'success',
      message: 'Device approved successfully!',
    });
  } catch (error) {
    const connectError = error as { code?: Code; message: string };
    const errorCode = connectError.code;
    let mappedCode: DeviceApprovalErrorCode = 'UNKNOWN';

    if (errorCode === Code.NotFound) {
      mappedCode = 'NOT_FOUND';
    } else if (errorCode === Code.DeadlineExceeded) {
      mappedCode = 'EXPIRED';
    } else if (errorCode === Code.AlreadyExists) {
      mappedCode = 'ALREADY_APPROVED';
    }
    // Note: UNAUTHORIZED case removed - handled by route guard

    setState({
      status: 'error',
      errorCode: mappedCode,
      message: connectError.message || 'An error occurred',
    });
  }
};
```

3. Remove UNAUTHORIZED case from errorMessages:

```ts
const errorMessages: Record<typeof state.errorCode, string> = {
  NOT_FOUND: 'Device code not found. It may have expired.',
  EXPIRED: 'This device code has expired. Please generate a new one.',
  ALREADY_APPROVED: 'This device code has already been approved.',
  UNKNOWN: state.message,
};
```

**Test Scenarios**:

| Scenario | API Error Code | Expected Error State | Expected Message |
|:---------|:--------------|:-------------------|:-----------------|
| Code not found | `Code.NotFound` | `NOT_FOUND` | "Device code not found. It may have expired." |
| Code expired | `Code.DeadlineExceeded` | `EXPIRED` | "This device code has expired. Please generate a new one." |
| Code already approved | `Code.AlreadyExists` | `ALREADY_APPROVED` | "This device code has already been approved." |
| Unknown error | Any other code | `UNKNOWN` | Error message from API |

---

### Step 6: Update Device Approval Type Definitions

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/feature/auth/type/device-code.ts`: Current type definitions

#### `/Users/jayce/team-attention/cops/web/src/feature/auth/type/device-code.ts` (MODIFY)

**Description**:
Remove UNAUTHORIZED from DeviceApprovalErrorCode type since it's now handled at route level.

**Changes**:

```ts
// DeviceApprovalErrorCode enumerates possible error conditions.
type DeviceApprovalErrorCode =
  | 'NOT_FOUND'
  | 'EXPIRED'
  | 'ALREADY_APPROVED'
  | 'UNKNOWN';
```

No other changes needed to this file.

---

## Environment Variables Required

The following environment variables must be configured in the web application:

```bash
# API endpoint (already exists)
VITE_API_URL=http://localhost:8080

# Google OAuth configuration (new)
VITE_GOOGLE_OAUTH_CLIENT_ID=your-client-id.apps.googleusercontent.com
VITE_GOOGLE_OAUTH_REDIRECT_URI=http://localhost:5173/auth/callback
```

For production deployment, update these values accordingly.

---

## Complete User Flow

1. **User runs CLI**: `cops auth login`
2. **CLI displays URL**: User sees `/auth/device?code=ABC123`
3. **User clicks URL**: Browser navigates to device approval page
4. **Route guard checks auth**: beforeLoad detects no token
5. **Redirect to login**: Navigate to `/auth/login?redirect=/auth/device?code=ABC123`
6. **Login page stores redirect**: Save redirect URL to sessionStorage
7. **Redirect to Google**: Browser navigates to Google OAuth consent screen
8. **User approves**: Google redirects to `/auth/callback?code=GOOGLE_AUTH_CODE`
9. **Callback exchanges code**: Call API googleAuth RPC with authorization code
10. **Store tokens**: Save access_token and refresh_token to localStorage
11. **Redirect to device page**: Navigate back to `/auth/device?code=ABC123`
12. **Route guard passes**: Token exists, load device approval page
13. **User approves device**: Click "Approve Device" button
14. **Success**: CLI receives tokens and completes login

---

## File Checklist

### New Files
- [ ] `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`
- [ ] `/Users/jayce/team-attention/cops/web/src/route/auth/login.tsx`
- [ ] `/Users/jayce/team-attention/cops/web/src/feature/auth/hook/use-google-auth.ts`
- [ ] `/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx`

### Modified Files
- [ ] `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx` - Add beforeLoad guard
- [ ] `/Users/jayce/team-attention/cops/web/src/feature/auth/component/device-approval.tsx` - Remove UNAUTHORIZED handling
- [ ] `/Users/jayce/team-attention/cops/web/src/feature/auth/type/device-code.ts` - Remove UNAUTHORIZED from type

### Configuration Files
- [ ] Create `.env.local` or document required environment variables

---

## Implementation Notes

1. **Token Storage**: All tokens use localStorage with consistent key naming (`cops_*` prefix)
2. **Redirect Preservation**: OAuth redirect URL stored in sessionStorage (temporary, cleared after use)
3. **Error Handling**: All error states use discriminated unions for type safety
4. **UI Consistency**: All auth pages use same Card/Alert components and color scheme
5. **Route Guards**: beforeLoad is synchronous, reads directly from localStorage
6. **Hook Pattern**: All gRPC calls wrapped in dedicated hooks following existing pattern
7. **Import Paths**: Use `@/` alias for all absolute imports as configured in tsconfig.json
8. **Component Exports**: All components use named exports (not default exports) per React rules
9. **File Naming**: All files use kebab-case (lowercase with hyphens)
10. **Directory Structure**: Features in `feature/auth/`, shared utilities in `shared/hook/`
