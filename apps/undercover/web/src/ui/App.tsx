import { useEffect, useRef } from "preact/hooks";
import { focusMain } from "./common";
import { GamePage } from "./GamePage";
import { HomePage } from "./HomePage";
import { ProposePage } from "./ProposePage";
import { Link, usePath } from "./router";
import { RulesPage } from "./RulesPage";

export function App() {
  const path = usePath();
  const first = useRef(true);

  // A route change replaces the content without a page load: move the focus to it so
  // keyboard and screen-reader users start at the top of the new page.
  useEffect(() => {
    if (first.current) {
      first.current = false;
      return;
    }
    focusMain();
  }, [path]);

  return (
    <>
      <a href="#main" class="skip-link">
        Aller au contenu
      </a>
      <header class="topbar">
        <Link href="/" class="brand">
          Undercover
        </Link>
      </header>
      <main id="main" class="page" tabIndex={-1}>
        <Page path={path} />
      </main>
    </>
  );
}

function Page({ path }: { path: string }) {
  switch (path) {
    case "/partie":
      return <GamePage />;
    case "/proposer":
      return <ProposePage />;
    case "/regles":
      return <RulesPage />;
    default:
      return <HomePage />;
  }
}
