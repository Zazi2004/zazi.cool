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
        button.querySelector(".box").textContent = result.done ? "✓" : "";
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
