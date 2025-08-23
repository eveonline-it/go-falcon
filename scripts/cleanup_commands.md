# Manual Cleanup Commands for Old Groups

## Option 1: Direct MongoDB Commands

Connect to your MongoDB and run these commands:

```javascript
// Connect to your MongoDB
use falcon  // or your database name

// Find the incorrectly named groups
db.groups.find({
  "$or": [
    {"name": /^Corp_[0-9]+$/},
    {"name": /^Alliance_[0-9]+$/}
  ]
})

// Delete memberships for these groups first
db.group_memberships.deleteMany({
  "group_id": {
    "$in": [
      ObjectId("68a972161a26917cc3af94f8"),  // Corp_98703632
      ObjectId("68a972161a26917cc3af94f9")   // Alliance_99003916
    ]
  }
})

// Delete the groups themselves
db.groups.deleteMany({
  "_id": {
    "$in": [
      ObjectId("68a972161a26917cc3af94f8"),  // Corp_98703632
      ObjectId("68a972161a26917cc3af94f9")   // Alliance_99003916
    ]
  }
})
```

## Option 2: Using mongosh Command Line

```bash
# Using mongosh (if you have it installed)
mongosh "mongodb://admin:password@localhost:27017/falcon?authSource=admin" --eval '
  db.group_memberships.deleteMany({
    "group_id": {
      "$in": [
        ObjectId("68a972161a26917cc3af94f8"),
        ObjectId("68a972161a26917cc3af94f9")
      ]
    }
  });
  
  db.groups.deleteMany({
    "_id": {
      "$in": [
        ObjectId("68a972161a26917cc3af94f8"),
        ObjectId("68a972161a26917cc3af94f9")
      ]
    }
  });
'
```

## Option 3: Via Groups API (Requires Authentication)

First get a super admin token, then:

```bash
# Delete corporation group
curl -X DELETE "http://localhost:3000/groups/68a972161a26917cc3af94f8" \
  -H "Authorization: Bearer <your_token>"

# Delete alliance group  
curl -X DELETE "http://localhost:3000/groups/68a972161a26917cc3af94f9" \
  -H "Authorization: Bearer <your_token>"
```

## Verification

After cleanup, verify with:

```bash
curl -s http://localhost:3000/groups | jq '.groups[] | select(.name | test("^(Corp_|Alliance_)[0-9]+$"))'
```

This should return empty (no results) if cleanup was successful.