# wallhaven_dl

this is a tool for downloading images from wallhaven and then passing the download path to a scrip to run(to set the wallpaper, run pywal, whatever)

```
Usage:
  wallhaven_dl set [flags]

Aliases:
  set, s

Flags:
      --at-least string        minimum resolution for results. (default "2560x1440")
  -c, --categories string      categories for the setList search. (default "010")
  -d, --download-path string   script to run after downloading the wallpaper
  -h, --help                   help for set
  -m, --maxPage int            number of pages to randomly choose wallpaper from. (default 5)
  -o, --order string           sort order for results, valid sorts: asc desc. (default "desc")
  -p, --purity string          purity for the setList search. (default "110")
  -r, --range string           range for setList search. (default "1y")
      --ratios strings         ratios to search for. (default [16x9,16x10])
  -t, --script string          script to run after downloading the wallpaper
  -s, --sort string            sort by for results, valid sorts: date_added, relevance, random, views, favorites, setlist. (default "toplist")
```
