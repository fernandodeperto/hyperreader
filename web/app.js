// html-mcp single-page UI — application script.
//
// T02 adds the documents table + live FTS5 search: on load it fetches
// GET /api/documents and renders name/description/tags (newest-first, as
// returned by the API); a debounced search input drives
// GET /api/documents?q=<encoded> (the real S01 FTS5 search — not a
// client-side filter) and re-renders live. Clearing the search restores the
// full list immediately. Fetch errors surface in the in-UI error region
// (R002).
//
// T03 adds the unsandboxed detail view + Back: clicking a row fetches
// GET /api/documents/{id}/content and renders that document's raw HTML
// full-page in a full-viewport iframe with NO sandbox attribute and no CSP
// (R006), so agent-authored inline scripts and absolute CDN references
// execute as authored. The Back button hides the detail view and restores
// the table with its state (rows + search input) preserved across the
// detail detour — the table DOM is never torn down, only hidden.
//
// All user-authored strings (name/description/tags) are inserted via
// textContent, never innerHTML, so agent-authored markup is rendered as
// inert text in the table. The detail view iframe is the sole place
// document HTML renders live, unsandboxed by design (R006).
//
// S04-T02 adds the live-update client: a native EventSource subscribes to
// GET /api/events and prepends each broadcast "document" event as a new
// top row with no fetch, no reload, and no manual refresh. #live-status
// (data-state="connecting"|"live"|"reconnecting") mirrors the browser's
// EventSource lifecycle so a dropped connection is visible in the UI
// instead of the table silently going stale — the browser's own automatic
// reconnect (another "open" event after the connection drops) drives
// reconnecting -> live without any app.js retry logic. A malformed event
// payload (non-JSON, or JSON that isn't a document-shaped object) is
// logged to the console via console.error and skipped rather than
// thrown, so one bad frame never breaks the page.
(function () {
  "use strict";

  var API = "/api/documents";
  var SEARCH_DEBOUNCE_MS = 250;

  function byId(id) {
    var el = document.getElementById(id);
    if (!el) {
      console.error("html-mcp: missing element #" + id);
    }
    return el;
  }

  // Surface fetch errors in the in-UI error region (R002). Kept stable from
  // T01 so fetch handlers share one error surface.
  window.htmlMcp = {
    showError: function (message) {
      var err = byId("error-message");
      if (err) {
        err.textContent = message || "Something went wrong.";
        err.hidden = false;
      }
    },
    clearError: function () {
      var err = byId("error-message");
      if (err) {
        err.hidden = true;
        err.textContent = "";
      }
    }
  };

  var state = {
    search: "",
    docs: [],
    pending: null, // in-flight AbortController
    debounce: null // debounce timeout id
  };

  // Fetch /api/documents (with optional ?q=) and re-render the table.
  // Aborts any prior in-flight request so a fast-typing user never sees a
  // stale response overwrite a newer one (last-write-wins via abort).
  function fetchDocs(query) {
    if (state.pending) {
      state.pending.abort();
    }
    var ctrl = new AbortController();
    state.pending = ctrl;

    var url = API + (query ? "?q=" + encodeURIComponent(query) : "");
    setLoading(true);

    fetch(url, { signal: ctrl.signal })
      .then(function (res) {
        if (!res.ok) {
          // Drain the body so the error message can include server detail.
          return res.text().then(function (body) {
            throw new Error("HTTP " + res.status + (body ? ": " + body : ""));
          });
        }
        return res.json();
      })
      .then(function (docs) {
        state.docs = Array.isArray(docs) ? docs : [];
        setLoading(false);
        render();
        window.htmlMcp.clearError();
      })
      .catch(function (err) {
        // AbortError is expected when a newer request supersedes this one;
        // leave loading to the newer request and do not surface an error.
        if (err && err.name === "AbortError") {
          return;
        }
        state.docs = [];
        setLoading(false);
        render();
        window.htmlMcp.showError(
          "Failed to load documents: " + (err && err.message ? err.message : err)
        );
      })
      .then(function () {
        if (state.pending === ctrl) {
          state.pending = null;
        }
      });
  }

  function setLoading(loading) {
    var ls = byId("loading-state");
    if (ls) {
      ls.hidden = !loading;
    }
  }

  // Render the current docs into the table. An empty list shows the empty
  // state (worded by whether a search is active); a non-empty list shows
  // the table. All strings are inserted via textContent so agent-authored
  // markup is inert here. Rows carry data-id for the T03 click handler.
  function render() {
    var table = byId("documents-table");
    var empty = byId("empty-state");
    var tbody = table ? table.querySelector("tbody") : null;
    if (!table || !empty || !tbody) {
      return;
    }

    if (state.docs.length === 0) {
      table.hidden = true;
      empty.hidden = false;
      empty.textContent = state.search
        ? "No documents match \u201c" + state.search + "\u201d."
        : "No documents yet.";
      return;
    }

    empty.hidden = true;
    tbody.textContent = "";

    for (var i = 0; i < state.docs.length; i++) {
      var doc = state.docs[i];
      var tr = document.createElement("tr");
      // Stash the id on the row for the detail-view click handler (T03).
      tr.dataset.id = doc.id;
      // Rows are keyboard-focusable + activatable so the detail view is
      // reachable without a mouse (a11y on the primary user loop).
      tr.tabIndex = 0;
      tr.setAttribute("role", "button");
      tr.setAttribute(
        "aria-label",
        "View rendered HTML for " + (doc.name || "document")
      );

      tr.appendChild(cell(doc.name || ""));
      tr.appendChild(cell(doc.description || ""));
      tr.appendChild(cell(doc.tags || ""));

      tbody.appendChild(tr);
    }
    table.hidden = false;
  }

  function cell(text) {
    var td = document.createElement("td");
    td.textContent = text;
    return td;
  }

  // --- Dark mode (S05) ---
  //
  // Single source of truth for the *current* theme is the "data-theme"
  // attribute on <html>, resolved synchronously by the inline script in
  // index.html's <head> (before first paint, avoiding a FOUC) using:
  // stored preference (localStorage) -> OS prefers-color-scheme -> light.
  // This module owns everything *after* that initial resolution: syncing
  // the toggle button's label/aria-pressed, persisting an explicit user
  // choice, and reacting live to OS-level scheme changes for users who
  // have never overridden it. All localStorage access is wrapped in
  // try/catch (same rationale as the inline script: private/sandboxed
  // contexts can throw) so a blocked store degrades to session-only
  // theming instead of breaking the page.
  var THEME_KEY = "html-mcp-theme";

  function getStoredTheme() {
    try {
      var v = window.localStorage.getItem(THEME_KEY);
      return v === "dark" || v === "light" ? v : null;
    } catch (e) {
      return null;
    }
  }

  function storeTheme(theme) {
    try {
      window.localStorage.setItem(THEME_KEY, theme);
    } catch (e) {
      // Ignore: theming still works for this session via the DOM
      // attribute, it just won't persist across reloads.
    }
  }

  function currentTheme() {
    return document.documentElement.dataset.theme === "dark" ? "dark" : "light";
  }

  // applyTheme updates the DOM only (attribute + toggle button state) --
  // it does NOT persist. Used both by explicit user toggles (paired with
  // storeTheme) and by the live OS-preference listener (which must NOT
  // persist, or it would silently convert "no preference" into a sticky
  // stored one).
  function applyTheme(theme) {
    document.documentElement.dataset.theme = theme;
    var btn = byId("theme-toggle");
    if (btn) {
      var isDark = theme === "dark";
      btn.textContent = isDark ? "\u2600\ufe0e Light mode" : "\u263D\ufe0e Dark mode";
      btn.setAttribute("aria-pressed", String(isDark));
    }
  }

  function setTheme(theme) {
    applyTheme(theme);
    storeTheme(theme);
  }

  function toggleTheme() {
    setTheme(currentTheme() === "dark" ? "light" : "dark");
  }

  // Live-follow the OS scheme only while the user has never explicitly
  // chosen a theme (no stored override). Once a user toggles, their choice
  // is sticky across OS changes until they toggle again.
  function watchSystemTheme() {
    if (typeof window.matchMedia !== "function") {
      return;
    }
    var mql = window.matchMedia("(prefers-color-scheme: dark)");
    var onChange = function (evt) {
      if (getStoredTheme() !== null) {
        return;
      }
      applyTheme(evt.matches ? "dark" : "light");
    };
    if (typeof mql.addEventListener === "function") {
      mql.addEventListener("change", onChange);
    } else if (typeof mql.addListener === "function") {
      // Safari < 14 / older engines.
      mql.addListener(onChange);
    }
  }

  // --- Live updates via SSE (S04-T02) ---
  //
  // setLiveStatus mirrors the EventSource connection lifecycle into
  // #live-status via its data-state attribute (and matching visible text),
  // so a dropped connection is visible in the toolbar instead of the table
  // silently going stale.
  function setLiveStatus(newState) {
    var el = byId("live-status");
    if (!el) {
      return;
    }
    el.dataset.state = newState;
    el.textContent =
      newState === "live"
        ? "Live"
        : newState === "reconnecting"
        ? "Reconnecting\u2026"
        : "Connecting\u2026";
  }

  // isValidDocPayload guards against a malformed "document" event payload
  // taking down the page: it must decode to a non-null object with a
  // present id (the field the table keys rows on).
  function isValidDocPayload(doc) {
    return (
      doc !== null &&
      typeof doc === "object" &&
      !Array.isArray(doc) &&
      doc.id !== undefined &&
      doc.id !== null
    );
  }

  // onDocumentEvent handles a broadcast "document" SSE frame: decode its
  // JSON data (the exact documentResponse shape POST/GET already return),
  // then prepend it as a new top row with no fetch and no page reload. A
  // frame that fails to decode, or decodes to something that isn't a
  // document-shaped object, is logged via console.error and skipped —
  // one bad frame must never break the live table.
  function onDocumentEvent(evt) {
    var doc;
    try {
      doc = JSON.parse(evt.data);
    } catch (err) {
      console.error(
        "html-mcp: malformed SSE document payload (invalid JSON), skipping",
        err,
        evt.data
      );
      return;
    }
    if (!isValidDocPayload(doc)) {
      console.error(
        "html-mcp: malformed SSE document payload (not a document object), skipping",
        doc
      );
      return;
    }

    // A search is active: the new document may or may not match it, and
    // this is a create-time broadcast, not a search result, so leave the
    // filtered view untouched rather than injecting a row the current
    // query would not have returned itself.
    if (state.search !== "") {
      return;
    }

    // De-dupe against an id that is already present (e.g. a page that
    // re-fetched between the create response and the broadcast landing).
    for (var i = 0; i < state.docs.length; i++) {
      if (state.docs[i].id === doc.id) {
        return;
      }
    }

    state.docs.unshift(doc);
    render();
  }

  // connectEvents opens the GET /api/events stream. The browser's native
  // EventSource retries automatically on drop (readyState cycles back to
  // CONNECTING then another "open" fires on reconnect), so app.js only
  // needs to mirror state transitions into #live-status — no manual retry
  // loop is implemented or needed here.
  function connectEvents() {
    if (typeof window.EventSource !== "function") {
      // No SSE support: leave the table fetch-driven only and surface that
      // live updates are unavailable rather than claiming a connection.
      setLiveStatus("reconnecting");
      return;
    }

    setLiveStatus("connecting");
    var es = new EventSource("/api/events");

    es.onopen = function () {
      setLiveStatus("live");
    };
    es.onerror = function () {
      // EventSource fires "error" both while retrying and when the retry
      // itself fails to connect; either way the connection is not live.
      setLiveStatus("reconnecting");
    };
    es.addEventListener("document", onDocumentEvent);

    return es;
  }

  // Debounced search handler. Each keystroke resets the timer; only after
  // SEARCH_DEBOUNCE_MS of quiet does the query hit the API. Clearing the
  // input fires immediately (no debounce) so restoring the full list is
  // snappy.
  function onSearchInput(evt) {
    var value = evt.target.value || "";
    state.search = value.trim();

    if (state.debounce) {
      clearTimeout(state.debounce);
    }
    if (state.search === "") {
      fetchDocs("");
      return;
    }
    state.debounce = setTimeout(function () {
      fetchDocs(state.search);
    }, SEARCH_DEBOUNCE_MS);
  }

  // --- Detail view (T03) + auto-resize (M002 / Branch A) ---
  //
  // showDetail fetches GET /api/documents/{id}/content and renders the raw
  // HTML full-page in #detail-frame. The iframe has NO sandbox attribute
  // (R006): agent-authored inline <script> executes and absolute CDN
  // references load as authored. The table view is hidden (not removed) so
  // its state survives the detail detour; showTable restores it untouched.
  //
  // M002 / Branch A: once the content document loads, the iframe's outer
  // height is set to its measured content height (plus a small buffer) so
  // the iframe has no internal scroll surface of its own — the outer page
  // provides the only scrollbar. A ResizeObserver on the content's
  // documentElement re-measures when late-loading fonts/images/CDN scripts
  // change the content height after initial load. Everything is guarded: if
  // contentDocument is unavailable or measurement throws, the CSS
  // min-height fallback (iframe keeps its own scroll) remains in effect.
  // Nothing inside frame.contentDocument is ever mutated (R006) — only read
  // from, and only the outer frame.style.height is written.
  var detailResizeObserver = null;
  var detailFrameLoadHandler = null;
  var FRAME_HEIGHT_BUFFER_PX = 4;

  // measureFrameHeight returns the larger of the iframe content's
  // documentElement.scrollHeight and body.scrollHeight, or null if neither
  // is measurable. It never throws: any access failure returns null so the
  // caller can fall back to the CSS min-height.
  function measureFrameHeight(frame) {
    var doc = frame.contentDocument;
    if (!doc) {
      return null;
    }
    var de = doc.documentElement;
    var body = doc.body;
    var deH = de && typeof de.scrollHeight === "number" ? de.scrollHeight : 0;
    var bodyH =
      body && typeof body.scrollHeight === "number" ? body.scrollHeight : 0;
    var h = Math.max(deH, bodyH);
    return h > 0 ? h : null;
  }

  // applyFrameHeight sets the iframe's outer height to the measured content
  // height (plus a small buffer) so the iframe has no internal scroll
  // surface of its own. Returns true when a height was applied, false when
  // measurement failed (caller keeps the CSS min-height fallback). Only the
  // outer frame.style.height is written — the content document is never
  // touched (R006).
  function applyFrameHeight(frame) {
    var h = measureFrameHeight(frame);
    if (h === null) {
      return false;
    }
    frame.style.height = h + FRAME_HEIGHT_BUFFER_PX + "px";
    return true;
  }

  // watchFrameContent attaches a ResizeObserver to the iframe's content
  // documentElement so late layout changes re-trigger a height update.
  // Returns true if attached, false if unavailable (caller relies on the
  // load-time measurement + CSS fallback). Guarded for same-origin/
  // contentDocument availability and ResizeObserver support.
  function watchFrameContent(frame) {
    if (typeof ResizeObserver !== "function") {
      return false;
    }
    var doc = frame.contentDocument;
    if (!doc || !doc.documentElement) {
      return false;
    }
    var ro = new ResizeObserver(function () {
      // A resize firing after the document has been cleared (e.g. srcdoc
      // reset on Back) must not throw — wrap so the fallback stays intact.
      try {
        applyFrameHeight(frame);
      } catch (err) {
        // Ignore: CSS min-height fallback remains in effect.
      }
    });
    ro.observe(doc.documentElement);
    detailResizeObserver = ro;
    return true;
  }

  // teardownFrameSizer disconnects any active ResizeObserver and removes
  // the load listener for the detail frame, and clears any explicit height
  // so the CSS min-height fallback applies cleanly when no document is
  // loaded. Called on Back and before loading a new document so doc ->
  // back -> doc never stacks observers on detached documents.
  function teardownFrameSizer(frame) {
    if (detailResizeObserver) {
      try {
        detailResizeObserver.disconnect();
      } catch (err) {
        // Ignore.
      }
      detailResizeObserver = null;
    }
    if (detailFrameLoadHandler) {
      try {
        frame.removeEventListener("load", detailFrameLoadHandler);
      } catch (err) {
        // Ignore.
      }
      detailFrameLoadHandler = null;
    }
    frame.style.height = "";
  }

  function showDetail(id) {
    var frame = byId("detail-frame");
    if (!frame) {
      return;
    }

    // Optimistically swap to the detail view so the user sees immediate
    // feedback; the iframe fills once the content resolves.
    setView("detail");
    window.htmlMcp.clearError();

    fetch(API + "/" + encodeURIComponent(id) + "/content")
      .then(function (res) {
        if (!res.ok) {
          return res.text().then(function (body) {
            throw new Error(
              "HTTP " + res.status + (body ? ": " + body : "")
            );
          });
        }
        return res.text();
      })
      .then(function (html) {
        // srcdoc renders the document's HTML as the iframe's own document.
        // With no sandbox attribute the content is same-origin and
        // unsandboxed: inline scripts run and external resources load.
        //
        // Auto-resize (M002): tear down any prior sizer, then attach a
        // load listener that measures the content height and keeps it in
        // sync via a ResizeObserver, so the iframe grows to its content
        // and the outer page provides the only scrollbar. The listener is
        // attached before setting srcdoc so it reliably catches the load.
        teardownFrameSizer(frame);
        detailFrameLoadHandler = function () {
          try {
            applyFrameHeight(frame);
            watchFrameContent(frame);
          } catch (err) {
            // Measurement/observer setup failed: leave the CSS min-height
            // in effect (iframe keeps its own scroll) — graceful fallback.
          }
        };
        frame.addEventListener("load", detailFrameLoadHandler);
        frame.srcdoc = html;
      })
      .catch(function (err) {
        // Render the failure in the detail view itself + surface it in the
        // shared error region (R002). Bounce back to the table so the user
        // is not left looking at a blank detail pane. Tear down the sizer
        // so no stale observer/listener remains on the cleared frame.
        teardownFrameSizer(frame);
        frame.srcdoc = "";
        setView("table");
        window.htmlMcp.showError(
          "Failed to load document: " +
            (err && err.message ? err.message : err)
        );
      });
  }

  // showTable restores the table view (Back button). The table DOM and
  // search input were never torn down — only hidden — so the previously-
  // clicked row is still present and the search input still holds its
  // value (still searchable). The iframe srcdoc is cleared to release the
  // rendered document and stop any scripts/CDN fetches it started. The
  // auto-size sizer is torn down first so no ResizeObserver/load listener
  // outlives the document it was tracking.
  function showTable() {
    var frame = byId("detail-frame");
    if (frame) {
      teardownFrameSizer(frame);
      frame.srcdoc = "";
    }
    setView("table");
  }

  // setView toggles the table/detail sections. The two sections are
  // siblings; only one is visible at a time.
  function setView(view) {
    var table = byId("table-view");
    var detail = byId("detail-view");
    if (table && detail) {
      table.hidden = view !== "table";
      detail.hidden = view !== "detail";
    }
  }

  // onTableClick handles clicks on document rows (event delegation on the
  // tbody so a single listener covers all rows, current and future). A
  // click on a row (or Enter/Space on a keyboard-focused row) opens the
  // detail view for that document.
  function onTableActivate(evt) {
    var target = evt.target;
    // Walk up to the TR — clicks may land on a TD.
    while (target && target.tagName !== "TR") {
      target = target.parentNode;
      if (!target || target.tagName === "TABLE") {
        return;
      }
    }
    if (!target) {
      return;
    }
    var id = target.dataset && target.dataset.id;
    if (!id) {
      return;
    }
    showDetail(id);
  }

  // onTableKeydown lets a keyboard user open the detail view with
  // Enter/Space on a focused row (the rows are role=button + tabIndex=0).
  function onTableKeydown(evt) {
    if (evt.key !== "Enter" && evt.key !== " ") {
      return;
    }
    var target = evt.target;
    if (!target || target.tagName !== "TR" || !target.dataset.id) {
      return;
    }
    evt.preventDefault();
    showDetail(target.dataset.id);
  }

  function init() {
    var search = byId("search");
    if (search) {
      search.addEventListener("input", onSearchInput);
    }

    // Detail-view wiring (T03): row clicks open the unsandboxed view; Back
    // restores the table. Both use event delegation on stable parent nodes.
    var tbody = document.querySelector("#documents-table tbody");
    if (tbody) {
      tbody.addEventListener("click", onTableActivate);
      tbody.addEventListener("keydown", onTableKeydown);
    }
    var back = byId("back-button");
    if (back) {
      back.addEventListener("click", showTable);
    }

    // Dark mode (S05): the inline head script already resolved and applied
    // the initial theme before paint; sync the toggle button's label to
    // that resolved state, wire the click handler, and start following OS
    // scheme changes for users who haven't set an explicit preference.
    applyTheme(currentTheme());
    var themeToggle = byId("theme-toggle");
    if (themeToggle) {
      themeToggle.addEventListener("click", toggleTheme);
    }
    watchSystemTheme();

    fetchDocs("");

    // Live updates (S04-T02): subscribe once at startup, independent of
    // the initial fetch above, so the table is kept live even while the
    // first GET /api/documents request is still in flight.
    connectEvents();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
