package main

import (
	"context"
	"log"
	"log/slog"

	"go-falcon/pkg/app"
	"os"
	"path/filepath"
)

const sdeURL = "https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/sde.zip"

func main() {
	ctx := context.Background()

	// Initialize application with shared components
	appCtx, err := app.InitializeApp("sde")
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer appCtx.Shutdown(ctx)

	slog.Info("Starting SDE utility...")

	// Define paths
	tmpDir := "tmp"
	sdeZipFile := filepath.Join(tmpDir, "sde.zip")
	extractDir := filepath.Join(tmpDir, "sde")

	// Create the tmp directory if it doesn't exist
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		slog.Error("Failed to create tmp directory", "error", err)
		os.Exit(1)
	}

	// Check if the SDE zip file exists, download if it doesn't
	if _, err := os.Stat(sdeZipFile); os.IsNotExist(err) {
		slog.Info("SDE zip file not found, downloading...", "url", sdeURL)
		if err := downloadFile(sdeZipFile, sdeURL); err != nil {
			slog.Error("Failed to download SDE zip file", "error", err)
			os.Exit(1)
		}
		slog.Info("SDE zip file downloaded successfully", "path", sdeZipFile)
	} else {
		slog.Info("SDE zip file already exists, skipping download", "path", sdeZipFile)
	}

	// Create the extraction directory if it doesn't exist
	if err := os.MkdirAll(extractDir, os.ModePerm); err != nil {
		slog.Error("Failed to create extraction directory", "error", err)
		os.Exit(1)
	}

	slog.Info("Extracting SDE zip file", "source", sdeZipFile, "destination", extractDir)

	if err := unzip(sdeZipFile, extractDir); err != nil {
		slog.Error("Failed to extract SDE zip file", "error", err)
		os.Exit(1)
	}

	// Define the list of YAML files to convert
	yamlFiles := []string{
		"fsd/agents.yaml",
		"fsd/blueprints.yaml",
		"fsd/categories.yaml",
		// "fsd/certificates.yaml",
		// "fsd/characterAttributes.yaml",
		// "fsd/constants.yaml",
		// "fsd/contraband.yaml",
		// "fsd/controlTowerResources.yaml",
		// "fsd/dogmaAttributes.yaml",
		// "fsd/dogmaEffects.yaml",
		// "fsd/dogmaExpressions.yaml",
		// "fsd/graphicIDs.yaml",
		// "fsd/groups.yaml",
		// "fsd/iconIDs.yaml",
		"fsd/marketGroups.yaml",
		"fsd/metaGroups.yaml",
		"fsd/npcCorporations.yaml",
		// "fsd/planetSchematics.yaml",
		// "fsd/races.yaml",
		// "fsd/skins.yaml",
		// "fsd/skinLicenses.yaml",
		// "fsd/skinMaterials.yaml",
		// "fsd/stationServices.yaml",
		"fsd/types.yaml",
		"fsd/typeDogma.yaml",
		"fsd/typeMaterials.yaml",
		// "fsd/universe.yaml",
	}

	// Convert selected YAML files to JSON
	jsonDir := "data/sde"
	for _, yamlFile := range yamlFiles {
		fullPath := filepath.Join(extractDir, yamlFile)
		if err := convertYamlToJson(fullPath, jsonDir); err != nil {
			slog.Error("Failed to convert file", "path", fullPath, "error", err)
		}
	}

	slog.Info("SDE processing completed successfully")
}
