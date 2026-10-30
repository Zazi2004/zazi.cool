// Checks a habit without reloading the page. Without this script the forms still
// work: they post and redirect back to the day.
(function () {
  "use strict";

  var status = document.getElementById("status");

  function say(text) {
    if (status) {
      status.textContent = text;
    }
  }

  // Spotlight position for .check buttons (pure decoration).
  document.addEventListener("pointermove", function (event) {
    var target = event.target;
    var el = target instanceof Element ? target.closest(".check") : null;
    if (!el) {
      return;
    }
    var rect = el.getBoundingClientRect();
    el.style.setProperty("--mx", event.clientX - rect.left + "px");
    el.style.setProperty("--my", event.clientY - rect.top + "px");
  });

  document.addEventListener("submit", function (event) {
    var form = event.target;
    if (!(form instanceof HTMLFormElement) || !form.hasAttribute("data-toggle")) {
      return;
    }
    var button = form.querySelector("button.check");
    var done = form.elements["done"];
    if (!button || !done || !window.fetch) {
      return;
    }
    event.preventDefault();
    if (button.getAttribute("aria-busy") === "true") {
      return;
    }
    button.setAttribute("aria-busy", "true");

    fetch(form.action, {
      method: "POST",
      body: new URLSearchParams(new FormData(form)),
      headers: { Accept: "application/json" },
      credentials: "same-origin",
    })
      .then(function (response) {
        if (response.status === 401 || response.redirected) {
          window.location.reload();
          return null;
        }
        if (!response.ok) {
          throw new Error(String(response.status));
        }
        return response.json();
      })
      .then(function (result) {
        if (!result) {
          return;
        }
        button.setAttribute("aria-pressed", result.done ? "true" : "false");
        var box = button.querySelector(".box");
        box.textContent = result.done ? "✓" : "";
        box.classList.remove("pop");
        if (result.done) {
          void box.offsetWidth;
          box.classList.add("pop");
        }
        done.value = result.done ? "0" : "1";
        say("");
      })
      .catch(function () {
        say("Enregistrement impossible, réessayez.");
      })
      .then(function () {
        button.removeAttribute("aria-busy");
      });
  });
})();
