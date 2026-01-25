package regolith

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bedrock-OSS/go-burrito/burrito"
)

// loadEnvFileFromArg loads environment variables from a .env file specified by the env argument.
// If env is empty, loads from ".env" in the current directory.
// Variables already present in the environment are not overwritten (standard priority).
// Returns an error if the file cannot be read (other than not existing).
func loadEnvFileFromArg(env string) error {
	envPath := ".env"
	if env != "" {
		envPath = env
	}
	return LoadEnvFileFromPath(envPath)
}

// LoadEnvFileFromPath loads environment variables from a .env file at a specified path.
// Variables already present in the environment are not overwritten (standard priority).
// Returns an error if the file cannot be read (other than not existing).
func LoadEnvFileFromPath(envFilePath string) error {
	// Resolve the path to be absolute
	absPath, err := filepath.Abs(envFilePath)
	if err != nil {
		return burrito.WrapErrorf(err, filepathAbsError, envFilePath)
	}

	// Check if the file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// File doesn't exist, which is fine - silently return
		return nil
	} else if err != nil {
		return burrito.WrapErrorf(err, osStatErrorAny, absPath)
	}

	// Open the .env file
	file, err := os.Open(absPath)
	if err != nil {
		return burrito.WrapErrorf(err, osOpenError, absPath)
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
				return burrito.WrapErrorf(
					err,
					"Failed to set environment variable:\nKey: %s\nValue: %s",
					key, value,
				)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return burrito.WrapErrorf(err, fileReadError, absPath)
	}

	return nil
}
