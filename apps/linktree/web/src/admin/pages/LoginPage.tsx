import { LockKeyhole } from "lucide-react";
import { m } from "motion/react";
import { useState, type FormEvent } from "react";
import { useLocation, useSearch } from "wouter";
import { Shape } from "@/components/visual/Shape";
import { Backdrop } from "@/public/components/Backdrop";
import { ApiError, api } from "../api";
import { Button } from "../ui/Button";
import { Field, Input } from "../ui/Field";

export function LoginPage() {
  const [, navigate] = useLocation();
  const next = new URLSearchParams(useSearch()).get("next");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api.login({ password });
      // Only paths of the admin itself: never an outside URL from the query string.
      navigate(next?.startsWith("/") && !next.startsWith("//") ? next : "/", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Connexion impossible.");
      setBusy(false);
    }
  }

  return (
    // The login page wears the zazi theme of the public pages.
    <div
      data-theme="zazi"
      data-accent="violet"
      className="lt-page relative grid min-h-dvh place-items-center px-4"
    >
      <Backdrop theme="zazi" accent="violet" />
      <m.form
        onSubmit={submit}
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        className="lt-card flex w-full max-w-sm flex-col gap-5 p-6"
      >
        <div className="flex flex-col items-center gap-3 text-center">
          <Shape id="star-1" size={64} />
          <h1 className="m-0 text-xl font-semibold">Administration</h1>
          <p className="m-0 text-sm text-lt-muted">link.zazi.cool</p>
        </div>
        <Field label="Mot de passe" error={error}>
          {(props) => (
            <Input
              {...props}
              type="password"
              autoComplete="current-password"
              autoFocus
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          )}
        </Field>
        <Button type="submit" variant="primary" disabled={busy || password === ""}>
          <LockKeyhole />
          Se connecter
        </Button>
      </m.form>
    </div>
  );
}
