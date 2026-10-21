import { Globe, Link2, Mail, Phone } from "lucide-react";
import { platformOf } from "@/shared/platforms";
import { cn } from "@/lib/cn";

interface PlatformIconProps {
  url: string;
  className?: string;
}

/** The icon of the link's platform, or a generic one. */
export function PlatformIcon({ url, className }: PlatformIconProps) {
  const platform = platformOf(url);
  const classes = cn("size-5 shrink-0", className);
  if (platform?.icon) {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true" className={classes} fill="currentColor">
        <path d={platform.icon.path} />
      </svg>
    );
  }
  if (platform?.id === "email") return <Mail aria-hidden="true" className={classes} />;
  if (platform?.id === "phone") return <Phone aria-hidden="true" className={classes} />;
  return platform ? (
    <Globe aria-hidden="true" className={classes} />
  ) : (
    <Link2 aria-hidden="true" className={classes} />
  );
}
