# Requirements Document: Account Deletion Feature

## Existing System Analysis

### Current Authentication & User Management

**Frontend (Web):**
- Settings page exists at `/settings` but is currently a placeholder ("Coming soon")
- User authentication via Google OAuth (web flow) and device flow (CLI)
- User state managed by:
  - `useUser` hook - provides user data, organizations, and state management
  - `useAuth` hook - handles authentication tokens
  - Zustand store for centralized user state
- User data includes: id, email, name, avatarUrl

**Backend (API):**
- User service (`api/internal/service/user/`) with:
  - `GetMe` RPC - fetches authenticated user info and organizations
  - User repository with MongoDB adapter
  - Organization repository for user memberships
- Auth service (`api/internal/service/auth/`) handles:
  - Google OAuth authentication
  - JWT token generation/refresh
  - Device code flow for CLI
- User domain model (`domain.v1.User`):
  ```protobuf
  message User {
    string id = 1;
    string email = 2;
    string name = 3;
    string avatar_url = 4;
  }
  ```

### Data Associated with Users

Based on the system architecture:
- **User record** - stored in users collection
- **Organizations** - user memberships in organizations
- **Projects** - projects owned by user's organizations
- **Sessions** - Claude Code session records collected by daemon
- **Device codes** - temporary codes for CLI authentication

---

---

## CONFIRMED REQUIREMENTS

### Request Summary

Implement account deletion feature that allows authenticated users to permanently delete their C-Ops account through the web interface. This includes hard deletion of the user's profile and cascade deletion of any organizations where the user is the sole member (along with all associated projects and sessions). User must type 'DELETE' to confirm the deletion action.

### Confirmed Acceptance Criteria

- [x] User can access account deletion option from Settings page
- [x] System requires explicit confirmation before deletion (user types 'DELETE')
- [x] System validates user is authenticated via JWT before allowing deletion
- [x] Backend performs cascade deletion:
  - Delete user profile (and all authentication accounts)
  - For each organization where user is SOLE member:
    - Delete all sessions for projects in that organization
    - Delete all projects in that organization
    - Delete the organization itself
  - For organizations where user is NOT sole member:
    - Remove user from membership only
    - Keep organization, projects, and sessions intact for other members
  - Invalidate all user tokens (logout all sessions)
- [x] Frontend shows clear warning modal about cascade deletion with explanation
- [x] API returns appropriate errors for unauthorized attempts
- [x] Hard delete is permanent - no grace period or recovery option

### Confirmed Scope

#### In Scope
- Settings page with "Delete Account" section (danger zone)
- Confirmation modal with detailed warning about what will be deleted
- Confirmation field: user must type 'DELETE' (case-sensitive)
- Backend RPC endpoint: `DeleteAccount()`
- User profile deletion with all authentication accounts
- Cascade delete organizations where user is sole member (along with projects/sessions)
- Remove user membership from shared organizations (non-destructive)
- Token invalidation and session logout
- Error handling for edge cases (e.g., validation errors)

#### Out of Scope
- Email notifications
- Grace period or account recovery
- Soft delete (data archiving)
- Data export before deletion
- Audit logging of deletion events
- Admin ability to recover deleted accounts
- Automatic organization transfer to other members

### Confirmed Constraints

**Technical:**
- Must use existing authentication (JWT in Authorization header)
- Must follow existing ConnectRPC patterns
- Must use MongoDB transactions for data consistency
- Frontend must use TanStack Query + ConnectRPC hooks
- Hard delete is atomic - all-or-nothing operation

**Business:**
- User confirmation required: must type 'DELETE' to proceed
- Cascade deletion applies only to organizations where user is SOLE member
- No recovery possible after deletion

**Security:**
- Validate user owns the account being deleted
- Invalidate all refresh tokens and access tokens after deletion
- Case-sensitive 'DELETE' confirmation phrase

### Deletion Logic Detail

When user deletes account:

1. Validate user is authenticated and confirmed deletion with 'DELETE' phrase
2. Retrieve all organizations where user is member
3. For each organization:
   - If user is the ONLY member:
     - Get all projects under this organization
     - For each project, delete all associated sessions
     - Delete all projects
     - Delete the organization
   - If user is NOT the only member:
     - Remove user from organization members only
4. Delete user profile and all authentication accounts
5. Invalidate all user's tokens

### Additional Context

**Related Services:**
- User service (`api/internal/service/user/`)
- Auth service (`api/internal/service/auth/`)
- RBAC service (`api/internal/service/core/rbac/`)
- Project service (`api/internal/service/project/`)
- Aggregation service (`api/internal/service/aggregation/`)

**Database Collections Affected:**
- `users` - user documents
- `organizations` - organization member arrays
- `projects` - projects owned by organizations
- `sessions` - session records for projects
- `device_codes` - temporary auth codes

---

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Q1: Hard delete vs. Soft delete? | Hard delete - permanent immediate deletion |
| Q2: What data should be deleted? | User profile, auth accounts, sole-member orgs with cascade, project/session cleanup |
| Q3: What happens to organizations owned by user? | Cascade delete if user is sole member, just remove membership if shared |
| Q4: What confirmation mechanism? | User must type 'DELETE' phrase (case-sensitive) |
| Q5: Grace period before deletion? | No - immediate permanent deletion |
| Q6: Email notifications? | No email notifications required |
| Q7: Where to place delete button? | Settings page under "Danger Zone" section |
| Q8: Warning modal? | Yes, with detailed explanation of cascade deletion impact |
| Q9: Single RPC or multi-step? | Single RPC call: `DeleteAccount()` with confirmation token/phrase |
