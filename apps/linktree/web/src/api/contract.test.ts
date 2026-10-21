import { describe, expect, it } from "vitest";
import { accentIds, themeIds } from "@/themes/tokens";
import goContract from "./contract.json";
import type { Contract, DocItem, ImageVisual, PublicLink, ShapeVisual } from "./types";

const id = "0199e6a8-0000-7000-8000-000000000001";
const image: ImageVisual = { kind: "image", id, url: `/images/${id}`, width: 512, height: 384 };
const shape: ShapeVisual = { kind: "shape", shape: "star-1" };
const link: PublicLink = {
  kind: "link",
  id,
  title: "Spotify",
  url: "https://open.spotify.com/track/1",
  visual: image,
};
const docLink: DocItem = {
  id,
  kind: "link",
  title: "Spotify",
  subtitle: "",
  url: "https://open.spotify.com/track/1",
  visual: image,
  openByDefault: false,
  published: true,
  links: [],
};

// Every field set, so that a field missing on either side shows up as a difference.
const tsContract: Contract = {
  themes: themeIds,
  accents: accentIds,
  imageVisual: image,
  shapeVisual: shape,
  indexView: {
    view: "index",
    pages: [{ slug: "musique", title: "Musique", description: "Mes sons", avatar: image }],
  },
  notFoundView: { view: "notFound" },
  pageView: {
    view: "page",
    page: {
      id,
      slug: "musique",
      title: "Musique",
      description: "Mes sons",
      avatar: image,
      theme: "nuit",
      accent: "violet",
      items: [
        link,
        {
          kind: "group",
          id,
          title: "Titre 1",
          subtitle: "Artiste",
          visual: shape,
          openByDefault: true,
          links: [link],
        },
      ],
    },
  },
  loginRequest: { password: "secret" },
  sessionResponse: { authenticated: true },
  pagesResponse: {
    pages: [
      {
        id,
        slug: "musique",
        title: "Musique",
        description: "Mes sons",
        avatar: shape,
        published: true,
        listed: true,
        linkCount: 3,
      },
    ],
  },
  createPageRequest: { title: "Musique", slug: "musique" },
  document: {
    id,
    slug: "musique",
    title: "Musique",
    description: "Mes sons",
    avatar: image,
    theme: "nuit",
    accent: "violet",
    published: true,
    listed: true,
    version: 2,
    items: [
      docLink,
      {
        id,
        kind: "group",
        title: "Titre 1",
        subtitle: "Artiste",
        url: "",
        visual: shape,
        openByDefault: true,
        published: true,
        links: [docLink],
      },
    ],
  },
  saveResponse: { version: 3 },
  reorderRequest: { ids: [id] },
  slugCheckResponse: { slug: "musique", available: false, error: "Ce slug est déjà utilisé." },
  imageResponse: { visual: image },
  errorResponse: {
    error: {
      code: "validation",
      message: "Certains champs sont invalides.",
      fields: { slug: "Ce slug est déjà utilisé." },
    },
  },
};

/** Replaces every value by its type, keeping the structure and the field names. */
function structure(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(structure);
  if (value !== null && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([k, v]) => [k, structure(v)]),
    );
  }
  return value === null ? "null" : typeof value;
}

describe("API contract", () => {
  it("matches the Go message types (contract.json)", () => {
    expect(structure(tsContract)).toEqual(structure(goContract));
  });

  it("uses the same values, theme and accent lists included", () => {
    // contract.json comes from Go: its values must be valid for the TS types.
    expect(tsContract).toEqual(goContract);
  });
});
