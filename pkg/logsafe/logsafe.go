// Package logsafe provides utilities for secure logging
package logsafe

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// SensitiveFields is a list of field names that should be redacted from logs
var SensitiveFields = []string{
	"api_key", "apikey", "key", "token", "secret", "password", "credential",
	"authorization", "auth", "bearer", "content", "data", "image", "audio",
}

// Redact replaces the content of a string with a redacted message
func Redact(value string) string {
	if len(value) == 0 {
		return ""
	}

	// Hash the value to create an identifier while hiding the actual content
	hash := sha256.Sum256([]byte(value))
	return fmt.Sprintf("[REDACTED:%s]", hex.EncodeToString(hash[:4]))
}

// RedactHeaders creates a safe copy of HTTP headers with sensitive info redacted
func RedactHeaders(headers map[string][]string) map[string][]string {
	result := make(map[string][]string)

	for key, values := range headers {
		lowerKey := strings.ToLower(key)

		// Check if the header might contain sensitive information
		isSensitive := false
		for _, field := range SensitiveFields {
			if strings.Contains(lowerKey, field) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			result[key] = []string{Redact(strings.Join(values, ","))}
		} else {
			result[key] = values
		}
	}

	return result
}

// DumpJSON logs a redacted version of a struct as JSON with sensitive fields removed
func DumpJSON(logger *slog.Logger, label string, v any) {
	// Convert to map for manipulation
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		logger.Error("failed to marshal object", "error", err)
		return
	}

	var data map[string]any
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		logger.Error("failed to unmarshal object", "error", err)
		return
	}

	// Redact sensitive fields
	safeData := redactMap(data)

	// Marshal back to JSON
	safeJSON, err := json.MarshalIndent(safeData, "", "  ")
	if err != nil {
		logger.Error("failed to marshal safe object", "error", err)
		return
	}

	logger.Debug(label, "data", string(safeJSON))
}

// redactMap recursively redacts sensitive fields in a map
func redactMap(data map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range data {
		lowerKey := strings.ToLower(key)

		// Check if this is a sensitive field
		isSensitive := false
		for _, field := range SensitiveFields {
			if strings.Contains(lowerKey, field) {
				isSensitive = true
				break
			}
		}

		switch v := value.(type) {
		case string:
			if isSensitive {
				result[key] = Redact(v)
			} else if len(v) > 500 {
				// Truncate very long strings
				result[key] = v[:100] + "... [TRUNCATED]"
			} else {
				result[key] = v
			}
		case map[string]any:
			result[key] = redactMap(v)
		case []any:
			result[key] = redactSlice(v)
		default:
			result[key] = v
		}
	}

	return result
}

// redactSlice recursively redacts sensitive fields in slices
func redactSlice(data []any) []any {
	result := make([]any, 0, len(data))

	for _, value := range data {
		switch v := value.(type) {
		case string:
			// Check if the string is very long (could be base64 encoded data)
			if len(v) > 500 {
				result = append(result, v[:100]+"... [TRUNCATED]")
			} else {
				result = append(result, v)
			}
		case map[string]any:
			result = append(result, redactMap(v))
		case []any:
			result = append(result, redactSlice(v))
		default:
			result = append(result, v)
		}
	}

	return result
}
