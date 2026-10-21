import { ArrowLeft, Check, ExternalLink, Save } from "lucide-react";
import { useCallback, useEffect, useMemo, useReducer, useState } from "react";
import { Link, useParams } from "wouter";
import type { Document } from "@/api/types";
import { cn } from "@/lib/cn";
import { ApiError, api } from "../api";
import { EditorContext, type EditorContextValue } from "../editor/context";
import { reduce } from "../editor/document";
import { ItemsEditor } from "../editor/ItemsEditor";
import { PageSettings } from "../editor/PageSettings";
import { Preview } from "../editor/Preview";
import { Button, buttonVariants } from "../ui/Button";
import { TopBar } from "../ui/TopBar";

export function EditorPage() {
  const { id } = useParams<{ id: string }>();
  const [doc, setDoc] = useState<Document | null>(null);
  const [loadError, setLoadError] = useState("");

  useEffect(() => {
    api
      .page(id)
      .then(setDoc, (err: unknown) =>
        setLoadError(err instanceof ApiError ? err.message : "Chargement impossible."),
      );
  }, [id]);

  if (loadError) {
    return (
      <>
        <TopBar>
          <BackLink />
        </TopBar>
        <p className="mx-auto max-w-xl p-6 text-center text-sm text-red-600">{loadError}</p>
      </>
    );
  }
  // The key reloads the whole editor when another page is opened.
  return doc ? <Editor key={doc.id} initial={doc} /> : null;
}

function BackLink({ onLeave }: { onLeave?: (e: React.MouseEvent) => void }) {
  return (
    <Link
      href="/"
      onClick={onLeave}
      aria-label="Retour aux pages"
      className={buttonVariants({ variant: "ghost", size: "icon" })}
    >
      <ArrowLeft />
    </Link>
  );
}

type Status =
  | { kind: "idle" }
  | { kind: "saving" }
  | { kind: "saved" }
  | { kind: "error"; message: string }
  | { kind: "conflict" };

