import { useEffect, useState } from "preact/hooks";
import { checkPair, createPair } from "../api/client";
import type { CheckResponse, PairView } from "../api/types";
import { ErrorText } from "./common";
import { ReportButtons } from "./ReportButtons";

type Category = "normal" | "extreme";

/** Delay after the last keystroke before checking redundancy. */
const CHECK_DELAY_MS = 400;

export function ProposePage() {
  const [wordA, setWordA] = useState("");
  const [wordB, setWordB] = useState("");
  const [category, setCategory] = useState<Category | null>(null);
  const [website, setWebsite] = useState(""); // honeypot
  const [check, setCheck] = useState<CheckResponse | null>(null);
  const [confirmSimilar, setConfirmSimilar] = useState(false);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<PairView | null>(null);

  // Live redundancy check, debounced; a newer keystroke cancels the pending request.
  useEffect(() => {
    setCheck(null);
    setConfirmSimilar(false);
    setError(null);
    if (wordA.trim() === "" || wordB.trim() === "") return;
    const controller = new AbortController();
    const timer = setTimeout(() => {
      checkPair(wordA, wordB, controller.signal)
        .then(setCheck)
        .catch((err: unknown) => {
          if (err instanceof DOMException && err.name === "AbortError") return;
          setError(err instanceof Error ? err.message : "Erreur inattendue.");
        });
    }, CHECK_DELAY_MS);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [wordA, wordB]);

  const blocked =
    check?.status === "duplicate" ||
    check?.status === "invalid" ||
    (check?.status === "similar" && !confirmSimilar);
  const canSubmit =
    wordA.trim() !== "" && wordB.trim() !== "" && category !== null && !blocked && !sending;

  const submit = async (e: Event) => {
    e.preventDefault();
    if (!canSubmit) return;
    setSending(true);
    setError(null);
    const result = await createPair({
      wordA,
      wordB,
      extreme: category === "extreme",
      confirmSimilar,
      website,
    });
    setSending(false);
    if (result.ok) {
      setSuccess(result.response.pair);
      setWordA("");
      setWordB("");
      setCategory(null);
    } else if ("check" in result) {
      setCheck(result.check);
    } else {
      setError(result.error);
    }
  };

  return (
    <form class="stack" onSubmit={submit}>
      <h1>Proposer une paire</h1>
      <p class="hint">
        Deux mots proches mais différents, comme « chat / tigre ». La paire est ajoutée au jeu
        immédiatement.
      </p>

      {success && (
        <p class="success" role="status">
          Merci ! La paire « {success.wordA} / {success.wordB} » est ajoutée.
        </p>
      )}

      <div class="card stack">
        <label class="field">
          <span>Premier mot</span>
          <input
            type="text"
            value={wordA}
            maxLength={40}
            autocomplete="off"
            onInput={(e) => {
              setWordA(e.currentTarget.value);
              setSuccess(null);
            }}
          />
        </label>
        <label class="field">
          <span>Second mot</span>
          <input
            type="text"
            value={wordB}
            maxLength={40}
            autocomplete="off"
            onInput={(e) => {
              setWordB(e.currentTarget.value);
              setSuccess(null);
            }}
          />
        </label>

        {/* Hidden from humans: only bots fill this field. */}
        <div class="honeypot" aria-hidden="true">
          <label>
            Site web
            <input
              type="text"
              tabIndex={-1}
              autocomplete="off"
              value={website}
              onInput={(e) => setWebsite(e.currentTarget.value)}
            />
          </label>
        </div>
      </div>

      <fieldset class="card stack">
        <legend>Catégorie (obligatoire)</legend>
        <label class="checkbox">
          <input
            type="radio"
            name="category"
            checked={category === "normal"}
            onChange={() => setCategory("normal")}
          />
          <span>Tout public</span>
        </label>
        <label class="checkbox">
          <input
            type="radio"
            name="category"
            checked={category === "extreme"}
            onChange={() => setCategory("extreme")}
          />
          <span>Extrême : politique, religion, sous-entendus, humour noir…</span>
        </label>
      </fieldset>

      {check && (
        <CheckResult check={check} confirmed={confirmSimilar} onConfirm={setConfirmSimilar} />
      )}
      {error && <ErrorText>{error}</ErrorText>}

      <button type="submit" class="button primary" disabled={!canSubmit}>
        {sending ? "Envoi…" : "Ajouter la paire"}
      </button>
    </form>
  );
}

interface CheckResultProps {
  check: CheckResponse;
  confirmed: boolean;
  onConfirm: (confirmed: boolean) => void;
}

function CheckResult({ check, confirmed, onConfirm }: CheckResultProps) {
  switch (check.status) {
    case "invalid":
      return <ErrorText>{check.error}</ErrorText>;
    case "duplicate":
      return (
        <div class="card stack">
          <ErrorText>Cette paire existe déjà.</ErrorText>
          {check.duplicate && <PairList pairs={[check.duplicate]} />}
        </div>
      );
    case "similar":
      return (
        <div class="card stack">
          <p class="warning">Des paires très proches existent déjà :</p>
          <PairList pairs={check.similar} />
          <label class="checkbox">
            <input
              type="checkbox"
              checked={confirmed}
              onChange={(e) => onConfirm(e.currentTarget.checked)}
            />
            <span>Ma paire est bien différente</span>
          </label>
          <Related pairs={check.related} />
        </div>
      );
    case "ok":
      return check.related.length > 0 ? (
        <div class="card stack">
          <Related pairs={check.related} />
        </div>
      ) : (
        <p class="success">Paire inédite.</p>
      );
  }
}

function Related({ pairs }: { pairs: PairView[] }) {
  if (pairs.length === 0) return null;
  return (
    <>
      <p class="hint">Pour info, ces paires utilisent déjà un de vos mots :</p>
      <PairList pairs={pairs} />
    </>
  );
}

function PairList({ pairs }: { pairs: PairView[] }) {
  return (
    <ul class="pair-list">
      {pairs.map((p) => (
        <li key={p.id}>
          <details>
            <summary>
              {p.wordA} / {p.wordB}
              {p.extreme && <span class="tag">extrême</span>}
            </summary>
            <ReportButtons pairId={p.id} extreme={p.extreme} />
          </details>
        </li>
      ))}
    </ul>
  );
}
