/**
 * Deduplicate users with the same email.
 *
 * For each duplicated email:
 * 1. Keep the earliest created user (smallest _id)
 * 2. Update all references in other collections to point to the kept user
 * 3. Delete the duplicate user documents
 *
 * Usage:
 *   mongosh "mongodb+srv://..." scripts/dedup-users.js
 */

// Find duplicate emails
const dupes = db.users.aggregate([
  { $group: { _id: "$email", count: { $sum: 1 }, ids: { $push: "$_id" } } },
  { $match: { count: { $gt: 1 } } },
]).toArray();

if (dupes.length === 0) {
  print("No duplicate users found.");
  quit(0);
}

print(`Found ${dupes.length} duplicate email(s):\n`);

let totalRemoved = 0;

for (const dupe of dupes) {
  const email = dupe._id;
  const sortedIds = dupe.ids.sort(); // smallest ObjectId = earliest created
  const keepId = sortedIds[0];
  const removeIds = sortedIds.slice(1);

  print(`${email}: keeping ${keepId}, removing ${removeIds.join(", ")}`);

  // Update references in all collections that have userId
  for (const removeId of removeIds) {
    // organizations.members[].userId
    const orgResult = db.organizations.updateMany(
      { "members.userId": removeId },
      { $set: { "members.$[elem].userId": keepId } },
      { arrayFilters: [{ "elem.userId": removeId }] }
    );
    if (orgResult.modifiedCount > 0) {
      print(`  - organizations: updated ${orgResult.modifiedCount} doc(s)`);
    }

    // apiKeys.userId
    const apiResult = db.apiKeys.updateMany(
      { userId: removeId },
      { $set: { userId: keepId } }
    );
    if (apiResult.modifiedCount > 0) {
      print(`  - apiKeys: updated ${apiResult.modifiedCount} doc(s)`);
    }

    // events.userId
    const eventResult = db.events.updateMany(
      { userId: removeId },
      { $set: { userId: keepId } }
    );
    if (eventResult.modifiedCount > 0) {
      print(`  - events: updated ${eventResult.modifiedCount} doc(s)`);
    }

    // sessions.userId
    const sessionResult = db.sessions.updateMany(
      { userId: removeId },
      { $set: { userId: keepId } }
    );
    if (sessionResult.modifiedCount > 0) {
      print(`  - sessions: updated ${sessionResult.modifiedCount} doc(s)`);
    }

    // deviceCodes.userId
    const dcResult = db.deviceCodes.updateMany(
      { userId: removeId },
      { $set: { userId: keepId } }
    );
    if (dcResult.modifiedCount > 0) {
      print(`  - deviceCodes: updated ${dcResult.modifiedCount} doc(s)`);
    }
  }

  // Remove duplicate org members (same userId after update)
  db.organizations.updateMany(
    { "members.userId": keepId },
    [
      {
        $set: {
          members: {
            $reduce: {
              input: "$members",
              initialValue: [],
              in: {
                $cond: [
                  {
                    $and: [
                      { $eq: ["$$this.userId", keepId] },
                      { $gt: [{ $size: { $filter: { input: "$$value", as: "v", cond: { $eq: ["$$v.userId", keepId] } } } }, 0] },
                    ],
                  },
                  "$$value",
                  { $concatArrays: ["$$value", ["$$this"]] },
                ],
              },
            },
          },
        },
      },
    ]
  );

  // Delete duplicate users
  const delResult = db.users.deleteMany({ _id: { $in: removeIds } });
  print(`  - users: deleted ${delResult.deletedCount} duplicate(s)\n`);
  totalRemoved += delResult.deletedCount;
}

print(`Done. Removed ${totalRemoved} duplicate user(s) total.`);
