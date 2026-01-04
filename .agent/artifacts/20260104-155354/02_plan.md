# Implementation Plan: MongoDB Migration Management with migrate-mongo

## Overview

This implementation sets up MongoDB migration management using `migrate-mongo` in the `ops/db/mongodb/` directory. The migration system will enable infrastructure-as-code management for database schema changes. The first migration script will create all required indexes for the C-Ops system across five collections: `users`, `organizations`, `organization_members`, `projects`, and `deviceCodes`.

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| Add | MongoDB migration management | `migrate-mongo` | Industry-standard Node.js-based migration tool specifically designed for MongoDB. Supports up/down migrations, changelog tracking, and CLI commands. |

## Step 1: Create Directory Structure

**Files to Read**: None required.

### Create directories

Create the following directory structure:

```
ops/
  db/
    mongodb/
      .meta/         (environment configuration files)
      migrations/    (will hold migration files)
```

**Command to execute**:
```bash
mkdir -p /Users/jayce/team-attention/cops/ops/db/mongodb/migrations
mkdir -p /Users/jayce/team-attention/cops/ops/db/mongodb/.meta
```

## Step 2: Create package.json and Install Dependencies

**Files to Read**: None required.

### `/Users/jayce/team-attention/cops/ops/db/mongodb/package.json`

**Description**: Create a package.json file for the MongoDB migrations directory with migrate-mongo as a local dependency.

```json
{
  "name": "cops-mongodb-migrations",
  "version": "1.0.0",
  "description": "MongoDB migrations for C-Ops system",
  "private": true,
  "scripts": {
    "migrate:up": "migrate-mongo up",
    "migrate:down": "migrate-mongo down",
    "migrate:status": "migrate-mongo status",
    "migrate:create": "migrate-mongo create"
  },
  "dependencies": {
    "migrate-mongo": "latest"
  }
}
```

**Commands to execute**:
```bash
cd /Users/jayce/team-attention/cops/ops/db/mongodb && npm install
```

## Step 3: Create Environment Configuration

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/Makefile`: Reference for Makefile pattern with environment files

### `/Users/jayce/team-attention/cops/ops/db/mongodb/.meta/.env.example`

**Description**: Create an example environment file with MongoDB connection variables. This follows the same pattern as `api/.meta/.env.example`.

```bash
# MongoDB connection configuration
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=cops
```

### `/Users/jayce/team-attention/cops/ops/db/mongodb/.meta/.gitignore`

**Description**: Create a .gitignore to exclude actual .env files while keeping .env.example.

```gitignore
.env
.env.*
!.env.example
```

## Step 4: Create Makefile

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/Makefile`: Reference for Makefile pattern

### `/Users/jayce/team-attention/cops/ops/db/mongodb/Makefile`

**Description**: Create a Makefile following the api/Makefile pattern with environment file loading and migration targets.

```makefile
ENV ?= local

# Environment file configuration
ENV_FILES := --env-file .meta/.env
ifdef ENV
	ENV_FILES += --env-file .meta/.env.$(ENV)
endif

include .meta/.env
-include .meta/.env.$(ENV)
export

## migration-up: Run all pending migrations
.PHONY: migration-up
migration-up:
	npm run migrate:up

## migration-down: Rollback the last applied migration
.PHONY: migration-down
migration-down:
	npm run migrate:down
```

## Step 5: Create migrate-mongo Configuration File

**Files to Read**: None required.

### `/Users/jayce/team-attention/cops/ops/db/mongodb/migrate-mongo-config.js`

**Description**: Create the migrate-mongo configuration file with environment variable support for MongoDB connection settings. Uses `MONGODB_URI` and `MONGODB_DATABASE` environment variables to match the api configuration pattern.

```javascript
// migrate-mongo configuration file
// Environment variables:
//   MONGODB_URI - MongoDB connection URL (default: mongodb://localhost:27017)
//   MONGODB_DATABASE - Database name (default: cops)

const config = {
  mongodb: {
    url: process.env.MONGODB_URI || "mongodb://localhost:27017",
    databaseName: process.env.MONGODB_DATABASE || "cops",
    options: {
      // Connection options (mongodb driver 4.x+ does not require useNewUrlParser/useUnifiedTopology)
    }
  },

  // The migrations dir, can be a relative or absolute path
  migrationsDir: "migrations",

  // The mongodb collection where the applied changes are stored
  changelogCollectionName: "changelog",

  // The mongodb collection where the lock will be created
  lockCollectionName: "changelog_lock",

  // The value in seconds for the TTL index that will be used for the lock
  // Value of 0 will disable the feature
  lockTtl: 0,

  // The file extension to create migrations and search for in migration dir
  migrationFileExtension: ".js",

  // Enable the algorithm to create a checksum of the file contents
  useFileHash: false,

  // Module system to use
  moduleSystem: 'commonjs',
};

module.exports = config;
```

