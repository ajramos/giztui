package tui

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"

	"github.com/ajramos/giztui/internal/services"
)

// aiJobStatus is the lifecycle state of a tracked AI background job.
type aiJobStatus string

const (
	aiJobRunning  aiJobStatus = "running"
	aiJobDone     aiJobStatus = "done"
	aiJobError    aiJobStatus = "error"
	aiJobCanceled aiJobStatus = "canceled"
)

// aiJob is a tracked bulk-prompt run. The durable result is persisted by the
// service layer (SaveBulkResult); this record tracks the run so the ":jobs"
// picker can browse, re-open, or remove it — the TUI-side mirror of the desktop
// AI-jobs feature.
type aiJob struct {
	id           int
	promptID     int
	promptName   string
	messageCount int
	status       aiJobStatus
	result       *services.BulkPromptResult
	errMsg       string
	createdAt    time.Time
	cancel       context.CancelFunc
}

// aiJobsRegistry is an in-memory, mutex-guarded list of AI jobs owned by the
// App (mirrors bulkState / aiPanelState). Results themselves live in the DB
// cache, so losing the registry on restart only loses the browse list, not data.
type aiJobsRegistry struct {
	mu     sync.RWMutex
	jobs   []*aiJob
	nextID int
}

func newAIJobsRegistry() *aiJobsRegistry { return &aiJobsRegistry{} }

// add registers a job, assigns it an id, and returns that id.
func (r *aiJobsRegistry) add(j *aiJob) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	j.id = r.nextID
	r.jobs = append(r.jobs, j)
	return j.id
}

// update mutates the job with the given id under the lock (no-op if absent).
func (r *aiJobsRegistry) update(id int, fn func(*aiJob)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, j := range r.jobs {
		if j.id == id {
			fn(j)
			return
		}
	}
}

// get returns a copy of the job with the given id, or nil.
func (r *aiJobsRegistry) get(id int) *aiJob {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, j := range r.jobs {
		if j.id == id {
			cp := *j
			return &cp
		}
	}
	return nil
}

// list returns copies of all jobs, newest first.
func (r *aiJobsRegistry) list() []*aiJob {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*aiJob, 0, len(r.jobs))
	for _, j := range r.jobs {
		cp := *j
		out = append(out, &cp)
	}
	sort.SliceStable(out, func(i, k int) bool { return out[i].createdAt.After(out[k].createdAt) })
	return out
}

// remove drops the job with the given id.
func (r *aiJobsRegistry) remove(id int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, j := range r.jobs {
		if j.id == id {
			r.jobs = append(r.jobs[:i], r.jobs[i+1:]...)
			return
		}
	}
}

// clearFinished drops every job that is no longer running.
func (r *aiJobsRegistry) clearFinished() {
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := make([]*aiJob, 0, len(r.jobs))
	for _, j := range r.jobs {
		if j.status == aiJobRunning {
			kept = append(kept, j)
		}
	}
	r.jobs = kept
}

// aiJobStatusLabel renders a short status glyph+word for a job row.
func aiJobStatusLabel(s aiJobStatus) string {
	switch s {
	case aiJobRunning:
		return "⏳ running"
	case aiJobDone:
		return "✓ done"
	case aiJobError:
		return "✗ error"
	case aiJobCanceled:
		return "⊘ canceled"
	}
	return string(s)
}

// humanizeJobAge renders a compact relative age ("12s", "3m", "2h").
func humanizeJobAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(d.Hours()))
}

// openAIJobsPicker opens (or toggles closed) the AI background-jobs side panel:
// a keyboard-navigable list of jobs. Enter re-opens a finished job's result in
// the AI panel, 'd' removes a job, 'c' clears finished ones, Esc closes.
func (a *App) openAIJobsPicker() {
	// Everything (including the toggle-close) runs on the UI goroutine so callers
	// can invoke this from a `go` in a key/command handler safely.
	a.QueueUpdateDraw(func() {
		if a.isAIJobsPickerActive() {
			a.closeAIJobsPicker()
			return
		}
		a.buildAIJobsPanel()
	})
}

