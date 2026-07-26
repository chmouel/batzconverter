# batzconverter - Batman Timezone Converter

`batz` shows one time across a configured set of timezones. It accepts GNU
`date` expressions such as `10h00 tomorrow`, keeps timezone output aligned in
terminals and pipes, and can emit JSON for launchers.

![See screenshot](./share/screenshot.png)

Read the original [blog post](https://blog.chmouel.com/2021/01/31/batzconverter-a-multiple-timezone-converter/)
for the idea behind the project.

## Install

Install the latest version with Go:

```bash
go install github.com/chmouel/batzconverter/cmd/batz@latest
batz 10h00 tomorrow
```

Tagged releases provide `batz_<version>_<os>-<arch>.tar.gz` archives for Linux
and macOS. Each archive contains the `batz` executable.

To build from a checkout:

```bash
make
./bin/batz 10h00 tomorrow
```

Run `make check` for formatting, vet, and tests. Contributors using
[`pre-commit`](https://pre-commit.com/) can enable the same checks with:

```bash
pre-commit install
```

## Requirements

`batz` delegates natural-language parsing and `strftime` formatting to GNU
`date`. Linux commonly provides it through coreutils. On macOS:

```bash
brew install coreutils
```

The command automatically prefers `gdate` when it is available.

Optional features require:

- `fzf` and `zdump` for interactive timezone selection with `-f`
- native macOS rendering tools for screenshots on macOS
- `rsvg-convert` or ImageMagick plus `wl-copy` for screenshots on Wayland

## Configuration

`batz` reads `~/.config/batz/config.toml`. Set `BATZ_CONFIG_FILE` to use a
different file. Start with [`config.example.toml`](./config.example.toml).

```toml
date_format = "%b %Hh%M"
default_timezone_emoji = "🌐 "
screenshot_theme = "light"

[[timezone]]
name = "India"
zone = "Asia/Calcutta"
emoji = "🇮🇳 "

[[timezone]]
name = "Europe"
zone = "Europe/Paris"
emoji = "🇪🇺 "
icon = "/path/to/icons/Europe.png"
```

Timezone entries retain their declared order. When `emoji` is omitted, `batz`
derives a country flag from the system IANA timezone database. An explicit
value overrides the inferred flag, so custom symbols are supported. UTC and
`Etc/*` zones use 🌍.

Configuration is strict: unknown keys, duplicate names, invalid themes, and
invalid IANA timezone identifiers are reported as errors.

## Usage

```text
batz
batz 10h30
batz 10h30 next week
batz 11:00 next Thursday
batz +America/Chicago +UTC 10h00 tomorrow
batz -t UTC 10h00 tomorrow
```

The first date argument must be a time written with `h` or `:`, followed by an
expression accepted by GNU `date`. With no expression, `batz` shows the current
time.

The local IANA timezone is the default base. Use `-t` with a configured name or
IANA identifier to interpret the expression in another timezone:

```bash
batz -t UTC 10h00 tomorrow
```

Use `+Area/Location` to add timezones for one invocation. The options can appear
before the date expression:

```bash
batz +America/Chicago +UTC -t Europe/Paris 10h00 tomorrow
```

Run `batz -h` for the complete flag reference.

## Output

Terminal output uses a rounded, Unicode-aware table with aligned flags and
custom emoji. Redirected and piped output is borderless, aligned, and
ANSI-free.

Use `-n` to disable colors, `-C` to force terminal colors, and `-E` to remove
emoji and local/base markers. `batz` also respects
[`NO_COLOR`](https://no-color.org/).

Use `-j` for Alfred-compatible JSON:

```bash
batz -j 10h00 tomorrow
```

### Interactive timezone selection

With `fzf` and `zdump` installed, `-f` opens the system timezone catalog.
Select multiple entries with `TAB`:

```bash
batz -f 10h00 tomorrow
```

The preview shows the current value and the next timezone transition within six
years.

### Screenshots

Use `-s` to render a world-clock card and copy it to the clipboard:

```bash
batz -s 10h30 tomorrow
batz -s -S dark 10h30 tomorrow
```

The theme precedence is `-S`, `BATZ_SCREENSHOT_THEME`, TOML
`screenshot_theme`, then `light`.

Linux screenshot support requires Wayland; X11 is not supported. Linux cards
use country codes instead of emoji flags because emoji rendering varies across
systems.

<img width="1200" height="630" alt="Example screenshot" src="https://github.com/user-attachments/assets/4e47c9ed-31b4-43ae-9ab0-f3457d899b49" />

## Launcher integrations

### Rofi

Install [`share/rofibatz.sh`](./share/rofibatz.sh), then launch it with:

```bash
rofi -modi batz:${HOME}/.config/rofi/rofibatz.sh -show batz
```

![Rofi screenshot](./share/rofibatz.png)

### Wofi

With `wofi` and `wl-copy` installed:

```bash
batz | wofi -d | wl-copy
```

### Alfred

Install [`TZ.alfredworkflow`](./alfredworkflow/TZ.alfredworkflow) and ensure
`batz` is available in `$HOME/bin`, `/usr/local/bin`, or another directory in
the workflow's `PATH`.

![Alfred screenshot](./alfredworkflow/screenshot.png)

Set a TOML `icon` on any timezone that should display a custom icon in Alfred:

```toml
[[timezone]]
name = "Paris"
zone = "Europe/Paris"
icon = "/path/to/icons/Paris.png"
```

### Raffi

Raffi can consume the JSON output:

```yaml
addons:
  script_filters:
    - name: "Timezone"
      command: "batz"
      args: ["-j"]
      keyword: "tz"
```

### Zenity

```bash
output=$(zenity --entry --text "🦇Time" --entry-text "$(date '+%H:%M')")
[[ -n $output ]] && batz "$output" | wl-copy
```

## License

[Apache-2.0](./LICENSE)

## Author

Chmouel Boudjnah [@chmouel](https://twitter.com/chmouel)
