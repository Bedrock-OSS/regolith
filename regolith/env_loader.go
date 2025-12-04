package regolith

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadEnvFile loads environment variables from a .env file in the current directory.
// Variables already present in the environment are not overwritten (standard priority).
// Returns an error if the file cannot be read (other than not existing).
func LoadEnvFile() error {
	// Look for .env file in the current directory
	envFilePath := ".env"

	// Check if the file exists
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		// File doesn't exist, which is fine - silently return
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking .env file: %w", err)
	}

	// Open the .env file
	file, err := os.Open(envFilePath)
	if err != nil {
		return fmt.Errorf("error opening .env file: %w", err)
	}
	defer file.Close()

	// Parse and load environment variables
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pair
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}

		// Only set the variable if it's not already set (standard priority)
		if _, exists := os.LookupEnv(key); !exists {
			err := os.Setenv(key, value)
			if err != nil {
				return fmt.Errorf("error setting environment variable %s: %w", key, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .env file: %w", err)
	}

	return nil
}

// LoadEnvFileFromPath loads environment variables from a .env file at a specified path.
// Variables already present in the environment are not overwritten (standard priority).
// Returns an error if the file cannot be read (other than not existing).
func LoadEnvFileFromPath(envFilePath string) error {
	// Resolve the path to be absolute
	absPath, err := filepath.Abs(envFilePath)
	if err != nil {
		return fmt.Errorf("error resolving path %s: %w", envFilePath, err)
	}

	// Check if the file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// File doesn't exist, which is fine - silently return
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking .env file at %s: %w", absPath, err)
	}

	// Open the .env file
	file, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("error opening .env file at %s: %w", absPath, err)
	}
	defer file.Close()

	// Parse and load environment variables
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pair
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}

		// Only set the variable if it's not already set (standard priority)
		if _, exists := os.LookupEnv(key); !exists {
			err := os.Setenv(key, value)
			if err != nil {
				return fmt.Errorf("error setting environment variable %s: %w", key, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .env file at %s: %w", absPath, err)
	}

	return nil
}
