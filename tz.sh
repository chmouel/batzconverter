#!/usr/bin/env bash
# needs bash v4
# you can do things like this :
# % tz
# Paris: Thu Apr  5 22:53:24 CEST 2018
# Brisbane: Thu Apr  5 22:53:24 CEST 2018
# Bangalore: Thu Apr  5 22:53:24 CEST 2018
# % tz 10h30
# Paris: Thu Apr  5 10:30:00 CEST 2018
# Brisbane: Fri Apr  6 02:30:00 CEST 2018
# Bangalore: Fri Apr  6 07:00:00 CEST 2018
# % tz 10h30 next week
# Paris: Thu Apr 12 10:30:00 CEST 2018
# Brisbane: Fri Apr 13 02:30:00 CEST 2018
# Bangalore: Fri Apr 13 07:00:00 CEST 2018
# % tz 11:00 next thursday
# Paris: Thu Apr 12 11:00:00 CEST 2018
# Brisbane: Thu Apr 12 03:00:00 CEST 2018
# Bangalore: Thu Apr 12 07:30:00 CEST 2018

set -e
set -f

date=date
type -p gdate >/dev/null 2>/dev/null && date=gdate

declare -A tzone

tzone=(
    ["Bangalore"]="Asia/Calcutta"
    ["Brisbane"]="Australia/Brisbane"
    ["Paris"]="Europe/Paris"
)
athour=now
args=($@)

if [[ -n ${1} ]];then
    [[ $1 != [0-9]*(:|h)[0-9]* ]] && { echo "Invalid date format: $1"; exit 1; }
    athour="${1/h/:} ${args[@]:1}"
fi


for i in ${!tzone[@]};do
    echo -n "$i: "
    ${date} --date="TZ=\"${tzone[$i]}\" ${athour}"
done
