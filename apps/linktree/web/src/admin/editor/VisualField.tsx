import { ImageUp, Shapes, X } from "lucide-react";
import { useRef, useState } from "react";
import type { Visual as VisualData } from "@/api/types";
import { Visual } from "@/components/visual/Visual";
import { ApiError, api } from "../api";
import { Button } from "../ui/Button";
import { shapeVisual } from "./document";
import { prepareImage } from "./image";
import { ShapePicker } from "./ShapePicker";

interface VisualFieldProps {
  label: string;
  value: VisualData | null;
  onChange: (visual: VisualData | null) => void;
  /** Links take images only; pages and groups also take shapes. */
  allowShape: boolean;
  error?: string;
}

/** Chooses an image (uploaded at once), a shape, or nothing. */
export function VisualField({ label, value, onChange, allowShape, error }: VisualFieldProps) {
  const input = useRef<HTMLInputElement>(null);
  const [picking, setPicking] = useState(false);
  const [busy, setBusy] = useState(false);
  const [uploadError, setUploadError] = useState("");

  async function upload(file: File) {
    setBusy(true);
    setUploadError("");
    try {
      onChange(await api.uploadImage(await prepareImage(file)));
    } catch (err) {
      setUploadError(
        err instanceof ApiError
          ? (Object.values(err.fields)[0] ?? err.message)
          : "Image illisible.",
      );
    } finally {
      setBusy(false);
      if (input.current) input.current.value = "";
    }
  }

  const message = uploadError || error;
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-sm font-medium">{label}</span>
      <div className="flex items-center gap-3">
        <div className="flex size-14 shrink-0 items-center justify-center rounded-xl border border-dashed border-zinc-300 dark:border-zinc-700">
          {value && <Visual visual={value} size={48} imageClassName="rounded-lg" />}
        </div>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" onClick={() => input.current?.click()} disabled={busy}>
            <ImageUp />
            {busy ? "Envoi…" : "Image"}
          </Button>
          {allowShape && (
            <Button size="sm" onClick={() => setPicking(true)}>
              <Shapes />
              Forme
            </Button>
          )}
          {value && (
            <Button size="sm" variant="ghost" onClick={() => onChange(null)}>
              <X />
              Retirer
            </Button>
          )}
        </div>
      </div>
      {message && <p className="m-0 text-xs text-red-600 dark:text-red-400">{message}</p>}
      <input
        ref={input}
        type="file"
        accept="image/*"
        hidden
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) void upload(file);
        }}
      />
      {allowShape && (
        <ShapePicker
          open={picking}
          onOpenChange={setPicking}
          value={value?.kind === "shape" ? value.shape : null}
          onPick={(shape) => onChange(shapeVisual(shape))}
        />
      )}
    </div>
  );
}
