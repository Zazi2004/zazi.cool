import {
  ChevronDown,
  Copy,
  CornerDownRight,
  Eye,
  EyeOff,
  FolderPlus,
  MoreHorizontal,
  Plus,
  Trash2,
} from "lucide-react";
import type { DocItem } from "@/api/types";
import { PlatformIcon } from "@/components/visual/PlatformIcon";
import { Visual } from "@/components/visual/Visual";
import { cn } from "@/lib/cn";
import { Badge } from "../ui/Badge";
import { Button } from "../ui/Button";
import { Field, Input } from "../ui/Field";
import { Menu, type MenuItem } from "../ui/Menu";
import { SortableItem, SortableList } from "../ui/Sortable";
import { Switch } from "../ui/Switch";
import { useEditor } from "./context";
import { completeLink, type Container } from "./document";
import { VisualField } from "./VisualField";

/** New IDs come from the browser: the server accepts them (see the spec, §4). */
export const newId = () => crypto.randomUUID();

/** The items of the page: links and groups, sortable, with buttons to add more. */
export function ItemsEditor({ items }: { items: DocItem[] }) {
  const { dispatch, toggle, errors } = useEditor();

  function add(kind: "link" | "group") {
    const id = newId();
    dispatch(kind === "link" ? { type: "addLink", id, container: null } : { type: "addGroup", id });
    toggle(id); // a new item opens for editing
  }

  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-2">
        <h2 className="m-0 text-base font-semibold">Liens et groupes</h2>
        <div className="flex gap-2">
          <Button size="sm" onClick={() => add("link")}>
            <Plus />
            Lien
          </Button>
          <Button size="sm" onClick={() => add("group")}>
            <FolderPlus />
            Groupe
          </Button>
        </div>
      </div>
      {errors.items && <p className="m-0 text-sm text-red-600 dark:text-red-400">{errors.items}</p>}
      {items.length === 0 ? (
        <p className="m-0 rounded-xl border border-dashed border-zinc-300 p-6 text-center text-sm text-zinc-500 dark:border-zinc-700">
          La page est vide : ajoutez un lien ou un groupe (par exemple une musique et ses
          plateformes).
        </p>
      ) : (
        <ItemList items={items} container={null} />
      )}
    </section>
  );
}

function ItemList({ items, container }: { items: DocItem[]; container: Container }) {
  const { dispatch } = useEditor();
  return (
    <SortableList
      ids={items.map((it) => it.id)}
      onMove={(from, to) => dispatch({ type: "reorder", container, from, to })}
    >
      <ul className="m-0 flex list-none flex-col gap-2 p-0">
        {items.map((it) => (
          <li key={it.id}>
            <SortableItem id={it.id} label={it.title || "l'élément"}>
              <ItemCard item={it} container={container} />
            </SortableItem>
          </li>
        ))}
      </ul>
    </SortableList>
  );
}

/** Whether some server error concerns this item (or one of its links). */
function hasErrors(errors: Record<string, string>, item: DocItem): boolean {
  const ids = [item.id, ...item.links.map((l) => l.id)];
  return Object.keys(errors).some((k) => ids.some((id) => k.startsWith(`items.${id}.`)));
}

function ItemCard({ item, container }: { item: DocItem; container: Container }) {
  const { dispatch, expanded, toggle, groups, errors } = useEditor();
  const open = expanded.has(item.id) || hasErrors(errors, item);
  const isGroup = item.kind === "group";

  const moveTargets: MenuItem[] = isGroup
    ? []
    : [
        ...(container !== null
          ? [
              {
                label: "Premier niveau",
                onSelect: () => dispatch({ type: "move", id: item.id, to: null }),
              },
            ]
          : []),
        ...groups
          .filter((g) => g.id !== container)
          .map((g) => ({
            label: g.title || "Groupe sans titre",
            onSelect: () => dispatch({ type: "move", id: item.id, to: g.id }),
          })),
      ];

  return (
    <div
      className={cn(
        "rounded-xl border bg-white dark:bg-zinc-900",
        hasErrors(errors, item) ? "border-red-400" : "border-zinc-200 dark:border-zinc-800",
        !item.published && "border-dashed",
      )}
    >
      <div className="flex items-center gap-2 p-2">
        <button
          type="button"
          onClick={() => toggle(item.id)}
          aria-expanded={open}
          className="flex min-w-0 flex-1 cursor-pointer items-center gap-3 rounded-lg p-1 text-left hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
        >
          <span className="flex size-9 shrink-0 items-center justify-center text-zinc-600 dark:text-zinc-300">
            {item.visual ? (
              <Visual visual={item.visual} size={36} imageClassName="rounded-lg" />
            ) : isGroup ? (
              <FolderPlus className="size-5" />
            ) : (
              <PlatformIcon url={item.url} />
            )}
          </span>
          <span className="min-w-0 flex-1">
            <span className="flex items-center gap-2">
              <span className="truncate text-sm font-medium">{item.title || "Sans titre"}</span>
              {!item.published && <Badge tone="warning">Brouillon</Badge>}
            </span>
            <span className="block truncate text-xs text-zinc-500">
              {isGroup
                ? `Groupe · ${item.links.length} lien${item.links.length > 1 ? "s" : ""}`
                : item.url || "Pas encore d'URL"}
            </span>
          </span>
          <ChevronDown
            className={cn(
              "size-4 shrink-0 text-zinc-400 transition-transform",
              open && "rotate-180",
            )}
          />
        </button>
        <Menu
          trigger={
            <Button
              variant="ghost"
              size="iconSm"
              aria-label={`Actions pour ${item.title || "l'élément"}`}
            >
              <MoreHorizontal />
            </Button>
          }
          items={[
            {
              label: "Dupliquer",
              icon: <Copy />,
              onSelect: () =>
                dispatch({
                  type: "duplicate",
                  id: item.id,
                  ids: [newId(), ...item.links.map(() => newId())],
                }),
            },
            {
              label: item.published ? "Passer en brouillon" : "Publier",
              icon: item.published ? <EyeOff /> : <Eye />,
              onSelect: () =>
                dispatch({ type: "update", id: item.id, patch: { published: !item.published } }),
            },
            ...(moveTargets.length > 0
              ? (["separator", { label: "Déplacer vers…", heading: true as const }] as const)
              : []),
            ...moveTargets.map((t) => ({ ...t, icon: <CornerDownRight /> })),
            "separator",
            {
              label: "Supprimer",
              icon: <Trash2 />,
              danger: true,
              onSelect: () => dispatch({ type: "remove", id: item.id }),
            },
          ]}
        />
      </div>
      {open && (
        <div className="border-t border-zinc-200 p-3 dark:border-zinc-800">
          {isGroup ? <GroupFields group={item} /> : <LinkFields link={item} />}
        </div>
      )}
    </div>
  );
}

