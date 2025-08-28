package migrations

import (
	"go.mongodb.org/mongo-driver/mongo"
	"strings"
)

// isIndexExistsError checks if error is due to index already existing
func isIndexExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return mongo.IsDuplicateKeyError(err) ||
		strings.Contains(errStr, "already exists") ||
		strings.Contains(errStr, "IndexKeySpecsConflict") ||
		strings.Contains(errStr, "IndexOptionsConflict") ||
		strings.Contains(errStr, "equivalent index already exists")
}
