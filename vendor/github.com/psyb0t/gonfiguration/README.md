# gonfiguration 🔧

A kickass configuration package for Golang. Because, why the hell not? Simplify setting defaults and setting struct field vals from env vars without the unnecessary bullshit.

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/psyb0t/gonfiguration"
)

type config struct {
	ListenAddress string `env:"LISTEN_ADDRESS"`
	DBDSN         string `env:"DB_DSN"`
	DBName        string `env:"DB_NAME"`
	DBUser        string `env:"DB_USER"`
	DBPass        string `env:"DB_PASS"`
}

func main() {
	cfg := config{}

	gonfiguration.SetDefaults(map[string]interface{}{
		"LISTEN_ADDRESS": "127.0.0.1:8080",
		"DB_DSN":         "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable",
	})

	if err := os.Setenv("DB_NAME", "postgres"); err != nil {
		log.Fatalf("holy fuque! can't set env: %v", err)
	}

	if err := os.Setenv("DB_USER", "postgres-user"); err != nil {
		log.Fatalf("holy fuque! can't set env: %v", err)
	}

	if err := os.Setenv("DB_PASS", "postgres-pass"); err != nil {
		log.Fatalf("holy fuque! can't set env: %v", err)
	}

	if err := gonfiguration.Parse(&cfg); err != nil {
		log.Fatalf("holy fuque! can't parse config: %v", err)
	}

	fmt.Printf("%+v\n", cfg) //nolint:forbidigo
}
```

## Installation

```bash
go get github.com/psyb0t/gonfiguration
```

## Usage

Alright, let's break it down:

### Step 1: Define Your Config Struct

Define your configuration struct. This badass package is all about keeping it simple. It only vibes with simple structs that don't piss me off. If your config's looking like a damn novel, maybe it's time to split that project into bite-sized chunks. And by the way, those env tags? They're your env var's alter ego. Get it right!

Here's a sweet little example:

```go
type MyAwesomeConfig struct {
    ListenAddress string `env:"LISTEN_ADDRESS"`
    DBDSN         string `env:"DB_DSN"`
    DBName        string `env:"DB_NAME"`
    DBUser        string `env:"DB_USER"`
    DBPass        string `env:"DB_PASS"`
}
```

### Step 2: Set Some Defaults (If You're Into That)

You can set defaults that'll make your life easier. They're like your wingman, always there when no one else is:

```go
gonfiguration.SetDefaults(map[string]interface{}{
    "LISTEN_ADDRESS": "127.0.0.1:8080",
    // ... and so on for your other config variables.
})
```

### Step 3: Parse It Like You Mean It

Make `gonfiguration` work for you. Tell it to grab those env vars and slap 'em into your config struct:

```go
cfg := MyAwesomeConfig{}
if err := gonfiguration.Parse(&cfg); err != nil {
    log.Fatalf("whoa there, partner! can't parse config: %v", err)
}
```

And BAM! Your app's environment is now fresher than a mint garden.

### Step 4: (Optional) Get All Values Because You're Nosy

Just wanna see all the values? We got you:

```go
allTheSecrets := gonfiguration.GetAllValues()
// Now go forth and spill those beans.
```

### Step 5: Hit The Reset Button When You Wanna Start Over

Overdid it? Call in a mulligan and reset those defaults and env vars:

```go
gonfiguration.Reset()
```

## Development

### Makefile

Run these bad boys:

- `make dep` for dependency management.
- `make lint` to lint all your Golang files because nobody likes messy code.
- `make test` for the usual tests.
- `make test-coverage` to see if you're covering your ass enough with tests.

For a full list of commands:

```bash
make help
```

## Contributing

Got some wicked improvements or just found a dumb bug? Open a PR or shoot an issue. Let's get chaotic together.

## License

Copyright 2023-2025 Ciprian Mandache ([ciprian.51k.eu](https://ciprian.51k.eu))

Listen up! Permission is straight-up given, no strings attached, to any badass out there snagging a copy of this masterpiece (let's call it the "Software"). You can rock out with the Software any damn way you please. Want to use it? Go for it. Modify it? Be my guest. Merge, publish, distribute, sublicense, or even make a quick buck selling it? Hell yeah, you can. Just if you're handing this gem to someone else, don't be a douche – include this copyright notice and my cool permission ramble in all copies or major parts of the Software.

Now, here's the kicker: the Software is provided "as is". I ain't making any pinky promises on how it'll perform or if it might royally screw things up. So, if some shit hits the fan, don't come crying to me or any other folks holding the copyright. We're just chilling and ain't responsible for whatever chaos you or this code might stir up.
