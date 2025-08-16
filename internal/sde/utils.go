package sde

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// unzipFile extracts a zip file to a destination directory
func (m *Module) unzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	totalFiles := len(r.File)
	processedFiles := 0

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outfile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outfile.Close()
			return err
		}

		_, err = io.Copy(outfile, rc)
		outfile.Close()
		rc.Close()

		if err != nil {
			return err
		}

		// Update progress
		processedFiles++
		progress := 0.3 + (0.2 * float64(processedFiles) / float64(totalFiles))
		m.updateProgress(progress, fmt.Sprintf("Extracting files... %d/%d", processedFiles, totalFiles))
	}

	return nil
}

// convertYAMLToJSON converts a YAML file to JSON, preserving path context for universe files
func convertYAMLToJSON(src, destDir string) error {
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

	// Generate JSON filename preserving universe path context
	jsonFileName := generateJSONFileName(src)
	destFile := filepath.Join(destDir, jsonFileName)

	if err := os.WriteFile(destFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file %s: %w", destFile, err)
	}

	return nil
}

// convertMapKeysToString recursively converts map keys from interface{} to string
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

// WriteCounter counts the number of bytes written to a stream
type WriteCounter struct {
	Total    uint64
	Expected uint64
	OnProgress func(current, total uint64)
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	
	if wc.OnProgress != nil {
		wc.OnProgress(wc.Total, wc.Expected)
	}
	
	return n, nil
}

// downloadFileWithProgress downloads a file with progress tracking
func (m *Module) downloadFileWithProgress(filepath string, url string) error {
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create progress counter
	counter := &WriteCounter{
		Expected: uint64(resp.ContentLength),
		OnProgress: func(current, total uint64) {
			if total > 0 {
				progress := 0.1 + (0.2 * float64(current) / float64(total))
				m.updateProgress(progress, fmt.Sprintf("Downloading... %d MB / %d MB", current/1024/1024, total/1024/1024))
			}
		},
	}

	// Create TeeReader to track progress
	reader := io.TeeReader(resp.Body, counter)

	_, err = io.Copy(out, reader)
	if err != nil {
		return err
	}

	return os.Rename(filepath+".tmp", filepath)
}

// collectYAMLFiles recursively collects all YAML files from a directory
func collectYAMLFiles(dirPath string) ([]string, error) {
	var yamlFiles []string
	
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return yamlFiles, nil // Return empty list if directory doesn't exist
	}
	
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check for YAML files
		ext := filepath.Ext(path)
		if ext == ".yaml" || ext == ".yml" {
			// Get relative path from extract directory
			relPath, err := filepath.Rel(filepath.Dir(dirPath), path)
			if err != nil {
				return err
			}
			yamlFiles = append(yamlFiles, relPath)
		}
		
		return nil
	})
	
	return yamlFiles, err
}

// generateJSONFileName generates appropriate JSON filename from YAML source path
// For universe files, preserves hierarchical path information in the filename
func generateJSONFileName(src string) string {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(src)
	
	// Check if this is a universe file
	if strings.Contains(normalizedPath, "/universe/") {
		// Extract universe path components
		parts := strings.Split(normalizedPath, "/")
		
		// Find universe directory index
		universeIndex := -1
		for i, part := range parts {
			if part == "universe" {
				universeIndex = i
				break
			}
		}
		
		if universeIndex >= 0 && len(parts) > universeIndex+1 {
			// Extract components: universe/{type}/{region}/{constellation}/{system}/{filename}
			universeParts := parts[universeIndex+1:]
			
			// Build filename: universe_{type}_{region}_{constellation}_{system}_{originalname}.json
			if len(universeParts) >= 4 {
				// Get original filename without extension
				originalName := universeParts[len(universeParts)-1]
				ext := filepath.Ext(originalName)
				nameWithoutExt := originalName[:len(originalName)-len(ext)]
				
				// Build hierarchical filename
				prefix := fmt.Sprintf("universe_%s_%s_%s_%s_%s", 
					universeParts[0], // type
					universeParts[1], // region
					universeParts[2], // constellation  
					universeParts[3], // system
					nameWithoutExt)   // original filename
				
				return prefix + ".json"
			} else if len(universeParts) >= 3 {
				// Constellation level
				originalName := universeParts[len(universeParts)-1]
				ext := filepath.Ext(originalName)
				nameWithoutExt := originalName[:len(originalName)-len(ext)]
				
				prefix := fmt.Sprintf("universe_%s_%s_%s_%s",
					universeParts[0], // type
					universeParts[1], // region
					universeParts[2], // constellation
					nameWithoutExt)   // original filename
				
				return prefix + ".json"
			} else if len(universeParts) >= 2 {
				// Region level
				originalName := universeParts[len(universeParts)-1]
				ext := filepath.Ext(originalName)
				nameWithoutExt := originalName[:len(originalName)-len(ext)]
				
				prefix := fmt.Sprintf("universe_%s_%s_%s",
					universeParts[0], // type
					universeParts[1], // region
					nameWithoutExt)   // original filename
				
				return prefix + ".json"
			}
		}
	}
	
	// For non-universe files, use standard naming
	baseName := filepath.Base(src)
	ext := filepath.Ext(baseName)
	return fmt.Sprintf("%s.json", baseName[:len(baseName)-len(ext)])
}