// buildAIJobsPanel (re)builds and mounts the jobs list in the shared side-panel
// slot. Must run on the UI goroutine (called from QueueUpdateDraw or a key
// handler). Re-callable to refresh after remove/clear.
func (a *App) buildAIJobsPanel() {
	jobs := a.aiJobs.list()
	colors := a.GetComponentColors("ai")
	bg := colors.Background.Color()

	list := tview.NewList().ShowSecondaryText(true)
	list.SetBackgroundColor(bg)
	list.SetMainTextColor(colors.Text.Color())
	list.SetSecondaryTextColor(a.GetComponentColors("general").Text.Color())
	list.SetSelectedTextColor(bg)
	list.SetSelectedBackgroundColor(colors.Accent.Color())

	if len(jobs) == 0 {
		list.AddItem("No AI jobs yet", "Run a bulk prompt (:prompt in bulk mode) to create one", 0, nil)
	} else {
		for _, j := range jobs {
			jobID := j.id
			main := fmt.Sprintf("%s  %s", aiJobStatusLabel(j.status), j.promptName)
			secondary := fmt.Sprintf("%d messages · %s", j.messageCount, humanizeJobAge(j.createdAt))
			if j.status == aiJobError && j.errMsg != "" {
				secondary = j.errMsg
			}
			list.AddItem(main, secondary, 0, func() { a.openAIJob(jobID) })
		}
	}

	list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch {
		case e.Key() == tcell.KeyEscape:
			a.closeAIJobsPicker()
			return nil
		case e.Rune() == 'd' || e.Rune() == 'D':
			idx := list.GetCurrentItem()
			if idx >= 0 && idx < len(jobs) {
				a.aiJobs.remove(jobs[idx].id)
				a.buildAIJobsPanel()
			}
			return nil
		case e.Rune() == 'c' || e.Rune() == 'C':
			a.aiJobs.clearFinished()
			a.buildAIJobsPanel()
			return nil
		}
		return e
	})

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(bg)
	container.SetBorder(true)
	container.SetBorderColor(colors.Border.Color())
	container.SetTitle(" 🤖 AI Jobs ")
	container.SetTitleColor(colors.Title.Color())
	container.AddItem(list, 0, 1, true)

	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Enter open · d delete · c clear finished · Esc close ")
	footer.SetTextColor(a.GetComponentColors("general").Text.Color())
	footer.SetBackgroundColor(bg)
	container.AddItem(footer, 1, 0, false)

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.SetBackgroundColor(bg)
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.markFocus("labels")
	a.setActivePicker(PickerAIJobs)
	a.SetFocus(list)
}

// closeAIJobsPicker collapses the jobs panel and restores focus.
func (a *App) closeAIJobsPicker() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok && a.labelsView != nil {
		split.ResizeItem(a.labelsView, 0, 0)
	}
	a.setActivePicker(PickerNone)
	a.restoreFocusAfterModal()
}

// openAIJob re-opens a job in the AI panel: a finished job renders its cached
// result, a running one just reports it's still streaming, an errored one shows
// the error.
func (a *App) openAIJob(id int) {
	job := a.aiJobs.get(id)
	a.closeAIJobsPicker()
	if job == nil {
		return
	}
	switch job.status {
	case aiJobRunning:
		go a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Job '%s' is still running…", job.promptName))
	case aiJobError:
		go a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Job '%s' failed: %s", job.promptName, job.errMsg))
	case aiJobCanceled:
		go a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Job '%s' was canceled", job.promptName))
	default:
		if job.result != nil {
			a.QueueUpdateDraw(func() { a.renderBulkJobResult(job) })
		}
	}
}

// renderBulkJobResult shows a finished job's stored result in the AI panel,
// matching the live bulk-prompt final render. Must run on the UI goroutine.
func (a *App) renderBulkJobResult(job *aiJob) {
	if job.result == nil || a.aiSummaryView == nil {
		return
	}
	if !a.aiPanel.visible.Load() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.aiSummaryView, 0, 1)
		}
		a.aiPanel.visible.Store(true)
	}
	a.aiPanel.inPromptMode = true
	a.aiSummaryView.SetTitle(fmt.Sprintf(" 🤖 Bulk: %s (%d messages) ", job.promptName, job.result.MessageCount))
	meta := fmt.Sprintf("🤖 Bulk Prompt Result: %s\n\n", job.promptName)
	meta += fmt.Sprintf("📊 Messages Processed: %d\n", job.result.MessageCount)
	meta += fmt.Sprintf("⏰ Processing Time: %v\n", job.result.Duration)
	meta += fmt.Sprintf("💾 From Cache: %v\n\n", job.result.FromCache)
	meta += "📝 Analysis:\n"
	a.aiSummaryView.SetText(meta + a.renderPromptResult(job.result.Summary))
	a.aiSummaryView.ScrollToBeginning()
	a.SetFocus(a.aiSummaryView)
	a.markFocus("summary")
}
