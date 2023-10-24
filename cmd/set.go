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
	rootCmd.AddCommand(setCmd)
	setCmd.PersistentFlags().StringVarP(
		&setRange,
		"range",
		"r",
		"1y",
		"range for setList search.",
	)
	setCmd.PersistentFlags().StringVarP(
		&setPurity,
		"purity",
		"p",
		"110",
		"purity for the setList search.",
	)
	setCmd.PersistentFlags().StringVarP(
		&setCategories,
		"categories",
		"c",
		"010",
		"categories for the setList search.",
	)
	setCmd.PersistentFlags().StringVarP(
		&setSorting,
		"sort",
		"s",
		"toplist",
		"sort by for results, valid sorts: date_added, relevance, random, views, favorites, setlist.",
	)
	setCmd.PersistentFlags().StringVarP(
		&setOrder,
		"order",
		"o",
		"desc",
		"sort order for results, valid sorts: asc desc.",
	)
	setCmd.PersistentFlags().IntVarP(
		&setPage,
		"maxPage",
		"m",
		5,
		"number of pages to randomly choose wallpaper from.",
	)
	setCmd.PersistentFlags().StringSliceVar(
		&setRatios,
		"ratios",
		[]string{"16x9", "16x10"},
		"ratios to search for.",
	)
	setCmd.PersistentFlags().StringVar(
		&setAtLeast,
		"at-least",
		"2560x1440",
		"minimum resolution for results.",
	)
	setCmd.PersistentFlags().StringVarP(
		&setScript,
		"script",
		"t",
		"",
		"script to run after downloading the wallpaper",
	)
	setCmd.PersistentFlags().StringVarP(
		&setPath,
		"download-path",
		"d",
		"",
		"script to run after downloading the wallpaper",
	)
}

var (
	setRange      string
	setPurity     string
	setCategories string
	setSorting    string
	setOrder      string
	setAtLeast    string
	setScript     string
	setPath       string
	setRatios     []string
	setPage       int
	setCmd        = &cobra.Command{
		Use:     "set",
		Aliases: []string{"s"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "Wallhaven downloader with the option to run a script after the image has been downloaded",
		RunE: func(cmd *cobra.Command, args []string) error {
			return set(args)
		},
	}
)

func set(args []string) error {
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	s := &wallhaven.Search{
		Categories: setCategories,
		Purities:   setPurity,
		Sorting:    setSorting,
		Order:      setOrder,
		TopRange:   setRange,
		AtLeast:    setAtLeast,
		Ratios:     setRatios,
		Page:       r.Intn(setPage) + 1,
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
	if setScript != "" {
		err = runScript(resultPath, setScript)
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
	downloadPath := path.Join(homedir, "Pictures/Wallpapers")
	if setPath != "" {
		downloadPath = setPath
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
