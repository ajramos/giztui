package main

import (
	"context"
	"errors"
	"log"
	"sync/atomic"

	"github.com/ajramos/giztui/pkg/desktop"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Wails events the frontend subscribes to for streaming AI output and progress.
const (
	summaryTokenEvent = "summary:token"
	promptTokenEvent  = "prompt:token"
	chatTokenEvent    = "chat:token"
	planProgressEvent = "plan:progress"
)

// App is the Wails-bound backend. Its exported methods are callable from the
// frontend (window.go.main.App.*). It is intentionally a thin wrapper: all real
// logic lives in the shared pkg/desktop.API, which is reused verbatim from the
// TUI's service layer.
type App struct {
	ctx       context.Context
	session   atomic.Pointer[desktop.Session]
	initErr   atomic.Pointer[string]
	ready     atomic.Bool
	authURL   atomic.Pointer[string]
	needCreds atomic.Bool // startup failed because credentials.json is missing
}

// NewApp creates the App in an unstarted state. Startup wiring happens in
// startup, once Wails has provided a context.
func NewApp() *App {
	return &App{}
}

// startup is invoked by Wails when the app launches. It builds the Gmail/service
// stack off the main thread: NewSession does OAuth + network + disk I/O, and
// doing that synchronously here blocks the WKWebView's first paint on macOS,
// which shows as a black window on the first (cold) launch. Building it in a
// goroutine lets the window paint immediately; the frontend waits on Ready().
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if path, err := desktop.SetupFileLogging(); err == nil {
		log.Printf("desktop: logging to %s", path)
	}
	// When interactive sign-in is needed, open the consent URL in the system
	// browser and expose it so the frontend can show a "Sign in" modal (the
	// local redirect server captures the code automatically once granted).
	desktop.SetAuthURLHook(func(url string) {
		a.authURL.Store(&url)
		wailsruntime.BrowserOpenURL(a.ctx, url)
		log.Printf("desktop: opened sign-in URL in browser")
	})
	a.initSession()
}

// initSession builds the Gmail/service stack in the background (off the main
// thread so WKWebView paints immediately) and records the outcome. It is safe to
// call again to retry — e.g. after the user imports credentials.json — and
// resets the ready/error state before doing so.
func (a *App) initSession() {
	a.ready.Store(false)
	a.initErr.Store(nil)
	a.needCreds.Store(false)
	go func() {
		sess, err := desktop.NewSession(a.ctx, desktop.Options{Logger: log.Default()})
		a.authURL.Store(nil) // sign-in finished (or failed); clear the prompt
		if err != nil {
			if errors.Is(err, desktop.ErrNoCredentials) {
				a.needCreds.Store(true)
			}
			msg := err.Error()
			a.initErr.Store(&msg)
			a.ready.Store(true)
			log.Printf("desktop: session init failed: %v", err)
			return
		}
		a.session.Store(sess)
		a.ready.Store(true)
		if email, err := sess.AccountEmail(a.ctx); err == nil {
			log.Printf("desktop: signed in as %s", email)
		}
	}()
}

// shutdown is invoked by Wails on exit; it releases the local database.
func (a *App) shutdown(_ context.Context) {
	if s := a.session.Load(); s != nil {
		_ = s.Close()
	}
}

// Ready reports whether the session has finished initializing (successfully or
// not). The frontend polls this before its first backend calls.
func (a *App) Ready() bool { return a.ready.Load() }

// Version returns the desktop build version so the UI can show it (like the TUI).
func (a *App) Version() string { return desktop.Version() }

// LogUI lets the frontend write a line to desktop.log — used to diagnose things
// that only reproduce in the packaged WKWebView (e.g. why an inline image or a
// shortcut didn't behave), where the browser console isn't visible.
func (a *App) LogUI(msg string) { log.Printf("ui: %s", msg) }

// PendingAuthURL returns the OAuth consent URL while interactive sign-in is in
// progress, or "" otherwise. The frontend polls it (alongside Ready) to show a
// sign-in modal with a button to (re)open the browser.
func (a *App) PendingAuthURL() string {
	if u := a.authURL.Load(); u != nil {
		return *u
	}
	return ""
}

// OpenAuthURL re-opens the pending sign-in URL in the system browser (the modal
// button), in case the automatic open was blocked or the user closed the tab.
func (a *App) OpenAuthURL() {
	if u := a.authURL.Load(); u != nil {
		wailsruntime.BrowserOpenURL(a.ctx, *u)
	}
}

// api returns the shared API or an error describing why startup failed / is
// still in progress.
func (a *App) api() (*desktop.API, error) {
	if s := a.session.Load(); s != nil {
		return s.API, nil
	}
	if e := a.initErr.Load(); e != nil {
		return nil, &startupError{*e}
	}
	return nil, &startupError{"connecting to Gmail…"}
}

