import { GamePage } from "./GamePage";
import { HomePage } from "./HomePage";
import { ProposePage } from "./ProposePage";
import { Link, usePath } from "./router";
import { RulesPage } from "./RulesPage";

export function App() {
  const path = usePath();
  return (
    <>
      <header class="topbar">
        <Link href="/" class="brand">
          Undercover
        </Link>
      </header>
      <main class="page">
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
