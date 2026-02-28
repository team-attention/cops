/**
 * Add users to the ralphton organization as members.
 *
 * Usage:
 *   mongosh "mongodb+srv://cops-api:iM6J8egL1DwkfJtF@production.xxcbpax.mongodb.net/cops?appName=cops-api" scripts/add-members-to-ralphton.js
 */

const ORG_ID = ObjectId("69a2cf5ff5ed45b819e3514d");

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

const org = db.organizations.findOne({ _id: ORG_ID });
if (!org) {
  print("ERROR: Organization not found");
  quit(1);
}

print(`Organization: ${org.name} (${org.slug})`);

const existingMemberIds = new Set(
  (org.members || []).map((m) => m.userId.toString())
);

const users = db.users.find({ email: { $in: MEMBER_EMAILS } }).toArray();
print(`Found ${users.length}/${MEMBER_EMAILS.length} users in DB`);

const notFound = MEMBER_EMAILS.filter(
  (e) => !users.some((u) => u.email === e)
);
if (notFound.length > 0) {
  print(`Not found (not yet registered): ${notFound.join(", ")}`);
}

const newMembers = users
  .filter((u) => !existingMemberIds.has(u._id.toString()))
  .map((u) => ({ userId: u._id, role: "member" }));

const alreadyMembers = users.filter((u) =>
  existingMemberIds.has(u._id.toString())
);
if (alreadyMembers.length > 0) {
  print(
    `Already members: ${alreadyMembers.map((u) => u.email).join(", ")}`
  );
}

if (newMembers.length === 0) {
  print("No new members to add.");
  quit(0);
}

db.organizations.updateOne(
  { _id: ORG_ID },
  { $push: { members: { $each: newMembers } } }
);

print(`Added ${newMembers.length} members:`);
users
  .filter((u) => !existingMemberIds.has(u._id.toString()))
  .forEach((u) => print(`  - ${u.email} (${u.name || "no name"})`));
