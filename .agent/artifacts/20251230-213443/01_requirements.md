# Requirements

## Request Summary

Migrate the codebase from the old `SessionRecord` model to the new `Record` model with custom JSON marshaling. This is a clean replacement with no backward compatibility - all usages of `SessionRecord`, `Message`, and related types in `session.go` and `message.go` will be replaced with the new `Record` model and its typed variants (`UserRecord`, `AssistantRecord`, `FileHistorySnapshotRecord`). After migration, the old domain files will be deleted completely.

## Acceptance Criteria

- [ ] All 17 files currently using `SessionRecord` are updated to use `Record` model
- [ ] CLI JSONL parser (`cli/internal/service/tracking/outbound/parser/jsonl/jsonl_parser.go`) parses all record types including `file-history-snapshot` (no filtering by type)
- [ ] Parser port interface returns `[]*domain.Record` instead of `[]*domain.SessionRecord`
- [ ] MongoDB adapter stores records in flattened format matching JSONL structure (all fields at root level with `type` as discriminator)
- [ ] MongoDB schema constants updated to reflect new Record model field names
- [ ] Protobuf definitions updated to match new Record model structure
- [ ] All generated gRPC stubs regenerated after proto updates
- [ ] Old domain files deleted: `shared/domain/session.go` and `shared/domain/message.go`
- [ ] All existing tests updated to use new Record model
- [ ] Code compiles successfully across all modules (cli, api, daemon, shared)
- [ ] No references to `SessionRecord`, `SessionType`, old `Message`, or old `Usage` types remain in codebase

## Scope

### In Scope

- **Parser Updates**:
  - Update JSONL parser to use `Record` unmarshaling
  - Remove type filtering logic (parse all record types)
  - Update parser port interface signature

- **Repository/Adapter Updates**:
  - Update MongoDB adapter to handle `Record` model
  - Maintain flattened document structure in MongoDB
  - Update field mapping logic for type-specific fields

- **Domain Model Changes**:
  - Use existing `Record`, `UserRecord`, `AssistantRecord`, `FileHistorySnapshotRecord` types
  - Update MongoDB schema constants to match new field names
  - Delete `session.go` and `message.go` after migration

- **Protobuf/gRPC Updates**:
  - Update proto definitions to use Record model structure
  - Regenerate gRPC stubs using `buf generate`
  - Update service handlers to use new types

- **Service Layer Updates**:
  - Update aggregation service to work with `Record` model
  - Update dashboard service to work with `Record` model
  - Update daemon log watcher to work with `Record` model

- **Test Updates**:
  - Update all existing tests to use new Record model
  - Ensure tests cover type-specific record handling

### Out of Scope

- **Backward Compatibility**: No migration path or dual support for old and new models
- **Data Migration**: No scripts to migrate existing MongoDB data (fresh start acceptable)
- **API Versioning**: No v1/v2 API versioning strategy
- **Gradual Rollout**: Clean break, not incremental migration
- **New Features**: Only model replacement, no new functionality or enhancements
- **Performance Optimization**: Focus on correctness, not performance improvements

## Constraints

- **Clean Break**: Acceptable to break existing data and APIs during migration
- **Module Structure**: Must maintain Go workspace structure with separate modules (cli, api, daemon, shared)
- **Code Generation**: Must use `buf generate` for protobuf code generation (commands in `idl/protobuf/`)
- **Build Process**: All modules must build successfully with `go build ./...` from root
- **JSON Format**: MongoDB storage must match JSONL flattened format (no nested type/data structure)
- **Type Safety**: Use type assertions or type switches when working with `Record.Data` field

## Additional Context

- **Reference Implementation**: The new Record model with custom marshaling is already implemented in `shared/domain/record.go`, `record_user.go`, `record_assistant.go`, and `record_file_history_snapshot.go`
- **Example JSONL Format**: `shared/domain/record_user.jsonl` and `record_assistant.jsonl` contain example flattened records
- **Custom Marshaling**: The `Record` type handles flattening/unflattening automatically via `MarshalJSON`/`UnmarshalJSON` methods
- **Type Registry**: `recordTypeRegistry` in `record.go` maps `RecordType` to concrete types for unmarshaling
- **MongoDB Collection**: Records are stored in `sessionRecords` collection (collection name unchanged)

## Questions Resolved

| Question | Answer |
| --- | --- |
| Should we keep the same flattened structure in MongoDB? | Yes, maintain flattened structure matching JSONL format. All fields at root level with `type` as discriminator. |
| What should happen to existing MongoDB data? | Fresh start acceptable. No migration script needed. |
| Should proto definitions be updated or keep conversion logic? | Update proto definitions to match new Record model structure. |
| Should we filter JSONL records by type? | No, parse all record types including `file-history-snapshot`. Remove type filtering logic. |
| Should we change ParserPort interface? | Yes, change to return `[]*domain.Record` instead of `[]*domain.SessionRecord`. |
| What level of test coverage is expected? | Update all existing tests to use new Record model. Ensure type-specific handling is tested. |
| Is backward compatibility required? | No, clean break acceptable. Can break existing data/APIs during migration. |
