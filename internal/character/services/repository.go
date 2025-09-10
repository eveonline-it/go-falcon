package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go-falcon/internal/character/dto"
	"go-falcon/internal/character/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles data persistence for characters
type Repository struct {
	mongodb               *database.MongoDB
	collection            *mongo.Collection
	attributesCollection  *mongo.Collection
	skillQueueCollection  *mongo.Collection
	skillsCollection      *mongo.Collection
	corpHistoryCollection *mongo.Collection
	clonesCollection      *mongo.Collection
}

// NewRepository creates a new repository instance
func NewRepository(mongodb *database.MongoDB) *Repository {
	return &Repository{
		mongodb:               mongodb,
		collection:            mongodb.Database.Collection("characters"),
		attributesCollection:  mongodb.Database.Collection("character_attributes"),
		skillQueueCollection:  mongodb.Database.Collection("character_skill_queues"),
		skillsCollection:      mongodb.Database.Collection("character_skills"),
		corpHistoryCollection: mongodb.Database.Collection("character_corporation_history"),
		clonesCollection:      mongodb.Database.Collection("character_clones"),
	}
}

// GetCharacterByID retrieves a character by character ID
func (r *Repository) GetCharacterByID(ctx context.Context, characterID int) (*models.Character, error) {
	filter := bson.M{"character_id": characterID}

	var character models.Character
	err := r.collection.FindOne(ctx, filter).Decode(&character)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &character, nil
}

