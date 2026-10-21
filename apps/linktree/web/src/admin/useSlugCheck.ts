import { useEffect, useState } from "react";
import { slugPattern } from "@/shared/slugify";
import { api } from "./api";

/**
 * Checks a slug while it is typed (after a short pause), against the same rules as
 * the server and the slugs already used. Returns the message to show, or "".
 * `except` is the page being edited, whose own slug is of course available.
 */
export function useSlugCheck(slug: string, except?: string): string {
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (slug === "") {
      setMessage("");
      return;
    }
    if (!slugPattern.test(slug)) {
      setMessage("Lettres minuscules sans accents, chiffres et tirets uniquement.");
      return;
    }
    let cancelled = false;
    const timer = setTimeout(() => {
      api.checkSlug(slug, except).then(
        (r) => !cancelled && setMessage(r.error),
        () => !cancelled && setMessage(""),
      );
    }, 300);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [slug, except]);

  return message;
}
