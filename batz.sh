#!/usr/bin/env bash
# Author: Chmouel Boudjnah <chmouel@chmouel.com>
set -eo pipefail
declare -A TIME_ZONES TIME_ZONES_EMOJI TIME_ZONES_ICONS
declare -a SCREENSHOT_NAMES SCREENSHOT_DATES SCREENSHOT_TIMES SCREENSHOT_ZONES
declare -a SCREENSHOT_EMOJIS SCREENSHOT_BADGES

TMP_DIR=$(mktemp -d /tmp/.batz.XXXXXX)
TMP="${TMP_DIR}/table.csv"
clean() { rm -rf "$TMP_DIR"; }
trap clean EXIT

## Change the default timezones here!
TIME_ZONES=(
  ["Bangalore"]="Asia/Calcutta"
  ["Brisbane"]="Australia/Brisbane"
  ["Paris"]="Europe/Paris"
  ["Boston"]="America/New_York"
  ["California"]="America/Los_Angeles"
)

# See man date(1) for time format
DATE_FORMAT="%c"

# Padding to add when showing the date format, useful when customizing DATE_FORMAT
DATE_FORMAT_PADDING=0

# Some emojis need a space while others don't 🤷🏼‍♂️
TIME_ZONES_EMOJI=(
  ["Bangalore"]="🇮🇳 "
  ["Brisbane"]="🇦🇺 "
  ["Paris"]="🇫🇷 "
  ["Boston"]="🇺🇸 "
  ["California"]="🐻 "
  ["UTC"]="🌍 "
)

# Icon paths for JSON output (used by Alfred workflow)
# Specify full paths to .png icon files for each timezone
TIME_ZONES_ICONS=(
  ["Bangalore"]="/path/to/icons/Bangalore.png"
  ["Brisbane"]="/path/to/icons/Brisbane.png"
  ["Paris"]="/path/to/icons/Paris.png"
  ["Boston"]="/path/to/icons/Boston.png"
  ["California"]="/path/to/icons/California.png"
)

# Whether to use gum tool to print
USE_GUM=

DEFAULT_TIME_ZONE_EMOJI="🌍 "
if [[ -t 1 ]]; then
  nocolor=
else
  nocolor=true
fi
noemoji=
fzf_selection=
base_time=

[[ -n ${NO_COLOR} ]] && nocolor=true

for f in ${BATZ_CONFIG_FILE:-} ~/.config/batz.sh ~/.config/batz/config; do
  [[ -e ${f} ]] && {
    source "${f}"
    break
  }
done

