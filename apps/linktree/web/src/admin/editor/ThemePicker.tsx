import { Check } from "lucide-react";
import { Shape } from "@/components/visual/Shape";
import { cn } from "@/lib/cn";
import { accentIds, accents, themeIds, themes, type AccentId, type ThemeId } from "@/themes/tokens";

interface ThemePickerProps {
  theme: ThemeId;
  accent: AccentId;
  onTheme: (theme: ThemeId) => void;
  onAccent: (accent: AccentId) => void;
}

/**
 * The themes as small cards drawn with their own CSS (gradient and card colors of
 * the chosen accent), then the accent colors.
 */
export function ThemePicker({ theme, accent, onTheme, onAccent }: ThemePickerProps) {
  return (
    <div className="flex flex-col gap-4">
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <legend className="mb-2 text-sm font-medium">Thème</legend>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
          {themeIds.map((id) => (
            <button
              key={id}
              type="button"
              aria-pressed={id === theme}
              onClick={() => onTheme(id)}
              className={cn(
                "group cursor-pointer overflow-hidden rounded-xl border-2 p-0 text-left transition",
                id === theme
                  ? "border-violet-500"
                  : "border-transparent hover:border-zinc-300 dark:hover:border-zinc-700",
              )}
            >
              <div
                data-theme={id}
                data-accent={accent}
                className="lt-backdrop relative h-20 overflow-hidden"
              >
                <Shape
                  id={id === "pop" ? "flower-3" : "star-1"}
                  size={34}
                  className="absolute top-2 left-2"
                />
                <div className="lt-card absolute right-2 bottom-2 left-12 h-6" />
              </div>
              <div className="bg-white px-2.5 py-2 dark:bg-zinc-900">
                <span className="flex items-center justify-between text-sm font-medium">
                  {themes[id].label}
                  {id === theme && <Check className="size-4 text-violet-600" />}
                </span>
                <span className="line-clamp-2 text-xs text-zinc-500">{themes[id].description}</span>
              </div>
            </button>
          ))}
        </div>
      </fieldset>

      <fieldset className="m-0 border-0 p-0">
        <legend className="mb-2 text-sm font-medium">Couleur d'accent</legend>
        <div className="flex flex-wrap gap-2">
          {accentIds.map((id) => (
            <button
              key={id}
              type="button"
              aria-pressed={id === accent}
              aria-label={accents[id].label}
              title={accents[id].label}
              onClick={() => onAccent(id)}
              style={{
                background: `linear-gradient(135deg, ${accents[id].bright}, ${accents[id].deep})`,
              }}
              className={cn(
                "size-9 cursor-pointer rounded-full ring-offset-2 transition hover:scale-110 dark:ring-offset-zinc-950",
                id === accent && "ring-2 ring-zinc-900 dark:ring-zinc-100",
              )}
            />
          ))}
        </div>
      </fieldset>
    </div>
  );
}
