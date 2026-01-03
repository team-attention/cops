# MongoDB Indexes

This document lists the MongoDB indexes that need to be created for the C-Ops system.

## Device Codes Collection

**Collection name:** `deviceCodes`

### TTL Index for Automatic Expiration

Device codes should automatically expire after their `expiresAt` time. This index enables MongoDB to automatically delete expired documents.

```javascript
db.deviceCodes.createIndex(
  { "expiresAt": 1 },
  { expireAfterSeconds: 0 }
);
```

**Important:** The `expireAfterSeconds: 0` means documents are deleted immediately when the `expiresAt` time is reached.

### Unique Index on User Code

Each user code must be unique to prevent conflicts during the device approval flow.

```javascript
db.deviceCodes.createIndex(
  { "userCode": 1 },
  { unique: true }
);
```

## Notes

- These indexes should be created during database setup or migration
- The TTL index is critical for preventing accumulation of expired device codes
- MongoDB's TTL monitor runs every 60 seconds, so there may be a slight delay before expired documents are removed
