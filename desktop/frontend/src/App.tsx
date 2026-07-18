import { useCallback, useEffect, useState } from "react";
import {
  backend,
  isWails,
  type MessageDetail,
  type MessageSummary,
} from "./api";
import Compose, { type ComposeInit } from "./Compose";

const PAGE_SIZE = 50;

export default function App() {
  const [account, setAccount] = useState("");
  const [initError, setInitError] = useState("");
  const [messages, setMessages] = useState<MessageSummary[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<MessageDetail | null>(null);
  const [query, setQuery] = useState("");
  const [loadingList, setLoadingList] = useState(false);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [aiEnabled, setAiEnabled] = useState(false);
  const [summary, setSummary] = useState<string | null>(null);
  const [summarizing, setSummarizing] = useState(false);
  const [compose, setCompose] = useState<ComposeInit | null>(null);
  const [toast, setToast] = useState("");

  const loadInbox = useCallback(async () => {
    setLoadingList(true);
    setError("");
    try {
      const list = await backend.ListInbox("", PAGE_SIZE);
      setMessages(list.messages ?? []);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingList(false);
    }
  }, []);

  const runSearch = useCallback(async () => {
    const q = query.trim();
    if (!q) {
      void loadInbox();
      return;
    }
    setLoadingList(true);
    setError("");
    try {
      const list = await backend.Search(q, "", PAGE_SIZE);
      setMessages(list.messages ?? []);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingList(false);
    }
  }, [query, loadInbox]);

  useEffect(() => {
    void (async () => {
      try {
        const ie = await backend.InitError();
        if (ie) {
          setInitError(ie);
          return;
        }
      } catch {
        /* mock backend never errors here */
      }
      try {
        setAccount(await backend.AccountEmail());
      } catch {
        /* non-fatal */
      }
      try {
        setAiEnabled(await backend.AIEnabled());
      } catch {
        /* non-fatal */
      }
      void loadInbox();
    })();
  }, [loadInbox]);

  const showToast = useCallback((msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(""), 2500);
  }, []);

  const openMessage = useCallback(async (m: MessageSummary) => {
    setSelectedId(m.id);
    setLoadingDetail(true);
    setError("");
    setSummary(null);
    try {
      const d = await backend.GetMessage(m.id);
      setDetail(d);
      if (m.unread) {
        void backend.MarkRead(m.id).catch(() => undefined);
        setMessages((prev) =>
          prev.map((x) => (x.id === m.id ? { ...x, unread: false } : x)),
        );
      }
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingDetail(false);
    }
  }, []);

  const removeFromList = useCallback(
    (id: string) => {
      setMessages((prev) => prev.filter((x) => x.id !== id));
      if (selectedId === id) {
        setSelectedId(null);
        setDetail(null);
      }
    },
    [selectedId],
  );

  const doAction = useCallback(
    async (action: "archive" | "trash" | "read" | "unread", id: string) => {
      setBusy(true);
      setError("");
      try {
        if (action === "archive") {
          await backend.Archive(id);
          removeFromList(id);
        } else if (action === "trash") {
          await backend.Trash(id);
          removeFromList(id);
        } else if (action === "read") {
          await backend.MarkRead(id);
          setMessages((prev) =>
            prev.map((x) => (x.id === id ? { ...x, unread: false } : x)),
          );
        } else if (action === "unread") {
          await backend.MarkUnread(id);
          setMessages((prev) =>
            prev.map((x) => (x.id === id ? { ...x, unread: true } : x)),
          );
        }
      } catch (e) {
        setError(String(e));
      } finally {
        setBusy(false);
      }
    },
    [removeFromList],
  );

  const summarize = useCallback(async (id: string) => {
    setSummarizing(true);
    setSummary(null);
    setError("");
    try {
      setSummary(await backend.Summarize(id));
    } catch (e) {
      setError(String(e));
    } finally {
      setSummarizing(false);
    }
  }, []);

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
      </div>
    );
  }

  return (
    <div className="app">
      <header className="topbar">
        <div className="brand">
          <span className="logo">✦</span> GizTUI
          <span className="subtitle">Desktop</span>
        </div>
        <form
          className="searchbox"
          onSubmit={(e) => {
            e.preventDefault();
            void runSearch();
          }}
        >
          <input
            type="text"
            placeholder="Search mail (Gmail operators supported: from:, has:attachment…)"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <button type="submit">Search</button>
          {query && (
            <button
              type="button"
              className="ghost"
              onClick={() => {
                setQuery("");
                void loadInbox();
              }}
            >
              Clear
            </button>
          )}
        </form>
        <div className="account">
          {!isWails() && <span className="badge">mock</span>}
          <span className="email">{account}</span>
          <button onClick={() => setCompose({ mode: "new" })} title="New message">
            Compose
          </button>
          <button className="ghost" onClick={() => void loadInbox()} title="Refresh">
            ⟳
          </button>
        </div>
      </header>

      {error && <div className="error-banner">{error}</div>}
      {toast && <div className="toast">{toast}</div>}

      <div className="body">
        <aside className="list">
          {loadingList ? (
            <div className="placeholder">Loading…</div>
          ) : messages.length === 0 ? (
            <div className="placeholder">No messages</div>
          ) : (
            <ul>
              {messages.map((m) => (
                <li
                  key={m.id}
                  className={
                    "row" +
                    (m.id === selectedId ? " selected" : "") +
                    (m.unread ? " unread" : "")
                  }
                  onClick={() => void openMessage(m)}
                >
                  <div className="row-top">
                    <span className="from">{displayName(m.from)}</span>
                    <span className="date">{formatDate(m.date)}</span>
                  </div>
                  <div className="subject">{m.subject || "(no subject)"}</div>
                  <div className="snippet">{m.snippet}</div>
                  {m.labels.length > 0 && (
                    <div className="labels">
                      {m.labels.map((l) => (
                        <span key={l} className="label-chip">
                          {l}
                        </span>
                      ))}
                    </div>
                  )}
                </li>
              ))}
            </ul>
          )}
        </aside>

        <main className="reader">
          {detail ? (
            <>
              <div className="reader-head">
                <h2>{detail.subject || "(no subject)"}</h2>
                <div className="meta">
                  <div>
                    <strong>{displayName(detail.from)}</strong>{" "}
                    <span className="muted">{emailAddr(detail.from)}</span>
                  </div>
                  <div className="muted">to {detail.to}</div>
                  <div className="muted">{formatFull(detail.date)}</div>
                </div>
                <div className="actions">
                  <button
                    onClick={() =>
                      setCompose({
                        mode: "reply",
                        originalId: detail.id,
                        to: detail.from,
                      })
                    }
                  >
                    Reply
                  </button>
                  {aiEnabled && (
                    <button
                      className="ghost"
                      disabled={summarizing}
                      onClick={() => void summarize(detail.id)}
                    >
                      {summarizing ? "Summarizing…" : "✦ Summarize"}
                    </button>
                  )}
                  <button
                    disabled={busy}
                    onClick={() => void doAction("archive", detail.id)}
                  >
                    Archive
                  </button>
                  <button
                    disabled={busy}
                    className="danger"
                    onClick={() => void doAction("trash", detail.id)}
                  >
                    Trash
                  </button>
                  <button
                    disabled={busy}
                    className="ghost"
                    onClick={() =>
                      void doAction(detail.unread ? "read" : "unread", detail.id)
                    }
                  >
                    Mark {detail.unread ? "read" : "unread"}
                  </button>
                </div>
              </div>
              <div className="reader-body">
                {(summarizing || summary) && (
                  <div className="summary-panel">
                    <div className="summary-head">
                      <span>✦ AI summary</span>
                      {summary && (
                        <button
                          className="ghost tiny"
                          onClick={() => setSummary(null)}
                        >
                          dismiss
                        </button>
                      )}
                    </div>
                    {summarizing ? (
                      <div className="muted">Generating…</div>
                    ) : (
                      <pre className="summary-text">{summary}</pre>
                    )}
                  </div>
                )}
                {loadingDetail ? (
                  <div className="placeholder">Loading…</div>
                ) : (
                  <pre className="plain">{detail.plainText || "(empty body)"}</pre>
                )}
              </div>
            </>
          ) : (
            <div className="empty-reader">
              <p>Select a message to read it here.</p>
            </div>
          )}
        </main>
      </div>

      {compose && (
        <Compose
          init={compose}
          onClose={() => setCompose(null)}
          onSent={(msg) => {
            setCompose(null);
            showToast(msg);
          }}
        />
      )}
    </div>
  );
}

// --- helpers -----------------------------------------------------------------

function displayName(from: string): string {
  const m = from.match(/^\s*"?([^"<]+?)"?\s*</);
  if (m) return m[1].trim();
  return from.split("@")[0] || from;
}

function emailAddr(from: string): string {
  const m = from.match(/<([^>]+)>/);
  return m ? m[1] : from;
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const now = new Date();
  const sameDay = d.toDateString() === now.toDateString();
  return sameDay
    ? d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    : d.toLocaleDateString([], { month: "short", day: "numeric" });
}

function formatFull(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString();
}
