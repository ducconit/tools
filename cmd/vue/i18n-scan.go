package vue

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Tạo command i18n:scan
var I18nScanCmd = &cobra.Command{
	Use:   "i18n:scan",
	Short: "Scan and update i18n keys",
	Long:  `Scans files for i18n keys and updates a specified JSON file, removing unused keys.`,
	Run: func(cmd *cobra.Command, args []string) {
		dir := cmd.Flag("dir").Value.String()
		output := cmd.Flag("output").Value.String()

		// Load current JSON file
		existingKeys, err := loadJSONKeys(output)
		if err != nil && !os.IsNotExist(err) {
			log.Fatalf("Error loading JSON: %v", err)
		}

		// Scan directory for keys
		foundKeys := scanForI18nKeys(dir, existingKeys)

		// Update the JSON file with found keys
		deletedKeys := updateJSONFile(output, foundKeys, existingKeys)

		// Display deleted keys
		fmt.Println("Deleted keys:")
		for _, key := range deletedKeys {
			fmt.Println(key)
		}

		fmt.Println("i18n scan and update complete.")
	},
}

func init() {
	I18nScanCmd.Flags().StringP("output", "o", "", "Output JSON file for i18n keys")
	I18nScanCmd.Flags().StringP("dir", "d", "", "Root directory to scan")

	I18nScanCmd.MarkFlagRequired("output")
	I18nScanCmd.MarkFlagRequired("dir")
}

// Updated regex to capture $t('key') or t('key') with various formats and ignore additional parameters
var tFuncRegexScan = regexp.MustCompile(`\b(\$t|t)\(\s*['"]([^'"]+)['"]`)

// scanForI18nKeys scans the directory for i18n keys in .js, .ts, .vue, .jsx files
func scanForI18nKeys(dir string, existingKeys map[string]string) map[string]string {
	keys := make(map[string]string)
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || (!strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".vue") && !strings.HasSuffix(path, ".jsx")) {
			return nil
		}

		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Printf("Failed to read file %s: %v", path, err)
			return nil
		}

		// Find all i18n keys in the file
		matches := tFuncRegexScan.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			key := match[2]

			// Nếu key không tồn tại trong existingKeys hoặc có giá trị rỗng, thêm vào map với giá trị rỗng
			if val, exists := existingKeys[key]; !exists || val == "" {
				keys[key] = "" // Đặt giá trị mặc định là rỗng cho key mới
			} else {
				keys[key] = val // Giữ giá trị cũ nếu key đã tồn tại và có giá trị
			}
		}
		return nil
	})
	return keys
}

// updateJSONFile updates the JSON file with found keys, removing unused ones and sorting keys alphabetically
func updateJSONFile(filePath string, foundKeys, existingKeys map[string]string) []string {
	updatedKeys := make(map[string]string)
	var deletedKeys []string

	// Update or add new keys
	for key, value := range foundKeys {
		if existingValue, exists := existingKeys[key]; exists && existingValue != "" {
			updatedKeys[key] = existingValue // Keep existing non-empty value
		} else {
			updatedKeys[key] = value // Add new key with default value if provided, or empty if not
		}
	}

	// Collect keys to delete
	for key := range existingKeys {
		if _, exists := foundKeys[key]; !exists {
			deletedKeys = append(deletedKeys, key)
		}
	}

	// Sort keys alphabetically
	sortedKeys := make([]string, 0, len(updatedKeys))
	for key := range updatedKeys {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	// Write updated keys back to file
	jsonData := make(map[string]string)
	for _, key := range sortedKeys {
		jsonData[key] = updatedKeys[key]
	}
	saveJSONFile(filePath, jsonData)

	return deletedKeys
}

// saveJSONFile writes the updated keys to the specified JSON file
func saveJSONFile(filePath string, data map[string]string) {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	err = ioutil.WriteFile(filePath, content, 0644)
	if err != nil {
		log.Fatalf("Failed to write JSON file: %v", err)
	}
}

// loadJSONKeys loads the existing keys from a JSON file
func loadJSONKeys(filePath string) (map[string]string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var jsonData map[string]string
	err = json.Unmarshal(content, &jsonData)
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}
