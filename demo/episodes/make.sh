#!/usr/bin/env bash
# Generate a flat directory (current working dir = season folder) for `rename-media episodes`.
set -euo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT="$DIR/data"
rm -rf "$OUT" && mkdir -p "$OUT"
# Put mixed episode naming forms directly inside the season dir
touch "$OUT/Show.Name.S03E01.mkv"
touch "$OUT/show.name.s03e02.mkv"
touch "$OUT/3x03.mkv"
touch "$OUT/3.04.mkv"
touch "$OUT/Show.Name.S03E07.en-US.srt"

echo "Demo dataset for 'episodes' created at $OUT"
echo "To test: cd '$OUT' && rename-media episodes"