## Step 6: Create Initial Migration Script for All Indexes

**Files to Read**:
- `/Users/jayce/team-attention/cops/TODO.md`: Contains index requirements for users, organizations, organization_members, and projects collections
- `/Users/jayce/team-attention/cops/doc/mongodb-indexes.md`: Contains index requirements for deviceCodes collection

### `/Users/jayce/team-attention/cops/ops/db/mongodb/migrations/20260104000000-create-initial-indexes.js`

**Description**: Create the first migration script that creates all required indexes for the C-Ops system. The script includes both `up` (create indexes) and `down` (drop indexes) functions. Index names follow MongoDB's default naming convention (`field1_1_field2_1` for compound indexes).

```javascript
/**
 * Initial migration to create all required indexes for C-Ops system.
 *
 * Collections and indexes:
 * - users: unique email, compound accounts.provider + accounts.providerId
 * - organizations: unique slug
 * - organization_members: unique compound organizationId + userId, userId
 * - projects: organizationId
 * - deviceCodes: TTL on expiresAt, unique userCode
 */
module.exports = {
  /**
   * Creates all required indexes for the C-Ops system.
   * @param {import('mongodb').Db} db - MongoDB database instance
   * @param {import('mongodb').MongoClient} client - MongoDB client instance
   * @returns {Promise<void>}
   */
  async up(db, client) {
    // Implementation outline:
    // 1. Create indexes for users collection
    //    a. Create unique index on email field
    //       - db.collection('users').createIndex({ email: 1 }, { unique: true })
    //    b. Create compound index on accounts.provider and accounts.providerId
    //       - db.collection('users').createIndex({ 'accounts.provider': 1, 'accounts.providerId': 1 })
    //
    // 2. Create indexes for organizations collection
    //    a. Create unique index on slug field
    //       - db.collection('organizations').createIndex({ slug: 1 }, { unique: true })
    //
    // 3. Create indexes for organization_members collection
    //    a. Create unique compound index on organizationId and userId
    //       - db.collection('organization_members').createIndex({ organizationId: 1, userId: 1 }, { unique: true })
    //    b. Create index on userId field
    //       - db.collection('organization_members').createIndex({ userId: 1 })
    //
    // 4. Create indexes for projects collection
    //    a. Create index on organizationId field
    //       - db.collection('projects').createIndex({ organizationId: 1 })
    //
    // 5. Create indexes for deviceCodes collection
    //    a. Create TTL index on expiresAt field with expireAfterSeconds: 0
    //       - db.collection('deviceCodes').createIndex({ expiresAt: 1 }, { expireAfterSeconds: 0 })
    //    b. Create unique index on userCode field
    //       - db.collection('deviceCodes').createIndex({ userCode: 1 }, { unique: true })
  },

  /**
   * Drops all indexes created by the up migration.
   * @param {import('mongodb').Db} db - MongoDB database instance
   * @param {import('mongodb').MongoClient} client - MongoDB client instance
   * @returns {Promise<void>}
   */
  async down(db, client) {
    // Implementation outline:
    // 1. Drop indexes from deviceCodes collection (reverse order)
    //    a. Drop index userCode_1
    //       - db.collection('deviceCodes').dropIndex('userCode_1')
    //    b. Drop index expiresAt_1
    //       - db.collection('deviceCodes').dropIndex('expiresAt_1')
    //
    // 2. Drop indexes from projects collection
    //    a. Drop index organizationId_1
    //       - db.collection('projects').dropIndex('organizationId_1')
    //
    // 3. Drop indexes from organization_members collection
    //    a. Drop index userId_1
    //       - db.collection('organization_members').dropIndex('userId_1')
    //    b. Drop index organizationId_1_userId_1
    //       - db.collection('organization_members').dropIndex('organizationId_1_userId_1')
    //
    // 4. Drop indexes from organizations collection
    //    a. Drop index slug_1
    //       - db.collection('organizations').dropIndex('slug_1')
    //
    // 5. Drop indexes from users collection
    //    a. Drop index accounts.provider_1_accounts.providerId_1
    //       - db.collection('users').dropIndex('accounts.provider_1_accounts.providerId_1')
    //    b. Drop index email_1
    //       - db.collection('users').dropIndex('email_1')
  }
};
```

