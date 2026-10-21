import type { PageData } from "@/api/types";

/** Reads the data the Go server put in <script id="page-data">. */
export function readPageData(): PageData {
  const el = document.getElementById("page-data");
  if (!el?.textContent) return { view: "notFound" };
  return JSON.parse(el.textContent) as PageData;
}
