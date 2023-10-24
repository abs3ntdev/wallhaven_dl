package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/spf13/cobra"

	"git.asdf.cafe/abs3nt/wallhaven_dl/src/wallhaven"
)

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.PersistentFlags().StringVarP(
		&searchRange,
		"range",
		"r",
		"1y",
		"range for search.",
	)
	searchCmd.PersistentFlags().StringVarP(
		&searchPurity,
		"purity",
		"p",
		"110",
		"purity for the search.",
	)
	searchCmd.PersistentFlags().StringVarP(
		&searchCategories,
		"categories",
		"c",
		"010",
		"categories for the search.",
	)
	searchCmd.PersistentFlags().StringVarP(
		&searchSorting,
		"sort",
		"s",
		"toplist",
		"sort by for results, valid sorts: date_added, relevance, random, views, favorites, searchlist.",
	)
	searchCmd.PersistentFlags().StringVarP(
		&searchOrder,
		"order",
		"o",
		"desc",
		"sort order for results, valid sorts: asc desc.",
	)
	searchCmd.PersistentFlags().IntVarP(
		&searchPage,
		"maxPage",
		"m",
		5,
		"number of pages to randomly choose wallpaper from.",
	)
	searchCmd.PersistentFlags().StringSliceVar(
		&searchRatios,
		"ratios",
		[]string{"16x9", "16x10"},
		"ratios to search for.",
	)
	searchCmd.PersistentFlags().StringVar(
		&searchAtLeast,
		"at-least",
		"2560x1440",
		"minimum resolution for results.",
	)
	searchCmd.PersistentFlags().StringVarP(
		&searchScript,
		"script",
		"t",
		"",
		"script to run after downloading the wallpaper",
	)
	searchCmd.PersistentFlags().StringVarP(
		&downloadPath,
		"download-path",
		"d",
		"",
		"directory to download the image too",
	)
}

var (
	searchRange      string
	searchPurity     string
	searchCategories string
	searchSorting    string
	searchOrder      string
	searchAtLeast    string
	searchScript     string
	downloadPath     string
	searchRatios     []string
	searchPage       int
	searchCmd        = &cobra.Command{
		Use:     "search",
		Aliases: []string{"s"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "Wallhaven downloader with the option to run a script after the image has been downloaded",
		RunE: func(cmd *cobra.Command, args []string) error {
			return search(args)
		},
	}
)

func search(args []string) error {
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	s := &wallhaven.Search{
		Categories: searchCategories,
		Purities:   searchPurity,
		Sorting:    searchSorting,
		Order:      searchOrder,
		TopRange:   searchRange,
		AtLeast:    searchAtLeast,
		Ratios:     searchRatios,
		Page:       r.Intn(searchPage) + 1,
	}
	if len(args) > 0 {
		s.Query = wallhaven.Q{
			Tags: []string{args[0]},
		}
	}
	results, err := wallhaven.SearchWallpapers(s)
	if err != nil {
		return err
	}
	resultPath, err := getOrDownload(results, r)
	if err != nil {
		return err
	}
	if searchScript != "" {
		err = runScript(resultPath, searchScript)
		if err != nil {
			return err
		}
	}
	return nil
}

func getOrDownload(results *wallhaven.SearchResults, r *rand.Rand) (string, error) {
	if len(results.Data) == 0 {
		return "", fmt.Errorf("no wallpapers found")
	}
	homedir, _ := os.UserHomeDir()
	if downloadPath == "" {
		downloadPath = path.Join(homedir, "Pictures/Wallpapers")
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
