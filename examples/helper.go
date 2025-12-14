package main

import (
	"fmt"
	"strings"
	"time"
)

// =============================================================================
// UI Helper Functions
// =============================================================================

type ExampleRunner struct {
	name        string
	description string
}

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiCyan   = "\033[36m"
	ansiBlue   = "\033[44m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
	ansiMag    = "\033[35m"
	ansiWhite  = "\033[37m"
)

func NewExample(name, description string) *ExampleRunner {
	fmt.Println()
	fmt.Printf("%s%sEXAMPLE: %s%s\n", ansiBold, ansiCyan, name, ansiReset)
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("%sDESCRIPTION: %s%s\n", ansiDim, description, ansiReset)
	fmt.Println(strings.Repeat("-", 70))
	return &ExampleRunner{name: name}
}

func (e *ExampleRunner) Start() {
	fmt.Printf("\n%s🚀 STARTING EXAMPLE: %s%s\n", ansiBlue, e.name, ansiReset)
}

func (e *ExampleRunner) Complete() {
	fmt.Println()
	fmt.Printf("%s✅ COMPLETED: %s%s\n", ansiGreen, e.name, ansiReset)
	fmt.Println("\n" + strings.Repeat("=", 70))
}

func (e *ExampleRunner) Section(title string) {
	fmt.Printf("\n%s📋 %s:%s\n", ansiWhite, title, ansiReset)
	fmt.Println(strings.Repeat("-", 40))
}

func (e *ExampleRunner) Info(message string, args ...interface{}) {
	fmt.Println()
	fmt.Printf("%sℹ️ INFO: %s%s\n", ansiCyan, fmt.Sprintf(message, args...), ansiReset)
}

func (e *ExampleRunner) Result(message string, args ...interface{}) {
	fmt.Println()
	fmt.Printf("%s📊 RESULT: %s%s\n", ansiYellow, fmt.Sprintf(message, args...), ansiReset)
}

func (e *ExampleRunner) Error(err error) {
	fmt.Println()
	fmt.Printf("%s❌ ERROR: %v%s\n", ansiRed, err, ansiReset)
}

func (e *ExampleRunner) Success(message string, args ...interface{}) {
	fmt.Println()
	fmt.Printf("%s✅ SUCCESS: %s%s\n", ansiGreen, fmt.Sprintf(message, args...), ansiReset)
}

func (e *ExampleRunner) CodeBlock(code string) {
	fmt.Println("\n" + ansiDim + "```go")
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		if len(line) > 70 {
			fmt.Println(line[:70] + "...")
		} else {
			fmt.Println(line)
		}
	}
	fmt.Println("```" + ansiReset)
}

func (e *ExampleRunner) KeyValue(key, value string) {
	fmt.Printf("  %s🔑 %-25s: %s%s\n", ansiMag, key, value, ansiReset)
}

// =============================================================================
// Data Structures
// =============================================================================

type AppConfig struct {
	App      AppInfo      `config:"app" validate:"required"`
	Server   ServerConfig `config:"server" validate:"required"`
	Database Database     `config:"database" validate:"required"`
	Logging  LogConfig    `config:"logging"`
	Features []Features   `config:"features" validate:"dive"`
}

type AppInfo struct {
	Name        string `config:"name" validate:"required,min=3"`
	Version     string `config:"version" validate:"required,semver"`
	Environment string `config:"environment" validate:"required,oneof=dev staging prod"`
	AdminEmail  string `config:"admin_email" validate:"omitempty,email"`
	APIEndpoint string `config:"api_endpoint" validate:"omitempty,uri"`
}

type ServerConfig struct {
	Host         string        `config:"host" validate:"required,hostname_rfc1123"`
	Port         int           `config:"port" validate:"required,min=1024,max=65535"`
	ReadTimeout  time.Duration `config:"read_timeout" validate:"required,gt=0"`
	WriteTimeout time.Duration `config:"write_timeout" validate:"required,gt=0"`
	MaxConns     int           `config:"max_connections" validate:"min=1,max=10000"`
	EnableSSL    bool          `config:"ssl"`
}

type Database struct {
	Driver   string `config:"driver" validate:"required,oneof=postgres mysql sqlite"`
	Host     string `config:"host" validate:"required,hostname_rfc1123"`
	Port     int    `config:"port" validate:"required,min=1024,max=65535"`
	Username string `config:"username" validate:"required,min=3"`
	Password string `config:"password" validate:"required,min=8"`
	Name     string `config:"name" validate:"required"`
	SSLMode  string `config:"ssl_mode" validate:"omitempty,oneof=disable require verify-ca verify-full"`
}

type LogConfig struct {
	Level  string `config:"level" validate:"omitempty,oneof=debug info warn error"`
	Format string `config:"format" validate:"omitempty,oneof=json text"`
	File   string `config:"file"`
}

type Features struct {
	Caching      bool `config:"caching"`
	Monitoring   bool `config:"monitoring"`
	Analytics    bool `config:"analytics"`
	RateLimiting bool `config:"rate_limiting"`
	DebugMode    bool `config:"debug_mode"`
}

// =============================================================================
