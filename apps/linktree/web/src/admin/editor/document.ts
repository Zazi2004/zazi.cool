// The editor state, as pure functions: every change to the page being edited goes
// through `reduce`, which never mutates its input. New IDs are passed in the
// actions, so that the functions stay pure and testable.
import type { DocItem, Document, Visual } from "@/api/types";
import { platformOf } from "@/shared/platforms";
import type { ShownItem, ShownLink, ShownPage } from "@/public/model";

/** Where an item lives: the top level of the page (null) or a group. */
export type Container = string | null;

export type PageFields = Pick<
  Document,
  "title" | "slug" | "description" | "avatar" | "theme" | "accent" | "published" | "listed"
>;

export type ItemPatch = Partial<
  Pick<DocItem, "title" | "subtitle" | "url" | "visual" | "openByDefault" | "published">
>;

export type Action =
  | { type: "load"; doc: Document }
  | { type: "setPage"; patch: Partial<PageFields> }
  | { type: "addLink"; id: string; container: Container }
  | { type: "addGroup"; id: string }
  | { type: "update"; id: string; patch: ItemPatch }
  | { type: "remove"; id: string }
  /** ids: one new ID for the item, then one per link for a group. */
  | { type: "duplicate"; id: string; ids: string[] }
  | { type: "move"; id: string; to: Container }
  | { type: "reorder"; container: Container; from: number; to: number }
  | { type: "saved"; version: number };

export function newLink(id: string): DocItem {
  return {
    id,
    kind: "link",
    title: "",
    subtitle: "",
    url: "",
    visual: null,
    openByDefault: false,
    published: true,
    links: [],
  };
}

export function newGroup(id: string): DocItem {
  return { ...newLink(id), kind: "group", title: "Nouveau groupe" };
}

/** The items of a container, or undefined if the group does not exist. */
export function itemsOf(doc: Document, container: Container): DocItem[] | undefined {
  if (container === null) return doc.items;
  return doc.items.find((it) => it.id === container && it.kind === "group")?.links;
}

/** Finds an item and the container that holds it. */
export function locate(doc: Document, id: string): { item: DocItem; container: Container } | null {
  for (const it of doc.items) {
    if (it.id === id) return { item: it, container: null };
    const link = it.links.find((l) => l.id === id);
    if (link) return { item: link, container: it.id };
  }
  return null;
}

/** Returns doc with the items of `container` replaced by fn(items). */
function withItems(doc: Document, container: Container, fn: (items: DocItem[]) => DocItem[]) {
  if (container === null) return { ...doc, items: fn(doc.items) };
  return {
    ...doc,
    items: doc.items.map((it) => (it.id === container ? { ...it, links: fn(it.links) } : it)),
  };
}

/** Moves the element at `from` to `to`, shifting the others. */
export function arrayMove<T>(list: T[], from: number, to: number): T[] {
  const copy = [...list];
  const [moved] = copy.splice(from, 1);
  if (moved === undefined) return list;
  copy.splice(to, 0, moved);
  return copy;
}

export function reduce(doc: Document, action: Action): Document {
  switch (action.type) {
    case "load":
      return action.doc;
    case "setPage":
      return { ...doc, ...action.patch };
    case "saved":
      return { ...doc, version: action.version };
    case "addLink":
      if (!itemsOf(doc, action.container)) return doc;
      return withItems(doc, action.container, (items) => [...items, newLink(action.id)]);
    case "addGroup":
      return { ...doc, items: [...doc.items, newGroup(action.id)] };
    case "update": {
      const found = locate(doc, action.id);
      if (!found) return doc;
      return withItems(doc, found.container, (items) =>
        items.map((it) => (it.id === action.id ? { ...it, ...action.patch } : it)),
      );
    }
    case "remove": {
      const found = locate(doc, action.id);
      if (!found) return doc;
      return withItems(doc, found.container, (items) => items.filter((it) => it.id !== action.id));
    }
    case "duplicate": {
      const found = locate(doc, action.id);
      const [id, ...linkIds] = action.ids;
      if (!found || !id) return doc;
      const copy: DocItem = {
        ...found.item,
        id,
        links: found.item.links.map((l, i) => ({ ...l, id: linkIds[i] ?? `${id}-${i}` })),
      };
      return withItems(doc, found.container, (items) => {
        const index = items.findIndex((it) => it.id === action.id);
        return [...items.slice(0, index + 1), copy, ...items.slice(index + 1)];
      });
    }
    case "move": {
      const found = locate(doc, action.id);
      // Only links move between containers: a group cannot go inside a group.
      if (!found || found.item.kind !== "link" || found.container === action.to) return doc;
      if (!itemsOf(doc, action.to)) return doc;
      const without = withItems(doc, found.container, (items) =>
        items.filter((it) => it.id !== action.id),
      );
      return withItems(without, action.to, (items) => [...items, found.item]);
    }
    case "reorder":
      return withItems(doc, action.container, (items) => arrayMove(items, action.from, action.to));
  }
}

/**
 * Prepares a link after its URL changed: adds https:// to a bare domain, and
 * takes the platform name as title while the title is empty.
 */
export function completeLink(url: string, title: string): { url: string; title: string } {
  let full = url.trim();
  if (full && !/^[a-z][a-z0-9+.-]*:/i.test(full) && /^[^\s/]+\.[^\s/]+/.test(full)) {
    full = "https://" + full;
  }
  const platform = platformOf(full);
  return { url: full, title: title.trim() === "" && platform ? platform.name : title };
}

/** What the preview draws: the public page, with or without the drafts. */
export function toShownPage(doc: Document, showDrafts: boolean): ShownPage {
  const shown = (it: DocItem) => showDrafts || it.published;
  const link = (it: DocItem): ShownLink => ({
    kind: "link",
    id: it.id,
    title: it.title || "Sans titre",
    url: it.url,
    visual: it.visual,
    draft: !it.published,
  });
  const items: ShownItem[] = doc.items.filter(shown).flatMap((it): ShownItem[] => {
    if (it.kind === "link") return [link(it)];
    const links = it.links.filter(shown).map(link);
    // Like the server: a group without any visible link is not shown.
    if (links.length === 0 && !showDrafts) return [];
    return [
      {
        kind: "group",
        id: it.id,
        title: it.title || "Sans titre",
        subtitle: it.subtitle,
        visual: it.visual,
        openByDefault: it.openByDefault,
        links,
        draft: !it.published,
      },
    ];
  });
  return {
    id: doc.id,
    slug: doc.slug,
    title: doc.title || "Sans titre",
    description: doc.description,
    avatar: doc.avatar,
    theme: doc.theme,
    accent: doc.accent,
    items,
  };
}

/** Counts the links of a page, those of the groups included. */
export function linkCount(items: DocItem[]): number {
  return items.reduce((n, it) => n + (it.kind === "link" ? 1 : it.links.length), 0);
}

/** A shape visual, as the API expects it. */
export function shapeVisual(shape: string): Visual {
  return { kind: "shape", shape };
}
