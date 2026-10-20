import type { ComponentChildren } from "preact";
import { useEffect, useRef } from "preact/hooks";
import { nextFocusIndex } from "./focusTrap";

interface ModalProps {
  title: string;
  children: ComponentChildren;
  /** Called on Escape and on a click on the backdrop. */
  onClose: () => void;
}

const FOCUSABLE =
  "button:not(:disabled), [href], input:not(:disabled), [tabindex]:not([tabindex='-1'])";

/**
 * An accessible modal dialog. Focus starts on the last action (the safe one is placed
 * there by callers), is trapped inside while open and goes back to the opener on close.
 */
export function Modal({ title, children, onClose }: ModalProps) {
  const dialog = useRef<HTMLDivElement>(null);
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  useEffect(() => {
    const opener = document.activeElement as HTMLElement | null;
    const root = dialog.current;
    if (!root) return;
    const focusables = () => [...root.querySelectorAll<HTMLElement>(FOCUSABLE)];
    (root.querySelector<HTMLElement>("[data-autofocus]") ?? focusables()[0] ?? root).focus();
    document.body.classList.add("modal-open");

    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onCloseRef.current();
        return;
      }
      if (e.key !== "Tab") return;
      const items = focusables();
      if (items.length === 0) {
        e.preventDefault();
        return;
      }
      const current = items.indexOf(document.activeElement as HTMLElement);
      const next = nextFocusIndex(items.length, current, e.shiftKey);
      // Only intervene at the edges (or if focus escaped): the browser handles the rest.
      if (current === -1 || next !== current + (e.shiftKey ? -1 : 1)) {
        e.preventDefault();
        items[next]!.focus();
      }
    };
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.classList.remove("modal-open");
      opener?.focus({ preventScroll: true });
    };
  }, []);

  return (
    <div
      class="modal-backdrop"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        class="modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-title"
        tabIndex={-1}
        ref={dialog}
      >
        <h2 id="modal-title">{title}</h2>
        {children}
      </div>
    </div>
  );
}
