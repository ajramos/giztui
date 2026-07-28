import { backend } from "./api";

// The pre-inbox screens: first-run onboarding (missing credentials.json),
// a fatal init-error screen, and the "connecting / sign-in" screen. Pure
// presentational — all state + the retry/import actions stay in App and are
// passed in. App renders <StartupScreens/> whenever one of these applies;
// it returns null once the app is connected so the inbox takes over.
export default function StartupScreens(props: {
  needCreds: boolean;
  initError: string;
  connecting: boolean;
  credsPath: string;
  importErr: string;
  importing: boolean;
  authUrl: string;
  importCreds: () => Promise<void>;
  retryInit: () => Promise<void>;
}) {
  const {
    needCreds, initError, connecting, credsPath, importErr, importing, authUrl,
    importCreds, retryInit,
  } = props;

  if (needCreds) {
    return (
      <div className="fatal onboarding">
        <span className="logo" aria-hidden="true">
          ✦
        </span>
        <h1>Welcome to GizTUI Desktop</h1>
        <p className="fatal-msg">
          To connect to Gmail, GizTUI needs your own Google API credentials — a
          one-time <code>credentials.json</code> (an OAuth client you create in
          Google Cloud). Your email never passes through anyone else's servers.
        </p>
        <ol className="onboarding-steps">
          <li>
            In the Google Cloud Console, <b>enable the Gmail API</b> and create
            an <b>OAuth client ID</b> of type <b>Desktop app</b>.
          </li>
          <li>
            <b>Download</b> the client's <code>credentials.json</code>.
          </li>
          <li>
            Click <b>Choose credentials.json…</b> below to import it (GizTUI
            copies it to <code>{credsPath || "~/.config/giztui/credentials.json"}</code>),
            then sign in.
          </li>
        </ol>
        {importErr && <p className="fatal-msg onboarding-err">{importErr}</p>}
        <div className="signin-actions">
          <button
            className="primary"
            disabled={importing}
            onClick={() => void importCreds()}
          >
            {importing ? "Importing…" : "Choose credentials.json…"}
          </button>
          <button
            onClick={() =>
              void backend.OpenURL(
                "https://github.com/ajramos/giztui/blob/main/docs/GETTING_STARTED.md#gmail-api-setup",
              )
            }
          >
            Open the setup guide
          </button>
          <button disabled={importing} onClick={() => void retryInit()}>
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (initError) {
    return (
      <div className="fatal">
        <h1>GizTUI Desktop</h1>
        <p className="fatal-msg">Could not start a Gmail session:</p>
        <pre>{initError}</pre>
        <p className="hint">
          Make sure GizTUI is configured (run <code>giztui --setup</code>) and
          that <code>~/.config/giztui/</code> holds valid credentials and token.
        </p>
        <div className="signin-actions">
          <button className="primary" onClick={() => void retryInit()}>
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (connecting) {
    return (
      <div className="connecting">
        <span className="logo">✦</span>
        <h1>GizTUI Desktop</h1>
        {authUrl ? (
          <div className="signin">
            <p>Sign in to your Google account to continue.</p>
            <p className="muted">
              We opened your browser to grant access. Once you approve, this
              window continues automatically.
            </p>
            <div className="signin-actions">
              <button
                className="primary"
                onClick={() => void backend.OpenAuthURL()}
              >
                Open sign-in in browser
              </button>
              <button
                onClick={() => void navigator.clipboard?.writeText(authUrl)}
              >
                Copy link
              </button>
            </div>
            <p className="muted signin-url">{authUrl}</p>
          </div>
        ) : (
          <p className="muted">Connecting to Gmail…</p>
        )}
      </div>
    );
  }

  return null;
}
