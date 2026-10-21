import type { Document } from "@/api/types";
import { Field, Input, Textarea } from "../ui/Field";
import { Switch } from "../ui/Switch";
import { useSlugCheck } from "../useSlugCheck";
import { useEditor } from "./context";
import { ThemePicker } from "./ThemePicker";
import { VisualField } from "./VisualField";

interface PageSettingsProps {
  doc: Document;
  /** The slug as last saved: changing it on a published page breaks shared links. */
  savedSlug: string;
  savedPublished: boolean;
}

export function PageSettings({ doc, savedSlug, savedPublished }: PageSettingsProps) {
  const { dispatch, errors } = useEditor();
  const set = (patch: Partial<Document>) => dispatch({ type: "setPage", patch });
  const slugMessage = useSlugCheck(doc.slug, doc.id);
  const slugWarning =
    savedPublished && doc.slug !== savedSlug
      ? "Les liens déjà partagés vers l'ancienne adresse ne marcheront plus."
      : undefined;

  return (
    <section className="flex flex-col gap-4">
      <h2 className="m-0 text-base font-semibold">Page</h2>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Titre" error={errors.title}>
          {(props) => (
            <Input
              {...props}
              maxLength={80}
              value={doc.title}
              onChange={(e) => set({ title: e.target.value })}
            />
          )}
        </Field>
        <Field
          label="Adresse"
          error={errors.slug || slugMessage}
          hint={slugWarning ?? `link.zazi.cool/${doc.slug}`}
        >
          {(props) => (
            <Input
              {...props}
              maxLength={40}
              value={doc.slug}
              onChange={(e) => set({ slug: e.target.value })}
            />
          )}
        </Field>
      </div>
      <Field
        label="Description"
        error={errors.description}
        hint={`${doc.description.length} / 280`}
      >
        {(props) => (
          <Textarea
            {...props}
            maxLength={280}
            value={doc.description}
            onChange={(e) => set({ description: e.target.value })}
          />
        )}
      </Field>
      <VisualField
        label="Avatar"
        value={doc.avatar}
        onChange={(avatar) => set({ avatar })}
        allowShape
        error={errors.avatar}
      />
      <ThemePicker
        theme={doc.theme}
        accent={doc.accent}
        onTheme={(theme) => set({ theme })}
        onAccent={(accent) => set({ accent })}
      />
      {(errors.theme || errors.accent) && (
        <p className="m-0 text-xs text-red-600 dark:text-red-400">
          {errors.theme || errors.accent}
        </p>
      )}
      <div className="flex flex-col gap-3 rounded-xl border border-zinc-200 p-3 dark:border-zinc-800">
        <Switch
          label="Publiée"
          hint="Visible à son adresse. Sinon, elle répond « page introuvable »."
          checked={doc.published}
          onChange={(published) => set({ published })}
        />
        <Switch
          label="Listée"
          hint="Présente dans l'index de link.zazi.cool et dans les moteurs de recherche."
          checked={doc.listed}
          onChange={(listed) => set({ listed })}
        />
      </div>
    </section>
  );
}
