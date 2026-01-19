/**
 * Initial migration to create all required indexes for C-Ops system.
 *
 * Collections and indexes:
 * - users: unique email, compound accounts.providerId + accounts.provider
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
    // 1. Create indexes for users collection
    await db.collection('users').createIndex({ email: 1 }, { unique: true });
    await db.collection('users').createIndex({ 'accounts.providerId': 1, 'accounts.provider': 1 });

    // 2. Create indexes for organizations collection
    await db.collection('organizations').createIndex({ slug: 1 }, { unique: true });

    // 3. Create indexes for organization_members collection
    await db.collection('organization_members').createIndex({ organizationId: 1, userId: 1 }, { unique: true });
    await db.collection('organization_members').createIndex({ userId: 1 });

    // 4. Create indexes for projects collection
    await db.collection('projects').createIndex({ organizationId: 1 });

    // 5. Create indexes for deviceCodes collection
    await db.collection('deviceCodes').createIndex({ expiresAt: 1 }, { expireAfterSeconds: 0 });
    await db.collection('deviceCodes').createIndex({ userCode: 1 }, { unique: true });
  },

  /**
   * Drops all indexes created by the up migration.
   * @param {import('mongodb').Db} db - MongoDB database instance
   * @param {import('mongodb').MongoClient} client - MongoDB client instance
   * @returns {Promise<void>}
   */
  async down(db, client) {
    // 1. Drop indexes from deviceCodes collection (reverse order)
    await db.collection('deviceCodes').dropIndex('userCode_1');
    await db.collection('deviceCodes').dropIndex('expiresAt_1');

    // 2. Drop indexes from projects collection
    await db.collection('projects').dropIndex('organizationId_1');

    // 3. Drop indexes from organization_members collection
    await db.collection('organization_members').dropIndex('userId_1');
    await db.collection('organization_members').dropIndex('organizationId_1_userId_1');

    // 4. Drop indexes from organizations collection
    await db.collection('organizations').dropIndex('slug_1');

    // 5. Drop indexes from users collection
    await db.collection('users').dropIndex('accounts.providerId_1_accounts.provider_1');
    await db.collection('users').dropIndex('email_1');
  }
};
