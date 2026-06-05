#!/usr/bin/env node
import { existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const root = join(here, "..");
const platformName = process.platform === "win32" ? "masterdatamate.exe" : "masterdatamate";
const candidates = [
  process.env.MASTERDATAMATE_BIN,
  join(root, "dist-native", platformName),
  join(root, "prebuilds", `${process.platform}-${process.arch}`, platformName)
].filter(Boolean);

const binary = candidates.find((candidate) => existsSync(candidate));
if (!binary) {
  console.error("MasterDataMate native binary was not found.");
  console.error("Run `npm run build:go` for local development, or install a package that includes prebuilt binaries.");
  process.exit(1);
}

const child = spawn(binary, process.argv.slice(2), { stdio: "inherit" });
child.on("exit", (code, signal) => {
  if (signal) process.kill(process.pid, signal);
  process.exit(code ?? 1);
});
