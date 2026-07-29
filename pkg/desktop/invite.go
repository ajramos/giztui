package desktop

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	gmail_v1 "google.golang.org/api/gmail/v1"
)

// inviteClient is the subset of *gmail.Client used to detect calendar invites.
type inviteClient interface {
	GetMessage(id string) (*gmail_v1.Message, error)
	GetAttachment(messageID, attachmentID string) ([]byte, string, error)
	ActiveAccountEmail(ctx context.Context) (string, error)
}

// calClient responds to calendar invites. It is satisfied by a thin wrapper over
// internal/calendar.Client, kept as an interface so the API stays testable and
// the internal calendar package doesn't leak into this public package.
type calClient interface {
	// FindEventID resolves a calendar event ID from an iCalUID.
	FindEventID(ctx context.Context, iCalUID string) (string, error)
	// RespondToInvite sets the attendee's response (accepted/declined/tentative).
	RespondToInvite(ctx context.Context, eventID, attendeeEmail, status string) error
}

// RSVPEnabled reports whether calendar RSVP is available (Calendar API wired).
func (a *API) RSVPEnabled() bool { return a.invite != nil && a.cal != nil }

// InviteInfo inspects a message and returns invitation details when it is a
// calendar REQUEST, otherwise an Invite with IsInvite=false.
func (a *API) InviteInfo(ctx context.Context, id string) (*Invite, error) {
	if a.invite == nil {
		return &Invite{}, nil
	}
	msg, err := a.invite.GetMessage(id)
	if err != nil {
		return nil, err
	}
	inv, ok := a.detectInvite(msg)
	inv.IsInvite = ok
	return &inv, nil
}

// RespondInvite finds the calendar event for a message's invite and sets the
// account's attendance status (accepted / declined / tentative).
func (a *API) RespondInvite(ctx context.Context, id, status string) error {
	if a.invite == nil || a.cal == nil {
		return fmt.Errorf("calendar RSVP is not available; re-authorize with Calendar permissions")
	}
	msg, err := a.invite.GetMessage(id)
	if err != nil {
		return err
	}
	inv, ok := a.detectInvite(msg)
	if !ok || inv.UID == "" {
		return fmt.Errorf("no calendar invite found in this message")
	}
	email, err := a.invite.ActiveAccountEmail(ctx)
	if err != nil || strings.TrimSpace(email) == "" {
		return fmt.Errorf("could not determine account email")
	}
	eventID, err := a.cal.FindEventID(ctx, inv.UID)
	if err != nil {
		return rsvpError(err)
	}
	return rsvpError(a.cal.RespondToInvite(ctx, eventID, email, status))
}

// rsvpError turns Google's raw "insufficient authentication scopes" 403 into an
// actionable message. The token was minted without the calendar.events scope
// (older tokens, or a desktop-only auth before that scope was requested), so the
// user must re-authorize — reusing the existing token can't add a scope.
func rsvpError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "ACCESS_TOKEN_SCOPE_INSUFFICIENT") ||
		strings.Contains(msg, "insufficient authentication scopes") ||
		strings.Contains(msg, "Insufficient Permission") {
		return fmt.Errorf("calendar access not granted for this account — re-authorize to enable RSVP (remove %s and restart, or run `giztui --setup`)", "~/.config/giztui/token.json")
	}
	return err
}

// detectInvite walks a message's MIME parts for a text/calendar REQUEST and
// extracts the event fields. Ported from the TUI's detectCalendarInvite.
func (a *API) detectInvite(msg *gmail_v1.Message) (Invite, bool) {
	if msg == nil || msg.Payload == nil {
		return Invite{}, false
	}
	var out Invite
	var found bool
	var walk func(p *gmail_v1.MessagePart)
	walk = func(p *gmail_v1.MessagePart) {
		if p == nil || found {
			return
		}
		mt := strings.ToLower(p.MimeType)
		fn := strings.ToLower(p.Filename)
		isICS := strings.Contains(mt, "text/calendar") ||
			strings.Contains(mt, "application/ics") ||
			(fn != "" && strings.HasSuffix(fn, ".ics")) ||
			(strings.Contains(mt, "application/octet-stream") && strings.HasSuffix(fn, ".ics"))
		if isICS {
			methodReq := false
			for _, h := range p.Headers {
				if strings.EqualFold(h.Name, "Content-Type") &&
					strings.Contains(strings.ToLower(h.Value), "method=request") {
					methodReq = true
					break
				}
			}
			var raw []byte
			if p.Body != nil {
				if p.Body.Data != "" {
					if data, err := base64.URLEncoding.DecodeString(p.Body.Data); err == nil {
						raw = data
					}
				} else if p.Body.AttachmentId != "" {
					if data, _, err := a.invite.GetAttachment(msg.Id, p.Body.AttachmentId); err == nil {
						raw = data
					}
				}
			}
			if len(raw) > 0 {
				s := string(raw)
				if strings.Contains(strings.ToUpper(s), "METHOD:REQUEST") {
					methodReq = true
				}
				out.UID = scanICSField(s, "UID")
				out.Summary = scanICSField(s, "SUMMARY")
				out.Organizer = scanICSField(s, "ORGANIZER")
				out.DtStart = resolveICSDateTime(scanICSField(s, "DTSTART"))
				out.DtEnd = resolveICSDateTime(scanICSField(s, "DTEND"))
			}
			if methodReq {
				found = true
				return
			}
		}
		for _, c := range p.Parts {
			walk(c)
			if found {
				return
			}
		}
	}
	walk(msg.Payload)
	return out, found
}

