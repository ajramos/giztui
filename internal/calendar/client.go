package calendar

import (
	"context"
	"fmt"
	"strings"

	cal "google.golang.org/api/calendar/v3"
)

// Client wraps the calendar.Service and provides convenience methods for RSVP
type Client struct {
	Service *cal.Service
}

// NewClient creates a new Calendar client
func NewClient(service *cal.Service) *Client {
	return &Client{Service: service}
}

// FindByICalUID finds an event by its iCalUID in the primary calendar
func (c *Client) FindByICalUID(ctx context.Context, iCalUID string) (*cal.Event, error) {
	if c == nil || c.Service == nil {
		return nil, fmt.Errorf("calendar client not initialized")
	}
	if iCalUID == "" {
		return nil, fmt.Errorf("empty iCalUID")
	}
	call := c.Service.Events.List("primary").ICalUID(iCalUID).Context(ctx)
	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list events by iCalUID: %w", err)
	}
	if resp == nil || len(resp.Items) == 0 {
		// Some invites come only as ICS and are not yet on calendar; try Events.Import as a best-effort
		// Note: We keep this method non-destructive; caller may decide to import using raw ICS if available.
		return nil, fmt.Errorf("event not found for iCalUID: %s", iCalUID)
	}
	return resp.Items[0], nil
}

// RespondToInvite updates the attendee responseStatus for the authenticated user
// status must be one of: "accepted", "declined", "tentative" (case-insensitive)
// If sendUpdates is true, organizer and attendees are notified.
func (c *Client) RespondToInvite(ctx context.Context, eventID, attendeeEmail, status string, sendUpdates bool) error {
	if c == nil || c.Service == nil {
		return fmt.Errorf("calendar client not initialized")
	}
	if eventID == "" || attendeeEmail == "" || status == "" {
		return fmt.Errorf("invalid RSVP inputs")
	}
	// Fetch current event to modify attendees
	evt, err := c.Service.Events.Get("primary", eventID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}
	// Normalize status
	var rs string
	switch strings.ToLower(status) {
	case "accepted", "accept", "yes", "y":
		rs = "accepted"
	case "declined", "decline", "no", "n":
		rs = "declined"
	case "tentative", "maybe", "m":
		rs = "tentative"
	default:
		return fmt.Errorf("unknown response status: %s", status)
	}
	// Update or add attendee
	found := false
	for _, a := range evt.Attendees {
		if strings.EqualFold(a.Email, attendeeEmail) || a.Self {
			a.ResponseStatus = rs
			found = true
			break
		}
	}
	if !found {
		att := &cal.EventAttendee{Email: attendeeEmail, ResponseStatus: rs}
		evt.Attendees = append(evt.Attendees, att)
	}
	// Update event
	upd := c.Service.Events.Update("primary", evt.Id, evt).Context(ctx)
	if sendUpdates {
		upd = upd.SendUpdates("all")
	}
	if _, err := upd.Do(); err != nil {
		return fmt.Errorf("failed to update event RSVP: %w", err)
	}
	return nil
}
