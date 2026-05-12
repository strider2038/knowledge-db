#!/usr/bin/env bash
set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
html="$script_dir/knowledge-db-overview.html"
print_html="$script_dir/knowledge-db-overview.print.html"
pdf="${1:-$script_dir/knowledge-db-overview.pdf}"

bash "$script_dir/build.sh"

find_chrome() {
  for bin in \
    chromium \
    chromium-browser \
    google-chrome \
    google-chrome-stable \
    "$HOME/.cache/ms-playwright/chromium-1224/chrome-linux64/chrome" \
    "$HOME/.cache/ms-playwright/chromium_headless_shell-1224/chrome-headless-shell-linux64/chrome-headless-shell"
  do
    if command -v "$bin" >/dev/null 2>&1; then
      command -v "$bin"
      return
    fi
    if [ -x "$bin" ]; then
      printf '%s\n' "$bin"
      return
    fi
  done
}

chrome="$(find_chrome || true)"
if [ -z "$chrome" ]; then
  echo "Chromium/Chrome not found. Install chromium or Playwright browsers first." >&2
  exit 1
fi

python3 - "$html" "$script_dir/knowledge-db-course.css" "$print_html" <<'PY'
import re
import sys
from pathlib import Path

html_path = Path(sys.argv[1])
css_path = Path(sys.argv[2])
out_path = Path(sys.argv[3])

html = html_path.read_text(encoding="utf-8")
course_css = css_path.read_text(encoding="utf-8")

slides_match = re.search(
    r'<div class="slides">\s*(.*?)\s*</div>\s*</div>\s*<script',
    html,
    flags=re.S,
)
if not slides_match:
    raise SystemExit("Could not find reveal slides in generated HTML")

slides = slides_match.group(1)

print_css = """
@page {
  size: 13.333in 7.5in;
  margin: 0;
}

html,
body {
  margin: 0;
  padding: 0;
  background: #f7f9fc;
}

.reveal {
  width: 100%;
  min-height: 100%;
  background: #f7f9fc;
}

.slides {
  width: 100%;
}

.slides > section {
  box-sizing: border-box;
  width: 13.333in;
  height: 7.5in;
  page-break-after: always;
  break-after: page;
  padding: 0.55in 0.72in;
  display: flex;
  flex-direction: column;
  justify-content: center;
  overflow: hidden;
}

.slides > section:last-child {
  page-break-after: auto;
  break-after: auto;
}

#title-slide {
  align-items: center;
  text-align: center;
}

.slides > section > h2 {
  margin-top: 0;
}

.diagram-svg {
  max-height: 4.8in;
}
"""

out = f"""<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Knowledge DB</title>
  <style>
{course_css}
{print_css}
  </style>
</head>
<body>
  <div class="reveal">
    <div class="slides">
{slides}
    </div>
  </div>
</body>
</html>
"""

out_path.write_text(out, encoding="utf-8")
PY

"$chrome" \
  --headless \
  --disable-gpu \
  --no-sandbox \
  --no-pdf-header-footer \
  --run-all-compositor-stages-before-draw \
  --virtual-time-budget=5000 \
  --print-to-pdf="$pdf" \
  "file://$print_html"

echo "PDF written to $pdf"
