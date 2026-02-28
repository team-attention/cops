/**
 * Add users to the ralphton organization as members.
 *
 * Finds users by email and adds them to organization 69a2cf5ff5ed45b819e3514d.
 * Skips users that are already members or don't exist yet.
 */
const { ObjectId } = require("mongodb");

const ORG_ID = new ObjectId("69a2cf5ff5ed45b819e3514d");

const MEMBER_EMAILS = [
  "changhoi0522@gmail.com",
  "vkehfdl1@gmail.com",
  "wyverselabs@gmail.com",
  "0xd669@gmail.com",
  "hurrc04@gmail.com",
  "subinium@gmail.com",
  "jeongmin1604@gmail.com",
  "kubony@gmail.com",
  "jqyu.lee@gmail.com",
  "ajunh7@gmail.com",
];

module.exports = {
  async up(db) {
    const users = await db
      .collection("users")
      .find({ email: { $in: MEMBER_EMAILS } })
      .toArray();

    if (users.length === 0) {
      console.log("No matching users found. Skipping.");
      return;
    }

    const org = await db
      .collection("organizations")
      .findOne({ _id: ORG_ID });

    if (!org) {
      throw new Error(`Organization ${ORG_ID} not found`);
    }

    const existingMemberIds = new Set(
      (org.members || []).map((m) => m.userId.toString())
    );

    const newMembers = users
      .filter((u) => !existingMemberIds.has(u._id.toString()))
      .map((u) => ({
        userId: u._id,
        role: "member",
      }));

    if (newMembers.length === 0) {
      console.log("All users are already members. Skipping.");
      return;
    }

    await db.collection("organizations").updateOne(
      { _id: ORG_ID },
      { $push: { members: { $each: newMembers } } }
    );

    console.log(
      `Added ${newMembers.length} members:`,
      users
        .filter((u) => !existingMemberIds.has(u._id.toString()))
        .map((u) => u.email)
    );
  },

  async down(db) {
    const users = await db
      .collection("users")
      .find({ email: { $in: MEMBER_EMAILS } })
      .toArray();

    const userIds = users.map((u) => u._id);

    await db.collection("organizations").updateOne(
      { _id: ORG_ID },
      { $pull: { members: { userId: { $in: userIds } } } }
    );

    console.log(`Removed ${userIds.length} members from organization`);
  },
};
