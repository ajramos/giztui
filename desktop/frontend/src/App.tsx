import { useCallback, useEffect, useRef, useState } from "react";
import {
  backend,
  isWails,
  type Attachment,
  type MessageDetail,
  type MessageSummary,
} from "./api";
import Compose, { type ComposeInit } from "./Compose";
import LabelsPicker from "./LabelsPicker";
import Help from "./Help";

const PAGE_SIZE = 50;

export default function App() {
  const [account, setAccount] = useState("");
  const [initError, setInitError] = useState("");
  const [messages, setMessages] = useState<MessageSummary[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<MessageDetail | null>(null);
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [query, setQuery] = useState("");
  const [activeQuery, setActiveQuery] = useState("");
  const [nextToken, setNextToken] = useState("");
  const [loadingList, setLoadingList] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [aiEnabled, setAiEnabled] = useState(false);
  const [summary, setSummary] = useState<string | null>(null);
  const [summarizing, setSummarizing] = useState(false);
  const [compose, setCompose] = useState<ComposeInit | null>(null);
  const [labelsFor, setLabelsFor] = useState<string | null>(null);
  const [showHelp, setShowHelp] = useState(false);
  const [toast, setToast] = useState("");

  const searchRef = useRef<HTMLInputElement>(null);
  const gPressedAt = useRef(0);

  const showToast = useCallback((msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(""), 2500);
  }, []);

  const load = useCallback(async (q: string) => {
    setLoadingList(true);
    setError("");
    setActiveQuery(q);
    try {
      const list = q
        ? await backend.Search(q, "", PAGE_SIZE)
        : await backend.ListInbox("", PAGE_SIZE);
      setMessages(list.messages ?? []);
      setNextToken(list.nextPageToken ?? "");
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingList(false);
    }
  }, []);

  const loadMore = useCallback(async () => {
    if (!nextToken || loadingMore) return;
    setLoadingMore(true);
    try {
      const list = activeQuery
        ? await backend.Search(activeQuery, nextToken, PAGE_SIZE)
        : await backend.ListInbox(nextToken, PAGE_SIZE);
      setMessages((prev) => [...prev, ...(list.messages ?? [])]);
      setNextToken(list.nextPageToken ?? "");
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingMore(false);
    }
  }, [nextToken, loadingMore, activeQuery]);

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
      void load("");
    })();
  }, [load]);

  const openMessage = useCallback(async (m: MessageSummary) => {
    setSelectedId(m.id);
    setLoadingDetail(true);
    setError("");
    setSummary(null);
    setAttachments([]);
    try {
      const d = await backend.GetMessage(m.id);
      setDetail(d);
      void backend
        .ListAttachments(m.id)
        .then(setAttachments)
        .catch(() => undefined);
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
          showToast("Archived");
        } else if (action === "trash") {
          await backend.Trash(id);
          removeFromList(id);
          showToast("Moved to trash");
        } else if (action === "read") {
          await backend.MarkRead(id);
          setMessages((prev) =>
            prev.map((x) => (x.id === id ? { ...x, unread: false } : x)),
          );
          setDetail((d) => (d && d.id === id ? { ...d, unread: false } : d));
        } else if (action === "unread") {
          await backend.MarkUnread(id);
          setMessages((prev) =>
            prev.map((x) => (x.id === id ? { ...x, unread: true } : x)),
          );
          setDetail((d) => (d && d.id === id ? { ...d, unread: true } : d));
        }
      } catch (e) {
        setError(String(e));
      } finally {
        setBusy(false);
      }
    },
    [removeFromList, showToast],
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

  const replyInit = (d: MessageDetail): ComposeInit => ({
    mode: "reply",
    originalId: d.id,
    to: d.from,
  });

  const forwardInit = (d: MessageDetail): ComposeInit => ({
    mode: "new",
    subject: d.subject.startsWith("Fwd:") ? d.subject : `Fwd: ${d.subject}`,
    body: `\n\n---------- Forwarded message ----------\nFrom: ${d.from}\nDate: ${d.date}\nSubject: ${d.subject}\nTo: ${d.to}\n\n${d.plainText}`,
  });

  const downloadAttachment = useCallback(
    async (att: Attachment) => {
      if (!detail) return;
      setBusy(true);
      try {
        const path = await backend.DownloadAttachment(
          detail.id,
          att.attachmentId,
          att.filename,
        );
        showToast(`Saved to ${path}`);
      } catch (e) {
        setError(String(e));
      } finally {
        setBusy(false);
      }
    },
    [detail, showToast],
  );

  // Keyboard shortcuts mirroring the GizTUI TUI defaults.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement | null)?.tagName;
      const typing = tag === "INPUT" || tag === "TEXTAREA";

      // Modals own their keys; the global handler only helps them close.
      if (showHelp) {
        if (e.key === "Escape" || e.key === "?") {
          setShowHelp(false);
          e.preventDefault();
        }
        return;
      }
      if (labelsFor) {
        if (e.key === "Escape") setLabelsFor(null);
        return;
      }
      if (compose) return;

      if (typing) {
        if (e.key === "Escape") (e.target as HTMLElement).blur();
        return;
      }

      const idx = selectedId
        ? messages.findIndex((m) => m.id === selectedId)
        : -1;
      const openAt = (i: number) => {
        if (i >= 0 && i < messages.length) void openMessage(messages[i]);
      };

      // Prevent handled shortcut keys from also typing into freshly-focused
      // inputs (e.g. 'l' opening the labels picker must not seed its filter).
      const HANDLED = "jkGgNRadtlcrfys/?";
      if (
        HANDLED.includes(e.key) ||
        e.key === "ArrowDown" ||
        e.key === "ArrowUp" ||
        e.key === "Enter"
      ) {
        e.preventDefault();
      }

      switch (e.key) {
        case "j":
        case "ArrowDown":
          openAt(Math.min(messages.length - 1, idx + 1) || 0);
          break;
        case "k":
        case "ArrowUp":
          openAt(Math.max(0, idx <= 0 ? 0 : idx - 1));
          break;
        case "Enter":
          if (idx >= 0) openAt(idx);
          break;
        case "G":
          openAt(messages.length - 1);
          break;
        case "g": {
          const now = Date.now();
          if (now - gPressedAt.current < 500) {
            openAt(0);
            gPressedAt.current = 0;
          } else {
            gPressedAt.current = now;
          }
          break;
        }
        case "N":
          void loadMore();
          break;
        case "R":
          void load(activeQuery);
          break;
        case "Escape":
          if (detail) {
            setSelectedId(null);
            setDetail(null);
          }
          break;
        case "a":
          if (detail) void doAction("archive", detail.id);
          break;
        case "d":
          if (detail) void doAction("trash", detail.id);
          break;
        case "t":
          if (detail) void doAction(detail.unread ? "read" : "unread", detail.id);
          break;
        case "l":
          if (detail) setLabelsFor(detail.id);
          break;
        case "c":
          setCompose({ mode: "new" });
          break;
        case "r":
          if (detail) setCompose(replyInit(detail));
          break;
        case "f":
          if (detail) setCompose(forwardInit(detail));
          break;
        case "y":
          if (detail && aiEnabled) void summarize(detail.id);
          break;
        case "s":
        case "/":
          e.preventDefault();
          searchRef.current?.focus();
          break;
        case "?":
          setShowHelp(true);
          break;
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [
    messages,
    selectedId,
    detail,
    aiEnabled,
    compose,
    labelsFor,
    showHelp,
    activeQuery,
    openMessage,
    doAction,
    summarize,
    load,
    loadMore,
  ]);

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
            const q = query.trim();
            searchRef.current?.blur();
            void load(q);
          }}
        >
          <input
            ref={searchRef}
            type="text"
            placeholder="Search mail (press / or s) — Gmail operators supported: from:, has:attachment…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <button type="submit">Search</button>
          {activeQuery && (
            <button
              type="button"
              className="ghost"
              onClick={() => {
                setQuery("");
                void load("");
              }}
            >
              Clear
            </button>
          )}
        </form>
        <div className="account">
          {!isWails() && <span className="badge">mock</span>}
          <span className="email">{account}</span>
          <button onClick={() => setCompose({ mode: "new" })} title="Compose (c)">
            Compose
          </button>
          <button className="ghost" onClick={() => setShowHelp(true)} title="Shortcuts (?)">
            ?
          </button>
          <button
            className="ghost"
            onClick={() => void load(activeQuery)}
            title="Refresh (R)"
          >
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
            <>
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
              {nextToken && (
                <button
                  className="load-more"
                  disabled={loadingMore}
                  onClick={() => void loadMore()}
                >
                  {loadingMore ? "Loading…" : "Load more (N)"}
                </button>
              )}
            </>
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
                  <button onClick={() => setCompose(replyInit(detail))}>
                    Reply
                  </button>
                  <button
                    className="ghost"
                    onClick={() => setCompose(forwardInit(detail))}
                  >
                    Forward
                  </button>
                  <button className="ghost" onClick={() => setLabelsFor(detail.id)}>
                    Labels
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
                {attachments.length > 0 && (
                  <div className="attach-bar">
                    {attachments.map((att) => (
                      <button
                        key={att.attachmentId}
                        className="attach-chip"
                        disabled={busy}
                        title={`${att.mimeType} · ${formatSize(att.size)}`}
                        onClick={() => void downloadAttachment(att)}
                      >
                        📎 {att.filename}
                        <span className="attach-size">{formatSize(att.size)}</span>
                      </button>
                    ))}
                  </div>
                )}
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
              <p className="muted">
                Press <kbd>?</kbd> for keyboard shortcuts.
              </p>
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
      {labelsFor && (
        <LabelsPicker
          messageId={labelsFor}
          onClose={() => setLabelsFor(null)}
          onChanged={() => {
            if (selectedId === labelsFor) void openMessage({ id: labelsFor } as MessageSummary);
          }}
        />
      )}
      {showHelp && <Help onClose={() => setShowHelp(false)} />}
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

function formatSize(bytes: number): string {
  if (bytes <= 0) return "";
  const units = ["B", "KB", "MB", "GB"];
  let n = bytes;
  let u = 0;
  while (n >= 1024 && u < units.length - 1) {
    n /= 1024;
    u++;
  }
  return `${n.toFixed(u === 0 ? 0 : 1)} ${units[u]}`;
}
