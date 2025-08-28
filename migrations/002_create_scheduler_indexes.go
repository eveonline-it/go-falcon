package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	Register(Migration{
		Version:     "002_create_scheduler_indexes",
		Description: "Create indexes for scheduler_tasks and scheduler_executions collections",
		Up:          up002,
		Down:        down002,
	})
}

func up002(ctx context.Context, db *mongo.Database) error {
	// Scheduler tasks collection indexes
	tasksCollection := db.Collection("scheduler_tasks")
	taskIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "enabled", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "next_run", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "metadata.is_system", Value: 1}},
		},
	}

	if _, err := tasksCollection.Indexes().CreateMany(ctx, taskIndexes); err != nil {
		return err
	}

	// Scheduler executions collection indexes
	executionsCollection := db.Collection("scheduler_executions")
	executionIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "task_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "started_at", Value: -1}}, // Descending for recent first
		},
		{
			Keys: bson.D{
				{Key: "task_id", Value: 1},
				{Key: "started_at", Value: -1},
			},
		},
	}

	if _, err := executionsCollection.Indexes().CreateMany(ctx, executionIndexes); err != nil {
		return err
	}

	return nil
}

func down002(ctx context.Context, db *mongo.Database) error {
	// Drop all indexes except _id
	tasksCollection := db.Collection("scheduler_tasks")
	if _, err := tasksCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}

	executionsCollection := db.Collection("scheduler_executions")
	if _, err := executionsCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}

	return nil
}
