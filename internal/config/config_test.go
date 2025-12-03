// Package config provides tests for configuration management.
package config

import (
"os"
"testing"
"time"
)

func TestLoad(t *testing.T) {
os.Setenv("DISCORD_TOKEN", "test-token")
defer os.Unsetenv("DISCORD_TOKEN")

cfg, err := Load()
if err != nil {
t.Fatalf("unexpected error: %v", err)
}

if cfg.Discord.Token != "test-token" {
t.Errorf("expected token test-token, got %s", cfg.Discord.Token)
}
}

func TestLoadMissingRequired(t *testing.T) {
os.Unsetenv("DISCORD_TOKEN")

_, err := Load()
if err == nil {
t.Error("expected error for missing DISCORD_TOKEN")
}
}

func TestMustLoadPanic(t *testing.T) {
os.Unsetenv("DISCORD_TOKEN")

defer func() {
if r := recover(); r == nil {
t.Error("expected panic for missing required config")
}
}()

MustLoad()
}

func TestDefaults(t *testing.T) {
os.Setenv("DISCORD_TOKEN", "test")
defer os.Unsetenv("DISCORD_TOKEN")

cfg, err := Load()
if err != nil {
t.Fatalf("unexpected error: %v", err)
}

if cfg.App.Name != "agis-bot" {
t.Errorf("expected app name agis-bot, got %s", cfg.App.Name)
}
if cfg.App.Environment != "development" {
t.Errorf("expected environment development, got %s", cfg.App.Environment)
}
if cfg.HTTP.Port != 8080 {
t.Errorf("expected port 8080, got %d", cfg.HTTP.Port)
}
if cfg.HTTP.ReadTimeout != 15*time.Second {
t.Errorf("expected read timeout 15s, got %v", cfg.HTTP.ReadTimeout)
}
if cfg.Database.Port != 5432 {
t.Errorf("expected db port 5432, got %d", cfg.Database.Port)
}
if !cfg.Features.EnableMetrics {
t.Error("expected metrics enabled by default")
}
if cfg.Log.Level != "info" {
t.Errorf("expected log level info, got %s", cfg.Log.Level)
}
}

func TestEnvOverrides(t *testing.T) {
os.Setenv("DISCORD_TOKEN", "test")
os.Setenv("APP_NAME", "custom-name")
os.Setenv("HTTP_PORT", "9090")
os.Setenv("HTTP_READ_TIMEOUT", "30s")
os.Setenv("APP_DEBUG", "true")
os.Setenv("DISCORD_ALLOWED_GUILDS", "guild1, guild2, guild3")

defer func() {
os.Unsetenv("DISCORD_TOKEN")
os.Unsetenv("APP_NAME")
os.Unsetenv("HTTP_PORT")
os.Unsetenv("HTTP_READ_TIMEOUT")
os.Unsetenv("APP_DEBUG")
os.Unsetenv("DISCORD_ALLOWED_GUILDS")
}()

cfg, err := Load()
if err != nil {
t.Fatalf("unexpected error: %v", err)
}

if cfg.App.Name != "custom-name" {
t.Errorf("expected app name custom-name, got %s", cfg.App.Name)
}
if cfg.HTTP.Port != 9090 {
t.Errorf("expected port 9090, got %d", cfg.HTTP.Port)
}
if cfg.HTTP.ReadTimeout != 30*time.Second {
t.Errorf("expected read timeout 30s, got %v", cfg.HTTP.ReadTimeout)
}
if !cfg.App.Debug {
t.Error("expected debug true")
}
if len(cfg.Discord.AllowedGuilds) != 3 {
t.Errorf("expected 3 allowed guilds, got %d", len(cfg.Discord.AllowedGuilds))
}
}

func TestDatabaseDSN(t *testing.T) {
dbCfg := &DatabaseConfig{
Host:     "localhost",
Port:     5432,
User:     "testuser",
Password: "testpass",
Name:     "testdb",
SSLMode:  "require",
}

dsn := dbCfg.DatabaseDSN()
expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=require"

if dsn != expected {
t.Errorf("expected DSN %s, got %s", expected, dsn)
}
}

func TestHTTPAddress(t *testing.T) {
httpCfg := &HTTPConfig{
Host: "0.0.0.0",
Port: 8080,
}

addr := httpCfg.HTTPAddress()
if addr != "0.0.0.0:8080" {
t.Errorf("expected address 0.0.0.0:8080, got %s", addr)
}
}

func TestIsProduction(t *testing.T) {
tests := []struct {
env    string
isProd bool
}{
{"production", true},
{"staging", false},
{"development", false},
}

for _, tt := range tests {
t.Run(tt.env, func(t *testing.T) {
cfg := &AppConfig{Environment: tt.env}
if cfg.IsProduction() != tt.isProd {
t.Errorf("IsProduction() = %v, want %v", cfg.IsProduction(), tt.isProd)
}
})
}
}

func TestIsDevelopment(t *testing.T) {
tests := []struct {
env   string
isDev bool
}{
{"development", true},
{"staging", false},
{"production", false},
}

for _, tt := range tests {
t.Run(tt.env, func(t *testing.T) {
cfg := &AppConfig{Environment: tt.env}
if cfg.IsDevelopment() != tt.isDev {
t.Errorf("IsDevelopment() = %v, want %v", cfg.IsDevelopment(), tt.isDev)
}
})
}
}

func TestEnvHelpers(t *testing.T) {
os.Setenv("TEST_STRING", "value")
if envString("TEST_STRING", "default") != "value" {
t.Error("envString failed")
}
if envString("MISSING", "default") != "default" {
t.Error("envString default failed")
}
os.Unsetenv("TEST_STRING")

os.Setenv("TEST_INT", "42")
if envInt("TEST_INT", 0) != 42 {
t.Error("envInt failed")
}
if envInt("MISSING", 10) != 10 {
t.Error("envInt default failed")
}
os.Unsetenv("TEST_INT")

os.Setenv("TEST_INT_INVALID", "not-a-number")
if envInt("TEST_INT_INVALID", 99) != 99 {
t.Error("envInt invalid should return default")
}
os.Unsetenv("TEST_INT_INVALID")

os.Setenv("TEST_FLOAT", "3.14")
if envFloat("TEST_FLOAT", 0) != 3.14 {
t.Error("envFloat failed")
}
os.Unsetenv("TEST_FLOAT")

os.Setenv("TEST_BOOL", "true")
if !envBool("TEST_BOOL", false) {
t.Error("envBool failed")
}
os.Unsetenv("TEST_BOOL")

os.Setenv("TEST_DUR", "5m")
if envDuration("TEST_DUR", time.Second) != 5*time.Minute {
t.Error("envDuration failed")
}
os.Unsetenv("TEST_DUR")

os.Setenv("TEST_SLICE", "a, b, c")
slice := envStringSlice("TEST_SLICE", nil)
if len(slice) != 3 || slice[0] != "a" || slice[1] != "b" || slice[2] != "c" {
t.Errorf("envStringSlice failed: %v", slice)
}
os.Unsetenv("TEST_SLICE")
}
