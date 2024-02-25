package wallhaven

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// WallpaperID is a string representing a wallpaper
type WallpaperID string

// Q is used to hold the Q params for various fulltext options that the WH Search supports
type Q struct {
	Tags       []string
	ExcudeTags []string
	UserName   string
	TagID      int
	Type       string // Type is one of png/jpg
	Like       WallpaperID
}

func (q Q) toQuery() url.Values {
	var sb strings.Builder
	for _, tag := range q.Tags {
		sb.WriteString("+")
		sb.WriteString(tag)
	}
	for _, etag := range q.ExcudeTags {
		sb.WriteString("-")
		sb.WriteString(etag)
	}
	if len(q.UserName) > 0 {
		sb.WriteString("@")
		sb.WriteString(q.UserName)
	}
	if len(q.Type) > 0 {
		sb.WriteString("type:")
		sb.WriteString(q.Type)
	}
	out := url.Values{}
	val := sb.String()
	if len(val) > 0 {
		out.Set("q", val)
	}
	return out
}

// Search provides various parameters to search for on wallhaven
type Search struct {
	Query       Q
	Categories  string
	Purities    string
	Sorting     string
	Order       string
	TopRange    string
	AtLeast     string
	Resolutions []string
	Ratios      []string
	Colors      []string // Colors is an array of hex colors represented as strings in #RRGGBB format
	Page        int64
}

func (s Search) toQuery() url.Values {
	v := s.Query.toQuery()
	if s.Categories != "" {
		v.Add("categories", s.Categories)
	}
	if s.Purities != "" {
		v.Add("purity", s.Purities)
	}
	if s.Sorting != "" {
		v.Add("sorting", s.Sorting)
	}
	if s.Order != "" {
		v.Add("order", s.Order)
	}
	if s.TopRange != "" && s.Sorting == "toplist" {
		v.Add("topRange", s.TopRange)
	}
	if s.AtLeast != "" {
		v.Add("atleast", s.AtLeast)
	}
	if len(s.Resolutions) > 0 {
		v.Add("resolutions", strings.Join(s.Ratios, ","))
	}
	if len(s.Ratios) > 0 {
		v.Add("ratios", strings.Join(s.Ratios, ","))
	}
	if len(s.Colors) > 0 {
		v.Add("colors", strings.Join([]string(s.Colors), ","))
	}
	if s.Page > 0 {
		v.Add("page", strconv.FormatInt(s.Page, 10))
	}
	return v
}

// SearchWallpapers performs a search on WH given a set of criteria.
// Note that this API behaves slightly differently than the various
// single item apis as it also includes the metadata for paging purposes
func SearchWallpapers(search *Search) (*SearchResults, error) {
	resp, err := getWithValues("/search/", search.toQuery())
	if err != nil {
		return nil, err
	}

	out := &SearchResults{}
	err = processResponse(resp, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func processResponse(resp *http.Response, out interface{}) error {
	byt, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return json.Unmarshal(byt, out)
}

// Result Structs -- server responses

// SearchResults a wrapper containing search results from wh
type SearchResults struct {
	Data []Wallpaper `json:"data"`
}

// Wallpaper information about a given wallpaper
type Wallpaper struct {
	Path string `json:"path"`
}

// Tag full data on a given wallpaper tag
type Tag struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Alias      string `json:"alias"`
	CategoryID int    `json:"category_id"`
	Category   string `json:"category"`
	Purity     string `json:"purity"`
	CreatedAt  string `json:"created_at"`
}

const baseURL = "https://wallhaven.cc/api/v1"

func getWithBase(p string) string {
	return baseURL + p
}

func getWithValues(p string, v url.Values) (*http.Response, error) {
	u, err := url.Parse(getWithBase(p))
	if err != nil {
		return nil, err
	}
	u.RawQuery = v.Encode()
	return getAuthedResponse(u.String())
}

func getAuthedResponse(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", os.Getenv("WH_API_KEY"))
	return client.Do(req)
}

var client = &http.Client{}

func download(filepath string, resp *http.Response) error {
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Download downloads a wallpaper given the local filepath to save the wallpaper to
func (w *Wallpaper) Download(dir string) error {
	resp, err := getAuthedResponse(w.Path)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, path.Base(w.Path))
	return download(path, resp)
}
