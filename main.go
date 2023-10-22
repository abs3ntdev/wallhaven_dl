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
		Purity string `name:"purity" optional:"" help:"purity of results"`
		Range  string `name:"range" optional:"" help:"range for toplist"`
	} `cmd:"" help:"random toplist wallpaper"`
	Img struct {
		Path string `arg:"" name:"path" help:"path to image or directory." type:"path"`
	} `cmd:"" help:"set from file"`
}

func main() {
	ctx := kong.Parse(&cli)
	switch ctx.Command() {
	case "search <query>":
		err := searchAndSet(cli.Search.Query)
		if err != nil {
			panic(err)
		}
	case "top":
		err := setTop(cli.Top.Purity, cli.Top.Range)
		if err != nil {
			panic(err)
		}
	case "img <path>":
		err := setFromPath(cli.Img.Path)
		if err != nil {
			panic(err)
		}
	default:
		panic(ctx.Command())
	}
}

func setFromPath(filePath string) error {
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
	err = setWallPaperAndRestartStuff(result.Path)
	if err != nil {
		return err
	}
	return nil
}

func setTop(purity, topRange string) error {
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	s := &wallhaven.Search{
		Categories: "010",
		Purities:   "110",
		Sorting:    wallhaven.Toplist,
		Order:      wallhaven.Desc,
		TopRange:   "6m",
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
	if topRange != "" {
		s.TopRange = topRange
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
