// Renders the Codist ad in all three layouts with headless Chrome.
// Usage: node docs/ad/render.mjs   (Chrome required; set CHROME_PATH to override)
import { execFileSync } from "node:child_process";
import { existsSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const page = pathToFileURL(resolve(here, "ad.html")).href;
const chrome =
  process.env.CHROME_PATH ??
  ["/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/usr/bin/google-chrome", "/usr/bin/chromium"].find(existsSync);
if (!chrome) throw new Error("Chrome not found — set CHROME_PATH");

for (const [name, w, h] of [["square", 1080, 1080], ["wide", 1200, 630], ["story", 1080, 1920]]) {
  const out = resolve(here, `codist-ad-${name}.png`);
  execFileSync(chrome, [
    "--headless=new", "--disable-gpu", "--hide-scrollbars",
    "--force-device-scale-factor=2", `--window-size=${w},${h}`,
    "--virtual-time-budget=6000", "--allow-file-access-from-files",
    `--screenshot=${out}`, `${page}#${name}`,
  ], { stdio: "ignore" });
  console.log(`wrote ${out}`);
}
