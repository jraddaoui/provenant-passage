// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0
import fs from "node:fs";
import path from "node:path";

const lock = JSON.parse(fs.readFileSync("package-lock.json", "utf8"));
let notices = "Provenant Passage UI dependency notices\n\n";
for (const name of Object.keys(lock.packages).sort()) {
  if (!name || !fs.existsSync(name)) continue;
  const pkg = JSON.parse(
    fs.readFileSync(path.join(name, "package.json"), "utf8"),
  );
  notices += `${pkg.name} ${pkg.version} (${pkg.license ?? "see package notices"})\n`;
  for (const file of fs.readdirSync(name).sort()) {
    if (
      /^(license|licence|copying|notice|copyright)([.-]|$)/i.test(file) &&
      fs.statSync(path.join(name, file)).isFile()
    ) {
      notices += fs.readFileSync(path.join(name, file), "utf8") + "\n";
    }
  }
  notices += "\n";
}
fs.mkdirSync("dist", { recursive: true });
fs.writeFileSync("dist/third-party-notices.txt", notices);
fs.cpSync("licenses", "dist/licenses", { recursive: true });
