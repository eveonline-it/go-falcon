package services

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go-falcon/internal/sde_admin/dto"
	"go-falcon/pkg/config"

	"gopkg.in/yaml.v3"
)

// SDESource represents a configured SDE data source
type SDESource struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"` // github, direct, hoboleaks
	URL         string            `json:"url"`
	Format      string            `json:"format"` // yaml, json
	Description string            `json:"description"`
	Enabled     bool              `json:"enabled"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// UpdateService handles SDE updates, downloads, and processing
type UpdateService struct {
	dataDir    string
	tempDir    string
	sources    []SDESource
	httpClient *http.Client
}

// NewUpdateService creates a new SDE update service
func NewUpdateService(dataDir string) *UpdateService {
	tempDir := filepath.Join(dataDir, "..", "sde-temp")

	// Ensure directories exist
	os.MkdirAll(tempDir, 0755)

	return &UpdateService{
		dataDir: dataDir,
		tempDir: tempDir,
		sources: getDefaultSDESources(),
		httpClient: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for large downloads
		},
	}
}

// getDefaultSDESources returns the default configured SDE sources
func getDefaultSDESources() []SDESource {
	sdeURL := config.GetSDEURL()

	return []SDESource{
		{
			Name:        "ccp-official",
			Type:        "direct",
			URL:         sdeURL,
			Format:      "yaml",
			Description: "CCP Games official SDE from S3 bucket",
			Enabled:     true,
			Metadata: map[string]string{
				"download_url": sdeURL,
			},
		},
	}
}

// CheckForUpdates checks the CCP official SDE source for updates by comparing hashes
func (u *UpdateService) CheckForUpdates(ctx context.Context, req *dto.CheckUpdatesRequest) (*dto.CheckUpdatesResponse, error) {
	slog.Info("Checking CCP official SDE for updates using checksum comparison")

	// Load the saved SDE zip hash for comparison
	currentZipHash, err := u.loadSDEZipHash()
	if err != nil {
		slog.Warn("Failed to load saved SDE zip hash", "error", err)
	}
	if currentZipHash == "" {
		slog.Debug("No saved SDE zip hash found - treating as no current version")
	} else {
		slog.Debug("Loaded current SDE zip hash", "hash", currentZipHash[:16]) // Log first 16 chars
	}

	// Get the CCP official source (there's only one)
	ccpSource := u.sources[0] // CCP official is the only source

	// Create initial status for CCP official source
	status := dto.SDESourceStatus{
		Name:        ccpSource.Name,
		Available:   false,
		URL:         ccpSource.URL,
		LastChecked: time.Now().Format(time.RFC3339),
	}

	// Check the CCP official source using checksum comparison
	status = u.checkSourceViaChecksum(ctx, ccpSource, status)

	updatesAvailable := false
	latestVersion := ""

	if status.Available && status.LatestVersion != "" {
		if currentZipHash == "" || status.LatestVersion != currentZipHash {
			updatesAvailable = true
			latestVersion = status.LatestVersion
		}
	}

	return &dto.CheckUpdatesResponse{
		UpdatesAvailable: updatesAvailable,
		CurrentVersion:   currentZipHash,
		LatestVersion:    latestVersion,
		CCPOfficial:      status,
		CheckedAt:        time.Now().Format(time.RFC3339),
	}, nil
}

// checkDirectSource checks a direct URL for updates by downloading and hashing the SDE zip
func (u *UpdateService) checkDirectSource(ctx context.Context, source SDESource, status dto.SDESourceStatus) dto.SDESourceStatus {
	downloadURL := source.Metadata["download_url"]
	if downloadURL == "" {
		downloadURL = source.URL
	}

	// First check if URL is accessible with HEAD request
	headReq, err := http.NewRequestWithContext(ctx, "HEAD", downloadURL, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to create HEAD request: %v", err)
		status.Error = &errorMsg
		return status
	}

	headReq.Header.Set("User-Agent", "go-falcon-sde-admin/1.0")

	headResp, err := u.httpClient.Do(headReq)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to check URL: %v", err)
		status.Error = &errorMsg
		return status
	}
	headResp.Body.Close()

	if headResp.StatusCode != http.StatusOK {
		errorMsg := fmt.Sprintf("URL returned status %d", headResp.StatusCode)
		status.Error = &errorMsg
		return status
	}

	// Get content length if available
	if headResp.ContentLength > 0 {
		status.LatestSize = headResp.ContentLength
	}

	// Now download the file to calculate its hash for proper version comparison
	slog.Debug("Downloading SDE zip to calculate hash for version comparison", "url", downloadURL)

	// Download to temporary file and calculate hash
	tempFile, fileSize, zipHash, err := u.downloadFile(ctx, downloadURL)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to download for hash calculation: %v", err)
		status.Error = &errorMsg
		return status
	}

	// Clean up temporary file immediately after hash calculation
	defer func() {
		if err := os.Remove(tempFile); err != nil {
			slog.Warn("Failed to clean up temporary file", "file", tempFile, "error", err)
		}
	}()

	// Set status with calculated hash as version
	status.Available = true
	status.URL = downloadURL
	status.LatestVersion = zipHash // Use actual SDE zip hash for proper comparison
	status.LatestSize = fileSize

	slog.Debug("Calculated SDE zip hash for version comparison", "hash", zipHash[:16], "size", fileSize)

	return status
}

// checkSourceViaChecksum checks for updates by fetching checksum file from SDE_CHECKSUMS_URL
func (u *UpdateService) checkSourceViaChecksum(ctx context.Context, source SDESource, status dto.SDESourceStatus) dto.SDESourceStatus {
	// Get the checksum URL from config
	checksumURL := config.GetSDEChecksumsURL()
	slog.Debug("Fetching checksum file for comparison", "url", checksumURL)

	// Fetch the checksum file
	req, err := http.NewRequestWithContext(ctx, "GET", checksumURL, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to create checksum request: %v", err)
		status.Error = &errorMsg
		return status
	}

	req.Header.Set("User-Agent", "go-falcon-sde-admin/1.0")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to fetch checksum: %v", err)
		status.Error = &errorMsg
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorMsg := fmt.Sprintf("checksum URL returned status %d", resp.StatusCode)
		status.Error = &errorMsg
		return status
	}

	// Read the checksum file
	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to read checksum data: %v", err)
		status.Error = &errorMsg
		return status
	}

	// Parse the checksum file to find the sde.zip hash
	sdeZipHash, err := u.parseSDEHashFromChecksums(string(checksumData))
	if err != nil {
		errorMsg := fmt.Sprintf("failed to parse SDE hash from checksum: %v", err)
		status.Error = &errorMsg
		return status
	}

	if sdeZipHash == "" {
		errorMsg := "sde.zip hash not found in checksum file"
		status.Error = &errorMsg
		return status
	}

	// Set status with found hash
	status.Available = true
	status.URL = checksumURL
	status.LatestVersion = sdeZipHash
	status.LatestSize = int64(len(checksumData)) // Size of checksum file (not the zip)

	slog.Debug("Found SDE zip hash from checksum file", "hash", sdeZipHash[:16])

	return status
}

// parseSDEHashFromChecksums parses the checksum file to extract the hash for sde.zip
func (u *UpdateService) parseSDEHashFromChecksums(checksumData string) (string, error) {
	lines := strings.Split(checksumData, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: <hash> <filename>
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		hash := parts[0]
		filename := parts[1]

		// Look for sde.zip
		if filename == "sde.zip" {
			slog.Debug("Found sde.zip hash in checksum file", "hash", hash[:16], "filename", filename)
			return hash, nil
		}
	}

	return "", fmt.Errorf("sde.zip not found in checksum file")
}

// getSDEHashFromChecksum fetches the checksum file and extracts the sde.zip hash
func (u *UpdateService) getSDEHashFromChecksum(ctx context.Context, checksumURL string) (string, error) {
	// Fetch the checksum file
	req, err := http.NewRequestWithContext(ctx, "GET", checksumURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create checksum request: %v", err)
	}

	req.Header.Set("User-Agent", "go-falcon-sde-admin/1.0")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch checksum: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum URL returned status %d", resp.StatusCode)
	}

	// Read the checksum file
	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read checksum data: %v", err)
	}

	// Parse and return the hash
	return u.parseSDEHashFromChecksums(string(checksumData))
}

// UpdateSDE downloads and processes SDE data from CCP official source
func (u *UpdateService) UpdateSDE(ctx context.Context, req *dto.UpdateSDERequest) (*dto.UpdateSDEResponse, error) {
	slog.Info("Starting SDE update from CCP official source")

	startTime := time.Now()
	response := &dto.UpdateSDEResponse{
		Source:        "ccp-official",
		UpdatedAt:     time.Now().Format(time.RFC3339),
		ProcessingLog: []dto.UpdateLogEntry{},
	}

	// Use the CCP official source (only source)
	source := &u.sources[0]

	// Get old version hash for comparison
	oldZipHash, _ := u.loadSDEZipHash()
	response.OldVersion = oldZipHash

	// Download the data from CCP official source
	downloadURL := source.Metadata["download_url"]
	if downloadURL == "" {
		downloadURL = source.URL
	}

	if downloadURL == "" {
		response.Success = false
		errorMsg := "no download URL configured for CCP official source"
		response.Error = &errorMsg
		response.Message = errorMsg
		return response, nil
	}

	downloadedFile, downloadSize, err := u.downloadFileSimple(ctx, downloadURL)
	if err != nil {
		response.Success = false
		errorMsg := fmt.Sprintf("failed to download: %v", err)
		response.Error = &errorMsg
		response.Message = errorMsg
		return response, nil
	}
	defer os.Remove(downloadedFile)

	response.DownloadedSize = downloadSize
	response.ProcessingLog = append(response.ProcessingLog, dto.UpdateLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Step:      "download",
		Message:   fmt.Sprintf("Downloaded %.2f MB from %s", float64(downloadSize)/1024/1024, downloadURL),
		Success:   true,
	})

	// Extract the archive
	extractedFiles, err := u.extractArchive(downloadedFile, u.tempDir)
	if err != nil {
		response.Success = false
		errorMsg := fmt.Sprintf("failed to extract: %v", err)
		response.Error = &errorMsg
		response.Message = errorMsg
		return response, nil
	}

	response.ExtractedFiles = extractedFiles
	response.ProcessingLog = append(response.ProcessingLog, dto.UpdateLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Step:      "extract",
		Message:   fmt.Sprintf("Extracted %d files", extractedFiles),
		Success:   true,
	})

	// Process and convert files (CCP provides YAML format)
	convertedFiles := 0
	if req.ConvertToJSON {
		converted, err := u.processSDEFiles(u.tempDir, u.dataDir, true) // CCP official provides YAML
		if err != nil {
			response.Success = false
			errorMsg := fmt.Sprintf("failed to process files: %v", err)
			response.Error = &errorMsg
			response.Message = errorMsg
			return response, nil
		}
		convertedFiles = converted
	} else {
		// Just copy files
		err := u.copySDEFiles(u.tempDir, u.dataDir)
		if err != nil {
			response.Success = false
			errorMsg := fmt.Sprintf("failed to copy files: %v", err)
			response.Error = &errorMsg
			response.Message = errorMsg
			return response, nil
		}
	}

	response.ConvertedFiles = convertedFiles
	response.ProcessingLog = append(response.ProcessingLog, dto.UpdateLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Step:      "process",
		Message:   fmt.Sprintf("Processed %d SDE files", convertedFiles),
		Success:   true,
	})

	// Get the official hash from CCP's checksum file
	checksumURL := config.GetSDEChecksumsURL()
	officialHash, err := u.getSDEHashFromChecksum(ctx, checksumURL)
	if err != nil {
		slog.Warn("Failed to get official SDE hash from checksum", "error", err)
		response.Success = false
		errorMsg := fmt.Sprintf("failed to get official SDE hash: %v", err)
		response.Error = &errorMsg
		response.Message = errorMsg
		return response, nil
	}

	response.NewVersion = officialHash

	// Save the official hash from CCP's checksum file for future update checks
	if err := u.saveSDEZipHash(officialHash); err != nil {
		slog.Warn("Failed to save official SDE zip hash", "error", err, "hash", officialHash)
		// Don't fail the entire update for this - it's just for optimization
	} else {
		slog.Info("Saved official SDE zip hash for update tracking", "hash", officialHash[:16]) // Log first 16 chars
	}

	// Clean up temp files
	os.RemoveAll(u.tempDir)
	os.MkdirAll(u.tempDir, 0755)

	response.Success = true
	response.Duration = time.Since(startTime).String()
	response.Message = "Successfully updated SDE from CCP official source"

	slog.Info("SDE update completed", "source", "ccp-official", "duration", response.Duration, "files", convertedFiles, "official_hash", officialHash[:16])

	return response, nil
}

// downloadFile downloads a file from URL and returns local path, size, and MD5 hash
func (u *UpdateService) downloadFile(ctx context.Context, url string) (string, int64, string, error) {
	slog.Info("Downloading SDE data", "url", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", 0, "", err
	}

	req.Header.Set("User-Agent", "go-falcon-sde-admin/1.0")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp(u.tempDir, "sde-download-*.zip")
	if err != nil {
		return "", 0, "", err
	}
	defer tmpFile.Close()

	// Copy with size tracking and hash calculation
	hash := md5.New()
	multiWriter := io.MultiWriter(tmpFile, hash)

	written, err := io.Copy(multiWriter, resp.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", 0, "", err
	}

	// Calculate final hash
	hashString := hex.EncodeToString(hash.Sum(nil))

	slog.Debug("Downloaded file hash calculated", "hash", hashString, "size_bytes", written)

	return tmpFile.Name(), written, hashString, nil
}

// downloadFileSimple downloads a file from URL without hash calculation
func (u *UpdateService) downloadFileSimple(ctx context.Context, url string) (string, int64, error) {
	slog.Info("Downloading SDE data", "url", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("User-Agent", "go-falcon-sde-admin/1.0")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp(u.tempDir, "sde-download-*.zip")
	if err != nil {
		return "", 0, err
	}
	defer tmpFile.Close()

	// Copy with size tracking only
	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", 0, err
	}

	slog.Debug("Downloaded file", "size_bytes", written)

	return tmpFile.Name(), written, nil
}

// extractArchive extracts a ZIP archive to the specified directory
func (u *UpdateService) extractArchive(archivePath, destDir string) (int, error) {
	slog.Info("Extracting SDE archive", "archive", archivePath, "destination", destDir)

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	// Clean destination
	os.RemoveAll(destDir)
	os.MkdirAll(destDir, 0755)

	extractedCount := 0
	for _, file := range reader.File {
		// Skip directories and non-data files
		if file.FileInfo().IsDir() || !u.isSDEDataFile(file.Name) {
			continue
		}

		err := u.extractFile(file, destDir)
		if err != nil {
			slog.Warn("Failed to extract file", "file", file.Name, "error", err)
			continue
		}
		extractedCount++
	}

	return extractedCount, nil
}

// saveSDEZipHash saves the SDE zip file hash to a persistent file
func (u *UpdateService) saveSDEZipHash(hash string) error {
	hashFile := filepath.Join(u.dataDir, ".sde-hash")
	slog.Debug("Saving SDE zip hash", "hash", hash, "file", hashFile)

	return os.WriteFile(hashFile, []byte(hash), 0644)
}

// loadSDEZipHash loads the saved SDE zip file hash
func (u *UpdateService) loadSDEZipHash() (string, error) {
	hashFile := filepath.Join(u.dataDir, ".sde-hash")

	data, err := os.ReadFile(hashFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No saved hash, treat as no current version
		}
		return "", err
	}

	hash := strings.TrimSpace(string(data))
	slog.Debug("Loaded SDE zip hash", "hash", hash, "file", hashFile)
	return hash, nil
}

// calculateFileHash calculates MD5 hash of a file (used for checksum verification)
func (u *UpdateService) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// GetCurrentZipHash returns the currently saved SDE zip hash (for testing/validation)
func (u *UpdateService) GetCurrentZipHash() (string, error) {
	return u.loadSDEZipHash()
}

// ValidateZipHash validates that a downloaded file matches the expected hash
func (u *UpdateService) ValidateZipHash(filePath, expectedHash string) (bool, error) {
	actualHash, err := u.calculateFileHash(filePath)
	if err != nil {
		return false, err
	}
	return actualHash == expectedHash, nil
}

// isSDEDataFile checks if a file is an SDE data file we care about
func (u *UpdateService) isSDEDataFile(filename string) bool {
	// Clean path for consistent checking
	cleanPath := filepath.Clean(filename)
	basename := filepath.Base(filename)

	// Check for SDE subdirectories (fsd/ and bsd/)
	isFSDFile := strings.Contains(cleanPath, "fsd/")
	isBSDFile := strings.Contains(cleanPath, "bsd/")

	// If not in expected SDE subdirectories, check for direct files (GitHub archives, etc.)
	isDirectFile := !strings.Contains(cleanPath, "/") || strings.Contains(cleanPath, "eve-sde-")

	if !isFSDFile && !isBSDFile && !isDirectFile {
		return false
	}

	// Check for known SDE file patterns
	knownFiles := []string{
		"agents.yaml", "agents.json",
		"agentsInSpace.yaml", "agentsInSpace.json",
		"ancestries.yaml", "ancestries.json",
		"bloodlines.yaml", "bloodlines.json",
		"blueprints.yaml", "blueprints.json",
		"categories.yaml", "categories.json",
		"certificates.yaml", "certificates.json",
		"characterAttributes.yaml", "characterAttributes.json",
		"contrabandTypes.yaml", "contrabandTypes.json",
		"controlTowerResources.yaml", "controlTowerResources.json",
		"corporationActivities.yaml", "corporationActivities.json",
		"dogmaAttributeCategories.yaml", "dogmaAttributeCategories.json",
		"dogmaAttributes.yaml", "dogmaAttributes.json",
		"dogmaEffects.yaml", "dogmaEffects.json",
		"factions.yaml", "factions.json",
		"graphicIDs.yaml", "graphicIDs.json",
		"groups.yaml", "groups.json",
		"iconIDs.yaml", "iconIDs.json",
		"invFlags.yaml", "invFlags.json",
		"invItems.yaml", "invItems.json",
		"invNames.yaml", "invNames.json",
		"invPositions.yaml", "invPositions.json",
		"invUniqueNames.yaml", "invUniqueNames.json",
		"marketGroups.yaml", "marketGroups.json",
		"metaGroups.yaml", "metaGroups.json",
		"npcCorporations.yaml", "npcCorporations.json",
		"npcCorporationDivisions.yaml", "npcCorporationDivisions.json",
		"planetResources.yaml", "planetResources.json",
		"planetSchematics.yaml", "planetSchematics.json",
		"races.yaml", "races.json",
		"researchAgents.yaml", "researchAgents.json",
		"skinLicenses.yaml", "skinLicenses.json",
		"skinMaterials.yaml", "skinMaterials.json",
		"skins.yaml", "skins.json",
		"sovereigntyUpgrades.yaml", "sovereigntyUpgrades.json",
		"staStations.yaml", "staStations.json",
		"stationOperations.yaml", "stationOperations.json",
		"stationServices.yaml", "stationServices.json",
		"translationLanguages.yaml", "translationLanguages.json",
		"typeDogma.yaml", "typeDogma.json",
		"typeMaterials.yaml", "typeMaterials.json",
		"types.yaml", "types.json",
	}

	// Check for universe data files (regions, constellations, solar systems)
	if strings.Contains(basename, "_region.yaml") || strings.Contains(basename, "_region.json") ||
		strings.Contains(basename, "_constellation.yaml") || strings.Contains(basename, "_constellation.json") ||
		strings.Contains(basename, "_solarsystem.json") {
		return true
	}

	return slices.Contains(knownFiles, basename)
}

// extractFile extracts a single file from archive
func (u *UpdateService) extractFile(file *zip.File, destDir string) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	// Get clean filename
	filename := filepath.Base(file.Name)
	destPath := filepath.Join(destDir, filename)

	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, reader)
	return err
}

// processSDEFiles processes extracted SDE files, converting YAML to JSON if needed
func (u *UpdateService) processSDEFiles(srcDir, destDir string, convertYAML bool) (int, error) {
	slog.Info("Processing SDE files", "source", srcDir, "destination", destDir, "convert_yaml", convertYAML)

	files, err := os.ReadDir(srcDir)
	if err != nil {
		return 0, err
	}

	processedCount := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		srcPath := filepath.Join(srcDir, file.Name())
		filename := file.Name()

		// Convert .yaml to .json if requested
		if convertYAML && strings.HasSuffix(filename, ".yaml") {
			jsonFilename := strings.TrimSuffix(filename, ".yaml") + ".json"
			destPath := filepath.Join(destDir, jsonFilename)

			err := u.convertYAMLToJSON(srcPath, destPath)
			if err != nil {
				slog.Warn("Failed to convert YAML file", "file", filename, "error", err)
				continue
			}
			processedCount++
		} else if strings.HasSuffix(filename, ".json") {
			// Copy JSON files directly
			destPath := filepath.Join(destDir, filename)
			err := u.copyFile(srcPath, destPath)
			if err != nil {
				slog.Warn("Failed to copy JSON file", "file", filename, "error", err)
				continue
			}
			processedCount++
		}
	}

	return processedCount, nil
}

// convertYAMLToJSON converts a YAML file to JSON format
func (u *UpdateService) convertYAMLToJSON(yamlPath, jsonPath string) error {
	// Read YAML file
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		return err
	}

	// Parse YAML
	var data any
	err = yaml.Unmarshal(yamlData, &data)
	if err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert YAML structure to JSON-compatible structure
	jsonCompatibleData := u.convertToJSONCompatible(data)

	// Convert to JSON
	jsonData, err := json.MarshalIndent(jsonCompatibleData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write JSON file
	err = os.WriteFile(jsonPath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	return nil
}

// convertToJSONCompatible recursively converts YAML interface{} structures to JSON-compatible ones
func (u *UpdateService) convertToJSONCompatible(data any) any {
	switch v := data.(type) {
	case map[any]any:
		// Convert map[any]any to map[string]any
		result := make(map[string]any)
		for key, value := range v {
			// Convert key to string
			keyStr := fmt.Sprintf("%v", key)
			result[keyStr] = u.convertToJSONCompatible(value)
		}
		return result
	case map[string]any:
		// Already string-keyed map, just convert values
		result := make(map[string]any)
		for key, value := range v {
			result[key] = u.convertToJSONCompatible(value)
		}
		return result
	case []any:
		// Convert slice elements
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = u.convertToJSONCompatible(item)
		}
		return result
	default:
		// Return as-is for primitive types (string, int, float, bool, nil)
		return v
	}
}

// copySDEFiles copies SDE files without conversion
func (u *UpdateService) copySDEFiles(srcDir, destDir string) error {
	files, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		srcPath := filepath.Join(srcDir, file.Name())
		destPath := filepath.Join(destDir, file.Name())

		err := u.copyFile(srcPath, destPath)
		if err != nil {
			return err
		}
	}

	return nil
}

// copyFile copies a single file
func (u *UpdateService) copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}
