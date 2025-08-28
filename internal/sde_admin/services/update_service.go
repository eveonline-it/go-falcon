package services

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
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
	backupDir  string
	tempDir    string
	sources    []SDESource
	httpClient *http.Client
}

// NewUpdateService creates a new SDE update service
func NewUpdateService(dataDir string) *UpdateService {
	backupDir := filepath.Join(dataDir, "..", "sde-backups")
	tempDir := filepath.Join(dataDir, "..", "sde-temp")

	// Ensure directories exist
	os.MkdirAll(backupDir, 0755)
	os.MkdirAll(tempDir, 0755)

	return &UpdateService{
		dataDir:   dataDir,
		backupDir: backupDir,
		tempDir:   tempDir,
		sources:   getDefaultSDESources(),
		httpClient: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for large downloads
		},
	}
}

// getDefaultSDESources returns the default configured SDE sources
func getDefaultSDESources() []SDESource {
	sdeURL := config.GetSDEPath()

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

// GetSources returns the list of configured SDE sources
func (u *UpdateService) GetSources() []SDESource {
	return u.sources
}

// CheckForUpdates checks all enabled sources for SDE updates
func (u *UpdateService) CheckForUpdates(ctx context.Context, req *dto.CheckUpdatesRequest) (*dto.CheckUpdatesResponse, error) {
	slog.Info("Checking for SDE updates", "sources", req.Sources, "force", req.Force)

	currentHash, err := u.calculateCurrentHash()
	if err != nil {
		slog.Warn("Failed to calculate current SDE hash", "error", err)
	}

	var sourcesToCheck []SDESource
	if len(req.Sources) == 0 {
		// Check all enabled sources
		for _, source := range u.sources {
			if source.Enabled {
				sourcesToCheck = append(sourcesToCheck, source)
			}
		}
	} else {
		// Check specified sources
		for _, sourceName := range req.Sources {
			for _, source := range u.sources {
				if source.Name == sourceName && source.Enabled {
					sourcesToCheck = append(sourcesToCheck, source)
					break
				}
			}
		}
	}

	sourceStatuses := make([]dto.SDESourceStatus, 0, len(sourcesToCheck))
	updatesAvailable := false
	latestVersion := ""

	for _, source := range sourcesToCheck {
		status := u.checkSourceForUpdates(ctx, source)
		sourceStatuses = append(sourceStatuses, status)

		if status.Available && status.LatestVersion != "" {
			if currentHash == "" || status.LatestVersion != currentHash {
				updatesAvailable = true
				if latestVersion == "" {
					latestVersion = status.LatestVersion
				}
			}
		}
	}

	return &dto.CheckUpdatesResponse{
		UpdatesAvailable: updatesAvailable,
		CurrentVersion:   currentHash,
		LatestVersion:    latestVersion,
		Sources:          sourceStatuses,
		CheckedAt:        time.Now().Format(time.RFC3339),
	}, nil
}

// checkSourceForUpdates checks a single source for updates
func (u *UpdateService) checkSourceForUpdates(ctx context.Context, source SDESource) dto.SDESourceStatus {
	slog.Debug("Checking source for updates", "source", source.Name, "url", source.URL)

	status := dto.SDESourceStatus{
		Name:        source.Name,
		Available:   false,
		URL:         source.URL,
		LastChecked: time.Now().Format(time.RFC3339),
	}

	switch source.Type {
	case "github":
		return u.checkGitHubSource(ctx, source, status)
	case "direct":
		return u.checkDirectSource(ctx, source, status)
	default:
		errorMsg := fmt.Sprintf("unsupported source type: %s", source.Type)
		status.Error = &errorMsg
		return status
	}
}

// checkGitHubSource checks GitHub API for latest commit/release
func (u *UpdateService) checkGitHubSource(ctx context.Context, source SDESource, status dto.SDESourceStatus) dto.SDESourceStatus {
	// Check latest commit on default branch
	commitURL := fmt.Sprintf("%s/commits", source.URL)

	req, err := http.NewRequestWithContext(ctx, "GET", commitURL, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to create request: %v", err)
		status.Error = &errorMsg
		return status
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "go-falcon-sde-admin/1.0")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to check GitHub API: %v", err)
		status.Error = &errorMsg
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorMsg := fmt.Sprintf("GitHub API returned status %d", resp.StatusCode)
		status.Error = &errorMsg
		return status
	}

	var commits []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Author struct {
				Date string `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		errorMsg := fmt.Sprintf("failed to decode GitHub response: %v", err)
		status.Error = &errorMsg
		return status
	}

	if len(commits) > 0 {
		status.Available = true
		status.LatestVersion = commits[0].SHA[:12] // Short SHA
		status.URL = source.Metadata["download_url"]
	}

	return status
}

// checkDirectSource checks a direct URL for updates (basic availability check)
func (u *UpdateService) checkDirectSource(ctx context.Context, source SDESource, status dto.SDESourceStatus) dto.SDESourceStatus {
	downloadURL := source.Metadata["download_url"]
	if downloadURL == "" {
		downloadURL = source.URL
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", downloadURL, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to create request: %v", err)
		status.Error = &errorMsg
		return status
	}

	req.Header.Set("User-Agent", "go-falcon-sde-admin/1.0")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to check URL: %v", err)
		status.Error = &errorMsg
		return status
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		status.Available = true
		status.URL = downloadURL

		// Use Last-Modified or ETag as version if available
		if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
			if t, err := time.Parse(time.RFC1123, lastModified); err == nil {
				status.LatestVersion = fmt.Sprintf("%d", t.Unix())
			}
		} else if etag := resp.Header.Get("ETag"); etag != "" {
			status.LatestVersion = strings.Trim(etag, `"`)
		}

		// Get content length if available
		if resp.ContentLength > 0 {
			status.LatestSize = resp.ContentLength
		}
	} else {
		errorMsg := fmt.Sprintf("URL returned status %d", resp.StatusCode)
		status.Error = &errorMsg
	}

	return status
}

