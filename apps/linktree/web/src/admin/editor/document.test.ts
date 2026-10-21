import { describe, expect, it } from "vitest";
import type { Document } from "@/api/types";
import {
  arrayMove,
  completeLink,
  linkCount,
  locate,
  newGroup,
  newLink,
  reduce,
  toShownPage,
} from "./document";

function link(id: string, title = id, published = true) {
  return { ...newLink(id), title, url: `https://exemple.fr/${id}`, published };
}

function doc(): Document {
  return {
    id: "page",
    slug: "musique",
    title: "Musique",
    description: "",
    avatar: null,
    theme: "zazi",
    accent: "violet",
    published: false,
    listed: true,
    version: 1,
    items: [link("a"), { ...newGroup("g"), title: "G", links: [link("b"), link("c")] }, link("d")],
  };
}

/** A compact view of the tree: "a,G[b,c],d". */
function tree(d: Document): string {
  return d.items
    .map((it) =>
      it.kind === "group" ? `${it.title}[${it.links.map((l) => l.id).join(",")}]` : it.id,
    )
    .join(",");
}

describe("editor document", () => {
  it("adds links at the top level and in groups, and groups", () => {
    let d = reduce(doc(), { type: "addLink", id: "x", container: null });
    d = reduce(d, { type: "addLink", id: "y", container: "g" });
    d = reduce(d, { type: "addGroup", id: "h" });
    expect(tree(d)).toBe("a,G[b,c,y],d,x,Nouveau groupe[]");
    // A link cannot be added to something that is not a group.
    expect(reduce(d, { type: "addLink", id: "z", container: "a" })).toBe(d);
  });

  it("updates an item wherever it is, without touching the original", () => {
    const before = doc();
    const after = reduce(before, { type: "update", id: "c", patch: { title: "Deezer" } });
    expect(locate(after, "c")?.item.title).toBe("Deezer");
    expect(locate(before, "c")?.item.title).toBe("c");
  });

  it("removes items; removing a group removes its links", () => {
    expect(tree(reduce(doc(), { type: "remove", id: "b" }))).toBe("a,G[c],d");
    expect(tree(reduce(doc(), { type: "remove", id: "g" }))).toBe("a,d");
  });

  it("duplicates a group with new IDs for its links", () => {
    const d = reduce(doc(), { type: "duplicate", id: "g", ids: ["g2", "b2", "c2"] });
    expect(tree(d)).toBe("a,G[b,c],G[b2,c2],d");
    expect(d.items[2]?.id).toBe("g2");
  });

  it("moves links between levels, never groups", () => {
    expect(tree(reduce(doc(), { type: "move", id: "a", to: "g" }))).toBe("G[b,c,a],d");
    expect(tree(reduce(doc(), { type: "move", id: "b", to: null }))).toBe("a,G[c],d,b");
    const d = doc();
    expect(reduce(d, { type: "move", id: "g", to: null })).toBe(d);
    expect(reduce(d, { type: "move", id: "a", to: "nowhere" })).toBe(d);
  });

  it("reorders within a container", () => {
    expect(tree(reduce(doc(), { type: "reorder", container: null, from: 2, to: 0 }))).toBe(
      "d,a,G[b,c]",
    );
    expect(tree(reduce(doc(), { type: "reorder", container: "g", from: 0, to: 1 }))).toBe(
      "a,G[c,b],d",
    );
    expect(arrayMove([1, 2, 3], 5, 0)).toEqual([1, 2, 3]);
  });

  it("keeps the saved version", () => {
    expect(reduce(doc(), { type: "saved", version: 7 }).version).toBe(7);
  });

  it("completes URLs and titles from the platform", () => {
    expect(completeLink("open.spotify.com/track/1", "")).toEqual({
      url: "https://open.spotify.com/track/1",
      title: "Spotify",
    });
    expect(completeLink("https://www.deezer.com/x", "Mon titre").title).toBe("Mon titre");
    expect(completeLink("mailto:moi@zazi.cool", "").title).toBe("E-mail");
    expect(completeLink("pas une url", "").url).toBe("pas une url");
  });

  it("builds the preview with or without drafts", () => {
    const d = doc();
    d.items[0] = link("a", "a", false);
    d.items[1] = { ...d.items[1]!, links: [link("b", "b", false)] };
    const withDrafts = toShownPage(d, true);
    expect(withDrafts.items.map((it) => it.id)).toEqual(["a", "g", "d"]);
    expect(withDrafts.items[0]).toMatchObject({ draft: true });
    // Without drafts, the group left empty disappears, as on the public page.
    expect(toShownPage(d, false).items.map((it) => it.id)).toEqual(["d"]);
  });

  it("counts links, those of groups included", () => {
    expect(linkCount(doc().items)).toBe(4);
  });
});
