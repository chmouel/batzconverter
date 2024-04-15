#!/usr/bin/env bash
# Author: Chmouel Boudjnah <chmouel@chmouel.com>
set -eo pipefail
declare -A TIME_ZONES TIME_ZONES_EMOJI

## Change the default timezones here!
TIME_ZONES=(
	["Bangalore"]="Asia/Calcutta"
	["Brisbane"]="Australia/Brisbane"
	["Paris"]="Europe/Paris"
	["Boston"]="America/New_York"
	["California"]="America/Los_Angeles"
)

# see man date(1) for time format
DATE_FORMAT="%c"

# How many padding to add when showing the date format, only useful when you customize the DATE_FORMAT
DATE_FORMAT_PADDING=0

# Not sure why some emojis need a space and the other doesnt 🤷🏼‍♂️
TIME_ZONES_EMOJI=(
	["Bangalore"]="🇮🇳 "
	["Brisbane"]="🇦🇺 "
	["Paris"]="🇫🇷 "
	["Boston"]="🇺🇸 "
	["California"]="🐻 "
	["UTC"]="🌍"
)

DEFAULT_TIME_ZOME_EMOJI="🌍"
if [[ -t 1 ]]; then
	nocolor=
else
	nocolor=true
fi
noemoji=

[[ -n ${NO_COLOR} ]] && nocolor=true

for f in ~/.config/batz.sh ~/.config/batz/config; do
	[[ -e ${f} ]] && { source ${f}; }
done

function help() {
	cat <<EOF
batz - If batman needed a TZ converted he would probably use this 🦇

batz allow to calculate different timezone, it allows you to do somethinge like this :
% batz
% batz 10h30
% batz 19h00 Monday 17 June
% batz 10h30 next week
% batz 11:00 next thursday

It will show all different timezone for the timeformat.

You can as well add multiple timezones directly on the command line like this :

% batz +America/Chicago +UTC 10h00 tomorrow

By default this script will try to detect your current timezone and base the
time conversion on your own timezone, if you want to say something like this,
show me the different times tomorrow at 10h00 UTC you can do :

% batz -t UTC 10h00 tomorrow

and so on,

This needs gnu date, on MacOSX just install gnuutils from brew
This needs bash v4 too, you need to install it from brew as well
on MacOSX

if '-j' is specified batz will generate a json output for 'Alfred' OSX
launcher.

If you don't want to have colours you can specify '-n' on the command line.
If you want explicit colors you can specify '-C' on the command line which are
disabled automatically on pipes or non stdin/tty.
If you don't want emojis you can specify '-E' on the command line.

configuration is located in ~/.config/batz/config
see variables TIME_ZONES and TIME_ZONES_EMOJI in this file to see how to
configure them.

Conditions to use: Be nice and helpful to other people 🤗
Author: Chmouel Boudjnah <chmouel@chmouel.com>
License: Apache

EOF
}

function c() {
	[[ -n ${nocolor} ]] && {
		printf "%s " "$2"
		return
	}
	BOLD='\033[1m'
	NONE='\033[00m'
	RED='\033[01;31m'

	case $1 in
	bold)
		color=$BOLD
		;;
	normal)
		color=$NONE
		;;
	red)
		color=$BOLD$RED
		;;
	*) ;;

	esac
	printf "%b" "${color}$2${NONE} "
}

# parse arguments
while getopts ":hjnCE" opt; do
	case $opt in
	E)
		noemoji=true
		;;
	C)
		nocolor=
		;;
	n)
		nocolor=true
		;;
	h)
		help
		exit 0
		;;
	j)
		jsonoutput=true
		;;
	\?)
		echo "Invalid option: -$OPTARG" >&2
		exit 1
		;;
	esac
done
shift $((OPTIND - 1))

# If that fails (old distros used to do a hardlink for /etc/localtime)
# you may want to specify your batz directly in currentz like
# currentz="America/Chicago"
currenttz=$(env ls -l /etc/localtime | env awk -F/ '{print $(NF-1)"/"$NF}')
date="date"
type -p gdate >/dev/null 2>/dev/null && date="gdate"
athour=

while [[ $1 == +* ]]; do
	noplus=${1#+}
	[[ -e /usr/share/zoneinfo/${noplus} ]] || {
		echo "${noplus} does not exist in /usr/share/zoneinfo"
		exit 1
	}
	TIME_ZONES[$(basename ${noplus})]=${1#+}
	shift
done

if [[ $1 == "-t" ]]; then
	done=
	specified=true

	for i in ${!TIME_ZONES[@]}; do
		if [[ ${2} == ${i} || ${2} == ${TIME_ZONES[$i]} ]]; then
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

args=($@)
if [[ -n ${1} ]]; then
	t=$1
	[[ $t != [0-9]*(:|h)[0-9]* ]] && {
		echo "Invalid date format: $t you need to specify a time first like tz 10h00 tomorrow!"
		exit 1
	}
	[[ $t == *h ]] && t=${t%h}
	athour="${t/h/:} ${args[@]:1}"
fi

if [[ ${jsonoutput} ]]; then
	cat <<EOF
{"items": [
EOF
fi

for i in ${!TIME_ZONES[@]}; do
	# bug in gnu date? 'now' doesn't take in consideration TZ :(
	[[ -n ${athour} ]] && res=$(TZ="${TIME_ZONES[$i]}" ${date} --date="TZ=\"${currenttz}\" ${athour}" "+${DATE_FORMAT}") ||
		res=$(TZ=${TIME_ZONES[$i]} ${date} "+${DATE_FORMAT}")
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
	else
		if [[ $currenttz == ${TIME_ZONES[$i]} ]]; then
			if [[ -n $nocolor ]]; then
				specified=""
			elif [[ -n $specified ]]; then
				specified="✈"
			else
				specified="🏠"
			fi
			printf "%-20s %-${DATE_FORMAT_PADDING}s %s%s\n" $(c bold ${i}) "$res" "$emoji" $specified
		else
			printf "%-20s %-${DATE_FORMAT_PADDING}s %s\n" $(c bold $i) "$res" "$emoji"
		fi
	fi
done

[[ -n ${jsonoutput} ]] && echo "]}"
