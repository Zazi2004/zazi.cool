import { useEffect, useState } from "react";
import { hasSession } from "../api";
import { matchRoute, navigate, usePath } from "../router";
import { Login } from "./Login";
import { Screen } from "./Screen";
import { TabBar } from "./TabBar";

type Auth = "loading" | "in" | "out" | "error";

export function App() {
  const route = matchRoute(usePath());
  const [auth, setAuth] = useState<Auth>("loading");

  useEffect(() => {
    let current = true;
    hasSession().then(
      (ok) => current && setAuth(ok ? "in" : "out"),
      () => current && setAuth("error"),
    );
    return () => {
      current = false;
    };
  }, []);

  useEffect(() => {
    if (auth === "out" && route.name !== "login") navigate("/login", { replace: true });
    if (auth === "in" && route.name === "login") navigate("/", { replace: true });
  }, [auth, route.name]);

  if (auth === "loading") return <main className="center" aria-busy="true" />;
  if (auth === "error") {
    return (
      <main className="center">
        <p role="alert">Impossible de joindre le serveur.</p>
        <button type="button" onClick={() => window.location.reload()}>
          Réessayer
        </button>
      </main>
    );
  }
  if (auth === "out") {
    return route.name === "login" ? <Login onSuccess={() => setAuth("in")} /> : null;
  }

  return (
    <>
      <Screen route={route} />
      <TabBar route={route} />
    </>
  );
}
