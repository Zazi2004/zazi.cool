// Development script: renders web/banners/*.svg to the 1600 × 533 JPEGs embedded in
// the binary (internal/web/static/assets/banners/<id>.jpg). Run it with
// `make export-banners` after changing an SVG, then commit the JPEGs.
//
// It needs no npm package: it drives a local Chromium-based browser (Chrome,
// Chromium or Brave) over the DevTools protocol, with the WebSocket client of
// Node. Set CHROME_BIN to use a specific browser.
import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { mkdir, mkdtemp, readdir, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { basename, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const root = resolve(fileURLToPath(import.meta.url), "../..");
const sourceDir = join(root, "banners");
const outDir = resolve(root, "../internal/web/static/assets/banners");

const WIDTH = 1600;
const HEIGHT = 533;
const MAX_BYTES = 250 * 1024; // F16b
const QUALITIES = [88, 82, 76, 70, 60];

const candidates = [
  process.env.CHROME_BIN,
  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
  "/Applications/Chromium.app/Contents/MacOS/Chromium",
  "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
  "/usr/bin/chromium",
  "/usr/bin/chromium-browser",
  "/usr/bin/google-chrome",
];
const browserPath = candidates.find((p) => p && existsSync(p));
if (!browserPath) {
  console.error("No Chromium-based browser found: set CHROME_BIN.");
  process.exit(1);
}

const profile = await mkdtemp(join(tmpdir(), "export-banners-"));
const browser = spawn(
  browserPath,
  [
    "--headless=new",
    "--disable-gpu",
    "--hide-scrollbars",
    "--no-first-run",
    "--remote-debugging-port=0",
    `--user-data-dir=${profile}`,
    "about:blank",
  ],
  { stdio: "ignore" },
);

try {
  const ws = new WebSocket(await debuggerURL());
  await new Promise((ok, fail) => {
    ws.onopen = ok;
    ws.onerror = () => fail(new Error("DevTools connection failed"));
  });
  const cdp = session(ws);

  const { targetId } = await cdp.send("Target.createTarget", { url: "about:blank" });
  const { sessionId } = await cdp.send("Target.attachToTarget", { targetId, flatten: true });
  await cdp.send("Page.enable", {}, sessionId);
  await cdp.send(
    "Emulation.setDeviceMetricsOverride",
    { width: WIDTH, height: HEIGHT, deviceScaleFactor: 1, mobile: false },
    sessionId,
  );

  await mkdir(outDir, { recursive: true });
  const sources = (await readdir(sourceDir)).filter((f) => f.endsWith(".svg")).sort();
  for (const file of sources) {
    const loaded = cdp.once("Page.loadEventFired", sessionId);
    await cdp.send("Page.navigate", { url: pathToFileURL(join(sourceDir, file)).href }, sessionId);
    await loaded;

    let jpeg;
    for (const quality of QUALITIES) {
      const { data } = await cdp.send(
        "Page.captureScreenshot",
        { format: "jpeg", quality, optimizeForSpeed: false },
        sessionId,
      );
      jpeg = Buffer.from(data, "base64");
      if (jpeg.length <= MAX_BYTES) break;
    }
    if (jpeg.length > MAX_BYTES)
      throw new Error(`${file}: ${jpeg.length} bytes, over ${MAX_BYTES}`);
    const out = join(outDir, basename(file, ".svg") + ".jpg");
    await writeFile(out, jpeg);
    console.log(`${basename(out)}  ${(jpeg.length / 1024).toFixed(0)} Kio`);
  }
  ws.close();
} finally {
  const exited = new Promise((done) => browser.once("exit", done));
  browser.kill();
  await exited;
  // The browser may still be flushing files: retry a few times.
  await rm(profile, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 });
}

// debuggerURL waits for the browser to write its DevTools endpoint in the profile.
async function debuggerURL() {
  const portFile = join(profile, "DevToolsActivePort");
  for (let i = 0; i < 100; i++) {
    if (existsSync(portFile)) {
      const [port, path] = (await readFile(portFile, "utf8")).trim().split("\n");
      if (path) return `ws://127.0.0.1:${port}${path}`;
    }
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error("the browser did not expose a DevTools endpoint");
}

// session wraps the DevTools WebSocket: request/response by id, and one-shot events.
function session(ws) {
  let nextId = 0;
  const pending = new Map();
  const waiting = [];
  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.id !== undefined) {
      const { ok, fail } = pending.get(msg.id);
      pending.delete(msg.id);
      if (msg.error) fail(new Error(msg.error.message));
      else ok(msg.result);
      return;
    }
    const i = waiting.findIndex((w) => w.method === msg.method && w.sessionId === msg.sessionId);
    if (i >= 0) waiting.splice(i, 1)[0].ok();
  };
  return {
    send: (method, params = {}, sessionId) =>
      new Promise((ok, fail) => {
        const id = ++nextId;
        pending.set(id, { ok, fail });
        ws.send(JSON.stringify({ id, method, params, sessionId }));
      }),
    once: (method, sessionId) => new Promise((ok) => waiting.push({ method, sessionId, ok })),
  };
}
