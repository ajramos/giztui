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

// ChatGPTLogin runs the OAuth (PKCE) flow: it opens the system browser and
// blocks until the callback completes (or errors), persisting the token.
func ChatGPTLogin(ctx context.Context, model string) error {
	_, wait, err := llm.NewChatGPT(model, 0).StartLogin(ctx)
	if err != nil {
		return err
	}
	return wait()
}

// ChatGPTLogout removes the stored ChatGPT subscription token.
func ChatGPTLogout() error {
	return llm.NewChatGPT("", 0).Logout()
}
