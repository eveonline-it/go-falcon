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

type CharStatsRepository struct {
	db         *database.MongoDB
	collection *mongo.Collection
}

func NewCharStatsRepository(db *database.MongoDB) *CharStatsRepository {
	return &CharStatsRepository{
		db:         db,
		collection: db.Database.Collection(models.KillmailsCharStatsCollection),
	}
}

// GetCharacterStats retrieves character stats by character ID
func (r *CharStatsRepository) GetCharacterStats(ctx context.Context, characterID int32) (*models.CharacterKillmailStats, error) {
	var stats models.CharacterKillmailStats
	err := r.collection.FindOne(ctx, bson.M{"character_id": characterID}).Decode(&stats)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &stats, nil
}

// UpsertCharacterStats inserts or updates character stats
func (r *CharStatsRepository) UpsertCharacterStats(ctx context.Context, stats *models.CharacterKillmailStats) error {
	filter := bson.M{"character_id": stats.CharacterID}
	update := bson.M{
		"$set": bson.M{
			"notable_ships": stats.NotableShips,
			"last_updated":  stats.LastUpdated,
		},
		"$setOnInsert": bson.M{
			"_id": primitive.NewObjectID(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// UpdateLastShipUsed updates only the last ship used for a specific category
func (r *CharStatsRepository) UpdateLastShipUsed(ctx context.Context, characterID int32, category string, shipTypeID int64) error {
	filter := bson.M{"character_id": characterID}

	// Prepare the update operations
	update := bson.M{
		"$set": bson.M{
			"last_updated":              time.Now(),
			"notable_ships." + category: shipTypeID,
		},
		"$setOnInsert": bson.M{
			"_id": primitive.NewObjectID(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetCharactersByShipCategory returns characters who have used ships in a specific category
func (r *CharStatsRepository) GetCharactersByShipCategory(ctx context.Context, category string, limit int) ([]*models.CharacterKillmailStats, error) {
	filter := bson.M{
		"notable_ships." + category: bson.M{"$exists": true},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "notable_ships." + category + ".killmail_time", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var stats []*models.CharacterKillmailStats
	if err := cursor.All(ctx, &stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetCharactersByShipType returns characters who last used a specific ship type in any category
func (r *CharStatsRepository) GetCharactersByShipType(ctx context.Context, shipTypeID int64, limit int) ([]*models.CharacterKillmailStats, error) {
	// Build OR query for all possible categories
	categories := []string{
		"interdictor", "forcerecon", "strategic", "hic", "monitor",
		"blackops", "marauders", "fax", "dread", "carrier", "super", "titan", "lancer",
	}

	orQueries := make([]bson.M, len(categories))
	for i, category := range categories {
		orQueries[i] = bson.M{"notable_ships." + category: shipTypeID}
	}

	filter := bson.M{"$or": orQueries}

	opts := options.Find().
		SetSort(bson.D{{Key: "last_updated", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var stats []*models.CharacterKillmailStats
	if err := cursor.All(ctx, &stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetRecentCharacterActivity returns characters with recent activity
func (r *CharStatsRepository) GetRecentCharacterActivity(ctx context.Context, since time.Time, limit int) ([]*models.CharacterKillmailStats, error) {
	filter := bson.M{
		"last_updated": bson.M{"$gte": since},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "last_updated", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var stats []*models.CharacterKillmailStats
	if err := cursor.All(ctx, &stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetCharacterLastShipByCategory gets the last ship used by a character in a specific category
func (r *CharStatsRepository) GetCharacterLastShipByCategory(ctx context.Context, characterID int32, category string) (*int64, error) {
	projection := bson.M{
		"notable_ships." + category: 1,
	}

	opts := options.FindOne().SetProjection(projection)

	var result struct {
		NotableShips map[string]int64 `bson:"notable_ships"`
	}

	err := r.collection.FindOne(ctx, bson.M{"character_id": characterID}, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	if result.NotableShips != nil {
		if shipTypeID, exists := result.NotableShips[category]; exists {
			return &shipTypeID, nil
		}
	}

	return nil, nil
}

// CountCharactersWithStats returns total count of characters with stats
func (r *CharStatsRepository) CountCharactersWithStats(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{})
}

// CountCharactersByCategory returns count of characters who have used ships in each category
func (r *CharStatsRepository) CountCharactersByCategory(ctx context.Context) (map[string]int64, error) {
	// Categories to check
	categories := []string{
		"interdictor", "forcerecon", "strategic", "hic", "monitor",
		"blackops", "marauders", "fax", "dread", "carrier", "super", "titan", "lancer",
	}

	counts := make(map[string]int64)

	for _, category := range categories {
		filter := bson.M{
			"notable_ships." + category: bson.M{"$exists": true},
		}

		count, err := r.collection.CountDocuments(ctx, filter)
		if err != nil {
			return nil, err
		}

		counts[category] = count
	}

	return counts, nil
}

// CreateIndexes creates necessary indexes for the character stats collection
func (r *CharStatsRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		// Primary character lookup
		{
			Keys:    bson.D{{Key: "character_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		// Last updated for maintenance queries
		{
			Keys: bson.D{{Key: "last_updated", Value: -1}},
		},
		// Compound indexes for ship category queries (simplified structure)
		{
			Keys: bson.D{
				{Key: "notable_ships.interdictor", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "notable_ships.titan", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "notable_ships.super", Value: 1},
			},
		},
		// Add more category-specific indexes as needed for performance
		{
			Keys: bson.D{
				{Key: "notable_ships.dread", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "notable_ships.carrier", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "notable_ships.fax", Value: 1},
			},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
