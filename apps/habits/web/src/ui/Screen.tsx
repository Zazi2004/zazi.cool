import type { Route } from "../router";
import { Habits } from "./Habits";
import { Stats } from "./Stats";
import { StatsHabit } from "./StatsHabit";
import { Today } from "./Today";

export function Screen({ route }: { route: Route }) {
  switch (route.name) {
    case "today":
      return <Today key={route.date ?? "today"} date={route.date} />;
    case "stats":
      return <Stats />;
    case "statsHabit":
      return <StatsHabit key={route.id} id={route.id} />;
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