**Index Specifications**:

| Collection | Index Name | Fields | Options |
| :--------- | :--------- | :----- | :------ |
| users | email_1 | `{ email: 1 }` | `{ unique: true }` |
| users | accounts.provider_1_accounts.providerId_1 | `{ "accounts.provider": 1, "accounts.providerId": 1 }` | None |
| organizations | slug_1 | `{ slug: 1 }` | `{ unique: true }` |
| organization_members | organizationId_1_userId_1 | `{ organizationId: 1, userId: 1 }` | `{ unique: true }` |
| organization_members | userId_1 | `{ userId: 1 }` | None |
| projects | organizationId_1 | `{ organizationId: 1 }` | None |
| deviceCodes | expiresAt_1 | `{ expiresAt: 1 }` | `{ expireAfterSeconds: 0 }` |
| deviceCodes | userCode_1 | `{ userCode: 1 }` | `{ unique: true }` |

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Fresh database (up) | Empty database | All 8 indexes created successfully | Happy path - up |
| Migration status after up | Database with indexes | Shows migration as applied in changelog | Changelog tracking |
| Rollback (down) | Database with indexes | All 8 indexes dropped successfully | Happy path - down |
| Re-run up after down | Database without indexes | All 8 indexes created again | Idempotency verification |
| Partial failure (up) | Invalid index spec | Migration fails, no partial state | Error handling |

## Step 7: Update Project .gitignore

**Files to Read**:
- `/Users/jayce/team-attention/cops/.gitignore`: Check existing gitignore patterns

### `/Users/jayce/team-attention/cops/.gitignore`

**Description**: Add node_modules exclusion for the ops directory to prevent committing dependencies.

Append the following lines to the existing .gitignore file:

```gitignore
# Node.js dependencies in ops directory
ops/**/node_modules/
```

## Step 8: Delete Completed Documentation Files

**Files to Read**: None required.

**Description**: Delete the documentation files that contained the index migration requirements, as these requirements are now implemented in the migration script.

**Commands to execute**:
```bash
rm /Users/jayce/team-attention/cops/TODO.md
rm /Users/jayce/team-attention/cops/doc/mongodb-indexes.md
```

**Files to delete**:
- `/Users/jayce/team-attention/cops/TODO.md` - Index migration requirements now implemented
- `/Users/jayce/team-attention/cops/doc/mongodb-indexes.md` - deviceCodes indexes now in migration script

## Execution Order Summary

| Step | Action | Output |
| :--- | :----- | :----- |
| 1 | Create directory structure | `ops/db/mongodb/migrations/` and `ops/db/mongodb/.meta/` created |
| 2 | Create package.json | `ops/db/mongodb/package.json` created |
| 3 | Install npm dependencies | `node_modules/` and `package-lock.json` created |
| 4 | Create .env.example | `ops/db/mongodb/.meta/.env.example` created |
| 5 | Create .meta/.gitignore | `ops/db/mongodb/.meta/.gitignore` created |
| 6 | Create Makefile | `ops/db/mongodb/Makefile` created |
| 7 | Create migrate-mongo config | `ops/db/mongodb/migrate-mongo-config.js` created |
| 8 | Create initial migration | `ops/db/mongodb/migrations/20260104000000-create-initial-indexes.js` created |
| 9 | Update .gitignore | `ops/**/node_modules/` pattern added |
| 10 | Delete TODO.md | `/Users/jayce/team-attention/cops/TODO.md` deleted |
| 11 | Delete mongodb-indexes.md | `/Users/jayce/team-attention/cops/doc/mongodb-indexes.md` deleted |

## Final Directory Structure

```
cops/
  .gitignore                    (updated with ops/**/node_modules/)
  ops/
    db/
      mongodb/
        .meta/
          .env.example
          .gitignore
        migrations/
          20260104000000-create-initial-indexes.js
        Makefile
        migrate-mongo-config.js
        package.json
        package-lock.json
        node_modules/           (gitignored)
```

## Quality Checklist

- [x] Every function has a concrete signature (not "something like X")
- [x] Detailed algorithm explanation included as comments in function bodies
- [x] Every function has test scenarios covering all branches
- [x] No "or" statements leaving choices to Implementation Agent
- [x] All packages are selected (migrate-mongo latest)
- [x] Execution order is clear and dependencies are explicit
