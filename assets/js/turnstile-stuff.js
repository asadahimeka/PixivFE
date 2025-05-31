/**
 * @file turnstile-stuff.js
 * Manages Cloudflare Turnstile integration using explicit rendering.
 * Ensures clean state by removing old widgets before rendering new ones.
 * Handles widget callbacks and form submission.
 */

(function () {
  // Scope variables to this IIFE
  let currentTurnstileWidgetId = null;
  let isTurnstileApiLoaded = false;

  /**
   * Callback for successful Turnstile challenge.
   * @param {string} token The Turnstile token.
   */
  function handleTurnstileSuccess(token) {
    if (!token) {
      console.error("Turnstile JS (Success): No token received.");
      return;
    }

    const challengeArea = document.getElementById("turnstile-challenge-area");
    if (!challengeArea) {
      console.error(
        "Turnstile JS (Success): #turnstile-challenge-area not found.",
      );
      return;
    }
    const verifyForm = challengeArea.querySelector("#turnstile-verify-form");
    if (!verifyForm) {
      console.error(
        "Turnstile JS (Success): #turnstile-verify-form not found.",
      );
      return;
    }

    // The token is automatically added to a hidden input `cf-turnstile-response` by Turnstile.
    // We just need to submit the form. HTMX will handle the POST due to hx-post attribute.
    htmx.process(verifyForm); // Ensure HTMX has processed the form
    if (typeof verifyForm.requestSubmit === "function") {
      verifyForm.requestSubmit(); // Modern way to submit, respects HTMX if processed
    } else {
      verifyForm.submit(); // Fallback for older browsers
    }
  }

  /**
   * Callback for Turnstile challenge errors.
   * @param {string} errorCode The error code from Turnstile.
   */
  function handleTurnstileError(errorCode) {
    console.error(`Turnstile JS: Challenge error. Error Code: ${errorCode}`);
    // Attempt a reset.
    if (
      errorCode === "timeout" &&
      currentTurnstileWidgetId &&
      typeof turnstile !== "undefined"
    ) {
      turnstile.reset(currentTurnstileWidgetId);
    }
  }

  /**
   * Callback for when the Turnstile token expires.
   */
  function handleTurnstileExpired() {
    console.log("Turnstile JS: Token expired.");
    // Widget will refresh automatically with `refresh-expired` defaulting to 'auto'.
  }

  /**
   * Callback for when an interactive challenge times out.
   */
  function handleTurnstileTimeout() {
    console.log("Turnstile JS: Interactive challenge timed out.");
    // Widget will refresh automatically with `refresh-timeout` defaulting to 'auto'.
  }

  /**
   * Renders the Turnstile widget.
   * Ensures any previous widget is removed first.
   */
  function renderTurnstileWidget() {
    if (
      !isTurnstileApiLoaded ||
      typeof turnstile === "undefined" ||
      !turnstile.render
    ) {
      console.warn(
        "Turnstile JS: API not ready or `turnstile.render` is undefined. Widget not rendered.",
      );
      return;
    }

    const container = document.getElementById("turnstile-widget-container");
    if (!container) {
      console.log(
        "Turnstile JS: #turnstile-widget-container not found. Widget not rendered.",
      );
      return;
    }

    // Clean up previous widget if any
    if (currentTurnstileWidgetId) {
      console.log(
        `Turnstile JS: Removing previous widget (ID: ${currentTurnstileWidgetId}) before rendering new one.`,
      );
      try {
        turnstile.remove(currentTurnstileWidgetId);
      } catch (e) {
        // This might happen if the widget was already removed or ID is stale.
        console.warn(
          `Turnstile JS: Error removing previous widget ${currentTurnstileWidgetId}: ${e.message}`,
        );
      }
      currentTurnstileWidgetId = null;
    }

    const siteKey = container.dataset.sitekey;
    if (!siteKey) {
      console.error(
        "Turnstile JS: data-sitekey not found on #turnstile-widget-container. Cannot render widget.",
      );
      return;
    }

    const renderParams = {
      sitekey: siteKey,
      callback: handleTurnstileSuccess,
      "error-callback": handleTurnstileError,
      "expired-callback": handleTurnstileExpired,
      "timeout-callback": handleTurnstileTimeout,

      // Read configuration from data attributes
      size: container.dataset.size,
      appearance: container.dataset.appearance,
      theme: container.dataset.theme,
    };

    try {
      console.log(
        "Turnstile JS: Rendering widget in #turnstile-widget-container with params:",
        renderParams,
      );
      currentTurnstileWidgetId = turnstile.render(container, renderParams); // Pass the DOM element itself

      if (currentTurnstileWidgetId) {
        console.log(
          `Turnstile JS: Widget rendered successfully. Widget ID: ${currentTurnstileWidgetId}`,
        );
      } else {
        // This can happen if the container is not visible or has 0 dimensions,
        // or if an issue occurred within turnstile.render()
        console.error(
          "Turnstile JS: turnstile.render() did not return a widgetId. Rendering may have failed.",
        );
      }
    } catch (e) {
      console.error("Turnstile JS: Exception during turnstile.render():", e);
      currentTurnstileWidgetId = null; // Ensure ID is null on failure
    }
  }

  // This function is assigned to `window.onloadTurnstileCallback`
  // It's called when the Turnstile API script (api.js?onload=onloadTurnstileCallback) finishes loading.
  window.onloadTurnstileCallback = function () {
    if (isTurnstileApiLoaded) {
      console.log(
        "Turnstile JS: API onload callback invoked again, but already processed. Ignoring.",
      );
      return; // Prevent re-initialization
    }
    console.log(
      "Turnstile JS: Cloudflare Turnstile API Loaded (onloadTurnstileCallback executed).",
    );
    isTurnstileApiLoaded = true;

    if (typeof turnstile !== "undefined" && turnstile.render) {
      renderTurnstileWidget(); // Attempt to render if the container is already in the DOM
    } else {
      console.error(
        "Turnstile JS: onloadTurnstileCallback fired, but `turnstile` object or `turnstile.render` is undefined!",
      );
    }
  };

  // Ensure HTMX event listener is added only once to `document.body`
  if (!window.isTurnstileHtmxListenerAttached) {
    document.body.addEventListener("htmx:afterSwap", function (event) {
      // Check if the Turnstile API is ready and if our widget container is now in the DOM
      const widgetContainer = document.getElementById(
        "turnstile-widget-container",
      );
      if (widgetContainer) {
        console.log(
          "Turnstile JS: htmx:afterSwap - #turnstile-widget-container is present. Attempting to render/re-render widget.",
        );
        renderTurnstileWidget(); // This will check isTurnstileApiLoaded and handle removal/re-render
      }
    });
    window.isTurnstileHtmxListenerAttached = true;
  }

  // Fallback for initial page load if scripts load in an unexpected order,
  // or if turnstile-stuff.js loads after api.js has already fired its onload.
  document.addEventListener("DOMContentLoaded", () => {
    if (
      isTurnstileApiLoaded &&
      typeof turnstile !== "undefined" &&
      turnstile.render
    ) {
      // This condition means api.js loaded, onloadTurnstileCallback ran,
      // and now the DOM is ready.
      console.log(
        "Turnstile JS: DOMContentLoaded - API loaded and DOM ready. Ensuring widget render if container exists.",
      );
      renderTurnstileWidget();
    } else if (
      typeof turnstile !== "undefined" &&
      turnstile.render &&
      !isTurnstileApiLoaded
    ) {
      // This means api.js might have loaded before this script defined onloadTurnstileCallback,
      // but after this script runs, turnstile object is available.
      console.warn(
        "Turnstile JS: DOMContentLoaded - `turnstile` object exists but API not marked 'loaded'. Attempting to set up.",
      );
      // Manually trigger the API loaded logic if `turnstile` object is present.
      // This is a safety net for unusual script loading orders.
      window.onloadTurnstileCallback();
    }
  });
})();
