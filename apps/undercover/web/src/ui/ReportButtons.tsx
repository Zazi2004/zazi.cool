import { useState } from "preact/hooks";
import { reportPair } from "../api/client";
import type { ReportReason } from "../api/types";
import { ErrorText } from "./common";

interface Props {
  pairId: number;
  /** An extreme pair cannot be reported as "should be extreme". */
  extreme: boolean;
}

/** Report buttons for a pair. A single report takes effect immediately. */
export function ReportButtons({ pairId, extreme }: Props) {
  const [done, setDone] = useState<ReportReason | null>(null);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const report = async (reason: ReportReason) => {
    setSending(true);
    setError(null);
    try {
      await reportPair(pairId, reason);
      setDone(reason);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur inattendue.");
    } finally {
      setSending(false);
    }
  };

  if (done) {
    return (
      <p class="hint">
        Merci, c'est noté :{" "}
        {done === "extreme" ? "la paire passe en mode extrême." : "la paire ne sera plus proposée."}
      </p>
    );
  }
  return (
    <div class="report">
      {!extreme && (
        <button
          type="button"
          class="button small"
          disabled={sending}
          onClick={() => report("extreme")}
        >
          Devrait être extrême
        </button>
      )}
      <button
        type="button"
        class="button small"
        disabled={sending}
        onClick={() => report("remove")}
      >
        À supprimer
      </button>
      {error && <ErrorText>{error}</ErrorText>}
    </div>
  );
}
