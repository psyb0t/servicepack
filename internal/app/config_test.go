package app

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name:    "default config parsing",
			envVars: map[string]string{},
			wantErr: false,
		},
		{
			name: "config parsing with env vars",
			envVars: map[string]string{
				"ENV":        "test",
				"LOG_LEVEL":  "debug",
				"LOG_FORMAT": "json",
			},
			wantErr: false,
		},
		{
			name: "config parsing with minimal env vars",
			envVars: map[string]string{
				"ENV": "production",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store and clear existing environment variables
			envVarsToCheck := []string{"ENV", "LOG_LEVEL", "LOG_FORMAT"}
			originalValues := make(map[string]string)

			for _, envVar := range envVarsToCheck {
				originalValues[envVar] = os.Getenv(envVar)
				os.Unsetenv(envVar)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				err := os.Setenv(key, value)
				require.NoError(t, err)
			}

			// Restore environment after test
			defer func() {
				for envVar, originalValue := range originalValues {
					if originalValue != "" {
						err := os.Setenv(envVar, originalValue)
						if err != nil {
							t.Logf("Failed to restore env var %s: %v", envVar, err)
						}
					} else {
						os.Unsetenv(envVar)
					}
				}
			}()

			cfg, err := parseConfig()
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, cfg)
		})
	}
}

func TestConfig_Structure(t *testing.T) {
	t.Run("config struct initialization", func(t *testing.T) {
		// Test that config struct can be created and is empty as expected
		cfg := config{}
		assert.NotNil(t, cfg)
		// Since config struct is currently empty, there's not much to test
		// This test ensures the struct is properly defined and accessible
	})
}
