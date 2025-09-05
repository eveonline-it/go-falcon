package services

import (
	"context"
	"time"

	"go-falcon/internal/killmails/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct {
	db         *database.MongoDB
	collection *mongo.Collection
}

func NewRepository(db *database.MongoDB) *Repository {
	return &Repository{
		db:         db,
		collection: db.Database.Collection(models.KillmailsCollection),
	}
}

// GetByKillmailID retrieves a killmail by its killmail ID
func (r *Repository) GetByKillmailID(ctx context.Context, killmailID int64) (*models.Killmail, error) {
	var killmail models.Killmail
	err := r.collection.FindOne(ctx, bson.M{"killmail_id": killmailID}).Decode(&killmail)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &killmail, nil
}

// GetByKillmailIDAndHash retrieves a killmail by ID and hash
func (r *Repository) GetByKillmailIDAndHash(ctx context.Context, killmailID int64, hash string) (*models.Killmail, error) {
	var killmail models.Killmail
	err := r.collection.FindOne(ctx, bson.M{
		"killmail_id":   killmailID,
		"killmail_hash": hash,
	}).Decode(&killmail)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &killmail, nil
}

// UpsertKillmail inserts or updates a killmail
func (r *Repository) UpsertKillmail(ctx context.Context, killmail *models.Killmail) error {
	// Use upsert to insert or update
	filter := bson.M{"killmail_id": killmail.KillmailID}
	update := bson.M{
		"$set": bson.M{
			"killmail_hash":   killmail.KillmailHash,
			"killmail_time":   killmail.KillmailTime,
			"solar_system_id": killmail.SolarSystemID,
			"moon_id":         killmail.MoonID,
			"war_id":          killmail.WarID,
			"victim":          killmail.Victim,
			"attackers":       killmail.Attackers,
		},
		"$setOnInsert": bson.M{
			"_id": primitive.NewObjectID(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetRecentKillmailsByCharacter gets recent killmails for a character
func (r *Repository) GetRecentKillmailsByCharacter(ctx context.Context, characterID int64, limit int) ([]models.Killmail, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"victim.character_id": characterID},
			{"attackers.character_id": characterID},
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "killmail_time", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var killmails []models.Killmail
	if err := cursor.All(ctx, &killmails); err != nil {
		return nil, err
	}

	return killmails, nil
}

// GetRecentKillmailsByCorporation gets recent killmails for a corporation
func (r *Repository) GetRecentKillmailsByCorporation(ctx context.Context, corporationID int64, limit int) ([]models.Killmail, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"victim.corporation_id": corporationID},
			{"attackers.corporation_id": corporationID},
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "killmail_time", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var killmails []models.Killmail
	if err := cursor.All(ctx, &killmails); err != nil {
		return nil, err
	}

	return killmails, nil
}

// GetRecentKillmailsByAlliance gets recent killmails for an alliance
func (r *Repository) GetRecentKillmailsByAlliance(ctx context.Context, allianceID int64, limit int) ([]models.Killmail, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"victim.alliance_id": allianceID},
			{"attackers.alliance_id": allianceID},
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "killmail_time", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var killmails []models.Killmail
	if err := cursor.All(ctx, &killmails); err != nil {
		return nil, err
	}

	return killmails, nil
}

// GetKillmailsBySystem gets killmails in a specific solar system
func (r *Repository) GetKillmailsBySystem(ctx context.Context, systemID int64, since time.Time, limit int) ([]models.Killmail, error) {
	filter := bson.M{
		"solar_system_id": systemID,
		"killmail_time":   bson.M{"$gte": since},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "killmail_time", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var killmails []models.Killmail
	if err := cursor.All(ctx, &killmails); err != nil {
		return nil, err
	}

	return killmails, nil
}

// CountKillmails counts total killmails in the collection
func (r *Repository) CountKillmails(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{})
}

// Exists checks if a killmail exists by ID and optionally hash
func (r *Repository) Exists(ctx context.Context, killmailID int64, hash string) (bool, error) {
	filter := bson.M{"killmail_id": killmailID}
	if hash != "" {
		filter["killmail_hash"] = hash
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CreateMany inserts multiple killmails in a batch operation
func (r *Repository) CreateMany(ctx context.Context, killmails []*models.Killmail) error {
	if len(killmails) == 0 {
		return nil
	}

	// Prepare documents for insertion
	documents := make([]interface{}, len(killmails))
	for i, killmail := range killmails {
		documents[i] = killmail
	}

	// Use ordered=false for better performance (continues on errors)
	opts := options.InsertMany().SetOrdered(false)
	_, err := r.collection.InsertMany(ctx, documents, opts)
	return err
}

// CreateIndexes creates necessary indexes for the killmails collection
func (r *Repository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "killmail_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "killmail_id", Value: 1},
				{Key: "killmail_hash", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "killmail_time", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "solar_system_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "victim.character_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "victim.corporation_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "victim.alliance_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "attackers.character_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "attackers.corporation_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "attackers.alliance_id", Value: 1}},
		},
		// Compound indexes for common query patterns
		{
			Keys: bson.D{
				{Key: "solar_system_id", Value: 1},
				{Key: "killmail_time", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "victim.character_id", Value: 1},
				{Key: "killmail_time", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "attackers.character_id", Value: 1},
				{Key: "killmail_time", Value: -1},
			},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
