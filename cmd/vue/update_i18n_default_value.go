package vue

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io/fs"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strings"
)

var UpdateI18nDefaultValueCmd = &cobra.Command{
	Use:   "update-i18n-default-value",
	Short: "Update i18n default value",
	Long:  `Update i18n default value`,
	Run: func(cmd *cobra.Command, args []string) {
		dir := cmd.Flag("dir").Value.String()
		from := cmd.Flag("from").Value.String()
		// Load JSON file
		jsonMap, err := loadJSON(from)
		if err != nil {
			log.Fatalf("Error loading JSON: %v", err)
		}

		// Process files in directory
		err = scanAndModifyFiles(dir, jsonMap)
		if err != nil {
			log.Fatalf("Error processing files: %v", err)
		}

		fmt.Println("Processing complete.")
	},
}

func init() {
	UpdateI18nDefaultValueCmd.Flags().StringP("from", "f", "", "File lang default value")
	UpdateI18nDefaultValueCmd.Flags().StringP("dir", "d", "", "Root dir scan")

	UpdateI18nDefaultValueCmd.MarkFlagRequired("from")
	UpdateI18nDefaultValueCmd.MarkFlagRequired("dir")
}

// Regex to match $t('key') or t('key') with optional parameters
var tFuncRegex = regexp.MustCompile(`(\$t|t)\(['"]([^'"]+)['"](,\s*([^)]*))?\)`)

// Updated regex to match attributes with $t or t function calls inside double-quoted values
var attrRegex = regexp.MustCompile(`(?i)(v-html|:placeholder|:label)=["']([^"']*\$t\(.*?\))["']`)

// Load JSON key-value pairs
func loadJSON(filePath string) (map[string]string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

	var jsonData map[string]string
	err = json.Unmarshal(content, &jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return jsonData, nil
}

// Traverse the directory and process relevant files
func scanAndModifyFiles(dir string, jsonMap map[string]string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Only process specific file types
		if !strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".ts") &&
			!strings.HasSuffix(path, ".vue") && !strings.HasSuffix(path, ".jsx") {
			return nil
		}

		// Read file content
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		modifiedContent, modified := processFileContent(string(content), jsonMap)
		if modified {
			// Write modified content back to the file
			err = ioutil.WriteFile(path, []byte(modifiedContent), 0644)
			if err != nil {
				return fmt.Errorf("failed to write modified content to file %s: %w", path, err)
			}
			fmt.Printf("Modified file: %s\n", path)
		}
		return nil
	})
}

func processFileContent(content string, jsonMap map[string]string) (string, bool) {
	modified := false

	// Process each attribute match that contains $t or t
	modifiedContent := attrRegex.ReplaceAllStringFunc(content, func(attrMatch string) string {
		return tFuncRegex.ReplaceAllStringFunc(attrMatch, func(match string) string {
			submatches := tFuncRegex.FindStringSubmatch(match)
			funcName := submatches[1] // "$t" or "t"
			key := submatches[2]
			param := submatches[4]

			// Check if the key exists in the JSON map
			if jsonValue, found := jsonMap[key]; found {
				// Determine how to format jsonValue
				formattedJsonValue := "'" + escapeSingleQuotes(jsonValue) + "'" // Default to single quotes
				if strings.Contains(jsonValue, "<") {
					// If jsonValue contains HTML, switch to backtick and replace double quotes with single quotes
					formattedJsonValue = "`" + strings.ReplaceAll(jsonValue, `"`, `'`) + "`"
				}

				// Apply modifications based on parameter existence
				if param == "" {
					modified = true
					return fmt.Sprintf(`%s('%s', %s)`, funcName, key, formattedJsonValue)
				} else if !strings.HasPrefix(param, `"`) && !strings.HasPrefix(param, `'`) {
					modified = true
					return fmt.Sprintf(`%s('%s', %s, %s)`, funcName, key, param, formattedJsonValue)
				}
			}
			return match // No modification if conditions aren't met
		})
	})

	return modifiedContent, modified
}

// Escape single quotes for JSON values used in single-quoted contexts
func escapeSingleQuotes(value string) string {
	return strings.ReplaceAll(value, `'`, `\'`)
}
