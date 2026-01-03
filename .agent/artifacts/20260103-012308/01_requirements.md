# Requirements: CLI Authentication via C-Ops Web Device Flow

## Request Summary

The current CLI authentication flow sends users directly to Google OAuth device flow, bypassing the C-Ops Web application. This needs to be changed so that CLI authentication goes through the C-Ops Web interface, allowing users to authenticate using their existing web session. The new flow will be: CLI initiates device code → API generates device code → User visits C-Ops Web to approve → Web validates using existing Google login session → CLI receives JWT tokens.

## Acceptance Criteria

### Device Code Domain Model
- [ ] DeviceCode domain model defined with fields (code, userCode, userID, expiresAt, approved, createdAt)
- [ ] MongoDB collection `device_codes` created with TTL index on expiresAt (15 minutes)
- [ ] Device code generation creates unique 6-8 character user-friendly code (e.g., "ABCD-EFGH")
- [ ] Device code has secure random identifier for CLI polling

### API Changes
- [ ] Remove direct Google OAuth device flow from API auth service
- [ ] API DeviceCode endpoint generates and stores device code in MongoDB instead of calling Google
- [ ] API DeviceCodePoll endpoint checks MongoDB approval status instead of polling Google
- [ ] New API endpoint DeviceCodeApprove for web to approve device codes
- [ ] DeviceCodeApprove validates user is authenticated (JWT from web session)
- [ ] DeviceCodeApprove links device code to authenticated user and marks as approved
- [ ] After approval, DeviceCodePoll returns JWT tokens for the approved user
- [ ] Device codes automatically expire after 15 minutes via MongoDB TTL index

### Web Device Approval Page
- [ ] New web route `/auth/device?code=USER_CODE` for device approval
- [ ] If user not logged in, redirect to Google login with return URL to device page
- [ ] After Google login, automatically redirect back to device approval page with code parameter
- [ ] If user already logged in, show simple device approval UI with user code display
- [ ] Approval UI shows: "Approve CLI Access for device code: {USER_CODE}"
- [ ] Approve button calls API DeviceCodeApprove endpoint with user's JWT token
- [ ] Success message shown after approval: "Device approved! You can return to your terminal."
- [ ] Error handling for expired, invalid, or already-approved codes

### CLI Flow Updates
- [ ] CLI InitiateLogin calls API DeviceCode endpoint (no changes to CLI logic needed)
- [ ] CLI displays C-Ops Web URL instead of Google URL: `https://cops.example.com/auth/device?code={USER_CODE}`
- [ ] CLI continues polling API DeviceCodePoll endpoint (no changes to polling logic)
- [ ] CLI receives and stores JWT tokens after approval (no changes to token storage)

### Security & Validation
- [ ] Device codes are single-use (cannot be approved multiple times)
- [ ] Device code validation checks expiration before approval
- [ ] API rejects device code approval if code is expired or invalid
- [ ] JWT tokens generated contain only user ID (no organization scoping)
- [ ] User can access all organizations they belong to with single token

### Protobuf Service Definitions
- [ ] Update `auth/v1/auth.proto` to modify DeviceCodeRequest/Response (remove Google-specific fields)
- [ ] Add DeviceCodeApproveRequest message (deviceCode, userCode fields)
- [ ] Add DeviceCodeApproveResponse message (success boolean)
- [ ] Add DeviceCodeApprove RPC to AuthService

## Scope

### In Scope
- Device code generation and storage in MongoDB
- Web-based device approval flow using existing Google login
- Automatic redirect after login to device approval page
- Device code expiration via MongoDB TTL index (15 minutes)
- Single-use device codes
- Multi-organization access via user ID in JWT (no org scoping in token)
- Simple approval UI with minimal device information
- Error handling for expired/invalid codes

### Out of Scope
- Device information display (OS, hostname, IP, location) - simple approval only
- Organization selection during approval - user gets access to all orgs
- Device management/revocation UI - separate feature
- Multiple device sessions tracking - no device registry
- Push notifications for approval requests
- Mobile app approval flow
- QR code scanning for device approval
- Device naming or labeling
- Audit log for device approvals - separate feature
- Rate limiting on device code generation
- Device fingerprinting or trust scoring

## Constraints

### Technical Constraints
- Must use existing Google OAuth login from C-Ops Web (no new OAuth flow)
- Device codes must expire after exactly 15 minutes
- JWT tokens must contain only user ID (no organization ID in token payload)
- User can access all organizations they belong to without re-authentication
- MongoDB TTL index handles automatic cleanup of expired device codes
- CLI polling interval should remain at current value (likely 5 seconds)
- Web URL must be configurable via environment variable for different deployments