// UpdateSDE downloads and processes SDE data from the specified source
func (u *UpdateService) UpdateSDE(ctx context.Context, req *dto.UpdateSDERequest) (*dto.UpdateSDEResponse, error) {
	slog.Info("Starting SDE update", "source", req.Source, "format", req.Format)

	startTime := time.Now()
	response := &dto.UpdateSDEResponse{
		Source:        req.Source,
		UpdatedAt:     time.Now().Format(time.RFC3339),
		ProcessingLog: []dto.UpdateLogEntry{},
	}

	// Find the source
	var source *SDESource
	for _, s := range u.sources {
		if s.Name == req.Source {
			source = &s
			break
		}
	}

	if source == nil && req.Source != "custom" {
		response.Success = false
		errorMsg := fmt.Sprintf("unknown source: %s", req.Source)
		response.Error = &errorMsg
		response.Message = errorMsg
		return response, nil
	}

	// Get old version for comparison
	oldVersion, _ := u.calculateCurrentHash()
	response.OldVersion = oldVersion

	// Create backup if requested
	if req.BackupCurrent {
		logEntry := dto.UpdateLogEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			Step:      "backup",
			Message:   "Creating backup of current SDE data",
		}

		backupIDResult, err := u.createBackup("pre-update-" + time.Now().Format("20060102-150405"))
		if err != nil {
			logEntry.Success = false
			logEntry.Message = fmt.Sprintf("Failed to create backup: %v", err)
			response.ProcessingLog = append(response.ProcessingLog, logEntry)

			response.Success = false
			errorMsg := fmt.Sprintf("failed to create backup: %v", err)
			response.Error = &errorMsg
			response.Message = errorMsg
			return response, nil
		}

		logEntry.Success = true
		logEntry.Message = fmt.Sprintf("Created backup: %s", backupIDResult)
		response.ProcessingLog = append(response.ProcessingLog, logEntry)
		response.BackupCreated = true
		response.BackupID = &backupIDResult
	}

	// Download the data
	downloadURL := req.URL
	if downloadURL == "" && source != nil {
		downloadURL = source.Metadata["download_url"]
		if downloadURL == "" {
			downloadURL = source.URL
		}
	}

	if downloadURL == "" {
		response.Success = false
		errorMsg := "no download URL specified"
		response.Error = &errorMsg
		response.Message = errorMsg
		return response, nil
	}

	downloadedFile, downloadSize, err := u.downloadFile(ctx, downloadURL)
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

	// Process and convert files
	convertedFiles := 0
	if req.ConvertToJSON {
		converted, err := u.processSDEFiles(u.tempDir, u.dataDir, req.Format == "" || req.Format == "yaml")
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

	// Calculate new version hash
	newVersion, err := u.calculateCurrentHash()
	if err == nil {
		response.NewVersion = newVersion
	}

	// Clean up temp files
	os.RemoveAll(u.tempDir)
	os.MkdirAll(u.tempDir, 0755)

	response.Success = true
	response.Duration = time.Since(startTime).String()
	response.Message = fmt.Sprintf("Successfully updated SDE from %s", req.Source)

	slog.Info("SDE update completed", "source", req.Source, "duration", response.Duration, "files", convertedFiles)

	return response, nil
}