// scanICSField extracts an iCalendar field value from within the VEVENT block,
// handling folded (continuation) lines. Ported from the TUI.
func scanICSField(s, key string) string {
	lines := strings.Split(s, "\n")
	inVEvent := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "BEGIN:VEVENT" {
			inVEvent = true
			continue
		}
		if line == "END:VEVENT" {
			inVEvent = false
			continue
		}
		if !inVEvent {
			continue
		}
		if strings.HasPrefix(line, key) {
			colonIdx := strings.Index(line, ":")
			if colonIdx > 0 {
				fullValue := strings.TrimSpace(line[colonIdx+1:])
				for j := i + 1; j < len(lines); j++ {
					next := lines[j]
					if len(next) > 0 && (next[0] == ' ' || next[0] == '\t') {
						fullValue += strings.TrimSpace(next)
					} else {
						break
					}
				}
				// For DTSTART/DTEND, preserve the parameter part (e.g.
				// ";TZID=Europe/Madrid:") so the timezone survives to
				// resolveICSDateTime — otherwise the wall clock is shown in the
				// organizer's zone instead of the viewer's.
				if key == "DTSTART" || key == "DTEND" {
					if p := strings.Index(line, ";"); p > 0 && p < colonIdx {
						return line[p:colonIdx] + ":" + fullValue
					}
				}
				return fullValue
			}
		}
	}
	return ""
}

// resolveICSDateTime turns an iCalendar DTSTART/DTEND value — as returned by
// scanICSField, which may carry a ";TZID=Zone:" or ";VALUE=DATE:" parameter —
// into an absolute instant formatted as RFC3339 (UTC), so the frontend can
// render it in the viewer's local timezone (matching Gmail/Calendar). All-day
// (date-only) values and anything unparseable are returned as their raw digits
// for the frontend's wall-clock fallback path.
func resolveICSDateTime(raw string) string {
	if raw == "" {
		return ""
	}
	params, val := "", raw
	if strings.HasPrefix(val, ";") {
		if idx := strings.LastIndex(val, ":"); idx >= 0 {
			params, val = val[:idx], val[idx+1:]
		}
	}
	val = strings.TrimSpace(val)
	// All-day / date-only: leave the raw date for the frontend to show sans time.
	if strings.Contains(strings.ToUpper(params), "VALUE=DATE") || len(val) == 8 {
		return val
	}
	// UTC (trailing Z).
	if strings.HasSuffix(val, "Z") {
		for _, f := range []string{"20060102T150405Z", "20060102T1504Z"} {
			if t, err := time.Parse(f, val); err == nil {
				return t.UTC().Format(time.RFC3339)
			}
		}
		return val
	}
	// Zoned (TZID=Area/City).
	if i := strings.Index(strings.ToUpper(params), "TZID="); i >= 0 {
		zone := params[i+len("TZID="):]
		if j := strings.IndexByte(zone, ';'); j >= 0 {
			zone = zone[:j]
		}
		if loc, err := time.LoadLocation(strings.TrimSpace(zone)); err == nil {
			for _, f := range []string{"20060102T150405", "20060102T1504"} {
				if t, err := time.ParseInLocation(f, val, loc); err == nil {
					return t.UTC().Format(time.RFC3339)
				}
			}
		}
	}
	// Floating (no zone declared): best effort — treat the wall clock as UTC so
	// the frontend still receives an absolute instant. Raw if unparseable.
	for _, f := range []string{"20060102T150405", "20060102T1504"} {
		if t, err := time.Parse(f, val); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return val
}
