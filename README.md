#  batzconverter - Batman Timezone Converter
Show times in different timezone with bash

* Author: Chmouel Boudjnah <chmouel@chmouel.com>
* License: GPL

## Demo

![See screenshot](./share/screenshot.png)

## Settings

Create a file in `~/.config/batz.sh` and add configure the `TIME_ZONES` and `TIME_ZONES_EMOJI` variable like this :

```
## Change the default timezones here!
TIME_ZONES=(
    ["India"]="Asia/Calcutta"
    ["Europe"]="Europe/Paris"
	["US-East"]="America/New_York"
	["US-West"]="America/Los_Angeles"
)
```


```
TIME_ZONES_EMOJI=(
    ["India"]="🇮🇳 "
    ["Europe"]="🇪🇺 "
	["US-East"]="🇺🇸 "
	["US-West"]="🐻"
)
```


The format is :

    "TZ_Alias_Name"="Timezone"

for example :

    "HomeSweetHome"="Europe/Paris"

## Requirement

Some pretty modern Bash >4.0 and modern GNU Date. On MacosX install those from brew (bash and gnuutils).

## Usage
```bash
% batz
% batz 10h30
% batz 10h30 next week
% batz 11:00 next thursday
```

BaTZ  will show all different timezone for the timeformat

You can as well add multiple timezones directly on the command line like this :
```bash
% batz +America/Chicago +UTC 10h00 tomorrow
```

By default this script will try to detect your current timezone, if you want
to say something like this, show me the different times tomorrow at 10h00 UTC
you can do :

```bash
% batz -t UTC 10
````

When you set another timezone than your current one, it wil show a nice ✈️emoji
near your different base timezone, or by default it will show a 🏠 emojis to
emphasis the current timezone in your copy and paste.

*If you want to add extra timezone with +TZ you need to do at first before the
options, cause bash getopt sucks*

## Alfred Support

BaTZ support alfred, it basically output nicely the timezone from alfred in a nice way.

![See screenshot](./alfredworkflow/screenshot.png)

Just install the [alfredworlflow file](./alfredworkflow/TZ.alfredworkflow) from the repository
and make sure the batz script is in one of these path: `$HOME/bin/` or `/usr/local/bin/`

## Install

This needs gnu date, on MacOSX just install gnuutils from brew

It needs bash v4 too, you need to install it from brew as well on MacOSX
