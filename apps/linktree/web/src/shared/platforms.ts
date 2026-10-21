// Finds the platform of a link from its URL: its icon on the public page, and the
// default title of a new link in the admin.
import {
  siApplemusic,
  siBandcamp,
  siBluesky,
  siDeezer,
  siDiscord,
  siEpicgames,
  siFacebook,
  siGithub,
  siInstagram,
  siItchdotio,
  siPlaystation,
  siReddit,
  siSoundcloud,
  siSpotify,
  siSteam,
  siThreads,
  siTidal,
  siTiktok,
  siTwitch,
  siX,
  siYoutube,
  siYoutubemusic,
} from "simple-icons";

/** An icon from Simple Icons (CC0): a 24×24 SVG path and the brand color. */
export interface BrandIcon {
  path: string;
  hex: string;
}

export interface Platform {
  id: string;
  name: string;
  /** Missing for brands that are not in Simple Icons: a generic icon is shown. */
  icon?: BrandIcon;
}

type Matcher = (host: string) => boolean;

/** Matches the domain itself and its subdomains (open.spotify.com…). */
function domain(...names: string[]): Matcher {
  return (host) => names.some((n) => host === n || host.endsWith("." + n));
}

/** Matches a brand under any top-level domain (music.amazon.fr, nintendo.co.uk…). */
function anyTld(pattern: RegExp): Matcher {
  return (host) => pattern.test(host);
}

// Order matters: the first match wins (YouTube Music before YouTube).
const platforms: (Platform & { match: Matcher })[] = [
  {
    id: "spotify",
    name: "Spotify",
    icon: siSpotify,
    match: domain("spotify.com", "spotify.link", "spoti.fi"),
  },
  { id: "deezer", name: "Deezer", icon: siDeezer, match: domain("deezer.com", "deezer.page.link") },
  { id: "applemusic", name: "Apple Music", icon: siApplemusic, match: domain("music.apple.com") },
  {
    id: "youtubemusic",
    name: "YouTube Music",
    icon: siYoutubemusic,
    match: domain("music.youtube.com"),
  },
  { id: "youtube", name: "YouTube", icon: siYoutube, match: domain("youtube.com", "youtu.be") },
  { id: "soundcloud", name: "SoundCloud", icon: siSoundcloud, match: domain("soundcloud.com") },
  { id: "bandcamp", name: "Bandcamp", icon: siBandcamp, match: domain("bandcamp.com") },
  { id: "tidal", name: "Tidal", icon: siTidal, match: domain("tidal.com") },
  { id: "amazonmusic", name: "Amazon Music", match: anyTld(/(^|\.)music\.amazon\.[a-z.]+$/) },
  { id: "qobuz", name: "Qobuz", match: domain("qobuz.com") },
  {
    id: "instagram",
    name: "Instagram",
    icon: siInstagram,
    match: domain("instagram.com", "instagr.am"),
  },
  { id: "tiktok", name: "TikTok", icon: siTiktok, match: domain("tiktok.com") },
  { id: "x", name: "X", icon: siX, match: domain("x.com", "twitter.com") },
  { id: "bluesky", name: "Bluesky", icon: siBluesky, match: domain("bsky.app") },
  { id: "threads", name: "Threads", icon: siThreads, match: domain("threads.net", "threads.com") },
  {
    id: "facebook",
    name: "Facebook",
    icon: siFacebook,
    match: domain("facebook.com", "fb.com", "fb.me"),
  },
  { id: "reddit", name: "Reddit", icon: siReddit, match: domain("reddit.com", "redd.it") },
  { id: "twitch", name: "Twitch", icon: siTwitch, match: domain("twitch.tv") },
  { id: "discord", name: "Discord", icon: siDiscord, match: domain("discord.gg", "discord.com") },
  { id: "github", name: "GitHub", icon: siGithub, match: domain("github.com") },
  { id: "linkedin", name: "LinkedIn", match: domain("linkedin.com", "lnkd.in") },
  {
    id: "steam",
    name: "Steam",
    icon: siSteam,
    match: domain("steampowered.com", "steamcommunity.com"),
  },
  { id: "itchio", name: "itch.io", icon: siItchdotio, match: domain("itch.io") },
  { id: "epicgames", name: "Epic Games", icon: siEpicgames, match: domain("epicgames.com") },
  { id: "playstation", name: "PlayStation", icon: siPlaystation, match: domain("playstation.com") },
  { id: "xbox", name: "Xbox", match: domain("xbox.com") },
  { id: "nintendo", name: "Nintendo", match: anyTld(/(^|\.)nintendo\.[a-z.]+$/) },
];

export const email: Platform = { id: "email", name: "E-mail" };
export const phone: Platform = { id: "phone", name: "Téléphone" };

/** Returns the platform of url, or null for an unknown site or an invalid URL. */
export function platformOf(url: string): Platform | null {
  let u: URL;
  try {
    u = new URL(url);
  } catch {
    return null;
  }
  if (u.protocol === "mailto:") return email;
  if (u.protocol === "tel:") return phone;
  if (u.protocol !== "https:" && u.protocol !== "http:") return null;
  const host = u.hostname.toLowerCase().replace(/^www\./, "");
  const found = platforms.find((p) => p.match(host));
  if (!found) return null;
  return { id: found.id, name: found.name, icon: found.icon };
}
