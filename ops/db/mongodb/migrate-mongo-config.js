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
