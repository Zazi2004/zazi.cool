import { useState } from "preact/hooks";
import { reportPair, toUiError, type UiError } from "../api/client";
import type { ReportReason } from "../api/types";
import { RetryableError } from "./common";

interface Props {
  pairId: number;
  /** An extreme pair cannot be reported as "should be extreme". */
  extreme: boolean;
}

/** Report buttons for a pair. A single report takes effect immediately. */
export function ReportButtons({ pairId, extreme }: Props) {
  const [done, setDone] = useState<ReportReason | null>(null);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<{ error: UiError; reason: ReportReason } | null>(null);

  const report = async (reason: ReportReason) => {
    if (sending) return;
    setSending(true);
    setError(null);
    try {
      await reportPair(pairId, reason);
      setDone(reason);
    } catch (err) {
      setError({ error: toUiError(err), reason });
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
      {error && (
        <RetryableError error={error.error} busy={sending} onRetry={() => report(error.reason)} />
      )}
    </div>
  );
}
