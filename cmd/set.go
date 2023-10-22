package cmd

import (
	"fmt"
	"io"
	"log"
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
		"range for setList search (default is 1 Year).",
	)
	setCmd.PersistentFlags().StringVarP(
		&setPurity,
		"purity",
		"p",
		"110",
		"purity for the setList search (default is 110).",
	)
	setCmd.PersistentFlags().
		StringVarP(
			&setCategories,
			"categories",
			"c",
			"010",
			"categories for the setList search (default is 010).",
		)
	setCmd.PersistentFlags().
		StringVarP(
			&setSorting,
			"sort",
			"s",
			"toplist",
			"sort by for results, valid sorts: date_added, relevance, random, views, favorites, setlist (default is toplist)",
		)
	setCmd.PersistentFlags().
		StringVarP(
			&setOrder,
			"order",
			"o",
			"desc",
			"sort order for results, valid sorts: asc desc (default is desc)",
		)
	setCmd.PersistentFlags().
		BoolVarP(
			&localPath,
			"localPath",
			"l",
			false,
			"set to true if the argument is to a directory or an image file (default is false)",
		)
}

var (
	setRange      string
	setPurity     string
	setCategories string
	setSorting    string
	setOrder      string
	localPath     bool
	setCmd        = &cobra.Command{
		Use:     "set",
		Aliases: []string{"s"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "set wallpaper from setlist",
		RunE: func(cmd *cobra.Command, args []string) error {
			return set(args)
		},
	}
)

func set(args []string) error {
	if localPath {
		if len(args) == 0 {
			return fmt.Errorf("you must provide a path to an image or directory of images to use this option")
		}
		filePath := args[0]
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		if fileInfo.IsDir() {
			files, err := os.ReadDir(filePath)
			if err != nil {
				return err
			}
			file := files[rand.Intn(len(files))]
			return setWallPaperAndRestartStuff(file.Name())
		}
		return setWallPaperAndRestartStuff(filePath)
	}
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	s := &wallhaven.Search{
		Categories: setCategories,
		Purities:   setPurity,
		Sorting:    setSorting,
		Order:      setOrder,
		TopRange:   setRange,
		AtLeast:    wallhaven.Resolution{Width: 2560, Height: 1440},
		Ratios: []wallhaven.Ratio{
			{Horizontal: 16, Vertical: 9},
			{Horizontal: 16, Vertical: 10},
		},
		Page: r.Intn(5) + 1,
	}
	log.Println(args)
	if len(args) > 0 {
		s.Query = wallhaven.Q{
			Tags: []string{args[0]},
		}
	}
	results, err := wallhaven.SearchWallpapers(s)
	if err != nil {
		return err
	}
	result, err := getOrDownload(results, r)
	if err != nil {
		return err
	}
	err = setWallPaperAndRestartStuff(result.Path)
	if err != nil {
		return err
	}
	return nil
}

func getOrDownload(results *wallhaven.SearchResults, r *rand.Rand) (wallhaven.Wallpaper, error) {
	if len(results.Data) == 0 {
		return wallhaven.Wallpaper{}, fmt.Errorf("no wallpapers found")
	}
	homedir, _ := os.UserHomeDir()
	result := results.Data[r.Intn(len(results.Data))]
	if _, err := os.Stat(path.Join(homedir, "Pictures/Wallpapers", path.Base(result.Path))); err != nil {
		err = result.Download(path.Join(homedir, "Pictures/Wallpapers"))
		if err != nil {
			return wallhaven.Wallpaper{}, err
		}
	}
	return result, nil
}

func setWallPaperAndRestartStuff(result string) error {
	homedir, _ := os.UserHomeDir()
	_, err := exec.Command("wal", "--cols16", "-i", path.Join(homedir, "Pictures/Wallpapers", path.Base(result)), "-n", "-a", "85").
		Output()
	if err != nil {
		return err
	}
	_, err = exec.Command("swww", "img", path.Join(homedir, "/Pictures/Wallpapers", path.Base(result))).
		Output()
	if err != nil {
		return err
	}
	_, err = exec.Command("restart_dunst").
		Output()
	if err != nil {
		return err
	}
	_, err = exec.Command("pywalfox", "update").
		Output()
	if err != nil {
		return err
	}
	source, err := os.Open(path.Join(homedir, ".cache/wal/discord-wal.theme.css"))
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.Create(path.Join(homedir, ".config/Vencord/themes/discord-wal.theme.css"))
	if err != nil {
		return err
	}
	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}
	return nil
}
