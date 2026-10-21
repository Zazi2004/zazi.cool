import { describe, expect, it } from "vitest";
import { maxSlug, slugify, slugPattern } from "./slugify";

describe("slugify", () => {
  it.each([
    ["Musique", "musique"],
    ["Jeux vidéo !", "jeux-video"],
    ["  Ça c'est l'été  ", "ca-c-est-l-ete"],
    ["Top 10 — 2026", "top-10-2026"],
    ["Œuvres", "uvres"],
    ["!!!", ""],
  ])("%s → %s", (title, slug) => {
    expect(slugify(title)).toBe(slug);
  });

  it("produces valid slugs of at most 40 characters", () => {
    const slug = slugify("Une très longue page de liens consacrée à la musique électronique");
    expect(slug.length).toBeLessThanOrEqual(maxSlug);
    expect(slug).toMatch(slugPattern);
    expect(slug).toBe("une-tres-longue-page-de-liens-consacree");
  });
});
