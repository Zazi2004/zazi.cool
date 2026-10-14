// A minimal client-side router: four pages do not justify a routing library.
import type { ComponentChildren } from "preact";
import { useEffect, useState } from "preact/hooks";

export function navigate(path: string): void {
  history.pushState(null, "", path);
  dispatchEvent(new PopStateEvent("popstate"));
  window.scrollTo(0, 0);
}

/** The current path, updated on navigation and on the browser's back button. */
export function usePath(): string {
  const [path, setPath] = useState(location.pathname);
  useEffect(() => {
    const onChange = () => setPath(location.pathname);
    addEventListener("popstate", onChange);
    return () => removeEventListener("popstate", onChange);
  }, []);
  return path;
}

interface LinkProps {
  href: string;
  class?: string;
  children: ComponentChildren;
}

export function Link({ href, class: className, children }: LinkProps) {
  return (
    <a
      href={href}
      class={className}
      onClick={(e) => {
        // Keep the browser's behaviour for "open in a new tab".
        if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
        e.preventDefault();
        navigate(href);
      }}
    >
      {children}
    </a>
  );
}
