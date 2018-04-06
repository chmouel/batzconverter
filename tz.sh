#!/usr/bin/env bash
# needs bash v4
# you can do things like this :
# % tz
# % tz 10h30
# % tz 10h30 next week
# % tz 11:00 next thursday
#
# and so on!

set -eo pipefail
declare -A tzone

## Change this
tzone=(
    ["Bengalore"]="Asia/Calcutta"
    ["Brisbane"]="Australia/Brisbane"
    ["Paris"]="Europe/Paris"
)
currenttz=$(/bin/ls -l /etc/localtime|awk -F/ '{print $(NF-1)"/"$NF}')
date=date
type -p gdate >/dev/null 2>/dev/null && date=gdate

athour=
args=($@)

if [[ -n ${1} ]];then
    [[ $1 != [0-9]*(:|h)[0-9]* ]] && { echo "Invalid date format: $1"; exit 1; }
    athour="${1/h/:} ${args[@]:1}"
fi


for i in ${!tzone[@]};do
    echo -n "$i: "
    # bug in gnu date? 'now' doesn't take in consideration TZ :(
    [[ -n ${athour} ]] && TZ="${tzone[$i]}" ${date} --date="TZ=\"$currenttz\" ${athour}" || \
            TZ=${tzone[$i]} ${date}
done
