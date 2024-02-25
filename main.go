package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/urfave/cli/v3"

	"git.asdf.cafe/abs3nt/wallhaven_dl/src/wallhaven"
)

func main() {
	app := cli.Command{
		EnableShellCompletion: true,
		Name:                  "wallhaven_dl",
		Usage:                 "Download wallpapers from wallhaven.cc",
		Commands: []*cli.Command{
			{
				Name:  "search",
				Usage: "Search for wallpapers",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "range",
						Aliases: []string{"r"},
						Value:   "1y",
						Validator: func(s string) error {
							if s != "1d" && s != "3d" && s != "1w" && s != "1M" && s != "3M" && s != "6M" && s != "1y" {
								return fmt.Errorf("range must be '1d', '3d', '1w', '1M', '3M', '6M' or '1y'")
							}
							return nil
						},
						Usage: "Time range for top sorting",
					},
					&cli.StringFlag{
						Name:    "purity",
						Aliases: []string{"p"},
						Value:   "110",
						Validator: func(s string) error {
							if len(s) != 3 {
								return fmt.Errorf("purity must be 3 characters long")
							}
							for _, c := range s {
								if c != '0' && c != '1' {
									return fmt.Errorf("purity must be 0 or 1")
								}
							}
							return nil
						},
						Usage: "Purity of the wallpapers",
					},
					&cli.StringFlag{
						Name:    "categories",
						Aliases: []string{"c"},
						Value:   "010",
						Validator: func(s string) error {
							if len(s) != 3 {
								return fmt.Errorf("categories must be 3 characters long")
							}
							for _, c := range s {
								if c != '0' && c != '1' {
									return fmt.Errorf("categories must be 0 or 1")
								}
							}
							return nil
						},
						Usage: "Categories of the wallpapers",
					},
					&cli.StringFlag{
						Name:    "sort",
						Aliases: []string{"s"},
						Value:   "toplist",
						Validator: func(s string) error {
							if s != "relevance" && s != "random" && s != "date_added" && s != "views" && s != "favorites" &&
								s != "toplist" {
								return fmt.Errorf(
									"sort must be 'relevance', 'random', 'date_added', 'views', 'favorites' or 'toplist'",
								)
							}
							return nil
						},
						Usage: "Sorting of the wallpapers",
					},
					&cli.StringFlag{
						Name:    "order",
						Aliases: []string{"o"},
						Value:   "desc",
						Validator: func(s string) error {
							if s != "asc" && s != "desc" {
								return fmt.Errorf("order must be 'asc' or 'desc'")
							}
							return nil
						},
						Usage: "Order of the wallpapers",
					},
					&cli.IntFlag{
						Name:    "page",
						Aliases: []string{"pg"},
						Value:   5,
						Usage:   "Pages to search",
					},
					&cli.StringSliceFlag{
						Name:    "ratios",
						Aliases: []string{"rt"},
						Value:   []string{"16x9", "16x10"},
						Usage:   "Ratios of the wallpapers",
					},
					&cli.StringFlag{
						Name:    "atLeast",
						Aliases: []string{"al"},
						Value:   "2560x1440",
						Usage:   "Minimum resolution",
					},
					&cli.StringFlag{
						Name:      "scriptPath",
						Aliases:   []string{"sp"},
						Value:     "",
						TakesFile: true,
						Usage:     "Path to the script to run after downloading",
					},
					&cli.StringFlag{
						Name:      "downloadPath",
						Aliases:   []string{"dp"},
						Value:     os.Getenv("HOME") + "/Pictures/Wallpapers",
						TakesFile: true,
						Usage:     "Path to download the wallpapers",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return search(c)
				},
			},
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func search(c *cli.Command) error {
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	s := &wallhaven.Search{
		Categories: c.String("categories"),
		Purities:   c.String("purity"),
		Sorting:    c.String("sort"),
		Order:      c.String("order"),
		TopRange:   c.String("range"),
		AtLeast:    c.String("atLeast"),
		Ratios:     c.StringSlice("ratios"),
		Page:       c.Int("page"),
	}
	query := c.Args().First()
	if query != "" {
		s.Query = wallhaven.Q{
			Tags: []string{query},
		}
	}
	results, err := wallhaven.SearchWallpapers(s)
	if err != nil {
		return err
	}
	resultPath, err := getOrDownload(results, r, c.String("downloadPath"))
	if err != nil {
		return err
	}
	searchScript := c.String("scriptPath")
	if searchScript != "" {
		err = runScript(resultPath, searchScript)
		if err != nil {
			return err
		}
	}
	return nil
}

func getOrDownload(results *wallhaven.SearchResults, r *rand.Rand, downloadPath string) (string, error) {
	if len(results.Data) == 0 {
		return "", fmt.Errorf("no wallpapers found")
	}
	result := results.Data[r.Intn(len(results.Data))]
	fullPath := path.Join(downloadPath, path.Base(result.Path))
	if _, err := os.Stat(fullPath); err != nil {
		err = result.Download(path.Join(downloadPath))
		if err != nil {
			return "", err
		}
	}
	return fullPath, nil
}

func runScript(imgPath, script string) error {
	_, err := exec.Command(script, imgPath).Output()
	if err != nil {
		return err
	}
	return nil
}
