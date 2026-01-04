# Requirements: Fix Google Login Authentication Bug

## Request Summary

Users attempting to sign in via the /auth page using Google OAuth encounter an authentication error after completing Google login. The error message "missing authorization header" appears, and the browser console shows an infinite loop of POST requests to the GoogleAuth endpoint, all returning 401 Unauthorized. This prevents users from successfully logging into the application.

## Root Cause Analysis

### Issue Identified

The API server has a **global authentication interceptor** applied to ALL ConnectRPC endpoints, including the authentication endpoints themselves. This creates a chicken-and-egg problem:

1. **Location**: `/api/cmd/internal/container/register_connectrpc.go` (lines 45-49)
2. **Problem**: The auth interceptor is applied globally to all handlers:
   ```go
   authInterceptor := interceptor.NewAuthInterceptor(logger, jwtCfg)
   opts := []connect.HandlerOption{connect.WithInterceptors(authInterceptor)}

   for _, handler := range params.ConnectHandlers {
       path, h := handler.GetHandler(opts...)  // Auth interceptor applied to ALL handlers
   ```

3. **Impact**: The `/auth.v1.AuthService/GoogleAuth` endpoint requires an Authorization header to pass the interceptor, but users calling this endpoint to LOGIN cannot have a token yet.

### Authentication Flow Analysis

**Expected Flow:**
1. User completes Google OAuth and is redirected to `/auth/callback?code=...`
2. Frontend calls `GoogleAuth` RPC with authorization code
3. Backend exchanges code with Google, creates/finds user, generates JWT tokens
4. Frontend stores tokens and redirects to dashboard

**Actual Flow (Broken):**
1. User completes Google OAuth and is redirected to `/auth/callback?code=...`
2. Frontend calls `GoogleAuth` RPC with authorization code
3. **Backend auth interceptor blocks request** with "missing authorization header" (401)
4. **Frontend auth interceptor detects 401** and attempts token refresh
5. Frontend checks if URL contains `/auth.v1.AuthService/` and skips refresh (line 136 in `connect-transport.ts`)
6. Error propagates to callback page, shown to user as error

**Why No Infinite Loop Occurs (Corrected Analysis):**

The infinite loop does NOT occur because:
- Frontend has a guard in `connect-transport.ts` (line 136): `if (req.url.includes(AUTH_SERVICE_PREFIX)) { throw error }`
- This prevents the refresh attempt for auth service endpoints
- However, the error still prevents successful login

### Backend Interceptor Behavior

**File**: `/api/internal/platform/interceptor/auth_interceptor.go`

The interceptor requires an Authorization header for ALL requests:
- Line 45-46: Returns 401 if `authHeader == ""`
- Line 49-51: Returns 401 if header doesn't start with "Bearer "
- Line 55-60: Validates JWT token

**No exemptions** for specific endpoints like GoogleAuth, RefreshToken, DeviceCode, or DevicePoll.

### Frontend Transport Behavior

**File**: `/web/src/shared/service/connect-transport.ts`

The frontend has proper guards to prevent infinite refresh loops:
- Line 136-138: Skips token refresh for auth service endpoints
- Line 142-145: Skips token refresh if no refresh token available
- These guards prevent infinite loops but don't solve the 401 error

## Acceptance Criteria

- [ ] Users can successfully log in via Google OAuth without receiving "missing authorization header" error
- [ ] GoogleAuth endpoint is accessible without requiring an Authorization header
- [ ] RefreshToken endpoint is accessible without requiring an Authorization header (but validates the refresh token in request body)
- [ ] DeviceCode and DevicePoll endpoints are accessible without requiring an Authorization header
- [ ] DeviceCodeApprove endpoint REQUIRES Authorization header (already has inline validation - this should remain protected)
- [ ] All other endpoints (dashboard, project, session, etc.) continue to require valid JWT tokens
- [ ] No infinite loop occurs during authentication or token refresh flows
- [ ] Existing authenticated users can continue using the application without interruption

## Scope

### In Scope
- Modify backend auth interceptor to exempt specific auth endpoints from JWT validation
- Ensure GoogleAuth, RefreshToken, DeviceCode, and DevicePoll endpoints are publicly accessible
- Verify DeviceCodeApprove endpoint remains protected (it already validates JWT inline at line 144-148 in handler)
- Test complete Google OAuth login flow from /auth → callback → dashboard

### Out of Scope
- Changes to frontend authentication logic (already correctly implemented)
- Changes to Google OAuth configuration
- Changes to JWT token generation or validation logic
- Changes to token refresh mechanism
- UI/UX improvements to error messages

