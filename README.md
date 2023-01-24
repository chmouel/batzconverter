#  batzconverter - Batman Timezone Converter
Show times in different timezone with bash

* Author: Chmouel Boudjnah <chmouel@chmouel.com>
* License: Apache-2.0

## Blog post 

https://blog.chmouel.com/2021/01/31/batzconverter-a-multiple-timezone-converter/

## Demo

![See screenshot](./share/screenshot.png)

## INSTALL

grab the shell script directly from this repo and put it in your path, or just copy and paste this : 

```sh
dest="/usr/local/bin"
[[ -w ${dest} ]] || { dest=${HOME}/bin;mkdir -p ${dest} ;}
curl -L -o ${dest}/batz https://raw.githubusercontent.com/chmouel/batzconverter/master/batz.sh && \
        chmod +x ${dest}/batz && \
	echo "'The' batz has been installed into: ${dest}/batz"
```

## Settings

Create a file in `~/.config/batz/config` and add configure the `TIME_ZONES` and `TIME_ZONES_EMOJI` variable like this :

```shell
## Change the default timezones here!
TIME_ZONES=(
  ["India"]="Asia/Calcutta"
  ["Europe"]="Europe/Paris"
  ["US-East"]="America/New_York"
  ["US-West"]="America/Los_Angeles"
)

TIME_ZONES_EMOJI=(
 ["India"]="🇮🇳 "
 ["Europe"]="🇪🇺 "
 ["US-East"]="🇺🇸 "
 ["US-West"]="🐻"
)

DATE_FORMAT="%b %Hh%M"
DATE_FORMAT_PADDING=10
DEFAULT_TIME_ZOME_EMOJI="🌐"
```


The format is :

    "TZ_Alias_Name"="Timezone"

for example :

    "HomeSweetHome"="Europe/Paris"

You can look an example in my config [here](https://gitlab.com/chmouel/rc-config/-/blob/main/batz/config)

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
options, cause bash getopt is kind of buggy in this regard*

You can use the -j option to output as json and may do some parsing with it. (this is used by the Alfred suppoort).

## [Rofi](https://github.com/davatorium/rofi) Support

Take this rofi script [rofibatz.sh](./share/rofibatz.sh) and place it somewhere, i.e: `~/.config/rofi/rofibatz.sh`, launch it up like this : 

`rofi -modi batz:${HOME}/.config/rofi/rofibatz.sh -show batz`

You can type batz string i.e: `13h00 tomorrow` when you press enter it will evaluate it.

I have a [custom theme](https://github.com/chmouel/mounch/blob/main/rofi/mounch.rasi) so your mileage may vary but for me it will be shown like this : 

![See screenshot](./share/rofibatz.png)

## [Wofi](https://hg.sr.ht/~scoopta/wofi) Support

![Screenshot_2021-11-23-23:24 03](https://user-images.githubusercontent.com/98980/143138764-574a7885-dc02-4c78-ad24-0c44d88f1bc0.png)


Just do something like this assuming you have wofi and wl-copy installed :

```shell
wl-copy $(batz|sed -e "s/\x1b\[.\{1,5\}m//g"|wofi -d|sed 's/.* : //')
```

will only get you the timezone since wofi doesnt support scripts (afaik)

## [Alfred](https://www.alfredapp.com/) Support

BaTZ support [alfred](https://www.alfredapp.com/), it basically output nicely the timezone from alfred in a nice way.

![See screenshot](./alfredworkflow/screenshot.png)

Just install the [alfredworlflow file](./alfredworkflow/TZ.alfredworkflow) from the repository
and make sure the batz script is in one of these path: `$HOME/bin/` or `/usr/local/bin/`
