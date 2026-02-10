#!/usr/bin/env node

/**
 * @fileoverview CLI wrapper for the tdd-ai Go binary.
 *
 * This is the entry point registered in package.json's "bin" field.
 * npm symlinks this script so that `tdd-ai` is available as a command.
 *
 * It spawns the Go binary with inherited stdio so that terminal detection
 * (used by tdd-ai for auto JSON/text format switching) works correctly.
 * Exit codes from the Go binary are propagated to the caller.
 */

"use strict";

const { execFileSync } = require("child_process");
const path = require("path");
const fs = require("fs");

const BINARY_NAME = "tdd-ai";

/**
 * Resolves the path to the downloaded Go binary.
 * @returns {string} Absolute path to the tdd-ai binary
 * @throws {Error} If the binary does not exist (postinstall may not have run)
 */
function getBinaryPath() {
  const ext = process.platform === "win32" ? ".exe" : "";
  const binPath = path.join(__dirname, "bin", BINARY_NAME + ext);

  if (!fs.existsSync(binPath)) {
    throw new Error(
      `tdd-ai binary not found at ${binPath}. ` +
        `Try reinstalling: npm install -g tdd-ai`
    );
  }

  return binPath;
}

try {
  execFileSync(getBinaryPath(), process.argv.slice(2), {
    stdio: "inherit",
  });
} catch (err) {
  if (err.status !== undefined) {
    process.exit(err.status);
  }
  throw err;
}
