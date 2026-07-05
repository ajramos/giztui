package gmail

import (
	gmail "google.golang.org/api/gmail/v1"
)

// CreateFilter creates a server-side Gmail filter (Settings → Filters) that applies
// action to FUTURE incoming mail matching query. Returns the created filter's ID.
// Requires the gmail.settings.basic OAuth scope — older tokens without it fail here
// and the user must re-authorize.
func (c *Client) CreateFilter(query string, action *gmail.FilterAction) (string, error) {
	user := "me"
	f := &gmail.Filter{
		Criteria: &gmail.FilterCriteria{Query: query},
		Action:   action,
	}
	created, err := c.Service.Users.Settings.Filters.Create(user, f).Do()
	if err != nil {
		return "", err
	}
	return created.Id, nil
}

// DeleteFilter removes a server-side Gmail filter by ID.
func (c *Client) DeleteFilter(id string) error {
	user := "me"
	return c.Service.Users.Settings.Filters.Delete(user, id).Do()
}
