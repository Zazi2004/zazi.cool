// Form fields in the style of shadcn/ui (MIT), written for this app.
import type { InputHTMLAttributes, ReactNode, TextareaHTMLAttributes } from "react";
import { useId } from "react";
import { cn } from "@/lib/cn";

const control =
  "w-full rounded-lg border border-zinc-200 bg-white px-3 py-2 text-sm outline-none placeholder:text-zinc-400 focus:border-violet-500 focus:ring-2 focus:ring-violet-500/20 aria-invalid:border-red-500 dark:border-zinc-800 dark:bg-zinc-900";

export function Input({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cn(control, "h-9", className)} {...props} />;
}

export function Textarea({ className, ...props }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={cn(control, "min-h-20 resize-y", className)} {...props} />;
}

interface FieldProps {
  label: string;
  error?: string;
  hint?: ReactNode;
  /** Renders the control, wired to the label and the error message. */
  children: (props: {
    id: string;
    "aria-invalid"?: true;
    "aria-describedby"?: string;
  }) => ReactNode;
  className?: string;
}

/** A label, a control and its error or hint, accessible together. */
export function Field({ label, error, hint, children, className }: FieldProps) {
  const id = useId();
  const messageId = `${id}-message`;
  const message = error || hint;
  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      <label htmlFor={id} className="text-sm font-medium">
        {label}
      </label>
      {children({
        id,
        ...(error ? { "aria-invalid": true as const } : {}),
        ...(message ? { "aria-describedby": messageId } : {}),
      })}
      {message && (
        <p
          id={messageId}
          className={cn("m-0 text-xs", error ? "text-red-600 dark:text-red-400" : "text-zinc-500")}
        >
          {message}
        </p>
      )}
    </div>
  );
}