### Security Constraints
- Device codes must be cryptographically random
- User codes must be human-friendly (6-8 characters, avoid ambiguous characters)
- Device approval requires valid web session JWT token
- Expired or already-approved codes must be rejected
- Device codes are single-use only

### UX Constraints
- If user not logged in, must redirect to login then back to approval page automatically
- If user already logged in, show approval page immediately
- After approval, show clear success message telling user to return to terminal
- Device code must be displayed clearly in approval UI for verification

## Additional Context

### Current Implementation (Incorrect Flow)

**CLI Flow:**
```go
// cli/internal/service/auth/inbound/cli/cobra/login.go
// Currently displays Google OAuth device URL
fmt.Println("To sign in, open this URL in your browser:")
fmt.Printf("  %s\n\n", result.VerificationURL)  // Google URL
```

**API Service:**
```go
// api/internal/service/auth/auth_service.go
// DeviceCode() - Currently calls Google OAuth device flow
resp, err := s.oauthPort.InitiateDeviceFlow(ctx)

// DevicePoll() - Currently polls Google for completion
tokenResp, err := s.oauthPort.PollDeviceCode(ctx, deviceCode)
```

### New Implementation (Correct Flow)

**Flow Diagram:**
```
1. User runs: cops auth login
   ↓
2. CLI → API.DeviceCode()
   ↓
3. API generates device code → Stores in MongoDB
   ↓
4. API returns { userCode: "ABCD-EFGH", verificationURL: "https://cops.web/auth/device?code=ABCD-EFGH" }
   ↓
5. CLI displays: "Visit https://cops.web/auth/device?code=ABCD-EFGH"
   ↓
6. User opens URL in browser
   ↓
7a. If NOT logged in:
    - Redirect to /auth/login with returnUrl=/auth/device?code=ABCD-EFGH
    - User logs in with Google
    - Auto redirect back to /auth/device?code=ABCD-EFGH
   ↓
7b. If logged in:
    - Show approval page immediately
   ↓
8. User clicks "Approve" button
   ↓
9. Web → API.DeviceCodeApprove(deviceCode, userCode) with JWT in headers
   ↓
10. API validates JWT, links device code to user, marks approved
   ↓
11. CLI polling → API.DeviceCodePoll(deviceCode)
   ↓
12. API finds approved device code → Generates JWT tokens for user
   ↓
13. CLI receives tokens → Stores in ~/.cops/auth.json
```

### Device Code Domain Model

```go
// shared/domain/device_code.go
type DeviceCode struct {
    ID         string    `bson:"_id"`           // Secure random ID for polling
    UserCode   string    `bson:"userCode"`      // Human-friendly code (e.g., "ABCD-EFGH")
    UserID     *UserID   `bson:"userId"`        // Set when approved
    Approved   bool      `bson:"approved"`      // Approval status
    ExpiresAt  time.Time `bson:"expiresAt"`     // 15 minutes from creation
    CreatedAt  time.Time `bson:"createdAt"`
}
```

### MongoDB Collection Schema

**Collection: `device_codes`**
```javascript
{
  _id: "random-secure-device-code-id",
  userCode: "ABCD-EFGH",
  userId: "user_id" | null,
  approved: false,
  expiresAt: ISODate("2026-01-03T01:38:00Z"),
  createdAt: ISODate("2026-01-03T01:23:00Z")
}
```

**Indexes:**
- TTL index on `expiresAt` (automatic deletion after expiration)
- Unique index on `userCode`
- Index on `_id` (for polling)

### Web Component Structure

**Route Configuration:**
```typescript
// web/src/routes/auth/device.tsx
<Route path="/auth/device" element={<DeviceApproval />} />
```

**DeviceApproval Component Flow:**
1. Check if user is authenticated (via auth context/hook)
2. If not authenticated:
   - Store current URL with code parameter
   - Redirect to `/auth/login?returnUrl=/auth/device?code=ABCD-EFGH`
3. If authenticated:
   - Parse `code` from query parameters
   - Display approval UI
   - On approve button click, call API DeviceCodeApprove
   - Show success/error message

### Protobuf Changes

**Before (Current):**
```protobuf
// idl/protobuf/auth/v1/auth.proto
message DeviceCodeResponse {
  string device_code = 1;
  string user_code = 2;
  string verification_url = 3;  // Google OAuth URL
  int32 expires_in = 4;
  int32 interval = 5;
}
```

