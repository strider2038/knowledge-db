#!/usr/bin/env bash
set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
html="$script_dir/knowledge-db-overview.html"

pandoc \
  -t revealjs \
  -s "$script_dir/knowledge-db-overview.md" \
  --slide-level=2 \
  -o "$html"

python3 - "$html" <<'PY'
import re
import sys
from pathlib import Path

path = Path(sys.argv[1])
html = path.read_text(encoding="utf-8")

# Pandoc's default reveal.js template includes Search and Zoom plugins.
# In Chromium they may be blocked by ORB when loaded from jsDelivr, and the
# presentation fails before Reveal.initialize(). They are not needed here.
html = re.sub(
    r'\n  <script src="[^"]+/plugin/(?:search|zoom)/[^"]+"></script>',
    "",
    html,
)
html = re.sub(r"\n\s*RevealSearch,\n\s*RevealZoom", "", html)

path.write_text(html, encoding="utf-8")
PY