function LinkFields({ link }: { link: DocItem }) {
  const { dispatch, errors } = useEditor();
  const update = (patch: Partial<DocItem>) => dispatch({ type: "update", id: link.id, patch });
  const err = (field: string) => errors[`items.${link.id}.${field}`];

  return (
    <div className="flex flex-col gap-3">
      <Field label="URL" error={err("url")} hint="https://…, mailto:… ou tel:…">
        {(props) => (
          <Input
            {...props}
            type="text"
            inputMode="url"
            placeholder="https://open.spotify.com/…"
            value={link.url}
            onChange={(e) => update({ url: e.target.value })}
            // On leaving the field: https:// added, title taken from the platform.
            onBlur={() => update(completeLink(link.url, link.title))}
          />
        )}
      </Field>
      <Field label="Titre" error={err("title")}>
        {(props) => (
          <Input
            {...props}
            maxLength={80}
            value={link.title}
            onChange={(e) => update({ title: e.target.value })}
          />
        )}
      </Field>
      <VisualField
        label="Image (sinon, l'icône de la plateforme)"
        value={link.visual}
        onChange={(visual) => update({ visual })}
        allowShape={false}
        error={err("visual")}
      />
      <Switch
        label="Publié"
        checked={link.published}
        onChange={(published) => update({ published })}
      />
    </div>
  );
}

function GroupFields({ group }: { group: DocItem }) {
  const { dispatch, errors, toggle } = useEditor();
  const update = (patch: Partial<DocItem>) => dispatch({ type: "update", id: group.id, patch });
  const err = (field: string) => errors[`items.${group.id}.${field}`];

  function addLink() {
    const id = newId();
    dispatch({ type: "addLink", id, container: group.id });
    toggle(id);
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label="Titre" error={err("title")}>
          {(props) => (
            <Input
              {...props}
              maxLength={80}
              value={group.title}
              onChange={(e) => update({ title: e.target.value })}
            />
          )}
        </Field>
        <Field label="Sous-titre" error={err("subtitle")} hint="Par exemple l'artiste.">
          {(props) => (
            <Input
              {...props}
              maxLength={80}
              value={group.subtitle}
              onChange={(e) => update({ subtitle: e.target.value })}
            />
          )}
        </Field>
      </div>
      <VisualField
        label="Visuel (pochette ou forme)"
        value={group.visual}
        onChange={(visual) => update({ visual })}
        allowShape
        error={err("visual")}
      />
      <Switch
        label="Ouvert par défaut"
        checked={group.openByDefault}
        onChange={(openByDefault) => update({ openByDefault })}
      />
      <Switch
        label="Publié"
        checked={group.published}
        onChange={(published) => update({ published })}
      />

      <div className="flex flex-col gap-2 rounded-xl bg-zinc-50 p-2 dark:bg-zinc-950">
        <div className="flex items-center justify-between px-1">
          <span className="text-sm font-medium">Liens du groupe</span>
          <Button size="sm" variant="ghost" onClick={addLink}>
            <Plus />
            Lien
          </Button>
        </div>
        {err("links") && (
          <p className="m-0 text-xs text-red-600 dark:text-red-400">{err("links")}</p>
        )}
        {group.links.length === 0 ? (
          <p className="m-0 px-1 pb-1 text-xs text-zinc-500">
            Aucun lien : le groupe n'apparaîtra pas sur la page publique.
          </p>
        ) : (
          <ItemList items={group.links} container={group.id} />
        )}
      </div>
    </div>
  );
}