function help() {
  local BLUE=
  local NONE=
  local ITALIC=
  local YELLOW=
  local YELLOWITALIC=
  local GREEN=
  if [[ -t 1 ]]; then
    BLUE='\033[01;34m'
    NONE='\033[00m'
    ITALIC='\033[3m'
    YELLOW='\033[01;33m'
    GREEN='\033[01;32m'
    GREY='\033[01;39m'
    YELLOWITALIC="${YELLOW}${ITALIC}"
    GREYITALIC="${GREY}${ITALIC}"
  fi
  echo -e "batz - The Ultimate TZ Converter 🦇🕒🌍

This script helps you calculate and display time across different timezones
in a flash! 

Here are some examples to ${GREEN}get you started${NONE} 🚀:

% batz                       ${GREYITALIC}# Display current local time    ${NONE}
% batz 10h30                 ${GREYITALIC}# Specify an exact time         ${NONE}
% batz 19h00 Monday 17 June  ${GREYITALIC}# Use calendar dates for planning ${NONE}
% batz 10h30 next week       ${GREYITALIC}# Look ahead to next week       ${NONE}
% batz 11:00 next Thursday   ${GREYITALIC}# Future time conversion        ${NONE}

This will shows the time in various timezones with delightful details and emojis.

You can also specify multiple timezones directly on the command line like this:

${YELLOWITALIC}% batz +America/Chicago +UTC 10h00 tomorrow${NONE}

${GREEN}More advanced examples:${NONE}
% batz 3pm next Friday       ${GREYITALIC}# Convert next Friday afternoon time                    ${NONE}
% batz 9:30 yesterday        ${GREYITALIC}# Check what time it was yesterday                      ${NONE}
% batz -t Asia/Tokyo 17:00   ${GREYITALIC}# Convert 5pm Tokyo time to your defaults               ${NONE}
% batz 22:00 -g              ${GREYITALIC}# Show tonight's time in a neat table format (need gum) ${NONE}
% batz -s 22:00              ${GREYITALIC}# Copy a polished world clock to the clipboard         ${NONE}
% batz -f +Europe/London     ${GREYITALIC}# Select custom timezones to display ${NONE}

${GREEN}Pro tips:${NONE}
• Combine flags for powerful results: ${ITALIC}batz -fg 15:00${NONE} (custom zones + table format)
• Natural language dates work too:    ${ITALIC}batz "next Tuesday at noon"${NONE} 📝
• Chain commands:                     ${ITALIC}batz | grep UTC${NONE} to filter specific timezones 🔍

By default, the script detects your current timezone and converts accordingly.
To specify a different timezone, use the '${BLUE}-t${NONE}' flag like so:

${YELLOWITALIC}% batz -t UTC 10h00 tomorrow${NONE}

Requirements:
- GNU date and Bash v4. On macOS, install them via Homebrew 🍀.

Additional flags:
${BLUE}-j${NONE}    Generate JSON output for the Alfred macOS launcher
${BLUE}-t${NONE}    Use another timezone as the base for conversion
${BLUE}-n${NONE}    Disable colors
${BLUE}-C${NONE}    Enable colors
${BLUE}-E${NONE}    Disable emojis 
${BLUE}-f${NONE}    Select one or more timezones interactively using fzf 
${BLUE}-g${NONE}    Use gum to format the output in a table 
${BLUE}-s${NONE}    Copy the output to the clipboard as a PNG screenshot
${BLUE}-h${NONE}    Show this help message

Interactive selection:
${YELLOWITALIC}% batz -f             → Replace default timezones with your selections${NONE}

Configuration is located in ~/.config/batz/config.
Customize it by modifying TIME_ZONES and TIME_ZONES_EMOJI 🛠️.

Be kind and helpful to others 🤗

Author: Chmouel Boudjnah <chmouel@chmouel.com>
License: Apache"
}

check_tools() {
  local tools=(awk zdump)
  [[ -n ${fzf_selection} ]] && tools+=(fzf)
  if [[ -n ${USE_GUM} ]]; then
    tools+=(gum)
  fi
  for tool in "${tools[@]}"; do
    if ! command -v "$tool" &>/dev/null; then
      echo "$tool is not installed. Please install it." >&2
      exit 1
    fi
  done
}

