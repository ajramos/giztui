module github.com/ajramos/giztui-desktop

go 1.25.11

require (
	github.com/ajramos/giztui v0.0.0
	github.com/wailsapp/wails/v2 v2.10.2
)

// The desktop app lives in a nested module so the Wails/CGO toolchain never
// interferes with the main module's `go build ./...` or `make test`. It reuses
// GizTUI's business logic through the public pkg/desktop adapter via this
// replace directive.
replace github.com/ajramos/giztui => ../