type startupError struct{ msg string }

func (e *startupError) Error() string { return e.msg }

// --- bound methods (called from the frontend) --------------------------------

// InitError returns the startup error, if any, so the UI can surface config or
// auth problems instead of silently showing an empty inbox.
func (a *App) InitError() string {
	if e := a.initErr.Load(); e != nil {
		return *e
	}
	return ""
}

// NeedsCredentials reports whether startup failed specifically because the OAuth
// client credentials file is missing (the desktop first-run case). The frontend
// uses it to show an onboarding screen with an import button instead of a raw
// filesystem error.
func (a *App) NeedsCredentials() bool { return a.needCreds.Load() }

// CredentialsPath returns the path where GizTUI expects credentials.json, so the
// onboarding screen can show it (and so the user knows where a manual copy goes).
func (a *App) CredentialsPath() string {
	return desktop.CredentialsPathFor(desktop.Options{})
}

// ImportCredentials opens a native file picker for the user's downloaded OAuth
// credentials.json, validates and copies it into place, then retries session
// init (which may in turn start the interactive OAuth consent flow). It returns
// the destination path, or "" if the user cancelled the dialog.
func (a *App) ImportCredentials() (string, error) {
	src, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select your Google OAuth credentials.json",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "JSON credentials (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return "", err
	}
	if src == "" {
		return "", nil // user cancelled
	}
	dest, err := desktop.InstallCredentials(desktop.Options{}, src)
	if err != nil {
		return "", err
	}
	a.initSession() // retry now that credentials exist
	return dest, nil
}

// RetryInit re-runs session initialization (the "Retry" button), e.g. after the
// user placed credentials.json manually or fixed their config.
func (a *App) RetryInit() { a.initSession() }

// Quit exits the application (the TUI's `q` / `:quit`).
func (a *App) Quit() { wailsruntime.Quit(a.ctx) }

// AccountEmail returns the active account's email address.
func (a *App) AccountEmail() (string, error) {
	s := a.session.Load()
	if s == nil {
		return "", a.notReady()
	}
	return s.AccountEmail(a.ctx)
}

// notReady returns the startup error if init failed, else a "connecting" error.
func (a *App) notReady() error {
	if e := a.initErr.Load(); e != nil {
		return &startupError{*e}
	}
	return &startupError{"connecting to Gmail…"}
}

// enabled reports a feature flag, returning false until the session is ready.
func (a *App) enabled(fn func(*desktop.API) bool) bool {
	if s := a.session.Load(); s != nil {
		return fn(s.API)
	}
	return false
}

// ThemesEnabled reports whether theming is available.
func (a *App) ThemesEnabled() bool {
	return a.enabled((*desktop.API).ThemesEnabled)
}

// ListThemes returns the available theme names.
func (a *App) ListThemes() ([]string, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListThemes(a.ctx)
}

// GetThemeColors returns a theme's palette (empty name = current theme).
func (a *App) GetThemeColors(name string) (*desktop.ThemeColors, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.GetThemeColors(a.ctx, name)
}

// TelemetryEnabled reports whether local usage analytics are on (opt-in).
func (a *App) TelemetryEnabled() bool {
	return a.enabled((*desktop.API).TelemetryEnabled)
}

// RecordCommand records a command invocation (name only) for local analytics.
// Fire-and-forget: silently ignored until the session is ready or when disabled.
func (a *App) RecordCommand(name string) {
	if api, err := a.api(); err == nil {
		api.RecordCommand(name)
	}
}

// RecordShortcut records a single shortcut keypress for local analytics.
// Fire-and-forget, like RecordCommand.
func (a *App) RecordShortcut(key string) {
	if api, err := a.api(); err == nil {
		api.RecordShortcut(key)
	}
}

// TelemetrySummary returns the local usage-analytics dashboard data for the last
// windowDays (0 → default 30).
func (a *App) TelemetrySummary(windowDays int) (*desktop.TelemetrySummary, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.TelemetrySummary(a.ctx, windowDays)
}

// TelemetryReset deletes all captured telemetry for the active account.
func (a *App) TelemetryReset() error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.TelemetryReset(a.ctx)
}

// ApplyBulkPromptStream applies a prompt across many messages, streaming tokens
// via the "prompt:token" event.
func (a *App) ApplyBulkPromptStream(ids []string, promptID int) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.ApplyBulkPromptStream(a.ctx, ids, promptID, func(tok string) {
		wailsruntime.EventsEmit(a.ctx, promptTokenEvent, tok)
	})
}

// ActionPlanEnabled reports whether the AI inbox action plan is available.
