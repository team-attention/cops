# Requirements

## Request Summary

Set up a MongoDB migration management system using `migrate-mongo` in the `ops/db/mongodb/` directory. This will enable infrastructure-as-code management for database schema changes. The first migration script will create all required indexes for the C-Ops system, including indexes for user authentication, organizations, projects, and device codes collections.

## Acceptance Criteria

- [ ] `ops/db/mongodb/` directory structure is created at the project root
- [ ] `migrate-mongo` is installed and initialized in the `ops/db/mongodb/` directory
- [ ] Configuration file (`migrate-mongo-config.js`) is properly configured with MongoDB connection settings
- [ ] First migration script is created that includes ALL index definitions from `TODO.md` and `doc/mongodb-indexes.md`
- [ ] Migration script creates indexes for all five collections: `users`, `organizations`, `organization_members`, `projects`, and `deviceCodes`
- [ ] Migration includes both `up` (create indexes) and `down` (drop indexes) functions
- [ ] Configuration uses environment variables for MongoDB connection (database name, URL)
- [ ] Documentation is provided on how to run migrations (up/down/status commands)
- [ ] `.gitignore` is updated to exclude `node_modules` in ops directory if needed

## Scope

### In Scope
- Creating `ops/db/mongodb/` directory structure
- Installing and configuring `migrate-mongo` tool
- Creating first migration script with all required indexes:
  - `users` collection: unique email index, compound index on accounts.provider + accounts.providerId
  - `organizations` collection: unique slug index
  - `organization_members` collection: unique compound index on organizationId + userId, index on userId
  - `projects` collection: index on organizationId
  - `deviceCodes` collection: TTL index on expiresAt, unique index on userCode
- Configuration setup using environment variables
- Basic documentation for running migrations

### Out of Scope
- Other `ops/` subdirectories (cloud, k8s, etc.) - only MongoDB migration management
- Automated CI/CD integration for running migrations
- Integration with Go application startup code
- Multiple migration files (all indexes in one migration)
- Environment-specific migration tracking beyond connection string configuration
- Data migrations (only schema/index migrations)

## Constraints

- Must use `migrate-mongo` as the migration tool (Node.js-based, industry standard for MongoDB)
- Migration scripts must be written in JavaScript using native MongoDB syntax
- Must follow the index specifications exactly as documented in `TODO.md` and `doc/mongodb-indexes.md`
- Directory must be located at `ops/db/mongodb/` relative to project root (`/Users/jayce/team-attention/cops/ops/db/mongodb/`)
- Configuration must support environment variable overrides for connection settings

## Additional Context

### Source Documentation
- **TODO.md**: Contains index requirements for users, organizations, organization_members, and projects collections
- **doc/mongodb-indexes.md**: Contains index requirements for deviceCodes collection (TTL index, unique userCode)

### Index Requirements Summary

**users collection:**
```javascript
{ "email": 1 }, { unique: true }
{ "accounts.provider": 1, "accounts.providerId": 1 }
```

**organizations collection:**
```javascript
{ "slug": 1 }, { unique: true }
```

**organization_members collection:**
```javascript
{ "organizationId": 1, "userId": 1 }, { unique: true }
{ "userId": 1 }
```

**projects collection:**
```javascript
{ "organizationId": 1 }
```

**deviceCodes collection:**
```javascript
{ "expiresAt": 1 }, { expireAfterSeconds: 0 }  // TTL index
{ "userCode": 1 }, { unique: true }
```

### migrate-mongo Tool Information
- Installation: `npm install -g migrate-mongo` or local to ops directory
- Initialization: `migrate-mongo init`
- Create migration: `migrate-mongo create <description>`
- Run migrations: `migrate-mongo up`
- Rollback: `migrate-mongo down`
- Check status: `migrate-mongo status`

### Configuration Requirements
- Database name should be configurable via environment variable
- MongoDB connection URL should be configurable via environment variable
- Default values should be provided for local development
- Configuration file: `migrate-mongo-config.js`

## Questions Resolved

| Question | Answer |
|----------|--------|
| Which migration tool to use? | migrate-mongo (JavaScript-based, MongoDB-specific, industry standard) |
| Should all indexes be in one migration or separate? | All indexes in one migration script |
| Should we create other ops/ subdirectories? | No, only `ops/db/mongodb/` for MongoDB migrations |
| How should migrations be executed? | Manual CLI execution (documentation provided, automation out of scope) |
| Environment configuration needed? | Yes, use environment variables for database name and connection URL with sensible defaults |
