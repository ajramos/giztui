package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/llm"
)

// executeLLMCommand handles the :llm command family for subscription-based
// providers that require an interactive OAuth login (currently ChatGPT):
//
//	:llm                     show the active account's engine + login status
//	:llm status              same as bare :llm
//	:llm login chatgpt       start the ChatGPT subscription OAuth (PKCE) flow
//	:llm logout chatgpt      drop the stored ChatGPT tokens
//
// Stateless providers (ollama, bedrock) need no login; the subcommands only
// apply to subscription providers. Credentials are machine-global (one login
// reused by every account that selects the provider), per the design spec.
func (a *App) executeLLMCommand(args []string) {
	if len(args) == 0 || strings.ToLower(args[0]) == "status" {
		a.llmStatus()
		return
	}

	sub := strings.ToLower(args[0])
	provider := "chatgpt"
	if len(args) > 1 {
		provider = strings.ToLower(args[1])
	}

	switch sub {
	case "login":
		a.llmLogin(provider)
	case "logout":
		a.llmLogout(provider)
	default:
		a.showError("Usage: :llm [status] | :llm login chatgpt | :llm logout chatgpt")
	}
}

// llmStatus reports the active account's effective engine and, for
// subscription providers, whether a login token is present.
func (a *App) llmStatus() {
	go func() {
		var accountID string
		if as := a.GetAccountService(); as != nil {
			if acct, err := as.GetActiveAccount(a.ctx); err == nil && acct != nil {
				accountID = acct.ID
			}
		}
		eff := a.Config.EffectiveLLM(accountID)
		provider := eff.Provider
		if provider == "" {
			provider = "ollama"
		}
		if !eff.Enabled || eff.Model == "" {
			a.GetErrorHandler().ShowInfo(a.ctx, "AI: disabled")
			return
		}
		msg := fmt.Sprintf("AI: %s · %s", provider, eff.Model)
		if provider == "chatgpt" {
			if llm.NewChatGPT(eff.Model, 0).LoggedIn() {
				msg += "  ·  logged in (:llm logout chatgpt)"
			} else {
				msg += "  ·  not logged in — run :llm login chatgpt"
			}
		}
		a.GetErrorHandler().ShowInfo(a.ctx, msg)
	}()
}

// llmLogin runs the OAuth login flow for a subscription provider. Only ChatGPT
// is supported today; other providers are stateless and report so.
func (a *App) llmLogin(provider string) {
	if provider != "chatgpt" {
		a.showError(fmt.Sprintf("Provider %q needs no login (only 'chatgpt' uses OAuth)", provider))
		return
	}
	go func() {
		client := llm.NewChatGPT(a.Config.LLM.Model, 0)
		authURL, wait, err := client.StartLogin(a.ctx)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("ChatGPT login failed to start: %v", err))
			return
		}
		// Open the default browser AND copy the URL to the clipboard, so the user
		// can either use the browser that popped up or paste it into a different
		// one/profile.
		_ = llm.OpenBrowser(authURL)
		a.copyToClipboard(authURL)
		a.GetErrorHandler().ShowInfo(a.ctx, "🔗 Opening browser for ChatGPT login (URL also copied to clipboard — paste it in another browser if you prefer)")
		if a.logger != nil {
			a.logger.Printf("LLM: ChatGPT login URL: %s", authURL)
		}
		a.GetErrorHandler().ShowProgress(a.ctx, "ChatGPT login: waiting for the browser callback…")
		if err := wait(); err != nil {
			a.GetErrorHandler().ClearProgress()
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("ChatGPT login failed: %v", err))
			return
		}
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowSuccess(a.ctx, "✓ ChatGPT login complete — subscription ready for accounts using the 'chatgpt' provider")
	}()
}

// llmLogout drops stored credentials for a subscription provider.
func (a *App) llmLogout(provider string) {
	if provider != "chatgpt" {
		a.showError(fmt.Sprintf("Provider %q has no stored login to remove", provider))
		return
	}
	go func() {
		if err := llm.NewChatGPT("", 0).Logout(); err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("ChatGPT logout failed: %v", err))
			return
		}
		a.GetErrorHandler().ShowSuccess(a.ctx, "✓ ChatGPT tokens removed")
	}()
}
