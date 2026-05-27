#!/usr/bin/env bash
set -euo pipefail

PROJECT_NAME="${CLOUDFLARE_PAGES_PROJECT:-glyph}"
EXPORT_ZIP="${MINT_EXPORT_ZIP:-/tmp/glyph-docs-export.zip}"
EXPORT_DIR="${MINT_EXPORT_DIR:-/tmp/glyph-docs-export}"

echo "==> Validating Mintlify docs"
npx mintlify@latest validate
npx mintlify@latest broken-links --check-anchors --check-redirects

echo "==> Exporting Mintlify docs to ${EXPORT_ZIP}"
rm -rf "${EXPORT_DIR}" "${EXPORT_ZIP}"
npx mintlify@latest export --output "${EXPORT_ZIP}"

echo "==> Unpacking export to ${EXPORT_DIR}"
mkdir -p "${EXPORT_DIR}"
unzip -q "${EXPORT_ZIP}" -d "${EXPORT_DIR}"

echo "==> Checking export boundary"
if find "${EXPORT_DIR}" \( \
  -path '*/.glyph/*' -o \
  -name 'store.db' -o \
  -name '*.go' -o \
  -name 'go.mod' -o \
  -name 'go.sum' \
\) -print -quit | grep -q .; then
  echo "Refusing to deploy: export contains Glyph store or Go source files." >&2
  exit 1
fi

echo "==> Deploying ${EXPORT_DIR} to Cloudflare Pages project ${PROJECT_NAME}"
npx wrangler pages deploy "${EXPORT_DIR}" --project-name "${PROJECT_NAME}"
