package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/alecthomas/kong"

	"main/wallhaven"
)

var cli struct {
	Search struct {
		Query string `arg:"" name:"query" help:"what to search for." type:"string"`
	} `cmd:"" help:"search for wallpaper"`
	Top struct {
		Purity string `arg:"" name:"purity" optional:"" help:"purity of results"`
	} `cmd:"" help:"random toplist wallpaper"`
}

func main() {
	ctx := kong.Parse(&cli)
	switch ctx.Command() {
	case "search <query>":
		err := searchAndSet(cli.Search.Query)
		if err != nil {
			panic(err)
		}
	case "top", "top <purity>":
		err := setTop(cli.Top.Purity)
		if err != nil {
			panic(err)
		}
	default:
		panic(ctx.Command())
	}
}

func searchAndSet(query string) error {
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	results, err := wallhaven.SearchWallpapers(&wallhaven.Search{
		Query: wallhaven.Q{
			Tags: []string{query},
		},
		Categories: "111",
		Purities:   "110",
		Sorting:    wallhaven.Relevance,
		Order:      wallhaven.Desc,
		AtLeast:    wallhaven.Resolution{Width: 2560, Height: 1400},
		Ratios: []wallhaven.Ratio{
			{Horizontal: 16, Vertical: 9},
			{Horizontal: 16, Vertical: 10},
		},
		Page: 1,
	})
	if err != nil {
		return err
	}
	result, err := getOrDownload(results, r)
	if err != nil {
		return nil
	}
	err = setWallPaperAndRestartStuff(result)
	if err != nil {
		return err
	}
	return nil
}

func setTop(purity string) error {
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	s := &wallhaven.Search{
		Categories: "010",
		Purities:   "110",
		Sorting:    wallhaven.Toplist,
		Order:      wallhaven.Desc,
		TopRange:   "1y",
		AtLeast:    wallhaven.Resolution{Width: 2560, Height: 1400},
		Ratios: []wallhaven.Ratio{
			{Horizontal: 16, Vertical: 9},
			{Horizontal: 16, Vertical: 10},
		},
		Page: r.Intn(5) + 1,
	}
	if purity != "" {
		s.Purities = purity
	}
	results, err := wallhaven.SearchWallpapers(s)
	if err != nil {
		return err
	}
	result, err := getOrDownload(results, r)
	if err != nil {
		return err
	}
	err = setWallPaperAndRestartStuff(result)
	if err != nil {
		return err
	}
	return nil
}

func setWallPaperAndRestartStuff(result wallhaven.Wallpaper) error {
	homedir, _ := os.UserHomeDir()
	_, err := exec.Command("wal", "--cols16", "-i", path.Join(homedir, "Pictures/Wallpapers", path.Base(result.Path)), "-n", "-a", "85").
		Output()
	if err != nil {
		return err
	}
	_, err = exec.Command("swww", "img", path.Join(homedir, "/Pictures/Wallpapers", path.Base(result.Path)), "--transition-step=20", "--transition-fps=60").
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
