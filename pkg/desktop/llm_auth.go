package desktop

import (
	"context"

	"github.com/ajramos/giztui/internal/llm"
)

// LLM subscription-auth helpers exposed to the desktop App (the nested Wails
// module cannot import internal/ directly). These wrap the ChatGPT subscription
// provider's machine-global login, mirroring the TUI's ":llm" commands. The
// token is shared by every account that selects the "chatgpt" provider.

// ChatGPTLoggedIn reports whether a stored ChatGPT subscription token exists.
func ChatGPTLoggedIn() bool {
	return llm.NewChatGPT("", 0).LoggedIn()
}

// ChatGPTStartLogin begins the OAuth (PKCE) flow and returns the authorization
// URL plus a wait func that blocks until the browser callback completes and the
// token is persisted. The desktop App copies the URL to the clipboard (so the
// user can paste it into whatever browser they want) and then calls wait().
func ChatGPTStartLogin(ctx context.Context, model string) (authURL string, wait func() error, err error) {
	return llm.NewChatGPT(model, 0).StartLogin(ctx)
}

// ChatGPTLogout removes the stored ChatGPT subscription token.
func ChatGPTLogout() error {
	return llm.NewChatGPT("", 0).Logout()
}

// OpenLoginBrowser opens the login URL in the system browser — a fallback for
// when copying the URL to the clipboard fails.
func OpenLoginBrowser(url string) error {
	return llm.OpenBrowser(url)
}
