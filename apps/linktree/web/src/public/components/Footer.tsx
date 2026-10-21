/** Discreet footer: the way back to the index and to the site. */
export function Footer() {
  return (
    <footer className="flex justify-center gap-4 pt-12 pb-6 text-sm text-lt-muted">
      <a href="/" className="lt-focus rounded no-underline hover:text-lt-text">
        Toutes les pages
      </a>
      <span aria-hidden="true">·</span>
      <a href="https://zazi.cool" className="lt-focus rounded no-underline hover:text-lt-text">
        zazi.cool
      </a>
    </footer>
  );
}
