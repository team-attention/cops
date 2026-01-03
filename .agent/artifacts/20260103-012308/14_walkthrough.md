# Walkthrough: CLI Authentication via C-Ops Web Device Flow

## Summary

Changed CLI authentication flow to route through C-Ops Web instead of directly to Google OAuth. Users now authenticate CLI by visiting a C-Ops Web device approval page, which uses their existing Google login session.

**Flow Change:**
- Before: CLI → API → Google OAuth Device Flow (direct)
- After: CLI → API → C-Ops Web `/auth/device` → API → JWT tokens

## Changes Overview

### Backend (Go)

#### Domain Layer
- **`shared/domain/device_code.go`**: New DeviceCode domain model with ID, UserCode, UserID (optional), Approved, ExpiresAt fields
- **`shared/domain/mongoschema/device_code.go`**: MongoDB schema with `deviceCodes` collection name and BSON field mappings

#### Repository Layer
- **`api/internal/service/auth/outbound/repository/device_code_repo_port.go`**: DeviceCodeRepositoryPort interface with Create, FindByID, FindByUserCode, Approve methods
- **`api/internal/service/auth/outbound/repository/mongodb/device_code_repo.go`**: MongoDB implementation of device code repository

#### Service Layer
- **`api/internal/service/auth/auth_service.go`**:
  - Added DeviceCodeApprove method for web-based approval
  - Added private helpers: generateDeviceCodeID, generateUserCode, normalizeUserCode
  - Modified DeviceCode to generate and store codes in MongoDB instead of calling Google
  - Modified DevicePoll to check MongoDB approval status

#### gRPC Layer
- **`api/internal/service/auth/inbound/grpc/connectrpc/handler.go`**: Added DeviceCodeApprove handler with JWT authentication
- **`idl/protobuf/auth/v1/auth.proto`**: Added DeviceCodeApprove RPC with request/response messages

#### Configuration
- **`api/internal/platform/setup/config/config.go`**:
  - Added DeviceCodeConfig for expiration and web URL settings
  - Added Authorization header to CORS allowed headers
- **`api/cmd/internal/container/module_auth.go`**: Updated DI container for new repository

### CLI (Go)

- **`cli/internal/service/auth/auth_service.go`**: Removed homeDir parameter from constructor, calls os.UserHomeDir() internally
- **`cli/internal/service/auth/outbound/api/connectrpc/auth_client.go`**: Changed to accept *config.Config and *httpclient.APIHTTPClient instead of primitive types

### Web (React/TypeScript)

#### Auth Feature
- **`web/src/feature/auth/component/device-approval.tsx`**: Device approval UI component with code display and approve button
- **`web/src/feature/auth/hook/use-approve-device.ts`**: TanStack Query mutation hook for device approval API
- **`web/src/feature/auth/hook/use-google-auth.ts`**: Google OAuth mutation hook
- **`web/src/feature/auth/type/device-code.ts`**: TypeScript type definitions

#### Routes
- **`web/src/route/auth/index.tsx`**: Login page with "Sign in with Google" button
- **`web/src/route/auth/callback.tsx`**: OAuth callback handler that stores tokens and redirects
- **`web/src/route/auth/device.tsx`**: Device approval page with beforeLoad auth guard

#### Shared
- **`web/src/shared/hook/use-auth.ts`**: Auth state hook (isAuthenticated, storeTokens, logout)
- **`web/src/shared/service/connect-transport.ts`**: Added auth interceptor for JWT tokens in API requests

### Documentation

- **`doc/mongodb-indexes.md`**: TTL index documentation for automatic device code expiration

## Key Implementation Details

### Device Code Flow

1. CLI calls `DeviceCode` API endpoint
2. API generates secure device code ID and human-friendly user code (XXXX-XXXX format)
3. API stores device code in MongoDB with 15-minute TTL
4. API returns verification URL: `{WEB_URL}/auth/device?code={USER_CODE}`
5. User visits URL in browser
6. If not logged in, redirected to `/auth` → Google OAuth → back to device page
7. User clicks "Approve Device" button
8. Web calls `DeviceCodeApprove` with JWT token
9. API marks device code as approved and links to user
10. CLI polling receives JWT tokens

### Security

- Device codes expire after 15 minutes (MongoDB TTL index)
- Single-use codes (cannot be approved twice)
- JWT tokens contain only user ID (no org scoping)
- Authorization header properly configured in CORS

## Files Changed

| Category | Files |
|----------|-------|
| Backend Domain | `shared/domain/device_code.go`, `shared/domain/mongoschema/device_code.go` |
| Backend Repository | `api/internal/service/auth/outbound/repository/device_code_repo_port.go`, `api/internal/service/auth/outbound/repository/mongodb/device_code_repo.go` |
| Backend Service | `api/internal/service/auth/auth_service.go` |
| Backend gRPC | `api/internal/service/auth/inbound/grpc/connectrpc/handler.go` |
| Backend Config | `api/internal/platform/setup/config/config.go`, `api/cmd/internal/container/module_auth.go` |
| Protobuf | `idl/protobuf/auth/v1/auth.proto` |
| CLI | `cli/internal/service/auth/auth_service.go`, `cli/internal/service/auth/outbound/api/connectrpc/auth_client.go` |
| Web Routes | `web/src/route/auth/index.tsx`, `web/src/route/auth/callback.tsx`, `web/src/route/auth/device.tsx` |
| Web Feature | `web/src/feature/auth/component/device-approval.tsx`, `web/src/feature/auth/hook/*.ts`, `web/src/feature/auth/type/device-code.ts` |
| Web Shared | `web/src/shared/hook/use-auth.ts`, `web/src/shared/service/connect-transport.ts` |
| Generated | `shared/gen/grpcstub/auth/v1/*.go`, `web/src/gen/grpcstub/auth/v1/*.ts`, `web/src/routeTree.gen.ts` |
| Docs | `doc/mongodb-indexes.md` |
