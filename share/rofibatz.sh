#!/usr/bin/env bash
# Need rofi >=1.6.0
set -x
echo -en "\0prompt\x1f⏲ Time around the world \n"

if [[ ${ROFI_RETV} == 2 ]];then
    tz $*|sed -e "s/\x1b\[.\{1,5\}m//g"
    exit 0
elif [[ ${ROFI_RETV} == 1 ]];then
    echo "$@" | xclip -i -selection clipboard
    exit 
fi

tz|sed -e "s/\x1b\[.\{1,5\}m//g"
