# Requirements: User and Organization Management with Authentication

## Request Summary

The current C-Ops system tracks Claude Code sessions across registered projects but lacks user authentication and organizational structure. This feature will introduce User, Account, and Organization domain models with Google OAuth-based authentication. Projects will be scoped to organizations. The CLI/Daemon will support single account authentication with logout/login workflow for account switching.

## Acceptance Criteria

### Domain Models
- [ ] User domain model defined with fields (ID, email, name, profile image URL, accounts array, creation timestamp)
- [ ] Account domain model defined with fields (provider name, provider ID, linked user ID, creation timestamp)
- [ ] Organization domain model defined with fields (ID, name, slug, owner ID, creation timestamp)
- [ ] OrganizationMember domain model defined with relationship fields (organization ID, user ID, role, joined timestamp)
- [ ] Project model updated to include organization ID foreign key
- [ ] MongoDB schemas created for all new domain models following existing patterns

### Authentication & Authorization
- [ ] Google OAuth 2.0 sign-in and sign-up flow implemented in API server
- [ ] JWT token generation and validation implemented
- [ ] Token refresh mechanism implemented
- [ ] Session management for web dashboard
- [ ] Authorization middleware for API endpoints checking organization membership
- [ ] RBAC-like permission system for organization roles (Owner, Admin, Member)

### gRPC Services (ConnectRPC)
- [ ] AuthService.GoogleAuth - Exchange Google OAuth code for JWT tokens
- [ ] AuthService.RefreshToken - Refresh access token using refresh token
- [ ] AuthService.GetCurrentUser - Get current authenticated user information
- [ ] OrganizationService.CreateOrganization - Create new organization
- [ ] OrganizationService.ListOrganizations - List user's organizations
- [ ] OrganizationService.GetOrganization - Get organization details
- [ ] OrganizationService.AddMember - Add member to organization
- [ ] ProjectService.RegisterProject - Updated to include organization ID and validate membership
- [ ] AggregationService.SendLogs - Updated to validate organization membership
- [ ] Protobuf definitions created in idl/protobuf following existing patterns

### CLI Authentication
- [ ] `cops auth login` command implemented for Google OAuth device flow
- [ ] `cops auth logout` command to remove authentication tokens
- [ ] `cops auth status` command to show current authenticated user and organization
- [ ] Authentication token storage in `~/.cops/auth.json` with secure permissions (0600)
- [ ] CLI commands automatically include authentication token in API requests
- [ ] Project registration scoped to authenticated user's organization
- [ ] Account switching requires logout and re-login workflow

### Daemon Authentication
- [ ] Daemon reads authentication from shared CLI config (`~/.cops/auth.json`)
- [ ] Daemon includes authentication token in API requests
- [ ] Daemon handles token refresh automatically
- [ ] Daemon logs authentication failures clearly for debugging
- [ ] Daemon stops sending data if authentication fails until user re-authenticates

## Scope

### In Scope
- User domain model with extensible account linking
- Account domain model for multi-provider support (provider, providerID)
- Organization domain model with basic RBAC (Owner, Admin, Member)
- OrganizationMember relationship model
- Google OAuth 2.0 sign-in and sign-up
- JWT-based authentication for API
- CLI single account authentication with logout/login for switching
- Daemon authentication using CLI auth config
- Project scoping to organizations
- Basic authorization middleware for gRPC endpoints
- Protobuf service definitions for auth and organization management
- Web dashboard authentication UI

### Out of Scope
- Other OAuth providers (GitHub, Microsoft, etc.) - Account model supports future expansion
- Email/password authentication - Google OAuth only for MVP
- Multi-factor authentication (MFA)
- Multiple context switching (kubectl-style) - single account only, use logout/login to switch
- Fine-grained permissions beyond role-based access
- Organization transfer or deletion workflows
- User deletion or account management
- Audit logging for organization activities
- Rate limiting per organization
- Organization billing or usage quotas
- Invitation email system (use manual invitation codes for MVP)
- SSO/SAML integration
- API key authentication (separate feature)
- Backwards compatibility with existing data (new system)

## Constraints

### Technical Constraints
- Must use Google OAuth 2.0 for authentication
- Account model must separate provider type and provider ID for future OAuth provider expansion
- JWT tokens must be short-lived (15-30 minutes) with refresh tokens (7-30 days)
- CLI authentication must work in headless/SSH environments (device flow)
- All CLI commands must include authentication
- Database queries must efficiently filter by organization ID
- Token storage must have appropriate file permissions (0600)
- All API definitions must be in protobuf format, not REST endpoints

### Security Constraints
- Tokens must never be logged or exposed in error messages
- Authentication tokens must be stored securely on local filesystem
- API must validate organization membership on every protected endpoint
- JWT tokens must include organization context to minimize database lookups
- Refresh tokens must be rotatable and revocable

### Compatibility Constraints
- Proto definitions should support forward compatibility for future enhancements
- Account model structure allows adding new OAuth providers without breaking changes

## Additional Context

### Current System Architecture

**Domain Models (Existing):**
- `Project`: ID, Name, Path, IsGitProject, RegisteredAt
- `Record`: Session records with user/assistant messages
- Configuration stored in `~/.cops/config.json`

**Authentication Flow (None Currently):**
- No authentication exists
- API endpoints are publicly accessible
- CLI and Daemon make unauthenticated requests