**After (New):**
```protobuf
// idl/protobuf/auth/v1/auth.proto
message DeviceCodeResponse {
  string device_code = 1;          // Secure ID for polling
  string user_code = 2;             // Human-friendly code
  string verification_url = 3;      // C-Ops Web URL
  int32 expires_in = 4;             // 900 (15 minutes)
  int32 interval = 5;               // Polling interval in seconds
}

message DeviceCodeApproveRequest {
  string device_code = 1;
  string user_code = 2;
}

message DeviceCodeApproveResponse {
  bool success = 1;
  string message = 2;  // Success or error message
}

service AuthService {
  rpc DeviceCode(DeviceCodeRequest) returns (DeviceCodeResponse);
  rpc DeviceCodePoll(DeviceCodePollRequest) returns (DeviceCodePollResponse);
  rpc DeviceCodeApprove(DeviceCodeApproveRequest) returns (DeviceCodeApproveResponse);  // NEW
  // ... other RPCs
}
```

### API Service Changes

**Remove:**
- Direct Google OAuth device flow integration (`oauthPort.InitiateDeviceFlow`)
- Google device code polling (`oauthPort.PollDeviceCode`)

**Add:**
- Device code generation and MongoDB storage
- Device code approval endpoint (validates JWT, links user, marks approved)
- Device code polling checks MongoDB instead of Google

**Updated Service Methods:**
```go
// api/internal/service/auth/auth_service.go

// DeviceCode generates a new device code and stores in MongoDB
func (s *Service) DeviceCode(ctx context.Context) (*DeviceCodeResult, error) {
    // Generate secure device code ID
    // Generate human-friendly user code (6-8 chars)
    // Calculate expiration (15 minutes from now)
    // Store in MongoDB
    // Return C-Ops Web URL with user code
}

// DevicePoll checks if device code has been approved in MongoDB
func (s *Service) DevicePoll(ctx context.Context, deviceCode string) (*DevicePollResult, error) {
    // Query MongoDB for device code by ID
    // Check if expired
    // Check if approved
    // If approved, generate JWT tokens for linked user
    // Return tokens or pending status
}

// DeviceCodeApprove approves a device code (NEW)
func (s *Service) DeviceCodeApprove(ctx context.Context, deviceCode string, userCode string, userID UserID) error {
    // Validate device code and user code match
    // Check not expired
    // Check not already approved
    // Update MongoDB: set userId, approved=true
    // Return success
}
```

### Configuration

**Environment Variables:**
```bash
# API Server
COPS_WEB_URL=https://cops.example.com  # Base URL for web application
COPS_DEVICE_CODE_EXPIRATION=15m        # Device code expiration (default: 15 minutes)

# Web Application
VITE_API_URL=https://api.cops.example.com  # API base URL
```

### Error Handling

**Device Approval Errors:**
- `DEVICE_CODE_NOT_FOUND`: Invalid or expired device code
- `DEVICE_CODE_EXPIRED`: Code expired (after 15 minutes)
- `DEVICE_CODE_ALREADY_APPROVED`: Code already used
- `INVALID_USER_CODE`: User code doesn't match device code
- `UNAUTHORIZED`: No valid web session JWT

**CLI Polling Errors:**
- Timeout after 15 minutes (device code expiration)
- Network errors (retry with exponential backoff)
- API errors (display and exit)

### Multi-Organization Access

**JWT Token Payload (No Organization Scoping):**
```json
{
  "sub": "user_id_12345",
  "exp": 1704326400,
  "iat": 1704324600
}
```

**Explanation:**
- JWT contains only user ID (`sub` claim)
- No organization ID in token payload
- API endpoints query user's organization memberships from database
- User can access all organizations they belong to
- Organization switching happens at application level, not token level

### Testing Checklist

**Manual Testing:**
- [ ] Run `cops auth login` and verify C-Ops Web URL is displayed
- [ ] Visit URL while logged out → Should redirect to login → Then back to approval page
- [ ] Visit URL while logged in → Should show approval page immediately
- [ ] Approve device → Should show success message
- [ ] Verify CLI receives tokens and stores in `~/.cops/auth.json`
- [ ] Try to approve same code twice → Should show error
- [ ] Wait 15 minutes → Code should be expired and rejected
- [ ] Verify user can access all organizations they belong to after CLI login

## Questions Resolved

| Question                           | Answer                      |
| ---------------------------------- | --------------------------- |
| Device approval UX when logged in? | Show simple "Approve Device" button with user code display |
| Device approval UX when not logged in? | Redirect to Google login, then automatically redirect back to approval page |
| Where to store device codes? | MongoDB in new `device_codes` collection with TTL index for auto-expiration |
| Device code expiration time? | 15 minutes (900 seconds) |
| Multi-organization handling? | User gets access to all organizations they belong to. JWT contains only user ID, no org scoping. |
| Web URL pattern for approval? | `/auth/device?code=USER_CODE` |
| Should we show device information? | No, simple approval only for MVP |
| Security confirmation details? | Just display user code and "Approve CLI Access" button, no device details |
| Can device codes be reused? | No, single-use only. Once approved, cannot be approved again. |
| What happens after user approves? | Show success message "Device approved! You can return to your terminal." |