function Editor({ initial }: { initial: Document }) {
  const [doc, dispatch] = useReducer(reduce, initial);
  // The document as last saved: comparing with it tells whether there are changes.
  const [saved, setSaved] = useState(initial);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [status, setStatus] = useState<Status>({ kind: "idle" });
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [showDrafts, setShowDrafts] = useState(true);
  const [tab, setTab] = useState<"edit" | "preview">("edit");

  const dirty = useMemo(() => JSON.stringify(doc) !== JSON.stringify(saved), [doc, saved]);

  const save = useCallback(async () => {
    setStatus({ kind: "saving" });
    try {
      const { version } = await api.savePage(doc);
      const next = { ...doc, version };
      dispatch({ type: "saved", version });
      setSaved(next);
      setErrors({});
      setStatus({ kind: "saved" });
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setStatus({ kind: "conflict" });
        return;
      }
      setErrors(err instanceof ApiError ? err.fields : {});
      setStatus({
        kind: "error",
        message: err instanceof ApiError ? err.message : "Enregistrement impossible.",
      });
    }
  }, [doc]);

  // Ctrl/Cmd + S saves, as in any editor.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "s") {
        e.preventDefault();
        if (status.kind !== "saving") void save();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [save, status.kind]);

  // Leaving the tab (or reloading) with unsaved changes asks for confirmation.
  useEffect(() => {
    if (!dirty) return;
    const onBeforeUnload = (e: BeforeUnloadEvent) => e.preventDefault();
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, [dirty]);

  async function reload() {
    const fresh = await api.page(doc.id);
    dispatch({ type: "load", doc: fresh });
    setSaved(fresh);
    setErrors({});
    setStatus({ kind: "idle" });
  }

  const context: EditorContextValue = {
    dispatch,
    errors,
    expanded,
    toggle: (id) =>
      setExpanded((set) => {
        const next = new Set(set);
        if (next.has(id)) next.delete(id);
        else next.add(id);
        return next;
      }),
    groups: doc.items
      .filter((it) => it.kind === "group")
      .map((g) => ({ id: g.id, title: g.title })),
  };

  return (
    <EditorContext.Provider value={context}>
      <TopBar>
        <BackLink
          onLeave={(e) => {
            if (dirty && !window.confirm("Quitter sans enregistrer les modifications ?"))
              e.preventDefault();
          }}
        />
        <div className="min-w-0 flex-1">
          <h1 className="m-0 truncate text-base font-semibold">{doc.title || "Sans titre"}</h1>
          <p className="m-0 text-xs text-zinc-500" role="status">
            {status.kind === "saving"
              ? "Enregistrement…"
              : dirty
                ? "Modifications non enregistrées"
                : status.kind === "saved"
                  ? "Enregistré"
                  : saved.published
                    ? "Publiée"
                    : "Brouillon"}
          </p>
        </div>
        {saved.published && (
          <Button
            variant="ghost"
            size="icon"
            aria-label="Voir la page publique"
            title="Voir la page publique"
            onClick={() => window.open(`/${saved.slug}`, "_blank", "noopener")}
          >
            <ExternalLink />
          </Button>
        )}
        <Button variant="primary" onClick={save} disabled={!dirty || status.kind === "saving"}>
          {dirty ? <Save /> : <Check />}
          <span className="hidden sm:inline">Enregistrer</span>
        </Button>
      </TopBar>

      {status.kind === "error" && (
        <Banner tone="error">
          {status.message}
          {Object.keys(errors).length > 0 && " Les champs concernés sont signalés en rouge."}
        </Banner>
      )}
      {status.kind === "conflict" && (
        <Banner tone="warning">
          Cette page a été modifiée ailleurs (un autre onglet ?). Rechargez-la pour repartir de la
          dernière version : vos modifications ici seront perdues.
          <Button size="sm" className="ml-2" onClick={() => void reload()}>
            Recharger
          </Button>
        </Banner>
      )}

      {/* On small screens, the editor and the preview share the screen through tabs. */}
      <div className="mx-auto flex max-w-6xl gap-2 px-4 pt-4 lg:hidden" role="tablist">
        {(["edit", "preview"] as const).map((t) => (
          <button
            key={t}
            type="button"
            role="tab"
            aria-selected={tab === t}
            onClick={() => setTab(t)}
            className={cn(
              "flex-1 cursor-pointer rounded-lg py-2 text-sm font-medium",
              tab === t
                ? "bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900"
                : "bg-zinc-200/60 dark:bg-zinc-800",
            )}
          >
            {t === "edit" ? "Modifier" : "Aperçu"}
          </button>
        ))}
      </div>

      <main className="mx-auto grid max-w-6xl gap-8 px-4 py-6 lg:grid-cols-[minmax(0,1fr)_400px]">
        <div className={cn("flex flex-col gap-8", tab === "preview" && "max-lg:hidden")}>
          <PageSettings doc={doc} savedSlug={saved.slug} savedPublished={saved.published} />
          <ItemsEditor items={doc.items} />
        </div>
        <aside
          className={cn("lg:sticky lg:top-20 lg:self-start", tab === "edit" && "max-lg:hidden")}
        >
          <Preview doc={doc} showDrafts={showDrafts} onShowDrafts={setShowDrafts} />
        </aside>
      </main>
    </EditorContext.Provider>
  );
}

function Banner({ tone, children }: { tone: "error" | "warning"; children: React.ReactNode }) {
  return (
    <div role="alert" className="mx-auto mt-4 max-w-6xl px-4 text-sm">
      <p
        className={cn(
          "m-0 rounded-lg p-3",
          tone === "error"
            ? "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300"
            : "bg-amber-50 text-amber-800 dark:bg-amber-950 dark:text-amber-200",
        )}
      >
        {children}
      </p>
    </div>
  );
}