// downloadFile downloads a file from URL and returns local path and size
func (u *UpdateService) downloadFile(ctx context.Context, url string) (string, int64, error) {
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

	// Copy with size tracking
	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", 0, err
	}

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

// isSDEDataFile checks if a file is an SDE data file we care about
func (u *UpdateService) isSDEDataFile(filename string) bool {
	// Remove directory prefixes (GitHub archives have eve-sde-master/ prefix)
	basename := filepath.Base(filename)

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

	for _, known := range knownFiles {
		if basename == known {
			return true
		}
	}

	return false
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

// calculateCurrentHash calculates SHA256 hash of current SDE data
func (u *UpdateService) calculateCurrentHash() (string, error) {
	files, err := filepath.Glob(filepath.Join(u.dataDir, "*.json"))
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no SDE files found")
	}

	hash := sha256.New()

	// Sort files for consistent hashing
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			if files[i] > files[j] {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files we can't read
		}
		hash.Write(data)
	}

	return hex.EncodeToString(hash.Sum(nil))[:16], nil // Use first 16 chars
}

// createBackup creates a backup of current SDE data
func (u *UpdateService) createBackup(backupID string) (string, error) {
	slog.Info("Creating SDE backup", "backup_id", backupID)

	backupPath := filepath.Join(u.backupDir, backupID+".zip")

	zipFile, err := os.Create(backupPath)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	files, err := filepath.Glob(filepath.Join(u.dataDir, "*.json"))
	if err != nil {
		return "", err
	}

	for _, file := range files {
		err := u.addFileToZip(zipWriter, file, filepath.Base(file))
		if err != nil {
			return "", err
		}
	}

	return backupID, nil
}

// addFileToZip adds a file to a zip archive
func (u *UpdateService) addFileToZip(zipWriter *zip.Writer, filePath, fileName string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := zipWriter.Create(fileName)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// ListBackups returns a list of available SDE backups
func (u *UpdateService) ListBackups(ctx context.Context) (*dto.ListBackupsResponse, error) {
	slog.Debug("Listing SDE backups", "backup_dir", u.backupDir)

	files, err := os.ReadDir(u.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &dto.ListBackupsResponse{
				Backups:     []dto.BackupInfo{},
				TotalCount:  0,
				TotalSizeMB: 0,
			}, nil
		}
		return nil, err
	}

	backups := make([]dto.BackupInfo, 0)
	totalSize := int64(0)

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".zip") {
			continue
		}

		backupID := strings.TrimSuffix(file.Name(), ".zip")
		filePath := filepath.Join(u.backupDir, file.Name())

		info, err := file.Info()
		if err != nil {
			slog.Warn("Failed to get backup file info", "file", file.Name(), "error", err)
			continue
		}

		backupInfo := dto.BackupInfo{
			BackupID:    backupID,
			CreatedAt:   info.ModTime().Format(time.RFC3339),
			SizeMB:      float64(info.Size()) / 1024 / 1024,
			FileCount:   u.countFilesInBackup(filePath),
			Description: "SDE data backup",
		}

		backups = append(backups, backupInfo)
		totalSize += info.Size()
	}

	return &dto.ListBackupsResponse{
		Backups:     backups,
		TotalCount:  len(backups),
		TotalSizeMB: float64(totalSize) / 1024 / 1024,
	}, nil
}