xml_escape() {
  local value=$1
  value=${value//&/\&amp;}
  value=${value//</\&lt;}
  value=${value//>/\&gt;}
  printf "%s" "$value"
}

screenshot_date() {
  local timezone=$1
  local format=$2

  if [[ -n ${athour} ]]; then
    TZ="$timezone" ${date} --date="TZ=\"${currenttz}\" ${athour}" "+${format}"
  else
    TZ="$timezone" ${date} "+${format}"
  fi
}

screenshot_fail() {
  echo "$1" >&2
  exit 1
}

render_screenshot() {
  local svg="${TMP_DIR}/batz.svg"
  local png="${TMP_DIR}/batz.png"
  local row_count=${#SCREENSHOT_NAMES[@]}
  local canvas_width=1200
  local canvas_height=$((210 + row_count * 84))
  local panel_x=32
  local panel_y=28
  local panel_width=1136
  local panel_height=$((canvas_height - 56))
  local row_x=58
  local row_width=1084
  local row_height=72
  local row_start=142
  local city_x=142
  local flag_x=92
  local header_title_x=120
  local header_label
  local city_font_size
  local i y

  if [[ -n ${noemoji} ]]; then
    header_title_x=72
    city_x=82
  fi
  header_label=$(TZ="$currenttz" ${date} "+%a, %d %B · %H:%M %Z")

  {
    printf '%s\n' '<?xml version="1.0" encoding="UTF-8"?>'
    printf '<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">\n' \
      "$canvas_width" "$canvas_height" "$canvas_width" "$canvas_height"
    printf '%s\n' '  <defs>'
    printf '%s\n' '    <linearGradient id="background" x1="0" y1="0" x2="1" y2="1">'
    printf '%s\n' '      <stop offset="0" stop-color="#090b18"/>'
    printf '%s\n' '      <stop offset="0.55" stop-color="#111127"/>'
    printf '%s\n' '      <stop offset="1" stop-color="#07131d"/>'
    printf '%s\n' '    </linearGradient>'
    printf '%s\n' '    <radialGradient id="glow">'
    printf '%s\n' '      <stop offset="0" stop-color="#8b5cf6" stop-opacity="0.24"/>'
    printf '%s\n' '      <stop offset="1" stop-color="#8b5cf6" stop-opacity="0"/>'
    printf '%s\n' '    </radialGradient>'
    printf '%s\n' '    <linearGradient id="local-row" x1="0" y1="0" x2="1" y2="0">'
    printf '%s\n' '      <stop offset="0" stop-color="#8b5cf6" stop-opacity="0.22"/>'
    printf '%s\n' '      <stop offset="1" stop-color="#22d3ee" stop-opacity="0.08"/>'
    printf '%s\n' '    </linearGradient>'
    printf '%s\n' '    <linearGradient id="accent" x1="0" y1="0" x2="1" y2="0">'
    printf '%s\n' '      <stop offset="0" stop-color="#a78bfa"/>'
    printf '%s\n' '      <stop offset="1" stop-color="#22d3ee"/>'
    printf '%s\n' '    </linearGradient>'
    printf '%s\n' '    <filter id="shadow" x="-20%" y="-20%" width="140%" height="160%">'
    printf '%s\n' '      <feDropShadow dx="0" dy="14" stdDeviation="20" flood-color="#000000" flood-opacity="0.38"/>'
    printf '%s\n' '    </filter>'
    printf '%s\n' '  </defs>'
    printf '%s\n' '  <rect width="100%" height="100%" fill="none"/>'
    printf '  <rect width="%d" height="%d" fill="url(#background)"/>\n' "$canvas_width" "$canvas_height"
    printf '%s\n' '  <circle cx="175" cy="35" r="250" fill="url(#glow)"/>'
    printf '%s\n' '  <circle cx="1110" cy="610" r="260" fill="url(#glow)" opacity="0.45"/>'
    printf '  <rect x="%d" y="%d" width="%d" height="%d" rx="28" fill="#141426" fill-opacity="0.94" stroke="#ffffff" stroke-opacity="0.10" filter="url(#shadow)"/>\n' \
      "$panel_x" "$panel_y" "$panel_width" "$panel_height"
    printf '  <rect x="%d" y="%d" width="%d" height="4" rx="2" fill="url(#accent)"/>\n' \
      "$((panel_x + 28))" "$panel_y" "$((panel_width - 56))"

    if [[ -z ${noemoji} ]]; then
      printf '%s\n' '  <text x="72" y="91" font-family="Apple Color Emoji, Noto Color Emoji, Segoe UI Emoji, sans-serif" font-size="32">🦇</text>'
    fi
    printf '  <text x="%d" y="82" fill="#f5f3ff" font-family="SF Pro Display, Helvetica Neue, Arial, sans-serif" font-size="29" font-weight="650">World Clock</text>\n' "$header_title_x"
    printf '  <text x="%d" y="105" fill="#8f89aa" font-family="SF Pro Text, Helvetica Neue, Arial, sans-serif" font-size="13" letter-spacing="2.4">TIME AROUND THE WORLD</text>\n' "$header_title_x"
    printf '%s\n' '  <rect x="735" y="57" width="375" height="49" rx="24" fill="#ffffff" fill-opacity="0.055" stroke="#a78bfa" stroke-opacity="0.32"/>'
    printf '  <circle cx="764" cy="81" r="5" fill="#22d3ee"/>\n'
    printf '  <text x="786" y="88" fill="#d9d5e8" font-family="SFMono-Regular, Menlo, Monaco, Consolas, monospace" font-size="16">%s</text>\n' "$(xml_escape "$header_label")"

    for ((i = 0; i < row_count; i++)); do
      y=$((row_start + i * 84))
      city_font_size=26
      ((${#SCREENSHOT_NAMES[$i]} > 18)) && city_font_size=21
      ((${#SCREENSHOT_NAMES[$i]} > 25)) && city_font_size=18

      if [[ -n ${SCREENSHOT_BADGES[$i]} ]]; then
        printf '  <rect x="%d" y="%d" width="%d" height="%d" rx="18" fill="url(#local-row)" stroke="#a78bfa" stroke-opacity="0.45"/>\n' \
          "$row_x" "$y" "$row_width" "$row_height"
        printf '  <rect x="%d" y="%d" width="4" height="40" rx="2" fill="url(#accent)"/>\n' \
          "$row_x" "$((y + 16))"
      else
        printf '  <rect x="%d" y="%d" width="%d" height="%d" rx="18" fill="#ffffff" fill-opacity="0.035" stroke="#ffffff" stroke-opacity="0.055"/>\n' \
          "$row_x" "$y" "$row_width" "$row_height"
      fi

      if [[ -n ${SCREENSHOT_EMOJIS[$i]} ]]; then
        printf '  <circle cx="%d" cy="%d" r="24" fill="#ffffff" fill-opacity="0.075"/>\n' "$flag_x" "$((y + 36))"
        printf '  <text x="%d" y="%d" font-family="Apple Color Emoji, Noto Color Emoji, Segoe UI Emoji, sans-serif" font-size="25" text-anchor="middle">%s</text>\n' \
          "$flag_x" "$((y + 45))" "$(xml_escape "${SCREENSHOT_EMOJIS[$i]}")"
      fi

      printf '  <text x="%d" y="%d" fill="#f5f3ff" font-family="SF Pro Display, Helvetica Neue, Arial, sans-serif" font-size="%d" font-weight="600">%s</text>\n' \
        "$city_x" "$((y + 31))" "$city_font_size" "$(xml_escape "${SCREENSHOT_NAMES[$i]}")"
      printf '  <text x="%d" y="%d" fill="#958faa" font-family="SF Pro Text, Helvetica Neue, Arial, sans-serif" font-size="15">%s</text>\n' \
        "$city_x" "$((y + 55))" "$(xml_escape "${SCREENSHOT_DATES[$i]}")"

      if [[ -n ${SCREENSHOT_BADGES[$i]} ]]; then
        printf '%s\n' "  <rect x=\"620\" y=\"$((y + 21))\" width=\"82\" height=\"28\" rx=\"14\" fill=\"#a78bfa\" fill-opacity=\"0.16\" stroke=\"#a78bfa\" stroke-opacity=\"0.38\"/>"
        printf '  <text x="661" y="%d" fill="#c4b5fd" font-family="SF Pro Text, Helvetica Neue, Arial, sans-serif" font-size="11" font-weight="700" letter-spacing="1.2" text-anchor="middle">%s</text>\n' \
          "$((y + 40))" "${SCREENSHOT_BADGES[$i]}"
      fi

      printf '  <text x="935" y="%d" fill="#ffffff" font-family="SFMono-Regular, Menlo, Monaco, Consolas, monospace" font-size="36" font-weight="500" text-anchor="end">%s</text>\n' \
        "$((y + 48))" "$(xml_escape "${SCREENSHOT_TIMES[$i]}")"
      printf '%s\n' "  <rect x=\"960\" y=\"$((y + 17))\" width=\"100\" height=\"38\" rx=\"19\" fill=\"#22d3ee\" fill-opacity=\"0.10\" stroke=\"#22d3ee\" stroke-opacity=\"0.28\"/>"
      printf '  <text x="1010" y="%d" fill="#67e8f9" font-family="SFMono-Regular, Menlo, Monaco, Consolas, monospace" font-size="15" font-weight="600" text-anchor="middle">%s</text>\n' \
        "$((y + 42))" "$(xml_escape "${SCREENSHOT_ZONES[$i]}")"
    done
    printf '%s\n' '</svg>'
  } >"$svg"

  if [[ $(uname -s) == Darwin ]] && command -v swift &>/dev/null; then
    swift - "$svg" "$png" <<'SWIFT' || screenshot_fail "Failed to render the screenshot with the native macOS renderer."
import AppKit

func fail(_ message: String) -> Never {
  FileHandle.standardError.write(Data(("batz: " + message + "\n").utf8))
  exit(1)
}

let source = CommandLine.arguments[1]
let destination = CommandLine.arguments[2]
guard let image = NSImage(contentsOfFile: source) else { fail("could not load the generated SVG") }
let width = Int(image.size.width)
let height = Int(image.size.height)
guard let bitmap = NSBitmapImageRep(
  bitmapDataPlanes: nil,
  pixelsWide: width,
  pixelsHigh: height,
  bitsPerSample: 8,
  samplesPerPixel: 4,
  hasAlpha: true,
  isPlanar: false,
  colorSpaceName: .deviceRGB,
  bytesPerRow: 0,
  bitsPerPixel: 0
) else { fail("could not allocate the PNG bitmap") }
NSGraphicsContext.saveGraphicsState()
NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: bitmap)
NSColor(red: 9 / 255, green: 11 / 255, blue: 24 / 255, alpha: 1).setFill()
NSBezierPath(rect: NSRect(x: 0, y: 0, width: width, height: height)).fill()
image.draw(in: NSRect(x: 0, y: 0, width: width, height: height))
NSGraphicsContext.restoreGraphicsState()
guard let data = bitmap.representation(using: .png, properties: [:]) else { fail("could not encode the PNG") }
try data.write(to: URL(fileURLWithPath: destination))
SWIFT
  elif command -v rsvg-convert &>/dev/null; then
    rsvg-convert --background-color "#090b18" "$svg" --output "$png" ||
      screenshot_fail "rsvg-convert failed to render the screenshot."
  elif command -v magick &>/dev/null; then
    magick -background "#090b18" "$svg" -alpha remove "$png" ||
      screenshot_fail "ImageMagick failed to render the screenshot."
  elif command -v convert &>/dev/null; then
    convert -background "#090b18" "$svg" -alpha remove "$png" ||
      screenshot_fail "ImageMagick failed to render the screenshot."
  else
    echo "Screenshot output needs rsvg-convert or ImageMagick." >&2
    exit 1
  fi

  case $(uname -s) in
  Darwin)
    if ! command -v osascript &>/dev/null; then
      echo "osascript is required to copy PNG images on macOS." >&2
      exit 1
    fi
    osascript - "$png" <<'APPLESCRIPT' >/dev/null || screenshot_fail "Failed to copy the PNG to the macOS clipboard."
on run argv
  set imageFile to POSIX file (item 1 of argv)
  set the clipboard to (read imageFile as «class PNGf»)
end run
APPLESCRIPT
    ;;
  Linux)
    if [[ -z ${WAYLAND_DISPLAY} ]]; then
      echo "Screenshot clipboard output on Linux is only supported on Wayland." >&2
      exit 1
    fi
    if ! command -v wl-copy &>/dev/null; then
      echo "wl-copy is required to copy PNG images on Wayland." >&2
      exit 1
    fi
    wl-copy --type image/png <"$png" ||
      screenshot_fail "wl-copy failed to copy the PNG to the Wayland clipboard."
    ;;
  *)
    echo "Screenshot clipboard output is supported on macOS and Wayland." >&2
    exit 1
    ;;
  esac

  echo "Screenshot copied to the clipboard." >&2
}

function c() {
  [[ -n ${nocolor} ]] && {
    printf "%s " "$2"
    return
  }
  local BOLD='\033[1m'
  local NONE='\033[00m'
  local RED='\033[01;31m'

  case $1 in
  bold) color=$BOLD ;;
  normal) color=$NONE ;;
  red) color=$BOLD$RED ;;
  *) ;;
  esac
  printf "%b" "${color}$2${NONE} "
}

# Parse arguments
while getopts ":ht:jnCEfgs" opt; do
  case $opt in
  g) USE_GUM=yes ;;
  s)
    screenshotoutput=true
    USE_GUM=
    ;;
  f) fzf_selection=true ;;
  E) noemoji=true ;;
  C) nocolor= ;;
  n) nocolor=true ;;
  t) base_time=${OPTARG} ;;
  h)
    help
    exit 0
    ;;
  j) jsonoutput=true ;;
  \?)
    echo "Invalid option: -$OPTARG" >&2
    exit 1
    ;;
  esac
