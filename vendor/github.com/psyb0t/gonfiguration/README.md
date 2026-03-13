# gonfiguration 🔧

A no-bullshit, thread-safe configuration library for Go that doesn't make you wanna punch your monitor. Tired of writing the same boring-ass env var parsing shit over and over? This badass package's got your back with reflection magic that actually works without making you cry.

## What This Beast Can Do

This ain't your granddad's config parser. Here's what makes this package fucking legendary:

### 🎯 **Supported Types (All The Good Shit)**

- **Basic Types**: `string`, `bool` - the bread and butter
- **Signed Integers**: `int`, `int8`, `int16`, `int32`, `int64` - all the flavors you need
- **Unsigned Integers**: `uint`, `uint8`, `uint16`, `uint32`, `uint64` - for when you don't do negative vibes
- **Floating Point**: `float32`, `float64` - because math is hard
- **Time Durations**: `time.Duration` - parsed with Go's native format (`"5s"`, `"10m"`, `"1h30m"`)
- **String Slices**: `[]string` - comma-separated values that get split automagically (`"val1,val2,val3"`)

### 🚀 **Core Features**

- **Thread-Safe**: Won't shit the bed under concurrent load
- **Default Values**: Set fallbacks so your app doesn't break when someone forgets to set an env var
- **Required Fields**: Mark fields as required and get errors when they're missing
- **Zero Dependencies**: Just stdlib, no external packages because we're not monsters
- **Reflection-Based**: Uses Go's reflection to automagically map env vars to struct fields
- **Type Safety**: Validates types and gives you proper error messages instead of cryptic bullshit

## Installation

```bash
go get github.com/psyb0t/gonfiguration
```

## Basic Usage Example

```go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/psyb0t/gonfiguration"
)

type AppConfig struct {
	// Basic types
	ListenAddress string `env:"LISTEN_ADDRESS"`
	Debug         bool   `env:"DEBUG"`
	Port          int    `env:"PORT"`

	// Advanced types
	Timeout       time.Duration `env:"TIMEOUT"`
	AllowedHosts  []string      `env:"ALLOWED_HOSTS"`

	// Database shit - required fields will error if not set
	DBDSN    string `env:"DB_DSN"`
	DBName   string `env:"DB_NAME,required"`
	DBUser   string `env:"DB_USER,required"`
	DBPass   string `env:"DB_PASS,required"`
}

func main() {
	cfg := AppConfig{}

	// Set some defaults because you're not a savage
	gonfiguration.SetDefaults(map[string]interface{}{
		"LISTEN_ADDRESS": "127.0.0.1:8080",
		"DEBUG":          false,
		"PORT":           8080,
		"TIMEOUT":        30 * time.Second,
		"ALLOWED_HOSTS":  []string{"localhost", "127.0.0.1"},
		"DB_DSN":         "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable",
	})

	// Set some env vars (in real life these come from your environment)
	os.Setenv("DB_NAME", "myapp")
	os.Setenv("DB_USER", "postgres-user")
	os.Setenv("DB_PASS", "super-secret-password")
	os.Setenv("DEBUG", "true")
	os.Setenv("ALLOWED_HOSTS", "api.example.com, cdn.example.com, *.example.com")

	// Parse that shit
	if err := gonfiguration.Parse(&cfg); err != nil {
		log.Fatalf("holy fuque! config parsing failed: %v", err)
	}

	fmt.Printf("Config loaded: %+v\n", cfg)
	fmt.Printf("Allowed hosts: %v\n", cfg.AllowedHosts) // ["api.example.com", "cdn.example.com", "*.example.com"]
	fmt.Printf("Timeout: %v\n", cfg.Timeout)           // 30s
}
```

## Complete API Reference

### Core Functions

#### `Parse(dst any) error`

The main function that does all the magic. Pass a pointer to your config struct and it'll populate it with env vars.

```go
cfg := MyConfig{}
err := gonfiguration.Parse(&cfg)
```

#### `MustParse(dst any)`

Same as `Parse()` but panics on error. Perfect for init code where you want to fail fast and loud.

```go
cfg := MyConfig{}
gonfiguration.MustParse(&cfg) // panics if something's wrong
```

#### `SetDefault(key string, val any)`

