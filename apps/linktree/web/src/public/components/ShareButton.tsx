import { Check, Share2 } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useState } from "react";

interface ShareButtonProps {
  title: string;
  disabled?: boolean;
}

/** Shares the page: the system share sheet on phones, otherwise copies the URL. */
export function ShareButton({ title, disabled }: ShareButtonProps) {
  const [copied, setCopied] = useState(false);

  async function share() {
    const url = window.location.href;
    if (navigator.share) {
      try {
        await navigator.share({ title, url });
      } catch {
        // Closing the share sheet rejects the promise: nothing to report.
      }
      return;
    }
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch (err) {
      console.error("copy failed", err);
    }
  }

  return (
    <m.button
      type="button"
      onClick={disabled ? undefined : share}
      whileTap={{ scale: 0.9 }}
      aria-label={copied ? "Lien copié" : "Partager la page"}
      className="lt-card lt-link lt-focus flex size-10 cursor-pointer items-center justify-center rounded-full"
    >
      <AnimatePresence mode="wait" initial={false}>
        <m.span
          key={copied ? "copied" : "share"}
          initial={{ scale: 0.4, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          exit={{ scale: 0.4, opacity: 0 }}
          transition={{ duration: 0.15 }}
        >
          {copied ? <Check className="size-4.5" /> : <Share2 className="size-4.5" />}
        </m.span>
      </AnimatePresence>
      <span role="status" className="sr-only">
        {copied ? "Lien copié dans le presse-papiers" : ""}
      </span>
    </m.button>
  );
}