// countFilesInBackup counts files in a backup archive
func (u *UpdateService) countFilesInBackup(backupPath string) int {
	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		return 0
	}
	defer reader.Close()

	count := 0
	for _, file := range reader.File {
		if !file.FileInfo().IsDir() {
			count++
		}
	}
	return count
}

// RestoreBackup restores SDE data from a backup
func (u *UpdateService) RestoreBackup(ctx context.Context, req *dto.RestoreBackupRequest) (*dto.RestoreBackupResponse, error) {
	slog.Info("Restoring SDE backup", "backup_id", req.BackupID)

	startTime := time.Now()
	response := &dto.RestoreBackupResponse{
		BackupID:   req.BackupID,
		RestoredAt: time.Now().Format(time.RFC3339),
	}

	backupPath := filepath.Join(u.backupDir, req.BackupID+".zip")

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		response.Success = false
		errorMsg := fmt.Sprintf("backup not found: %s", req.BackupID)
		response.Error = &errorMsg
		response.Message = errorMsg
		return response, nil
	}

	// Open backup archive
	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		response.Success = false
		errorMsg := fmt.Sprintf("failed to open backup: %v", err)
		response.Error = &errorMsg
		response.Message = errorMsg
		return response, nil
	}
	defer reader.Close()

	// Create temporary restore directory
	restoreDir := filepath.Join(u.tempDir, "restore-"+req.BackupID)
	os.MkdirAll(restoreDir, 0755)
	defer os.RemoveAll(restoreDir)

	// Extract backup files to temp directory
	restoredFiles := 0
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		err := u.extractFile(file, restoreDir)
		if err != nil {
			slog.Warn("Failed to extract backup file", "file", file.Name, "error", err)
			continue
		}
		restoredFiles++
	}

	if restoredFiles == 0 {
		response.Success = false
		errorMsg := "no files were restored from backup"
		response.Error = &errorMsg
		response.Message = errorMsg
		return response, nil
	}

	// Backup current data before restore (safety measure)
	currentBackupID := fmt.Sprintf("pre-restore-%s-%s", req.BackupID, time.Now().Format("20060102-150405"))
	_, err = u.createBackup(currentBackupID)
	if err != nil {
		slog.Warn("Failed to create safety backup before restore", "error", err)
		// Continue with restore anyway
	}

	// Move restored files to data directory
	err = u.replaceDataFiles(restoreDir, u.dataDir)
	if err != nil {
		response.Success = false
		errorMsg := fmt.Sprintf("failed to replace data files: %v", err)
		response.Error = &errorMsg
		response.Message = errorMsg
		return response, nil
	}

	// Delete backup if requested
	if req.DeleteBackup {
		err = os.Remove(backupPath)
		if err != nil {
			slog.Warn("Failed to delete backup after restore", "backup_id", req.BackupID, "error", err)
		} else {
			response.BackupDeleted = true
		}
	}

	response.Success = true
	response.Duration = time.Since(startTime).String()
	response.RestoredFiles = restoredFiles
	response.Message = fmt.Sprintf("Successfully restored %d files from backup %s", restoredFiles, req.BackupID)

	slog.Info("SDE backup restore completed", "backup_id", req.BackupID, "files", restoredFiles, "duration", response.Duration)

	return response, nil
}

// replaceDataFiles replaces files in the data directory with restored files
func (u *UpdateService) replaceDataFiles(srcDir, destDir string) error {
	// Remove existing JSON files
	existingFiles, err := filepath.Glob(filepath.Join(destDir, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range existingFiles {
		os.Remove(file)
	}

	// Copy restored files
	return u.copySDEFiles(srcDir, destDir)
}
