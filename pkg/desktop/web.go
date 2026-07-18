package desktop

// GmailWebURL returns the Gmail web URL for a message, or "" when the web
// service is unavailable. The Wails layer opens it in the system browser.
func (a *API) GmailWebURL(messageID string) string {
	if a.web == nil {
		return ""
	}
	return a.web.GenerateGmailWebURL(messageID)
}
