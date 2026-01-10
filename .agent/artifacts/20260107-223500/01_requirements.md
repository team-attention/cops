# Requirements: TA-139 API Key Authentication Design

## Task Summary

Design the API key-based authentication system to replace OAuth tokens for project-level authentication. The web dashboard issues/revokes API keys, and CLI/Hook uses these keys to authenticate with the API server.

## Acceptance Criteria

- [ ] API key issuance/revocation protobuf service definition (`idl/protobuf/apikey/v1/apikey.proto`)
- [ ] API key schema definition (key format, metadata fields)
- [ ] MongoDB Collection schema design (`api_keys` collection)
- [ ] Run `buf generate` to confirm Go code generation

## Scope

### In Scope

* API key issuance/revocation RPC service protobuf definition
* API key data model design (MongoDB Collection schema)
* Key verification requirements definition (Header-based, Bearer token format)
* Client-side key storage location specification (`~/.cops/auth.json`)

### Out of Scope

* API server handler implementation (handled in TA-140)
* Web dashboard UI implementation
* Rate limiting, key expiration policy (Phase 2 consideration)
* `~/.cops/auth.json` parsing logic (handled in TA-138)

## Constraints

* **Protocol**: Use ConnectRPC (following existing project patterns)
* **Storage**: MongoDB (using existing infrastructure)
* **Authentication header**: `Authorization: Bearer {api_key}` format
* **File location**: protobuf defined at `idl/protobuf/apikey/v1/` path

## Related Tasks

* **TA-131**: Parent task - Hook Phase 1 design stage
* **TA-137**: Hook event protocol design (authentication separated via Header)
* **TA-138**: Hook configuration structure design (`~/.cops/auth.json` schema definition)
* **TA-140**: API key issuance/verification implementation (based on this design)

## Source

Linear Issue: [TA-139](https://linear.app/team-attention/issue/TA-139/api-키-인증-설계)