Set a single default value for when the env var doesn't exist.

```go
gonfiguration.SetDefault("PORT", 8080)
gonfiguration.SetDefault("DEBUG", false)
gonfiguration.SetDefault("TIMEOUT", 30*time.Second)
```

#### `SetDefaults(defaults map[string]any)`

Set multiple defaults at once because batch operations are cooler.

```go
gonfiguration.SetDefaults(map[string]interface{}{
    "PORT":     8080,
    "DEBUG":    false,
    "TIMEOUT":  30*time.Second,
    "HOSTS":    []string{"localhost", "127.0.0.1"}, // for []string fields
})
```

#### `GetDefaults() map[string]any`

Get all the default values you've set. Useful for debugging or just being nosy.

```go
defaults := gonfiguration.GetDefaults()
fmt.Printf("All defaults: %+v\n", defaults)
```

#### `GetEnvVars() map[string]string`

Get all the environment variables that were processed. Again, useful for debugging.

```go
envVars := gonfiguration.GetEnvVars()
fmt.Printf("Processed env vars: %+v\n", envVars)
```

#### `GetAllValues() map[string]any`

Get everything - defaults merged with env vars. Env vars override defaults because that's how the world works.

```go
allValues := gonfiguration.GetAllValues()
fmt.Printf("All config values: %+v\n", allValues)
```

#### `Reset()`

Nuke everything and start fresh. Clears all defaults and cached env vars.

```go
gonfiguration.Reset() // Back to square one
```

## Error Handling (When Shit Goes Wrong)

The library returns descriptive errors when things fuck up. All errors are exported sentinel errors so you can use `errors.Is()` like a civilized person:

```go
// Available sentinel errors
gonfiguration.ErrNilDestination       // "destination is nil"
gonfiguration.ErrInvalidEnvVar        // "invalid environment variable"
gonfiguration.ErrTargetNotPointer     // "destination must be a pointer"
gonfiguration.ErrDestinationNotStruct // "destination must be a struct"
gonfiguration.ErrUnsupportedFieldType // "unsupported field type"
gonfiguration.ErrRequiredFieldNotSet  // "required field not set"
gonfiguration.ErrDefaultTypeMismatch  // "default value type mismatch"

// Check for specific errors
err := gonfiguration.Parse(&cfg)
if errors.Is(err, gonfiguration.ErrRequiredFieldNotSet) {
    // handle missing required field
}

// Invalid struct (not a pointer)
err := gonfiguration.Parse(cfg) // Missing &
// Error: "destination must be a pointer"

// Required field not set
type Config struct {
    APIKey string `env:"API_KEY,required"`
}
err := gonfiguration.Parse(&Config{})
// Error: "required field not set"

// Invalid env var value
os.Setenv("PORT", "not-a-number")
err := gonfiguration.Parse(&cfg)
// Error: "failed to parse int: ..."
```

## Thread Safety (Because Concurrency Is Hard)

This package is thread-safe using `sync.RWMutex`. You can safely:

- Call `Parse()` from multiple goroutines
- Set defaults concurrently
- Get values from different goroutines

```go
// This won't blow up your app
go func() {
    gonfiguration.SetDefault("KEY1", "value1")
}()

go func() {
    gonfiguration.SetDefault("KEY2", "value2")
}()

go func() {
    cfg := MyConfig{}
    gonfiguration.Parse(&cfg)
}()
```

## Rules and Limitations (Read This Shit)

1. **Struct fields MUST have `env:"ENV_VAR_NAME"` tags** - no tag, no parsing
2. **Required fields use `env:"ENV_VAR_NAME,required"`** - errors if no value set
3. **Only supports simple structs** - no nested structs, no complex types, no maps
4. **Pass a pointer to `Parse()`** - not the struct itself, you savage
5. **String slices use comma separation** - `"val1,val2,val3"` becomes `["val1", "val2", "val3"]`
6. **Time durations use Go format** - `"30s"`, `"5m"`, `"2h30m"`, etc.
7. **Empty string slices become empty slices** - `""` becomes `[]string{}`
8. **Default value types must match field types** - don't be an idiot

## License

Copyright 2023-2025 Ciprian Mandache ([ciprian.51k.eu](https://ciprian.51k.eu))

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
