#!/usr/bin/env bash
# Author: Chmouel Boudjnah <chmouel@chmouel.com>
set -eo pipefail
declare -A TIME_ZONES TIME_ZONES_EMOJI

TMP=$(mktemp /tmp/.batz.XXXXXX)
clean() { rm -f "$TMP"; }
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

[[ -n ${NO_COLOR} ]] && nocolor=true

for f in ~/.config/batz.sh ~/.config/batz/config; do
  [[ -e ${f} ]] && source "${f}"
done

function help() {
  local BLUE=
  local NONE=
  local ITALIC=
  local YELLOW=
  local YELLOWITALIC=
  if [[ -t 1 ]]; then
    BLUE='\033[01;34m'
    NONE='\033[00m'
    ITALIC='\033[3m'
    YELLOW='\033[01;33m'
    YELLOWITALIC="${YELLOW}${ITALIC}"
  fi
  echo -e "batz - The Ultimate TZ Converter 🦇🕒🌍

This script helps you calculate and display time across different timezones
in a flash! Here are some examples to get you started 💡:

${YELLOWITALIC}% batz                      → Display current local time 🕰️
% batz 10h30                → Specify an exact time ⏰
% batz 19h00 Monday 17 June → Use calendar dates for planning 📅
% batz 10h30 next week       → Look ahead to next week 🔜
% batz 11:00 next Thursday   → Future time conversion 🗓️${NONE}

It shows the time in various timezones with delightful details and emojis 🚀.

You can also specify multiple timezones directly on the command line like this:

${YELLOWITALIC}% batz +America/Chicago +UTC 10h00 tomorrow${NONE}

By default, the script detects your current timezone and converts accordingly.
To specify a different timezone, use the '-t' flag like so:

${YELLOWITALIC}% batz -t UTC 10h00 tomorrow${NONE}

Requirements:
- GNU date and Bash v4. On macOS, install them via Homebrew 🍀.

Additional flags:
${BLUE}-j${NONE}    Generate JSON output for the Alfred macOS launcher
${BLUE}-n${NONE}    Disable colors
${BLUE}-C${NONE}    Enable colors
${BLUE}-E${NONE}    Disable emojis 😞
${BLUE}-h${NONE}    Show this help message

Configuration is located in ~/.config/batz/config.
Customize it by modifying TIME_ZONES and TIME_ZONES_EMOJI 🛠️.

Be kind and helpful to others 🤗

Author: Chmouel Boudjnah <chmouel@chmouel.com>
License: Apache"
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
while getopts ":hjnCEfg" opt; do
  case $opt in
  g) USE_GUM=yes ;;
  f) fzf_selection=true ;;
  E) noemoji=true ;;
  C) nocolor= ;;
  n) nocolor=true ;;
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

# If that fails (old distros used to do a hardlink for /etc/localtime)
# you may want to specify your timezone directly in currenttz like
# currenttz="America/Chicago"
currenttz=$(env ls -l /etc/localtime | env awk -F/ '{print $(NF-1)"/"$NF}')
date="date"
type -p gdate >/dev/null 2>&1 && date="gdate"
athour=

while [[ $1 == +* ]]; do
  noplus=${1#+}
  [[ -e /usr/share/zoneinfo/${noplus} ]] || {
    echo "${noplus} does not exist in /usr/share/zoneinfo"
    exit 1
  }
  TIME_ZONES[$(basename "${noplus}")]=${noplus}
  shift
done

if [[ -n ${fzf_selection} ]]; then
  # I don't know how to do this on non-standard distros like NixOS
  [[ -e /usr/share/zoneinfo/ ]] || {
    echo "/usr/share/zoneinfo/ does not exist"
    exit 1
  }
  type -p fzf >/dev/null 2>&1 || {
    echo "fzf is not installed, please install it"
    exit 1
  }

  IFS=$'\n'
  mapfile -t selected < <(find /usr/share/zoneinfo/ -type f | sed -n '/^[A-Z]*/ { s,/usr/share/zoneinfo/,,;p;}' | fzf --prompt="Select timezone: " --preview="echo {}" --preview-window=up:1:wrap -m)
  [[ -z ${selected[*]} ]] && {
    echo "No timezone selected"
    exit 1
  }
  for i in "${selected[@]}"; do
    TIME_ZONES[$(basename "$i")]=$i
  done
fi

if [[ $1 == "-t" ]]; then
  done=
  specified=true

  for i in "${!TIME_ZONES[@]}"; do
    if [[ ${2} == "${i}" || ${2} == "${TIME_ZONES[$i]}" ]]; then
      done=1
      currenttz=${TIME_ZONES[$i]}
    fi
  done

  if ((!done)); then
    currenttz=$2
    TIME_ZONES[$2]=$2
  fi

  shift
  shift
fi

[[ -e /usr/share/zoneinfo/${currenttz} ]] || {
  echo "${currenttz} does not exist in /usr/share/zoneinfo"
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
    cat <<EOF
    {
        "uid": "",
        "title": "$i",
        "arg": "$res",
        "subtitle": "$res",
EOF
    if [[ -e "$PWD/$i.png" ]]; then
      cat <<EOF
        "icon": {
            "path": "$PWD/$i.png"
        },
EOF
    fi
    echo "},"
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

[[ -n ${jsonoutput} ]] && echo "]}"
[[ -n ${USE_GUM} ]] && gum table -p <"$TMP"
