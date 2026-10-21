import { useState } from "react";
import { Shape } from "@/components/visual/Shape";
import { cn } from "@/lib/cn";
import { shapeFamilies, shapeIds } from "@/shapes/list";
import { Dialog } from "../ui/Dialog";

const familyLabels: Record<(typeof shapeFamilies)[number], string> = {
  star: "Étoiles",
  flower: "Fleurs",
  ellipse: "Ellipses",
  wheel: "Roues",
  moon: "Lunes",
  misc: "Divers",
  triangle: "Triangles",
  polygon: "Polygones",
  rectangle: "Rectangles",
  number: "Chiffres",
};

interface ShapePickerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  value: string | null;
  onPick: (shape: string) => void;
}

/** A grid of the coolshapes shapes, family by family. */
export function ShapePicker({ open, onOpenChange, value, onPick }: ShapePickerProps) {
  const [family, setFamily] = useState<(typeof shapeFamilies)[number]>(
    (value?.split("-")[0] as (typeof shapeFamilies)[number] | undefined) ?? "star",
  );
  const shapes = shapeIds.filter((id) => id.startsWith(family + "-"));

  return (
    <Dialog open={open} onOpenChange={onOpenChange} title="Choisir une forme">
      <div role="tablist" aria-label="Familles de formes" className="flex flex-wrap gap-1.5">
        {shapeFamilies.map((f) => (
          <button
            key={f}
            type="button"
            role="tab"
            aria-selected={f === family}
            onClick={() => setFamily(f)}
            className={cn(
              "cursor-pointer rounded-full px-3 py-1 text-xs font-medium",
              f === family
                ? "bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900"
                : "bg-zinc-100 hover:bg-zinc-200 dark:bg-zinc-800 dark:hover:bg-zinc-700",
            )}
          >
            {familyLabels[f]}
          </button>
        ))}
      </div>
      <div className="grid grid-cols-4 gap-2 sm:grid-cols-5">
        {shapes.map((id) => (
          <button
            key={id}
            type="button"
            aria-label={`Forme ${id}`}
            aria-pressed={id === value}
            onClick={() => {
              onPick(id);
              onOpenChange(false);
            }}
            className={cn(
              "flex aspect-square cursor-pointer items-center justify-center rounded-xl border-2 p-2 transition hover:scale-105",
              id === value
                ? "border-violet-500 bg-violet-50 dark:bg-violet-950"
                : "border-transparent hover:bg-zinc-100 dark:hover:bg-zinc-800",
            )}
          >
            <Shape id={id} size={56} />
          </button>
        ))}
      </div>
    </Dialog>
  );
}
