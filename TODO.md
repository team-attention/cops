# TODO: Database Index Migration

This document tracks database index requirements that need to be implemented via migration scripts.

**Note**: These indexes are NOT managed in Go codebase. They should be created via separate migration scripts.

## User Authentication Indexes

### Collection: `users`

1. **Unique index on email**
   - Field: `email`
   - Unique: `true`
   - Purpose: Prevent duplicate user accounts

2. **Index on embedded accounts array**
   - Fields: `accounts.provider`, `accounts.providerId`
   - Purpose: Efficient lookup by OAuth provider account
   - Query pattern: `{ "accounts": { "$elemMatch": { "provider": X, "providerId": Y } } }`

### Collection: `organizations`

1. **Unique index on slug**
   - Field: `slug`
   - Unique: `true`
   - Purpose: URL-friendly unique organization identifier

### Collection: `organization_members`

1. **Unique compound index on organization + user**
   - Fields: `organizationId`, `userId`
   - Unique: `true`
   - Purpose: Prevent duplicate memberships

2. **Index on userId**
   - Field: `userId`
   - Purpose: Efficient lookup of user's organizations

### Collection: `projects`

1. **Index on organizationId**
   - Field: `organizationId`
   - Purpose: Efficient lookup of organization's projects

## Migration Script Template

```javascript
// Example MongoDB migration script
db.users.createIndex({ "email": 1 }, { unique: true });
db.users.createIndex({ "accounts.provider": 1, "accounts.providerId": 1 });

db.organizations.createIndex({ "slug": 1 }, { unique: true });

db.organization_members.createIndex(
  { "organizationId": 1, "userId": 1 },
  { unique: true }
);
db.organization_members.createIndex({ "userId": 1 });

db.projects.createIndex({ "organizationId": 1 });
```

## Status

- [ ] Create migration scripts
- [ ] Test migrations in development environment
- [ ] Apply to staging
- [ ] Apply to production
