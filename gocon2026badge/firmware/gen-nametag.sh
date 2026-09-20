#!/usr/bin/env zsh
set -euo pipefail

name=${1:-}
x_id=${2:-}
github_id=${3:-}

if [[ -z "$name" || -z "$x_id" || -z "$github_id" ]]; then
	print -u2 "usage: $0 NAME X_ID GITHUB_ID"
	print -u2 "example: $0 'さご' '@sago35tk' 'sago35'"
	exit 2
fi

script_dir=${0:A:h}
out_png="$script_dir/nametag.png"
out_raw="$script_dir/images/nametag.rgb565"
font=${NAMETAG_FONT:-/System/Library/Fonts/ヒラギノ角ゴシック W6.ttc}
mono_font=${NAMETAG_MONO_FONT:-/System/Library/Fonts/Menlo.ttc}

magick -size 240x240 canvas:white \
	-fill '#00ADD8' -draw 'rectangle 0,0 239,25' \
	-fill white -font "$mono_font" -pointsize 14 -gravity North -annotate +0+5 'Go Conference 2026' \
	-fill '#20242A' -font "$font" -pointsize 72 -gravity Center -annotate +0-18 "$name" \
	-fill '#5C6670' -font "$mono_font" -pointsize 20 -gravity Center -annotate +0+48 "X:$x_id" \
	-fill '#5C6670' -font "$mono_font" -pointsize 20 -gravity Center -annotate +0+78 "GitHub:$github_id" \
	-fill '#00ADD8' -draw 'rectangle 0,232 239,239' \
	"$out_png"

go run ./gocon2026badge/firmware/cmd/png2rgb565 -in "$out_png" -out "$out_raw"

print "generated $out_png"
print "generated $out_raw"
