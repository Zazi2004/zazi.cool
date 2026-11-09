import type { MouseEvent } from "react";
import { navigate, type Route } from "../router";

const tabs = [
  { href: "/", label: "Aujourd'hui", icon: "◉", names: ["today"] },
  { href: "/stats", label: "Stats", icon: "▦", names: ["stats", "statsHabit"] },
  { href: "/habits", label: "Habitudes", icon: "☰", names: ["habits"] },
];

export function TabBar({ route }: { route: Route }) {
  function go(e: MouseEvent<HTMLAnchorElement>, href: string) {
    // Let the browser handle new-tab clicks; navigate in place otherwise.
    if (e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    e.preventDefault();
    navigate(href);
  }

  return (
    <nav className="tabbar" aria-label="Navigation principale">
      {tabs.map((tab) => (
        <a
          key={tab.href}
          href={tab.href}
          aria-current={tab.names.includes(route.name) ? "page" : undefined}
          onClick={(e) => go(e, tab.href)}
        >
          <span aria-hidden="true">{tab.icon}</span>
          {tab.label}
        </a>
      ))}
    </nav>
  );
}
