const miniRadius = 14;
const miniLength = 2 * Math.PI * miniRadius;

/** A small progress ring tinted by the enclosing `.color-*` class. */
export function MiniRing({ progress }: { progress: number }) {
  return (
    <svg className="mini-ring" viewBox="0 0 36 36" aria-hidden="true">
      <circle className="mini-track" cx="18" cy="18" r={miniRadius} />
      <circle
        className="mini-value"
        cx="18"
        cy="18"
        r={miniRadius}
        strokeDasharray={miniLength}
        strokeDashoffset={miniLength * (1 - progress)}
      />
    </svg>
  );
}
