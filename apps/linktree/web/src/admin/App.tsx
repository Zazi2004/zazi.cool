import { useEffect, useState } from "react";
import { Redirect, Route, Router, Switch, useLocation } from "wouter";
import { api, setUnauthorizedHandler } from "./api";
import { EditorPage } from "./pages/EditorPage";
import { LoginPage } from "./pages/LoginPage";
import { PagesPage } from "./pages/PagesPage";

/** Every admin URL lives under /admin; routes below are relative to it. */
export function App() {
  return (
    <Router base="/admin">
      <Switch>
        <Route path="/login" component={LoginPage} />
        <Route>
          <RequireSession>
            <Switch>
              <Route path="/" component={PagesPage} />
              <Route path="/pages/:id" component={EditorPage} />
              <Route>
                <Redirect to="/" />
              </Route>
            </Switch>
          </RequireSession>
        </Route>
      </Switch>
    </Router>
  );
}

/** Shows its children once the session is confirmed, else goes to the login page. */
function RequireSession({ children }: { children: React.ReactNode }) {
  const [location, navigate] = useLocation();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    // Any 401 later on (expired session) also comes back to the login page, then to
    // the page that was open.
    const toLogin = () =>
      navigate(`/login?next=${encodeURIComponent(location)}`, { replace: true });
    setUnauthorizedHandler(toLogin);
    api.session().then(
      () => setReady(true),
      () => toLogin(),
    );
    // The check runs once, on arrival; later 401s are handled by the API client.
  }, []);

  return ready ? <>{children}</> : null;
}
