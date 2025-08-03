// MongoDB initialization script
// This script creates separate databases for each microservice

// Switch to admin database for authentication
db = db.getSiblingDB('admin');

// Create databases for each service
const databases = [
  'gateway',
  'auth', 
  'users',
  'notifications'
];

databases.forEach(dbName => {
  const serviceDb = db.getSiblingDB(dbName);
  
  // Create a dummy collection to initialize the database
  serviceDb.createCollection('_init');
  
  // Create indexes or initial data as needed
  if (dbName === 'auth') {
    serviceDb.users.createIndex({ email: 1 }, { unique: true });
    serviceDb.sessions.createIndex({ expiresAt: 1 }, { expireAfterSeconds: 0 });
  }
  
  if (dbName === 'users') {
    serviceDb.profiles.createIndex({ userId: 1 }, { unique: true });
  }
  
  if (dbName === 'notifications') {
    serviceDb.notifications.createIndex({ userId: 1, createdAt: -1 });
  }
  
  print(`Database '${dbName}' initialized successfully`);
});

print('MongoDB initialization completed');