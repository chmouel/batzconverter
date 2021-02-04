#!/usr/bin/env bash
set -x
echo -en "\0prompt\x1f⏲ Time around the world \n"

if [[ ${ROFI_RETV} == 2 ]];then
    tz $*|sed -e "s/\x1b\[.\{1,5\}m//g"
    exit 0
fi

tz|sed -e "s/\x1b\[.\{1,5\}m//g"
