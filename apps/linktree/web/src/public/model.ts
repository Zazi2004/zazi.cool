import type { PublicGroup, PublicLink, PublicPage } from "@/api/types";

// The page as the components draw it: the public data, plus a "draft" flag that
// only the admin preview sets (visitors never receive drafts).
export type ShownLink = PublicLink & { draft?: boolean };
export type ShownGroup = Omit<PublicGroup, "links"> & { links: ShownLink[]; draft?: boolean };
export type ShownItem = ShownLink | ShownGroup;
export type ShownPage = Omit<PublicPage, "items"> & { items: ShownItem[] };
