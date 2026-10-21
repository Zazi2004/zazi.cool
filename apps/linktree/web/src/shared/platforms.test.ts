import { describe, expect, it } from "vitest";
import { platformOf } from "./platforms";

describe("platformOf", () => {
  it.each([
    ["https://open.spotify.com/track/1?si=a", "spotify", true],
    ["https://spotify.link/abc", "spotify", true],
    ["https://www.deezer.com/fr/track/1", "deezer", true],
    ["https://music.apple.com/fr/album/x/1", "applemusic", true],
    ["https://music.youtube.com/watch?v=1", "youtubemusic", true],
    ["https://www.youtube.com/watch?v=1", "youtube", true],
    ["https://youtu.be/1", "youtube", true],
    ["https://soundcloud.com/a/b", "soundcloud", true],
    ["https://artiste.bandcamp.com/album/x", "bandcamp", true],
    ["https://tidal.com/browse/track/1", "tidal", true],
    ["https://music.amazon.fr/albums/1", "amazonmusic", false],
    ["https://open.qobuz.com/album/1", "qobuz", false],
    ["https://www.instagram.com/zazi", "instagram", true],
    ["https://www.tiktok.com/@zazi", "tiktok", true],
    ["https://twitter.com/zazi", "x", true],
    ["https://x.com/zazi", "x", true],
    ["https://bsky.app/profile/zazi", "bluesky", true],
    ["https://www.threads.net/@zazi", "threads", true],
    ["https://fb.me/zazi", "facebook", true],
    ["https://www.reddit.com/r/x", "reddit", true],
    ["https://www.twitch.tv/zazi", "twitch", true],
    ["https://discord.gg/abc", "discord", true],
    ["https://github.com/Zazi2004", "github", true],
    ["https://www.linkedin.com/in/zazi", "linkedin", false],
    ["https://store.steampowered.com/app/1", "steam", true],
    ["https://zazi.itch.io/jeu", "itchio", true],
    ["https://store.epicgames.com/fr/p/x", "epicgames", true],
    ["https://www.playstation.com/fr-fr/", "playstation", true],
    ["https://www.xbox.com/fr-FR", "xbox", false],
    ["https://www.nintendo.co.uk/Games", "nintendo", false],
    ["mailto:moi@zazi.cool", "email", false],
    ["tel:+33612345678", "phone", false],
  ])("%s → %s", (url, id, hasIcon) => {
    const p = platformOf(url);
    expect(p?.id).toBe(id);
    expect(Boolean(p?.icon)).toBe(hasIcon);
  });

  it.each([
    "https://exemple.fr",
    "https://notspotify.com/track",
    "https://spotify.com.evil.example/x",
    "javascript:alert(1)",
    "pas une url",
    "",
  ])("%s → unknown", (url) => {
    expect(platformOf(url)).toBeNull();
  });
});
