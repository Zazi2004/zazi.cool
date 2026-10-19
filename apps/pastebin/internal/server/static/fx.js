// Visual polish only: a spotlight that follows the pointer over the cards (.fx-card, .code).
// It sets two CSS variables; the page is fully usable without it, and it is skipped
// when the visitor asks for reduced motion.
const motion = matchMedia("(prefers-reduced-motion: no-preference)");
if (motion.matches) {
  for (const card of document.querySelectorAll(".fx-card, .code")) {
    card.addEventListener("pointermove", (event) => {
      const box = card.getBoundingClientRect();
      card.style.setProperty("--mx", `${event.clientX - box.left}px`);
      card.style.setProperty("--my", `${event.clientY - box.top}px`);
    });
  }
}
