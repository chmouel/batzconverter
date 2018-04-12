#!/usr/bin/env bash
# you can do things like this :
# % tz
# % tz 10h30
# % tz 10h30 next week
# % tz 11:00 next thursday
#
# You can as well add multiple timezones directly on the command line like this :
#
# tz +America/Chicago +UTC 10h00 tomorrow
#
#
# and so on,
#
# This needs gnu date, on MacOSX just install gnuutils from brew
#
# This needs bash v4 too, you need to install it from brew as well
# on MacOSX
#

set -eo pipefail
declare -A tzone

## Change this
tzone=(
    ["Bengalore"]="Asia/Calcutta"
    ["Brisbane"]="Australia/Brisbane"
    ["Paris"]="Europe/Paris"
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

args=($@)
if [[ -n ${1} ]];then
    [[ $1 != [0-9]*(:|h)[0-9]* ]] && {
        echo "Invalid date format: $1 you need to specify a time first like tz 10h00 tomorrow!"
        exit 1
    }
    athour="${1/h/:} ${args[@]:1}"
fi


for i in ${!tzone[@]};do
    echo -n "$i: "
    # bug in gnu date? 'now' doesn't take in consideration TZ :(
    [[ -n ${athour} ]] && TZ="${tzone[$i]}" ${date} --date="TZ=\"$currenttz\" ${athour}" || \
            TZ=${tzone[$i]} ${date}
done
