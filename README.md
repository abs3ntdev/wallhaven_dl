# wallhaven_dl

A CLI tool for downloading wallpapers from [wallhaven.cc](https://wallhaven.cc) and passing the downloaded path to a script (to set the wallpaper, run pywal, whatever). It keeps a local SQLite cache of everything it downloads so you can browse history, jump back and forth between wallpapers, favorite, rate, and clean up.

## Install

```sh
go install git.asdf.cafe/abs3nt/wallhaven_dl@latest
```

Or build from source:

```sh
make build   # outputs dist/wallhaven_dl
sudo make install # installs to /usr/bin with zsh completions
```

## Usage

```
wallhaven_dl search [options] [query]     Search and download a wallpaper
wallhaven_dl previous (prev, p)           Switch back to the previous wallpaper
wallhaven_dl next (n)                     Switch forward to the next wallpaper in history
wallhaven_dl history (hist)               View history and select a wallpaper to apply
wallhaven_dl stats                        Show wallpaper statistics
wallhaven_dl cleanup (clean)              Clean up old or unused wallpapers
wallhaven_dl favorite (fav) add|list|random
wallhaven_dl rate --rating N              Rate the current wallpaper (1-5)
```

### search

Picks a random wallpaper from the search results and downloads it (or reuses the cached copy). If `--scriptPath` is set, the script is run with the wallpaper's path as its first argument.

```
wallhaven_dl search --scriptPath ~/.local/bin/set-wallpaper landscape
```

| Flag | Alias | Default | Description |
|------|-------|---------|-------------|
| `--range` | `-r` | `1y` | Time range for toplist sorting (`1d`, `3d`, `1w`, `1M`, `3M`, `6M`, `1y`) |
| `--purity` | `-p` | `110` | 3 chars for SFW\|Sketchy\|NSFW (e.g. `100` = SFW only) |
| `--categories` | `-c` | `010` | 3 chars for General\|Anime\|People (e.g. `010` = Anime only) |
| `--sort` | `-s` | `toplist` | `relevance`, `random`, `date_added`, `views`, `favorites`, `toplist` |
| `--order` | `-o` | `desc` | `asc` or `desc` |
| `--page` | `--pg`, `--maxPages` | `5` | Maximum number of result pages to randomly sample from. A random page between 1 and this value is chosen, automatically capped at the number of pages the search actually has |
| `--ratios` | `--rt` | `16x9,16x10` | Aspect ratios |
| `--atLeast` | `--al` | `2560x1440` | Minimum resolution |
| `--downloadPath` | `--dp` | `~/Pictures/Wallpapers` | Download directory |
| `--scriptPath` | `--sp` | | Script to run with the downloaded wallpaper path |

### cleanup

```
wallhaven_dl cleanup --mode unused --dryRun
wallhaven_dl cleanup --mode old --olderThan 30d
wallhaven_dl cleanup --mode invalid
```

### favorites and ratings

```
wallhaven_dl fav add        # toggle favorite on the current wallpaper
wallhaven_dl fav list
wallhaven_dl fav random --scriptPath ~/.local/bin/set-wallpaper
wallhaven_dl rate -r 5
```

## Environment variables

- `WH_API_KEY` — wallhaven API key; required for NSFW results
- `DEBUG` — set to any value to enable debug logging

## Purity / categories reference

categories:
|general|anime|people|
|-|-|-|
|1/0|1/0|1/0|

purity:
|sfw|sketchy|nsfw|
|-|-|-|
|1/0|1/0|1/0|
