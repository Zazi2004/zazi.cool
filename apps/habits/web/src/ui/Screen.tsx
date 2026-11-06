import type { Route } from "../router";
import { Habits } from "./Habits";

// Today and Stats are empty for now: the following tickets fill them in.
export function Screen({ route }: { route: Route }) {
  switch (route.name) {
    case "today":
      return <Page title="Aujourd'hui" />;
    case "stats":
    case "statsHabit":
      return <Page title="Stats" />;
    case "habits":
      return <Habits />;
    default:
      return <Page title="Page introuvable" />;
  }
}

function Page({ title }: { title: string }) {
  return (
    <main className="page">
      <h1>{title}</h1>
      <p className="muted">Rien à afficher pour l'instant.</p>
    </main>
  );
}
