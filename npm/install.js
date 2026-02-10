#!/usr/bin/env node

/**
 * @fileoverview Postinstall script for the tdd-ai npm package.
 *
 * Runs automatically after `npm install`. Downloads the pre-built tdd-ai
 * Go binary from GitHub Releases for the user's platform and architecture.
 *
 * Uses zero npm dependencies — only Node.js built-ins (https, zlib, fs, path).
 *
 * Supported platforms: darwin-x64, darwin-arm64, linux-x64, linux-arm64, win32-x64, win32-arm64
 */

"use strict";

const { execFileSync } = require("child_process");
const fs = require("fs");
const https = require("https");
const path = require("path");
const zlib = require("zlib");

/** @const {string} GitHub owner/repo for release downloads */
const REPO = "mauricioTechDev/tdd-ai";

/** @const {string} Name of the Go binary to download */
const BINARY_NAME = "tdd-ai";

/** @const {Object<string, string>} Maps Node.js process.platform to GoReleaser OS names */
const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

/** @const {Object<string, string>} Maps Node.js process.arch to GoReleaser architecture names */
const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

/**
 * Detects the current platform and architecture, mapped to GoReleaser naming.
 * @returns {{ platform: string, arch: string }} GoReleaser-compatible OS and arch names
 * @throws {Error} If the current platform/arch combination is not supported
 */
function getPlatformInfo() {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];

  if (!platform || !arch) {
    throw new Error(
      `Unsupported platform: ${process.platform}-${process.arch}. ` +
        `Supported: darwin-x64, darwin-arm64, linux-x64, linux-arm64, win32-x64, win32-arm64`
    );
  }

  return { platform, arch };
}

/**
 * Builds the GitHub Releases download URL for a specific version and platform.
 * @param {string} version - Semver version without "v" prefix (e.g. "0.4.0")
 * @param {string} platform - GoReleaser OS name (darwin, linux, windows)
 * @param {string} arch - GoReleaser arch name (amd64, arm64)
 * @returns {string} Full download URL for the release archive
 */
function getDownloadUrl(version, platform, arch) {
  const ext = platform === "windows" ? "zip" : "tar.gz";
  return `https://github.com/${REPO}/releases/download/v${version}/${BINARY_NAME}_${version}_${platform}_${arch}.${ext}`;
}

/**
 * Downloads a file from the given URL, following up to 5 redirects.
 * GitHub Releases URLs redirect to a CDN (objects.githubusercontent.com),
 * so redirect following is required.
 * @param {string} url - The URL to download
 * @returns {Promise<Buffer>} The downloaded file contents as a Buffer
 * @throws {Error} If the download fails or too many redirects occur
 */
function downloadFile(url) {
  return new Promise((resolve, reject) => {
    const follow = (url, redirects) => {
      if (redirects > 5) return reject(new Error("Too many redirects"));

      https
        .get(url, (res) => {
          if (
            res.statusCode >= 300 &&
            res.statusCode < 400 &&
            res.headers.location
          ) {
            return follow(res.headers.location, redirects + 1);
          }
          if (res.statusCode !== 200) {
            return reject(
              new Error(`Download failed: HTTP ${res.statusCode} from ${url}`)
            );
          }

          const chunks = [];
          res.on("data", (chunk) => chunks.push(chunk));
          res.on("end", () => resolve(Buffer.concat(chunks)));
          res.on("error", reject);
        })
        .on("error", reject);
    };

    follow(url, 0);
  });
}

/**
 * Extracts a binary from a gzipped tar archive (.tar.gz).
 * Uses a minimal tar parser that reads 512-byte headers to locate the binary,
 * avoiding the need for an external `tar` dependency.
 * @param {Buffer} buffer - The gzipped tar archive contents
 * @param {string} destDir - Directory to write the extracted binary to
 * @param {string} binaryName - Name of the binary file to find in the archive
 * @returns {string} Absolute path to the extracted binary
 * @throws {Error} If the binary is not found in the archive
 */
function extractTarGz(buffer, destDir, binaryName) {
  const tar = zlib.gunzipSync(buffer);

  let offset = 0;
  while (offset < tar.length) {
    const header = tar.slice(offset, offset + 512);
    if (header.every((b) => b === 0)) break;

    const name = header.toString("utf8", 0, 100).replace(/\0/g, "");
    const sizeStr = header
      .toString("utf8", 124, 136)
      .replace(/\0/g, "")
      .trim();
    const size = parseInt(sizeStr, 8) || 0;

    offset += 512;

    if (name === binaryName || name.endsWith("/" + binaryName)) {
      const data = tar.slice(offset, offset + size);
      const destPath = path.join(destDir, binaryName);
      fs.writeFileSync(destPath, data);
      fs.chmodSync(destPath, 0o755);
      return destPath;
    }

    offset += Math.ceil(size / 512) * 512;
  }

  throw new Error(`Binary '${binaryName}' not found in archive`);
}

/**
 * Extracts a binary from a zip archive (used for Windows releases).
 * Uses PowerShell on Windows or the `unzip` command on other platforms.
 * @param {Buffer} buffer - The zip archive contents
 * @param {string} destDir - Directory to extract into
 * @param {string} binaryName - Name of the binary (without .exe extension)
 * @returns {string} Absolute path to the extracted .exe binary
 * @throws {Error} If the binary is not found after extraction
 */
function extractZip(buffer, destDir, binaryName) {
  const tmpZip = path.join(destDir, "tmp.zip");
  fs.writeFileSync(tmpZip, buffer);

  try {
    if (process.platform === "win32") {
      execFileSync("powershell", [
        "-NoProfile",
        "-Command",
        `Expand-Archive -Path '${tmpZip}' -DestinationPath '${destDir}' -Force`,
      ]);
    } else {
      execFileSync("unzip", ["-o", tmpZip, "-d", destDir]);
    }
  } finally {
    fs.unlinkSync(tmpZip);
  }

  const destPath = path.join(destDir, binaryName + ".exe");
  if (!fs.existsSync(destPath)) {
    throw new Error(`Binary '${binaryName}.exe' not found after extraction`);
  }
  return destPath;
}

/**
 * Entry point. Reads the version from package.json, detects the platform,
 * downloads the matching binary from GitHub Releases, and extracts it
 * to ./bin/. Skips download if the binary already exists.
 */
async function main() {
  const pkg = require("./package.json");
  const version = pkg.version;
  const { platform, arch } = getPlatformInfo();
  const binDir = path.join(__dirname, "bin");

  const ext = platform === "windows" ? ".exe" : "";
  const binaryPath = path.join(binDir, BINARY_NAME + ext);
  if (fs.existsSync(binaryPath)) {
    console.log(`tdd-ai binary already exists at ${binaryPath}`);
    return;
  }

  const url = getDownloadUrl(version, platform, arch);
  console.log(`Downloading tdd-ai v${version} for ${platform}-${arch}...`);

  fs.mkdirSync(binDir, { recursive: true });

  const buffer = await downloadFile(url);

  if (platform === "windows") {
    extractZip(buffer, binDir, BINARY_NAME);
  } else {
    extractTarGz(buffer, binDir, BINARY_NAME);
  }

  console.log(`tdd-ai v${version} installed successfully`);
}

main().catch((err) => {
  console.error(`Failed to install tdd-ai: ${err.message}`);
  process.exit(1);
});