## Constraints

- Must maintain backward compatibility with existing authenticated sessions
- Cannot weaken security for protected endpoints
- Must follow existing hexagonal architecture patterns
- Auth interceptor should remain a single, reusable component

## Additional Context

### Related Files

**Backend:**
- `/api/internal/platform/interceptor/auth_interceptor.go` - Auth interceptor implementation
- `/api/cmd/internal/container/register_connectrpc.go` - Interceptor registration
- `/api/internal/service/auth/inbound/grpc/connectrpc/handler.go` - Auth handler with methods

**Frontend:**
- `/web/src/shared/service/connect-transport.ts` - Frontend transport with auth interceptor
- `/web/src/route/auth/callback.tsx` - OAuth callback handler
- `/web/src/shared/store/auth-store.ts` - Token storage

### Endpoints That Need Exemption

Based on auth service analysis:
1. **GoogleAuth** (POST /auth.v1.AuthService/GoogleAuth) - Exchanges Google auth code for JWT tokens
2. **RefreshToken** (POST /auth.v1.AuthService/RefreshToken) - Exchanges refresh token for new token pair
3. **DeviceCode** (POST /auth.v1.AuthService/DeviceCode) - Initiates device flow (returns user code)
4. **DevicePoll** (POST /auth.v1.AuthService/DevicePoll) - Polls device code status

### Endpoint That Must Remain Protected

1. **DeviceCodeApprove** (POST /auth.v1.AuthService/DeviceCodeApprove) - Approves device code (requires authenticated user)
   - Already validates JWT inline at handler level (lines 144-148)
   - Should remain protected by global interceptor as defense-in-depth

## Questions Resolved

| Question                                                                                      | Answer                                                                                                                                                                                        |
| --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| What is the root cause of the authentication error?                                          | Global auth interceptor requires Authorization header for ALL endpoints, including authentication endpoints like GoogleAuth that need to be publicly accessible.                             |
| Why does the infinite loop occur in the console?                                             | Corrected: No infinite loop occurs. Frontend has guards to prevent refresh on auth endpoints. However, the 401 error still blocks successful login.                                          |
| Which endpoints should be exempt from JWT validation?                                        | GoogleAuth, RefreshToken, DeviceCode, and DevicePoll should be publicly accessible. DeviceCodeApprove must remain protected.                                                                 |
| Should we modify the frontend or backend?                                                    | Backend only. The frontend is correctly implemented with proper guards. The issue is the backend interceptor blocking public auth endpoints.                                                 |
| How should we implement the exemption?                                                       | Modify auth interceptor to check request procedure/method name and skip validation for specific auth endpoints while maintaining protection for all other endpoints.                         |
| Will this affect existing authenticated users?                                               | No. Existing users with valid tokens will continue to work normally. The change only affects public auth endpoints.                                                                          |
| Does DeviceCodeApprove need special handling?                                                | DeviceCodeApprove should remain protected by the global interceptor. It already has inline JWT validation (defense-in-depth), and it requires an authenticated user to approve device codes. |
| Are there any security concerns with exempting these endpoints?                              | No. These endpoints are designed to be publicly accessible for authentication flows. GoogleAuth validates with Google, RefreshToken validates refresh token, DeviceCode/DevicePoll are safe. |
| Should we add any logging or monitoring for the exempt endpoints?                            | Yes. The interceptor should log when requests are exempted to aid in debugging and security monitoring.                                                                                      |
| What testing is required after the fix?                                                      | Test complete Google OAuth flow, token refresh flow, device code flow, and verify protected endpoints still require authentication.                                                          |
| Should the frontend guards in connect-transport.ts be modified or removed after this fix?    | No. Keep the frontend guards as they provide client-side protection against unnecessary refresh attempts. Defense-in-depth is good practice.                                                 |
| Is there a standardized pattern for endpoint exemption in ConnectRPC interceptors?           | Yes. Check request procedure name (e.g., `/auth.v1.AuthService/GoogleAuth`) and skip validation for specific procedures.                                                                     |
| Should we use a whitelist or pattern-matching approach for exempting endpoints?              | Use an explicit whitelist of procedure names for security and clarity. Avoid pattern matching to prevent accidental exemptions.                                                              |
| What should happen if an exempted endpoint receives an Authorization header?                  | Accept it but don't validate it. The endpoint implementation should decide if it needs the token (e.g., RefreshToken uses the token in request body, not header).                            |
| How do we ensure this fix doesn't introduce regression in other authentication flows?        | Test all authentication flows: Google OAuth, token refresh, device code flow, and verify all protected endpoints still require valid tokens.                                                 |
