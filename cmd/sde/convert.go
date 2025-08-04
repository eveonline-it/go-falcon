package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// convertMapKeysToString recursively converts map keys from interface{} to string.
func convertMapKeysToString(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[fmt.Sprintf("%v", k)] = convertMapKeysToString(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convertMapKeysToString(v)
		}
	}
	return i
}

func convertYamlToJson(src, destDir string) error {
	// Read the YAML file
	yamlData, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read YAML file %s: %w", src, err)
	}

	// Unmarshal the YAML data into a generic interface
	var data interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal YAML from %s: %w", src, err)
	}

	// Convert map keys to strings
	convertedData := convertMapKeysToString(data)

	// Marshal the data to JSON
	jsonData, err := json.MarshalIndent(convertedData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for %s: %w", src, err)
	}

	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	// Write the JSON data to a new file
	baseName := filepath.Base(src)
	ext := filepath.Ext(baseName)
	jsonFileName := fmt.Sprintf("%s.json", baseName[0:len(baseName)-len(ext)])
	destFile := filepath.Join(destDir, jsonFileName)

	if err := os.WriteFile(destFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file %s: %w", destFile, err)
	}

	log.Printf("Converted %s to %s", src, destFile)
	return nil
}
