// MongoDB Database Optimization Script
// Run with: docker exec go-falcon-mongodb mongosh -u admin -p password123 --authenticationDatabase admin falcon /path/to/optimize_database.js

// Switch to falcon database
db = db.getSiblingDB('falcon');

print("🔧 Go Falcon Database Optimization Script");
print("=========================================\n");

// 1. Verify indexes exist and are optimal
print("1. Checking database indexes...");

const collections = ['user_profiles', 'casbin_policies', 'permission_hierarchies', 'auth_states', 'scheduler_tasks', 'scheduler_executions'];

collections.forEach(collName => {
    if (db.getCollectionNames().includes(collName)) {
        print(`\n📊 ${collName} collection:`);
        print(`   Documents: ${db[collName].countDocuments()}`);
        print(`   Indexes: ${db[collName].getIndexes().length}`);
        
        // Show index details
        db[collName].getIndexes().forEach(index => {
            const sizeBytes = db[collName].totalIndexSize();
            print(`   - ${index.name}: ${JSON.stringify(index.key)}`);
        });
    }
});

// 2. Clean up expired authentication states
print("\n2. Cleaning up expired authentication states...");
const expiredStates = db.auth_states.deleteMany({
    "expires_at": { "$lt": new Date() }
});
print(`   Removed ${expiredStates.deletedCount} expired auth states`);

// 3. Analyze CASBIN policies structure
print("\n3. Analyzing CASBIN policies...");
const casbinStats = db.casbin_policies.aggregate([
    { "$group": { "_id": "$ptype", "count": { "$sum": 1 } } },
    { "$sort": { "_id": 1 } }
]).toArray();

casbinStats.forEach(stat => {
    const type = stat._id === 'p' ? 'Policies' : 'Role Assignments';
    print(`   ${type}: ${stat.count}`);
});

// 4. Check for orphaned permission hierarchies
print("\n4. Checking permission hierarchies integrity...");
const hierarchyCount = db.permission_hierarchies.countDocuments();
const userProfileCount = db.user_profiles.countDocuments();
print(`   Permission hierarchies: ${hierarchyCount}`);
print(`   User profiles: ${userProfileCount}`);

// 5. Identify invalid user profiles
print("\n5. Analyzing user profile validity...");
const profileStats = db.user_profiles.aggregate([
    { "$group": { 
        "_id": "$valid", 
        "count": { "$sum": 1 },
        "avgTokenExpiry": { "$avg": { "$subtract": ["$token_expiry", new Date()] } }
    } },
    { "$sort": { "_id": 1 } }
]).toArray();

profileStats.forEach(stat => {
    const status = stat._id ? 'Valid' : 'Invalid';
    const avgDays = stat.avgTokenExpiry ? Math.round(stat.avgTokenExpiry / (1000 * 60 * 60 * 24)) : 'N/A';
    print(`   ${status} profiles: ${stat.count} (avg token expires in ${avgDays} days)`);
});

// 6. Check for duplicate character IDs (should not exist)
print("\n6. Checking for duplicate character IDs...");
const duplicates = db.user_profiles.aggregate([
    { "$group": { 
        "_id": "$character_id", 
        "count": { "$sum": 1 },
        "user_ids": { "$addToSet": "$user_id" }
    } },
    { "$match": { "count": { "$gt": 1 } } }
]).toArray();

if (duplicates.length > 0) {
    print(`   ⚠️  Found ${duplicates.length} duplicate character IDs:`);
    duplicates.forEach(dup => {
        print(`   - Character ${dup._id}: ${dup.count} profiles, users: ${JSON.stringify(dup.user_ids)}`);
    });
} else {
    print("   ✅ No duplicate character IDs found");
}

// 7. Performance recommendations
print("\n7. Performance Recommendations:");

// Check if compound indexes could be beneficial
const expiresSoon = db.user_profiles.countDocuments({
    "valid": true,
    "token_expiry": { "$lt": new Date(Date.now() + 24 * 60 * 60 * 1000) }
});
print(`   - Tokens expiring in 24h: ${expiresSoon}`);

// Check auth state cleanup frequency
const oldStates = db.auth_states.countDocuments({
    "expires_at": { "$lt": new Date(Date.now() - 60 * 60 * 1000) }
});
if (oldStates > 100) {
    print(`   ⚠️  Consider more frequent auth state cleanup (${oldStates} old states)`);
}

// 8. Storage usage analysis
print("\n8. Storage Analysis:");
db.runCommand({ "collStats": "user_profiles" }).then ? 
    print("   Using async stats...") : 
    print(`   user_profiles size: ${Math.round(db.user_profiles.dataSize() / 1024)} KB`);

print("\n✅ Database optimization analysis complete!");
print("\nNext steps:");
print("1. Monitor query performance with db.setProfilingLevel(2)");
print("2. Consider TTL indexes for auth_states collection");
print("3. Implement Redis caching for frequently accessed user profiles");
print("4. Set up regular cleanup jobs for expired data");