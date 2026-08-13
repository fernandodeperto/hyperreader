// HyperReader single-page UI: application script.
//
// Fetches GET /api/pages on load and renders name/description
// (most-recently-changed-first, as returned by the API); a debounced search
// input drives GET /api/pages?q=<encoded> (the real FTS5 search — not a
// client-side filter) and re-renders live. Clearing the search restores the
// full list immediately. Fetch errors surface in the in-UI error region
// (R002).
//
// Activating a row opens the page's raw rendered HTML in a new browser tab
// via window.open to GET /api/pages/{slug}/content, with zero app chrome.
// There is no in-app detail view, iframe, or Back button; the browser
// navigates the existing content endpoint directly, so the rendered page is
// full-page and unsandboxed (R006) by the browser itself, not by an
// app-owned iframe.
//
// All user-authored strings (name/description) are inserted via
// textContent, never innerHTML, so agent-authored markup is rendered as
// inert text in the table. Page HTML renders live and unsandboxed (R006) in
// its own top-level browser tab opened via window.open — there is no in-app
// rendering surface for it.
//
// A native EventSource subscribes to GET /api/events and reconciles the
// table live by slug: a "page-created" event adds a new top row (if that
// slug isn't already present); a "page-updated" event moves that slug's
// row to the front with its patched name/description, in place of a second
// row. #live-status (data-state="connecting"|"live"|"reconnecting") mirrors
// the browser's EventSource lifecycle so a dropped connection is visible in
// the UI instead of the table silently going stale — the browser's own
// automatic reconnect (another "open" event after the connection drops)
// drives reconnecting -> live without any app.js retry logic. A malformed
// event payload (non-JSON, or JSON that isn't a page-shaped object) is
// logged to the console via console.error and skipped rather than thrown,
// so one bad frame never breaks the page.
(function () {
  "use strict";

  var API = "/api/pages";
  var SEARCH_DEBOUNCE_MS = 250;

  function byId(id) {
    var el = document.getElementById(id);
    if (!el) {
      console.error("hyperreader: missing element #" + id);
    }
    return el;
  }

  // Surface fetch errors in the in-UI error region (R002). Kept stable from
  // T01 so fetch handlers share one error surface.
  window.hyperReader = {
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
    pages: [],
    pending: null, // in-flight AbortController
    debounce: null // debounce timeout id
  };

  // Fetch /api/pages (with optional ?q=) and re-render the table.
  // Aborts any prior in-flight request so a fast-typing user never sees a
  // stale response overwrite a newer one (last-write-wins via abort).
  function fetchPages(query) {
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
      .then(function (pages) {
        state.pages = Array.isArray(pages) ? pages : [];
        setLoading(false);
        render();
        window.hyperReader.clearError();
      })
      .catch(function (err) {
        // AbortError is expected when a newer request supersedes this one;
        // leave loading to the newer request and do not surface an error.
        if (err && err.name === "AbortError") {
          return;
        }
        state.pages = [];
        setLoading(false);
        render();
        window.hyperReader.showError(
          "Failed to load pages: " + (err && err.message ? err.message : err)
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

  // Render the current pages into the table. An empty list shows the empty
  // state (worded by whether a search is active); a non-empty list shows
  // the table. All strings are inserted via textContent so agent-authored
  // markup is inert here. Rows carry data-slug for the row-activation
  // handler.
  function render() {
    var table = byId("pages-table");
    var empty = byId("empty-state");
    var tbody = table ? table.querySelector("tbody") : null;
    if (!table || !empty || !tbody) {
      return;
    }

    if (state.pages.length === 0) {
      table.hidden = true;
      empty.hidden = false;
      empty.textContent = state.search
        ? "No pages match \u201c" + state.search + "\u201d."
        : "No pages yet.";
      return;
    }

    empty.hidden = true;
    tbody.textContent = "";

    for (var i = 0; i < state.pages.length; i++) {
      var page = state.pages[i];
      var tr = document.createElement("tr");
      // Stash the slug on the row for the row-activation handler.
      tr.dataset.slug = page.slug;
      // Rows are keyboard-focusable + activatable so opening a page is
      // reachable without a mouse (a11y on the primary user loop).
      tr.tabIndex = 0;
      tr.setAttribute("role", "button");
      tr.setAttribute(
        "aria-label",
        "View rendered HTML for " + (page.name || "page")
      );

      tr.appendChild(cell(page.name || ""));
      tr.appendChild(cell(page.description || ""));

      tbody.appendChild(tr);
    }
    table.hidden = false;
  }

  function cell(text) {
    var td = document.createElement("td");
    td.textContent = text;
    return td;
  }

  // --- Live updates via SSE ---
  //
  // setLiveStatus mirrors the EventSource connection lifecycle into
  // #live-status via its data-state attribute (and matching visible text),
  // so a dropped connection is visible in the top bar instead of the table
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

  // isValidPagePayload guards against a malformed "page-created"/
  // "page-updated" event payload taking down the page: it must decode to a
  // non-null object with a non-empty slug (the field the table keys rows
  // on).
  function isValidPagePayload(page) {
    return (
      page !== null &&
      typeof page === "object" &&
      !Array.isArray(page) &&
      typeof page.slug === "string" &&
      page.slug !== ""
    );
  }

  // decodePagePayload parses evt.data and validates its shape, logging and
  // returning null on a malformed frame (invalid JSON, or JSON that isn't a
  // page-shaped object) so a bad frame never breaks the live table.
  function decodePagePayload(evt, eventName) {
    var page;
    try {
      page = JSON.parse(evt.data);
    } catch (err) {
      console.error(
        "hyperreader: malformed SSE " + eventName + " payload (invalid JSON), skipping",
        err,
        evt.data
      );
      return null;
    }
    if (!isValidPagePayload(page)) {
      console.error(
        "hyperreader: malformed SSE " + eventName + " payload (not a page object), skipping",
        page
      );
      return null;
    }
    return page;
  }

  // onPageCreated handles a broadcast "page-created" SSE frame: decode its
  // JSON data (the exact pageResponse shape POST/GET already return), then
  // prepend it as a new top row with no fetch and no page reload, unless
  // that slug is already present (e.g. a page that re-fetched between the
  // create response and the broadcast landing).
  function onPageCreated(evt) {
    var page = decodePagePayload(evt, "page-created");
    if (!page) {
      return;
    }

    // A search is active: the new page may or may not match it, and this
    // is a create-time broadcast, not a search result, so leave the
    // filtered view untouched rather than injecting a row the current
    // query would not have returned itself.
    if (state.search !== "") {
      return;
    }

    for (var i = 0; i < state.pages.length; i++) {
      if (state.pages[i].slug === page.slug) {
        return;
      }
    }

    state.pages.unshift(page);
    render();
  }

  // onPageUpdated handles a broadcast "page-updated" SSE frame: decode its
  // JSON data, remove the existing row for that slug (if shown), and
  // re-insert the patched metadata at the front — consistent with list
  // ordering by recency of change.
  function onPageUpdated(evt) {
    var page = decodePagePayload(evt, "page-updated");
    if (!page) {
      return;
    }

    // A search is active: leave the filtered view untouched until the
    // filter clears or is re-run (same rationale as onPageCreated).
    if (state.search !== "") {
      return;
    }

    state.pages = state.pages.filter(function (p) {
      return p.slug !== page.slug;
    });
    state.pages.unshift(page);
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
    es.addEventListener("page-created", onPageCreated);
    es.addEventListener("page-updated", onPageUpdated);

    return es;
  }

  // Debounced search handler. Each keystroke resets the timer; only after
  // SEARCH_DEBOUNCE_MS of quiet does the query hit the API. Clearing the
  // input fires immediately (no debounce) so restoring the full list is
  // snappy.
  function onSearchInput(evt) {
    var value = evt.target.value || "";
    state.search = value.trim();

    clearTimeout(state.debounce);
    if (state.search === "") {
      fetchPages("");
      return;
    }
    state.debounce = setTimeout(function () {
      fetchPages(state.search);
    }, SEARCH_DEBOUNCE_MS);
  }

  // --- Open page in a new tab ---
  //
  // openPage opens the page's raw rendered HTML in a new top-level browser
  // tab via the existing GET /api/pages/{slug}/content endpoint, with zero
  // app chrome. The browser navigates the endpoint directly, so the
  // rendered page is full-page and unsandboxed (R006) by the browser
  // itself — no in-app iframe, fetch, or srcdoc is involved. The table view
  // is never hidden or torn down; the user returns to it by closing the
  // new tab. window.open is called with "_blank" as the target and no
  // window features, which the browser treats as a normal new tab.
  function openPage(slug) {
    window.open(API + "/" + encodeURIComponent(slug) + "/content", "_blank");
  }

  // onTableActivate handles clicks on page rows (event delegation on the
  // tbody so a single listener covers all rows, current and future). A
  // click on a row (or Enter/Space on a keyboard-focused row) opens the
  // page in a new browser tab via openPage.
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
    var slug = target.dataset && target.dataset.slug;
    if (!slug) {
      return;
    }
    openPage(slug);
  }

  // onTableKeydown lets a keyboard user open the page in a new tab with
  // Enter/Space on a focused row (the rows are role=button + tabIndex=0).
  // Space scrolling the page is suppressed via preventDefault so
  // activation is the only Space behavior on a row.
  function onTableKeydown(evt) {
    if (evt.key !== "Enter" && evt.key !== " ") {
      return;
    }
    var target = evt.target;
    if (!target || target.tagName !== "TR" || !target.dataset.slug) {
      return;
    }
    evt.preventDefault();
    openPage(target.dataset.slug);
  }

  function init() {
    var search = byId("search");
    if (search) {
      search.addEventListener("input", onSearchInput);
    }

    // Row-activation wiring: a click (or Enter/Space on a focused row)
    // opens the page in a new browser tab. Uses event delegation on the
    // stable tbody so a single listener covers all rows, current and
    // future.
    var tbody = document.querySelector("#pages-table tbody");
    if (tbody) {
      tbody.addEventListener("click", onTableActivate);
      tbody.addEventListener("keydown", onTableKeydown);
    }

    // Single dark theme, no toggle: write hyperreader-theme once so
    // generate-html reports opened from this reader stay in sync with the
    // app's single theme instead of falling back to OS preference inside
    // the report. Wrapped in try/catch (private/sandboxed contexts can
    // throw) so a blocked store never blocks page load.
    try {
      window.localStorage.setItem("hyperreader-theme", "dark");
    } catch (e) {
      // Ignore: the app itself is always dark regardless of this key.
    }

    fetchPages("");

    // Live updates: subscribe once at startup, independent of the initial
    // fetch above, so the table is kept live even while the first
    // GET /api/pages request is still in flight.
    connectEvents();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