done
shift $((OPTIND - 1))

check_tools

if [[ -n ${screenshotoutput} && -n ${jsonoutput} ]]; then
  echo "The -s and -j output modes cannot be combined." >&2
  exit 1
fi

# If that fails (old distros used to do a hardlink for /etc/localtime)
# you may want to specify your timezone directly in currenttz like
# currenttz="America/Chicago"
currenttz=$(env ls -l /etc/localtime | env awk -F/ '{print $(NF-1)"/"$NF}')
date="date"
type -p gdate >/dev/null 2>&1 && date="gdate"
athour=
zoneinfo=/usr/share/zoneinfo

if [[ -L ${zoneinfo} ]]; then
  zoneinfo=$(readlink -f ${zoneinfo})
fi

while [[ $1 == +* ]]; do
  noplus=${1#+}
  [[ -e ${zoneinfo}/${noplus} ]] || {
    echo "${noplus} does not exist in ${zoneinfo}"
    exit 1
  }
  TIME_ZONES[$(basename "${noplus}")]=${noplus}
  shift
done

if [[ -n ${fzf_selection} ]]; then
  # I don't know how to do this on non-standard distros like NixOS
  [[ -e ${zoneinfo} ]] || {
    echo "${zoneinfo} does not exist"
    exit 1
  }
  type -p fzf >/dev/null 2>&1 || {
    echo "fzf is not installed, please install it"
    exit 1
  }

  IFS=$'\n'
  mapfile -t selected < <(find ${zoneinfo} -type f | sed -n "/^[A-Z]*/ { s,${zoneinfo},,;s,^/,,;p}" |
    fzf --prompt="Select timezone: " -s \
      --preview="zdump {}|sed 's,${zoneinfo},,';echo;echo 'Next Transition:';zdump -V {} |head -1|sed 's,${zoneinfo},,'" -m)
  [[ -z ${selected[*]} ]] && {
    echo "No timezone selected"
    exit 1
  }
  for i in "${selected[@]}"; do
    TIME_ZONES[$(basename "$i")]=$i
  done
fi

if [[ -n ${base_time} ]]; then
  done=
  specified=true

  for i in "${!TIME_ZONES[@]}"; do
    if [[ ${base_time} == "${i}" ]]; then
      done=1
      currenttz=${TIME_ZONES[$i]}
    fi
  done

  if ((!done)); then
    currenttz=${base_time}
    TIME_ZONES[${base_time}]=${base_time}
  fi

fi

[[ -e ${zoneinfo}/${currenttz} ]] || {
  echo "${currenttz} does not exist in ${zoneinfo}"
  exit 1
}

args=("$@")
if [[ -n ${1} ]]; then
  t=$1
  [[ $t != [0-9]*(:|h)[0-9]* ]] && {
    echo "Invalid date format: $t. You need to specify a time first like batz 10h00 tomorrow!"
    exit 1
  }
  [[ $t == *h ]] && t=${t%h}
  athour="${t/h/:} ${args[*]:1}"
fi

if [[ ${jsonoutput} ]]; then
  json_sep=""
  cat <<EOF
{"items": [
EOF
elif [[ -n ${USE_GUM} ]]; then
  echo "Timezone,Date" >"$TMP"
fi

for i in "${!TIME_ZONES[@]}"; do
  # Bug in GNU date? 'now' doesn't take into consideration TZ :(
  [[ -n ${athour} ]] && res=$(TZ="${TIME_ZONES[$i]}" ${date} --date="TZ=\"${currenttz}\" ${athour}" "+${DATE_FORMAT}") ||
    res=$(TZ=${TIME_ZONES[$i]} ${date} "+${DATE_FORMAT}")
  emoji="${DEFAULT_TIME_ZONE_EMOJI}"
  [[ -n "${TIME_ZONES_EMOJI[$i]}" ]] && emoji="${TIME_ZONES_EMOJI[$i]} "
  [[ -n ${noemoji} ]] && emoji=""

  if [[ ${jsonoutput} ]]; then
    subtitle_comma=""
    icon_path=""
    if [[ -n "${TIME_ZONES_ICONS[$i]}" ]]; then
      subtitle_comma=","
      icon_path="${TIME_ZONES_ICONS[$i]}"
    fi
    cat <<EOF
$json_sep{
        "uid": "",
        "title": "$i",
        "arg": "$res",
        "subtitle": "$res"$subtitle_comma
EOF
    if [[ -n "$icon_path" ]]; then
      cat <<EOF
        "icon": {
            "path": "$icon_path"
        }
EOF
    fi
    echo "}"
    json_sep=","
  elif [[ -n ${screenshotoutput} ]]; then
    screenshot_emoji=$emoji
    screenshot_emoji=${screenshot_emoji%"${screenshot_emoji##*[![:space:]]}"}
    screenshot_badge=
    if [[ $currenttz == "${TIME_ZONES[$i]}" ]]; then
      if [[ -n ${specified} ]]; then
        screenshot_badge=BASE
      else
        screenshot_badge=LOCAL
      fi
    fi
    SCREENSHOT_NAMES+=("$i")
    SCREENSHOT_DATES+=("$(screenshot_date "${TIME_ZONES[$i]}" "%a, %d %B")")
    SCREENSHOT_TIMES+=("$(screenshot_date "${TIME_ZONES[$i]}" "%H:%M")")
    SCREENSHOT_ZONES+=("$(screenshot_date "${TIME_ZONES[$i]}" "%Z")")
    SCREENSHOT_EMOJIS+=("$screenshot_emoji")
    SCREENSHOT_BADGES+=("$screenshot_badge")
  elif [[ -n ${USE_GUM} ]]; then
    echo "$emoji $i,$res" >>"$TMP"
  else
    if [[ $currenttz == "${TIME_ZONES[$i]}" ]]; then
      if [[ -n $nocolor ]]; then
        specified=""
      elif [[ -n $specified ]]; then
        specified="✈"
      else
        specified="🏠"
      fi
      printf "%-20s %-${DATE_FORMAT_PADDING}s %s%s\n" "$(c bold "${i}")" "$res" "$emoji" "$specified"
    else
      printf "%-20s %-${DATE_FORMAT_PADDING}s %s\n" "$(c bold "$i")" "$res" "$emoji"
    fi
  fi
done

[[ -n ${jsonoutput} ]] && {
  echo "]}"
  exit 0
}
[[ -n ${screenshotoutput} ]] && {
  render_screenshot
  exit 0
}
[[ -n ${USE_GUM} ]] && gum table -p <"$TMP"
