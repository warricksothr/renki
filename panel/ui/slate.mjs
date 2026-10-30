import { chromium } from "@playwright/test";
import { mkdir, readFile } from "node:fs/promises";
const out = new URL("../../.checks/streaming/", import.meta.url);
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true });
try {
  const page = await browser.newPage({
    viewport: { width: 1280, height: 720 },
    reducedMotion: "reduce",
  });
  console.log(
    "Chromium video codecs:",
    JSON.stringify(
      await page.evaluate(() =>
        RTCRtpReceiver.getCapabilities("video").codecs.map((c) => ({
          mimeType: c.mimeType,
          fmtp: c.sdpFmtpLine,
        })),
      ),
    ),
  );
  await page.setContent(
    await readFile(
      new URL("../../internal/panel/static/slate.html", import.meta.url),
      "utf8",
    ),
  );
  await page.getByText("Please stand by", { exact: true }).waitFor();
  await page.screenshot({
    path: new URL("standby.png", out).pathname,
    animations: "disabled",
  });
} finally {
  await browser.close();
}
