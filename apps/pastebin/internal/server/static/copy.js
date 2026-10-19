// Shows the "copy link" buttons and copies their data-copy value on click.
// Without JavaScript the buttons stay hidden and the link field can be copied by hand.
for (const button of document.querySelectorAll("button[data-copy]")) {
  button.hidden = false;
  button.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(button.dataset.copy);
      button.textContent = "Lien copié";
    } catch {
      button.textContent = "Copie impossible";
    }
  });
}
