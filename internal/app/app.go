// Package app provides application lifecycle management for AGIS.
package app

import (
"context"
"fmt"
"os"
"os/signal"
"sync"
"syscall"
"time"
)

// App represents the main application with lifecycle management.
type App struct {
name     string
version  string
services []Service
mu       sync.Mutex
started  bool

shutdownTimeout time.Duration
onStart         []func(context.Context) error
onStop          []func(context.Context) error
}

// Service represents a component that can be started and stopped.
type Service interface {
Start(ctx context.Context) error
Stop(ctx context.Context) error
Name() string
}

// Option configures the App.
type Option func(*App)

// New creates a new App with the given options.
func New(name, version string, opts ...Option) *App {
app := &App{
name:            name,
version:         version,
shutdownTimeout: 30 * time.Second,
}
for _, opt := range opts {
opt(app)
}
return app
}

// WithShutdownTimeout sets the graceful shutdown timeout.
func WithShutdownTimeout(d time.Duration) Option {
return func(a *App) {
a.shutdownTimeout = d
}
}

// WithOnStart adds a startup hook.
func WithOnStart(fn func(context.Context) error) Option {
return func(a *App) {
a.onStart = append(a.onStart, fn)
}
}

// WithOnStop adds a shutdown hook.
func WithOnStop(fn func(context.Context) error) Option {
return func(a *App) {
a.onStop = append(a.onStop, fn)
}
}

// Register adds a service to the application.
func (a *App) Register(svc Service) {
a.mu.Lock()
defer a.mu.Unlock()
a.services = append(a.services, svc)
}

// Run starts all services and blocks until shutdown signal.
func (a *App) Run(ctx context.Context) error {
a.mu.Lock()
if a.started {
a.mu.Unlock()
return fmt.Errorf("app already started")
}
a.started = true
a.mu.Unlock()

// Run startup hooks
for _, hook := range a.onStart {
if err := hook(ctx); err != nil {
return fmt.Errorf("startup hook failed: %w", err)
}
}

// Start all services
var wg sync.WaitGroup
errCh := make(chan error, len(a.services))

for _, svc := range a.services {
wg.Add(1)
go func(s Service) {
defer wg.Done()
if err := s.Start(ctx); err != nil {
errCh <- fmt.Errorf("service %s failed: %w", s.Name(), err)
}
}(svc)
}

// Wait for shutdown signal
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

select {
case sig := <-sigCh:
fmt.Printf("\nReceived signal %v, initiating graceful shutdown...\n", sig)
case err := <-errCh:
fmt.Printf("Service error: %v, initiating shutdown...\n", err)
case <-ctx.Done():
fmt.Printf("Context cancelled, initiating shutdown...\n")
}

return a.Shutdown()
}

// Shutdown gracefully stops all services.
func (a *App) Shutdown() error {
ctx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
defer cancel()

var errs []error

// Stop services in reverse order
for i := len(a.services) - 1; i >= 0; i-- {
svc := a.services[i]
if err := svc.Stop(ctx); err != nil {
errs = append(errs, fmt.Errorf("stopping %s: %w", svc.Name(), err))
}
}

// Run shutdown hooks
for _, hook := range a.onStop {
if err := hook(ctx); err != nil {
errs = append(errs, fmt.Errorf("shutdown hook: %w", err))
}
}

if len(errs) > 0 {
return fmt.Errorf("shutdown errors: %v", errs)
}
return nil
}

// Name returns the application name.
func (a *App) Name() string {
return a.name
}

// Version returns the application version.
func (a *App) Version() string {
return a.version
}

// ServiceFunc wraps simple start/stop functions as a Service.
type ServiceFunc struct {
name    string
startFn func(context.Context) error
stopFn  func(context.Context) error
}

// NewServiceFunc creates a Service from functions.
func NewServiceFunc(name string, start, stop func(context.Context) error) *ServiceFunc {
return &ServiceFunc{
name:    name,
startFn: start,
stopFn:  stop,
}
}

func (s *ServiceFunc) Name() string                       { return s.name }
func (s *ServiceFunc) Start(ctx context.Context) error    { return s.startFn(ctx) }
func (s *ServiceFunc) Stop(ctx context.Context) error     { return s.stopFn(ctx) }

// BackgroundService runs a function in the background until stopped.
type BackgroundService struct {
name   string
fn     func(context.Context) error
cancel context.CancelFunc
done   chan struct{}
}

// NewBackgroundService creates a service that runs in the background.
func NewBackgroundService(name string, fn func(context.Context) error) *BackgroundService {
return &BackgroundService{
name: name,
fn:   fn,
done: make(chan struct{}),
}
}

func (s *BackgroundService) Name() string { return s.name }

func (s *BackgroundService) Start(ctx context.Context) error {
ctx, s.cancel = context.WithCancel(ctx)
go func() {
defer close(s.done)
_ = s.fn(ctx)
}()
return nil
}

func (s *BackgroundService) Stop(ctx context.Context) error {
if s.cancel != nil {
s.cancel()
}
select {
case <-s.done:
return nil
case <-ctx.Done():
return ctx.Err()
}
}
