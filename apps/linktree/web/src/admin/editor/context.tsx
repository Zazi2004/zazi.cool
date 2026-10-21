import { createContext, useContext, type Dispatch } from "react";
import type { Action } from "./document";

/** What the parts of the editor share, to avoid passing it through every level. */
export interface EditorContextValue {
  dispatch: Dispatch<Action>;
  /** Server validation errors, by field ("title", "items.<id>.url"…). */
  errors: Record<string, string>;
  expanded: Set<string>;
  toggle: (id: string) => void;
  /** The groups of the page, as destinations of "Déplacer vers…". */
  groups: { id: string; title: string }[];
}

export const EditorContext = createContext<EditorContextValue | null>(null);

export function useEditor(): EditorContextValue {
  const value = useContext(EditorContext);
  if (!value) throw new Error("useEditor outside of the editor");
  return value;
}
