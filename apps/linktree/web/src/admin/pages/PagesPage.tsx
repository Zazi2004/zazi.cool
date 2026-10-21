import { ExternalLink, MoreHorizontal, Pencil, Plus, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useLocation } from "wouter";
import type { PageSummary } from "@/api/types";
import { Visual } from "@/components/visual/Visual";
import { slugify } from "@/shared/slugify";
import { ApiError, api } from "../api";
import { arrayMove } from "../editor/document";
import { Badge } from "../ui/Badge";
import { Button } from "../ui/Button";
import { Confirm, Dialog } from "../ui/Dialog";
import { Field, Input } from "../ui/Field";
import { Menu } from "../ui/Menu";
import { SortableItem, SortableList } from "../ui/Sortable";
import { TopBar } from "../ui/TopBar";
import { useSlugCheck } from "../useSlugCheck";

export function PagesPage() {
  const [pages, setPages] = useState<PageSummary[] | null>(null);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<PageSummary | null>(null);

  useEffect(() => {
    api.pages().then(setPages, (err: unknown) => setError(message(err)));
  }, []);

  async function reorder(from: number, to: number) {
    if (!pages) return;
    const previous = pages;
    const next = arrayMove(pages, from, to);
    setPages(next); // optimistic: the list moves at once, and comes back on error
    try {
      await api.reorder({ ids: next.map((p) => p.id) });
    } catch (err) {
      setPages(previous);
      setError(message(err));
    }
  }

  async function remove(page: PageSummary) {
    try {
      await api.deletePage(page.id);
      setPages((list) => list?.filter((p) => p.id !== page.id) ?? null);
      setDeleting(null);
    } catch (err) {
      setError(message(err));
    }
  }

  return (
    <>
      <TopBar>
        <h1 className="m-0 flex-1 text-base font-semibold">Pages de liens</h1>
        <Button variant="primary" onClick={() => setCreating(true)}>
          <Plus />
          Nouvelle page
        </Button>
      </TopBar>

      <main className="mx-auto flex max-w-3xl flex-col gap-4 px-4 py-6">
        {error && (
          <p className="m-0 rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950 dark:text-red-300">
            {error}
          </p>
        )}
        {pages?.length === 0 && (
          <p className="m-0 rounded-xl border border-dashed border-zinc-300 p-8 text-center text-sm text-zinc-500 dark:border-zinc-700">
            Aucune page pour l'instant. Créez la première !
          </p>
        )}
        {pages && pages.length > 0 && (
          <>
            <p className="m-0 text-sm text-zinc-500">
              L'ordre ci-dessous est celui de l'index public. Glissez une page pour le changer.
            </p>
            <SortableList ids={pages.map((p) => p.id)} onMove={reorder}>
              <ul className="m-0 flex list-none flex-col gap-2 p-0">
                {pages.map((p) => (
                  <li key={p.id}>
                    <SortableItem id={p.id} label={p.title}>
                      <PageRow page={p} onDelete={() => setDeleting(p)} />
                    </SortableItem>
                  </li>
                ))}
              </ul>
            </SortableList>
          </>
        )}
      </main>

      <CreateDialog open={creating} onOpenChange={setCreating} />
      <Confirm
        open={deleting !== null}
        onOpenChange={(open) => !open && setDeleting(null)}
        title="Supprimer la page ?"
        description={`« ${deleting?.title ?? ""} » et tous ses liens seront supprimés définitivement.`}
        confirmLabel="Supprimer"
        onConfirm={() => deleting && remove(deleting)}
      />
    </>
  );
}

function PageRow({ page, onDelete }: { page: PageSummary; onDelete: () => void }) {
  const [, navigate] = useLocation();
  return (
    <div className="flex items-center gap-3 rounded-xl border border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-900">
      <div className="flex size-12 shrink-0 items-center justify-center">
        {page.avatar && <Visual visual={page.avatar} size={44} imageClassName="rounded-full" />}
      </div>
      <Link href={`/pages/${page.id}`} className="min-w-0 flex-1 text-inherit no-underline">
        <span className="block truncate font-medium">{page.title}</span>
        <span className="flex flex-wrap items-center gap-1.5 text-xs text-zinc-500">
          /{page.slug} · {page.linkCount} lien{page.linkCount > 1 ? "s" : ""}
          {page.published ? (
            <Badge tone="success">Publiée</Badge>
          ) : (
            <Badge tone="warning">Brouillon</Badge>
          )}
          {!page.listed && <Badge>Non listée</Badge>}
        </span>
      </Link>
      <Menu
        trigger={
          <Button variant="ghost" size="icon" aria-label={`Actions pour ${page.title}`}>
            <MoreHorizontal />
          </Button>
        }
        items={[
          { label: "Modifier", icon: <Pencil />, onSelect: () => navigate(`/pages/${page.id}`) },
          {
            label: "Voir la page",
            icon: <ExternalLink />,
            disabled: !page.published,
            onSelect: () => window.open(`/${page.slug}`, "_blank", "noopener"),
          },
          "separator",
          { label: "Supprimer", icon: <Trash2 />, danger: true, onSelect: onDelete },
        ]}
      />
    </div>
  );
}

function CreateDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [, navigate] = useLocation();
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  // Until the slug is edited by hand, it follows the title.
  const [slugEdited, setSlugEdited] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const slugMessage = useSlugCheck(slug);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      const doc = await api.createPage({ title, slug });
      navigate(`/pages/${doc.id}`);
    } catch (err) {
      setErrors(
        err instanceof ApiError ? { ...err.fields, form: err.message } : { form: message(err) },
      );
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Nouvelle page"
      description="Elle sera créée en brouillon."
    >
      <form onSubmit={submit} className="flex flex-col gap-4">
        <Field label="Titre" error={errors.title}>
          {(props) => (
            <Input
              {...props}
              autoFocus
              required
              maxLength={80}
              value={title}
              onChange={(e) => {
                setTitle(e.target.value);
                if (!slugEdited) setSlug(slugify(e.target.value));
              }}
            />
          )}
        </Field>
        <Field
          label="Adresse"
          error={errors.slug || slugMessage}
          hint={slug ? `link.zazi.cool/${slug}` : undefined}
        >
          {(props) => (
            <Input
              {...props}
              required
              maxLength={40}
              value={slug}
              onChange={(e) => {
                setSlug(e.target.value);
                setSlugEdited(true);
              }}
            />
          )}
        </Field>
        {errors.form && !errors.title && !errors.slug && (
          <p className="m-0 text-sm text-red-600 dark:text-red-400">{errors.form}</p>
        )}
        <div className="flex justify-end gap-2">
          <Button onClick={() => onOpenChange(false)}>Annuler</Button>
          <Button
            type="submit"
            variant="primary"
            disabled={busy || !title || !slug || slugMessage !== ""}
          >
            Créer
          </Button>
        </div>
      </form>
    </Dialog>
  );
}

function message(err: unknown): string {
  return err instanceof ApiError ? err.message : "Une erreur est survenue.";
}