// SaveCharacter saves or updates a character profile
func (r *Repository) SaveCharacter(ctx context.Context, character *models.Character) error {
	character.UpdatedAt = time.Now()
	if character.CreatedAt.IsZero() {
		character.CreatedAt = time.Now()
	}

	filter := bson.M{"character_id": character.CharacterID}
	update := bson.M{"$set": character}
	opts := options.Update().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// SearchCharactersByName searches characters by name using optimized search strategies
func (r *Repository) SearchCharactersByName(ctx context.Context, name string) ([]*models.Character, error) {
	var filter bson.M
	var findOptions *options.FindOptions

	// Use different search strategies based on the search pattern
	if len(name) >= 3 {
		// For partial matches, try text search first (faster for full-text queries)
		// If the query looks like a text search (multiple words or special characters)
		if strings.Contains(name, " ") || len(strings.Fields(name)) > 1 {
			// Use text search for multi-word queries
			filter = bson.M{
				"$text": bson.M{
					"$search": name,
				},
			}
			// Sort by text score for relevance
			findOptions = options.Find().
				SetProjection(bson.M{"score": bson.M{"$meta": "textScore"}}).
				SetSort(bson.M{"score": bson.M{"$meta": "textScore"}}).
				SetLimit(50) // Limit results for performance
		} else {
			// Use case-insensitive regex for single-word prefix/contains search
			// Optimize with anchored regex for prefix search if possible
			regexPattern := "^" + strings.ToLower(name) // Start with prefix search
			if !strings.HasPrefix(name, "^") {
				regexPattern = strings.ToLower(name) // Contains search
			}

			filter = bson.M{
				"name": bson.M{
					"$regex":   regexPattern,
					"$options": "i", // case-insensitive
				},
			}
			// Sort by name for consistent results and limit
			findOptions = options.Find().
				SetSort(bson.M{"name": 1}).
				SetLimit(50) // Limit results for performance
		}
	} else {
		// For very short queries, use prefix search only
		filter = bson.M{
			"name": bson.M{
				"$regex":   "^" + strings.ToLower(name),
				"$options": "i",
			},
		}
		findOptions = options.Find().
			SetSort(bson.M{"name": 1}).
			SetLimit(20) // Smaller limit for short queries
	}

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var characters []*models.Character
	if err := cursor.All(ctx, &characters); err != nil {
		return nil, err
	}

	return characters, nil
}

// GetAllCharacterIDs retrieves all character IDs from the database
func (r *Repository) GetAllCharacterIDs(ctx context.Context) ([]int, error) {
	// Find all documents, but only return the character_id field
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"character_id": 1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var characterIDs []int
	for cursor.Next(ctx) {
		var doc struct {
			CharacterID int `bson:"character_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		characterIDs = append(characterIDs, doc.CharacterID)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return characterIDs, nil
}

// UpdateCharacterAffiliation updates a character's corporation and alliance affiliations
func (r *Repository) UpdateCharacterAffiliation(ctx context.Context, affiliation *dto.CharacterAffiliation) error {
	filter := bson.M{"character_id": affiliation.CharacterID}

	// Get the existing character to compare changes
	var existingCharacter models.Character
	err := r.collection.FindOne(ctx, filter).Decode(&existingCharacter)
	foundExisting := err == nil

	update := bson.M{
		"$set": bson.M{
			"corporation_id": affiliation.CorporationID,
			"alliance_id":    affiliation.AllianceID,
			"faction_id":     affiliation.FactionID,
			"updated_at":     time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// Debug logging for updates
	if result.MatchedCount > 0 && foundExisting {
		// Check what changed
		var changes []string
		if existingCharacter.CorporationID != affiliation.CorporationID {
			changes = append(changes, fmt.Sprintf("corp: %d→%d", existingCharacter.CorporationID, affiliation.CorporationID))
		}
		if existingCharacter.AllianceID != affiliation.AllianceID {
			changes = append(changes, fmt.Sprintf("alliance: %d→%d", existingCharacter.AllianceID, affiliation.AllianceID))
		}
		if existingCharacter.FactionID != affiliation.FactionID {
			changes = append(changes, fmt.Sprintf("faction: %d→%d", existingCharacter.FactionID, affiliation.FactionID))
		}

		if len(changes) > 0 {
			log.Printf("🔄 Character %d affiliation UPDATED: %s", affiliation.CharacterID, strings.Join(changes, ", "))
		}
	}

	// If no document was found, we might want to create a minimal character record
	if result.MatchedCount == 0 {
		log.Printf("➕ Character %d NOT FOUND in database, creating new record (corp: %d, alliance: %d, faction: %d)",
			affiliation.CharacterID, affiliation.CorporationID, affiliation.AllianceID, affiliation.FactionID)

		// Create a new character with minimal information
		character := &models.Character{
			CharacterID:   affiliation.CharacterID,
			CorporationID: affiliation.CorporationID,
			AllianceID:    affiliation.AllianceID,
			FactionID:     affiliation.FactionID,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		_, err = r.collection.InsertOne(ctx, character)
		if err != nil {
			// Check if it's a duplicate key error (character was created concurrently)
			if mongo.IsDuplicateKeyError(err) {
				log.Printf("⚠️  Character %d was created concurrently, retrying update", affiliation.CharacterID)
				// Try the update again
				_, err = r.collection.UpdateOne(ctx, filter, update)
				return err
			}
			return err
		}
		log.Printf("✅ Character %d successfully created", affiliation.CharacterID)
	}

	return nil
}

// BatchUpdateAffiliations updates multiple character affiliations in a single operation
func (r *Repository) BatchUpdateAffiliations(ctx context.Context, affiliations []*dto.CharacterAffiliation) error {
	if len(affiliations) == 0 {
		return nil
	}

	// Use bulk write for better performance
	models := make([]mongo.WriteModel, 0, len(affiliations))
	now := time.Now()

	for _, aff := range affiliations {
		filter := bson.M{"character_id": aff.CharacterID}
		update := bson.M{
			"$set": bson.M{
				"corporation_id": aff.CorporationID,
				"alliance_id":    aff.AllianceID,
				"faction_id":     aff.FactionID,
				"updated_at":     now,
			},
			"$setOnInsert": bson.M{
				"character_id": aff.CharacterID,
				"created_at":   now,
			},
		}

		model := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true)

		models = append(models, model)
	}

	opts := options.BulkWrite().SetOrdered(false)
	_, err := r.collection.BulkWrite(ctx, models, opts)
	return err
}

// CheckHealth verifies database connectivity
func (r *Repository) CheckHealth(ctx context.Context) error {
	// Perform a simple ping to check database connectivity
	return r.mongodb.Client.Ping(ctx, nil)
}

// GetCharacterCount returns the total number of characters in the database
func (r *Repository) GetCharacterCount(ctx context.Context) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	return count, err
}

// GetRecentlyUpdatedCount returns the number of characters updated within the specified duration
func (r *Repository) GetRecentlyUpdatedCount(ctx context.Context, duration time.Duration) (int64, error) {
	threshold := time.Now().Add(-duration)
	filter := bson.M{
		"updated_at": bson.M{
			"$gte": threshold,
		},
	}
	count, err := r.collection.CountDocuments(ctx, filter)
	return count, err
}

// GetCharactersByIDs retrieves multiple characters by their IDs
func (r *Repository) GetCharactersByIDs(ctx context.Context, characterIDs []int) ([]*models.Character, error) {
	filter := bson.M{"character_id": bson.M{"$in": characterIDs}}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var characters []*models.Character
	if err := cursor.All(ctx, &characters); err != nil {
		return nil, err
	}

	return characters, nil
}

// CountCharacters returns the total number of characters in the database
func (r *Repository) CountCharacters(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{})
}

// CreateIndexes creates necessary database indexes for the characters collection
func (r *Repository) CreateIndexes(ctx context.Context) error {
	// Create unique index on character_id
	characterIDIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "character_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	// Create text index on name field for full-text search (multi-word queries)
	nameTextIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: "text"}},
		Options: options.Index().SetName("name_text"),
	}

	// Create case-insensitive index on name for prefix/regex searches
	// This supports both prefix searches (^pattern) and general regex searches
	nameRegularIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "name", Value: 1}},
		Options: options.Index().
			SetName("name_regular").
			SetCollation(&options.Collation{
				Locale:   "en",
				Strength: 2, // Case-insensitive comparison
			}),
	}

	// Create compound index for efficient sorting with search
	nameWithTimestampIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "name", Value: 1},
			{Key: "created_at", Value: -1}, // Newest first as secondary sort
		},
		Options: options.Index().
			SetName("name_created_compound").
			SetCollation(&options.Collation{
				Locale:   "en",
				Strength: 2, // Case-insensitive
			}),
	}

	indexModels := []mongo.IndexModel{
		characterIDIndex,
		nameTextIndex,
		nameRegularIndex,
		nameWithTimestampIndex,
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return err
	}

	// Create indexes for character_attributes collection
	attributesIndexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "character_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetBackground(true),
		},
		{
			Keys:    bson.D{{Key: "updated_at", Value: -1}},
			Options: options.Index().SetBackground(true),
		},
	}

	_, err = r.attributesCollection.Indexes().CreateMany(ctx, attributesIndexModels)
	if err != nil {
		return err
	}

	// Create indexes for character_skill_queues collection
	skillQueueIndexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "character_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetBackground(true),
		},
		{
			Keys:    bson.D{{Key: "updated_at", Value: -1}},
			Options: options.Index().SetBackground(true),
		},
	}

	_, err = r.skillQueueCollection.Indexes().CreateMany(ctx, skillQueueIndexModels)
	if err != nil {
		return err
	}

	// Create indexes for character_skills collection
	skillsIndexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "character_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetBackground(true),
		},
		{
			Keys:    bson.D{{Key: "updated_at", Value: -1}},
			Options: options.Index().SetBackground(true),
		},
	}

	_, err = r.skillsCollection.Indexes().CreateMany(ctx, skillsIndexModels)
	return err
}

// GetCharacterAttributes retrieves character attributes by character ID
func (r *Repository) GetCharacterAttributes(ctx context.Context, characterID int) (*models.CharacterAttributes, error) {
	filter := bson.M{"character_id": characterID}

	var attributes models.CharacterAttributes
	err := r.attributesCollection.FindOne(ctx, filter).Decode(&attributes)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}

	return &attributes, nil
}

// SaveCharacterAttributes saves or updates character attributes
func (r *Repository) SaveCharacterAttributes(ctx context.Context, attributes *models.CharacterAttributes) error {
	attributes.UpdatedAt = time.Now()
	if attributes.CreatedAt.IsZero() {
		attributes.CreatedAt = time.Now()
	}

	filter := bson.M{"character_id": attributes.CharacterID}
	update := bson.M{"$set": attributes}
	opts := options.Update().SetUpsert(true)

	_, err := r.attributesCollection.UpdateOne(ctx, filter, update, opts)
	return err
}

// DeleteCharacterAttributes deletes character attributes by character ID
func (r *Repository) DeleteCharacterAttributes(ctx context.Context, characterID int) error {
	filter := bson.M{"character_id": characterID}
	_, err := r.attributesCollection.DeleteOne(ctx, filter)
	return err
}

// GetCharacterSkillQueue retrieves character skill queue by character ID
func (r *Repository) GetCharacterSkillQueue(ctx context.Context, characterID int) (*models.CharacterSkillQueue, error) {
	filter := bson.M{"character_id": characterID}

	var skillQueue models.CharacterSkillQueue
	err := r.skillQueueCollection.FindOne(ctx, filter).Decode(&skillQueue)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}

	return &skillQueue, nil
}

// SaveCharacterSkillQueue saves or updates character skill queue
func (r *Repository) SaveCharacterSkillQueue(ctx context.Context, skillQueue *models.CharacterSkillQueue) error {
	skillQueue.UpdatedAt = time.Now()
	if skillQueue.CreatedAt.IsZero() {
		skillQueue.CreatedAt = time.Now()
	}

	filter := bson.M{"character_id": skillQueue.CharacterID}
	update := bson.M{"$set": skillQueue}
	opts := options.Update().SetUpsert(true)

	_, err := r.skillQueueCollection.UpdateOne(ctx, filter, update, opts)
	return err
}

// DeleteCharacterSkillQueue deletes character skill queue by character ID
func (r *Repository) DeleteCharacterSkillQueue(ctx context.Context, characterID int) error {
	filter := bson.M{"character_id": characterID}
	_, err := r.skillQueueCollection.DeleteOne(ctx, filter)
	return err
}

// GetCharacterSkills retrieves character skills by character ID
func (r *Repository) GetCharacterSkills(ctx context.Context, characterID int) (*models.CharacterSkills, error) {
	filter := bson.M{"character_id": characterID}

	var skills models.CharacterSkills
	err := r.skillsCollection.FindOne(ctx, filter).Decode(&skills)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}

	return &skills, nil
}

// SaveCharacterSkills saves or updates character skills
func (r *Repository) SaveCharacterSkills(ctx context.Context, skills *models.CharacterSkills) error {
	skills.UpdatedAt = time.Now()
	if skills.CreatedAt.IsZero() {
		skills.CreatedAt = time.Now()
	}

	filter := bson.M{"character_id": skills.CharacterID}
	update := bson.M{"$set": skills}
	opts := options.Update().SetUpsert(true)

	_, err := r.skillsCollection.UpdateOne(ctx, filter, update, opts)
	return err
}

// DeleteCharacterSkills deletes character skills by character ID
func (r *Repository) DeleteCharacterSkills(ctx context.Context, characterID int) error {
	filter := bson.M{"character_id": characterID}
	_, err := r.skillsCollection.DeleteOne(ctx, filter)
	return err
}

// GetCharacterCorporationHistory retrieves character corporation history by character ID
func (r *Repository) GetCharacterCorporationHistory(ctx context.Context, characterID int) (*models.CharacterCorporationHistory, error) {
	filter := bson.M{"character_id": characterID}

	var history models.CharacterCorporationHistory
	err := r.corpHistoryCollection.FindOne(ctx, filter).Decode(&history)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}

	return &history, nil
}

// SaveCharacterCorporationHistory saves or updates character corporation history
func (r *Repository) SaveCharacterCorporationHistory(ctx context.Context, history *models.CharacterCorporationHistory) error {
	history.UpdatedAt = time.Now()
	if history.CreatedAt.IsZero() {
		history.CreatedAt = time.Now()
	}

	filter := bson.M{"character_id": history.CharacterID}
	update := bson.M{"$set": history}
	opts := options.Update().SetUpsert(true)

	_, err := r.corpHistoryCollection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetCharacterClones retrieves character clones by character ID
func (r *Repository) GetCharacterClones(ctx context.Context, characterID int) (*models.CharacterClones, error) {
	filter := bson.M{"character_id": characterID}

	var clones models.CharacterClones
	err := r.clonesCollection.FindOne(ctx, filter).Decode(&clones)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}

	return &clones, nil
}

// SaveCharacterClones saves or updates character clones
func (r *Repository) SaveCharacterClones(ctx context.Context, clones *models.CharacterClones) error {
	clones.UpdatedAt = time.Now()
	if clones.CreatedAt.IsZero() {
		clones.CreatedAt = time.Now()
	}

	filter := bson.M{"character_id": clones.CharacterID}
	update := bson.M{"$set": clones}
	opts := options.Update().SetUpsert(true)

	_, err := r.clonesCollection.UpdateOne(ctx, filter, update, opts)
	return err
}
