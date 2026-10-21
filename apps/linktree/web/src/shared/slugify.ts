// Same rules as pages.ValidateSlug in Go (which has the last word).
export const slugPattern = /^[a-z0-9]+(-[a-z0-9]+)*$/;
export const maxSlug = 40;

/** Proposes a slug from a title: "Jeux vidéo !" → "jeux-video". */
export function slugify(title: string): string {
  const slug = title
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "") // accents, split from their letter by NFD
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  if (slug.length <= maxSlug) return slug;
  // Cut at a word boundary when possible.
  const cut = slug.slice(0, maxSlug);
  const lastDash = cut.lastIndexOf("-");
  return (lastDash > maxSlug / 2 ? cut.slice(0, lastDash) : cut).replace(/-+$/, "");
}
