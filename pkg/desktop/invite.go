package desktop

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

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
		return err
	}
	return a.cal.RespondToInvite(ctx, eventID, email, status)
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
				out.DtStart = scanICSField(s, "DTSTART")
				out.DtEnd = scanICSField(s, "DTEND")
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
				return fullValue
			}
		}
	}
	return ""
}
