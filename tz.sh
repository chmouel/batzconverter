#!/usr/bin/env bash
# License: GPL
# Author: Chmouel Boudjnah <chmouel@chmouel.com>
set -eo pipefail
declare -A tzone

function help() {
    cat <<EOF
tz allow to calculate different timezone, it allows you to do somethinge like this :
% tz
% tz 10h30
% tz 10h30 next week
% tz 11:00 next thursday

It will show all different timezone for the timeformat

You can as well add multiple timezones directly on the command line like this :

% tz +America/Chicago +UTC 10h00 tomorrow

By default this script will try to detect your current timezone, if you want
to say something like this, show me the different times tomorrow at 10h00 UTC
you can do :

% tz -t UTC 10h00 tomorrow

and so on,

This needs gnu date, on MacOSX just install gnuutils from brew
This needs bash v4 too, you need to install it from brew as well
on MacOSX

if '-j' is specified tz will generate a json output for 'Alfred' OSX
launcher.

EOF
}


function c() {
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
        *)

    esac
    printf "%b" "${color}$2${NONE} "
}


if [[ $1 == "-h" || $1 == "--help" ]];then
    help
    exit 0
elif [[ $1 == "-j" || $1 == "--json" ]];then
    jsonoutput=true
    shift
fi

## Change this
tzone=(
    ["Bangalore"]="Asia/Calcutta"
    ["Brisbane"]="Australia/Brisbane"
    ["Paris"]="Europe/Paris"
	["Boston"]="America/New_York"
	["California"]="America/Los_Angeles"
)

# If that fails (old distros used to do a hardlink for /etc/localtime)
# you may want to specify your tz directly in currentz like
# currentz="America/Chicago"
currenttz=$(/bin/ls -l /etc/localtime|awk -F/ '{print $(NF-1)"/"$NF}')
date=date
type -p gdate >/dev/null 2>/dev/null && date=gdate

athour=

while [[ $1 == +* ]];do
    tzone[${1#+}]=${1#+}
    shift
done

if [[ $1 == "-t" ]];then
    done=
    specified=true

    for i in ${!tzone[@]};do
        if [[ ${2} == ${i} ]];then
            done=1
            currenttz=${tzone[$i]}
        fi
    done

    if (( !done ));then
        currenttz=$2
        tzone[$2]=$2
    fi

    shift
    shift
fi

args=($@)
if [[ -n ${1} ]];then
    t=$1
    [[ $t != [0-9]*(:|h)[0-9]* ]] && {
        echo "Invalid date format: $t you need to specify a time first like tz 10h00 tomorrow!"
        exit 1
    }
    [[ $t == *h ]] && t=${t%h}
    athour="${t/h/:} ${args[@]:1}"
fi

if [[ ${jsonoutput} ]];then
    cat <<EOF
{"items": [
EOF
fi


for i in ${!tzone[@]};do
    # bug in gnu date? 'now' doesn't take in consideration TZ :(
    [[ -n ${athour} ]] && res=$(TZ="${tzone[$i]}" ${date} --date="TZ=\"$currenttz\" ${athour}") || \
            res=$(TZ=${tzone[$i]} ${date})
    if [[ ${jsonoutput} ]];then
        cat <<EOF
    {
        "uid": "",
        "title": "$i",
        "arg": "$res",
        "subtitle": "$res",
EOF
        if [[ -e "$PWD/$i.png" ]];then
            cat <<EOF
		"icon": {
			"path": "$PWD/$i.png"
		},
EOF
        fi
        echo "},"
    else
        if [[ $currenttz == ${tzone[$i]} ]];then
            if [[ -n $specified ]];then
                specified="✈"
            else
                specified="🏠"
            fi
            printf "%-20s: %s %s\n" `c bold $i` "$res" $specified
        else
            printf "%-20s: %s\n" `c bold $i` "$res"
        fi
    fi
done


[[ -n ${jsonoutput} ]] && echo "]}"