**CLI Structure:**
- Uses Cobra for command handling
- Stores config in `~/.cops/config.json`
- Uses dependency injection via `dig`

**Daemon Architecture:**
- Watches `~/.cops/config.json` for changes
- Monitors Claude Code JSONL logs
- Sends records to API via ConnectRPC

**API Architecture:**
- Fiber HTTP server with ConnectRPC handlers
- MongoDB for persistence
- Services follow hexagonal architecture
- All APIs defined as protobuf services

### Account Model Design

The system separates user identity from OAuth provider accounts to support future multi-provider login:

**User Model:**
- Core identity (ID, email, name, profile image)
- References to linked accounts via accounts array

**Account Model:**
- Provider name (e.g., "google", "github")
- Provider-specific user ID
- Linked to parent User via userID

**Benefits:**
- Single user can link multiple OAuth providers (future)
- Easy to add new providers without schema changes
- Clear separation between identity and authentication method

### CLI Authentication Storage

Single account authentication stored in `~/.cops/auth.json`:

```json
{
  "user": {
    "id": "user_id",
    "email": "user@example.com",
    "name": "User Name"
  },
  "organization": {
    "id": "org_id",
    "name": "Organization Name",
    "slug": "org-slug"
  },
  "tokens": {
    "accessToken": "jwt_access_token",
    "refreshToken": "jwt_refresh_token",
    "expiresAt": "2025-01-15T10:00:00Z"
  }
}
```

**Account Switching:**
- User must run `cops auth logout` to clear current session
- Then run `cops auth login` to authenticate with different account
- No context switching - simple single account model

### Google OAuth Integration

**Web Dashboard Flow:**
1. User clicks "Sign in with Google"
2. Redirect to Google OAuth consent screen
3. Google redirects back with authorization code
4. Frontend exchanges code for tokens via API
5. API validates with Google, creates/finds user, returns JWT

**CLI Device Flow:**
1. User runs `cops auth login`
2. CLI displays Google device code URL and code
3. User opens URL in browser and enters code
4. User authenticates with Google
5. CLI polls Google for completion
6. CLI exchanges authorization for tokens via API
7. Tokens stored in `~/.cops/auth.json`

### Database Schema

**New Collections:**
- `users`: User documents (ID, email, name, profileImageURL, createdAt, updatedAt)
- `accounts`: OAuth account documents (ID, provider, providerID, userID, createdAt)
- `organizations`: Organization documents (ID, name, slug, ownerID, createdAt, updatedAt)
- `organization_members`: Membership relationships (ID, organizationID, userID, role, joinedAt)

**Modified Collections:**
- `projects`: Add `organizationId` field (required)

**Indexes:**
- `accounts`: Unique index on (provider, providerID)
- `accounts`: Index on userID
- `organization_members`: Unique index on (organizationID, userID)
- `organization_members`: Index on userID
- `projects`: Index on organizationID

### Protobuf Service Definitions

New proto files to be created following existing patterns:

**idl/protobuf/auth/v1/auth.proto:**
- AuthService with GoogleAuth, RefreshToken, GetCurrentUser RPCs
- Message types for requests/responses

**idl/protobuf/organization/v1/organization.proto:**
- OrganizationService with CreateOrganization, ListOrganizations, GetOrganization, AddMember RPCs
- Message types for organization and member data

**Modified proto files:**
- `project/v1/project.proto`: Add organizationId to RegisterProjectReq
- `aggregation/v1/aggregation.proto`: Add authentication context validation

### Related Documentation

- [Project Structure](.agent/rules/project.md)
- [Protobuf Guidelines](.agent/rules/idl/protobuf.md)
- [Platform Domain Guidelines](.agent/rules/go/go-platform-domain.md)
- [Hexagonal Architecture](.agent/rules/go/go-hexagonal-layout.md)

## Questions Resolved

| Question | Answer |
| ---------------------------------- | --------------------------- |
| How should OAuth provider info be stored in User model? | Separate into Account model with provider and providerID fields. User has accounts array for future multi-provider support. |
| Should API endpoints be defined as REST? | No, all APIs must be defined as protobuf services following ConnectRPC pattern. |
| Should we support multiple account contexts like kubectl? | No, single account only. Users must logout and re-login to switch accounts. |
| Do we need backwards compatibility for existing data? | No, this is a new system. No migration needed. |
| What authentication provider should be used? | Google OAuth 2.0 only for MVP. Account model designed for future providers. |
| How should CLI handle authentication in headless environments? | Use Google OAuth device flow - CLI displays URL and code for user to enter in browser. |
| What organization roles are needed? | Three roles for MVP: Owner (full control), Admin (manage members and projects), Member (view and create projects). |
| How should tokens be stored on local machine? | In `~/.cops/auth.json` with file permissions 0600, single account structure. |
| Should API support multiple authentication methods? | No, only JWT tokens for MVP. API keys can be added as separate feature. |
| What happens if token refresh fails? | CLI should prompt user to re-authenticate with `cops auth login`. Daemon should log error and stop sending data until re-authenticated. |
| How should organization be selected during project creation? | Use organization from current authenticated session. |
| What token expiration times should be used? | Access token: 30 minutes, Refresh token: 30 days. These can be configured via environment variables. |
| How should the web dashboard handle authentication? | Use standard OAuth redirect flow with JWT tokens stored in HTTP-only cookies or localStorage. |
