/*
archivr downloads and saves tumblr text posts to disk via the tumblr api
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"
	// TODO: "github.com/hashicorp/go-retryablehttp"
	// "github.com/.../goquery"
)

func main() {
	oauthKey := flag.String("key", "", "your oauth consumer key from https://www.tumblr.com/oauth/apps")
	blogName := flag.String("blog", "", "the name of the blog you want to archive (the tumblr subdomain or custom domain)")
	outputDir := flag.String("o", "", "output directory, defaults to [blogname].archive (will be created if it doesn't already exist)")
	_ = outputDir
	// qps := flag.Int("limit", 0, "how quickly we can query the api")
	flag.Parse()
	if *oauthKey == "" {
		fmt.Println("-key, your oauth key, is required to use the tumblr api")
		return
	}
	if *blogName == "" {
		fmt.Println("-blog, the name of the blog you want to archive, is required")
		return
	}

	scraper := &archiver{
		key:      *oauthKey,
		blogname: *blogName,
	}
	info, err := scraper.BlogInfo()
	if err != nil {
		fmt.Println("failed to get blog info for ", scraper.blogname, err)
		return
	}

	dirName, err := setUpDir(*outputDir, info.Name)
	if err != nil {
		fmt.Printf("failed to initialize output directory: %s\n", err)
		return
	}

	templ, err := template.New("post").Parse(postTempl)
	if err != nil {
		fmt.Println("failed to parse post template:", err)
		return
	}

	fmt.Printf("writing %s's %d posts to %s\n", info.Name, info.Posts, dirName)

	offset := 0
	for offset < info.Posts {
		fmt.Printf("getting page at offset %d / %d\n", offset, info.Posts)

		page, err := scraper.Page(offset)
		if err != nil {
			fmt.Printf("failed to get page at offset %d: %s", offset, err)
			return
		}
		for _, post := range page {
			// make the file
			f, err := os.Create(filepath.Join(dirName, post.Slug))
			if err != nil {
				fmt.Println("failed to write", filepath.Join(dirName, post.Slug), err)
				return
			}

			err = templ.Execute(f, post)
			f.Close()
			if err != nil {
				fmt.Println("failed to write post template to file:", err)
				return
			}
			err = os.Chtimes(f.Name(), post.Date, post.Date)
			if err != nil {
				fmt.Println("failed to change the modification date of", f.Name(), err)
			}

			// TODO: save the images with some known naming scheme
		}

		offset += len(page)
	}

}

func (a *archiver) BlogInfo() (*BlogMetadata, error) {
	u := fmt.Sprintf("https://api.tumblr.com/v2/blog/%s/info?api_key=%s", a.blogname, a.key)
	fmt.Println("made blog info url:", u)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	info := &tumblrBlogInfo{}
	err = json.NewDecoder(resp.Body).Decode(info)
	if err != nil {
		return nil, err
	}
	return &BlogMetadata{
		Name:  info.Resp.Blog.Name,
		Posts: info.Resp.Blog.Posts,
	}, nil
}

func (a *archiver) Page(offset int) ([]*Post, error) {
	u := fmt.Sprintf("https://api.tumblr.com/v2/blog/%s/posts?api_key=%s", a.blogname, a.key)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pages tumblrBlogPosts
	err = json.NewDecoder(resp.Body).Decode(&pages)
	if err != nil {
		return nil, err
	}

	result := make([]*Post, len(pages.Response.Posts))
	for i, page := range pages.Response.Posts {
		result[i] = &Post{
			Body: page.Body, // only the <p>, may have to template out a html shell with a comment about tumbrl source
			ID:   page.ID,
			Slug: page.Slug, // can make this the file name
			Url:  page.Url,
			Date: time.Unix(page.Timestamp, 0),
			// summary?
		}
	}
	return result, nil
}

type Archiver interface {
	BlogInfo() (*BlogMetadata, error)
	Page(offset int) (posts []Post, nextOffset int, err error) // gets up to 20 posts
}

type archiver struct {
	key      string
	blogname string
	// client   *retryablehttp.Client
}

type tumblrBlogInfo struct {
	Resp struct {
		Blog struct {
			Posts int    `json:"posts"`
			Name  string `json:"name"`
		} `json:"blog"`
	} `json:"response"`
}

type tumblrBlogPosts struct {
	Response struct {
		Posts []*tumblrBlogPost `json:"posts"`
	} `json:"response"`
}

type tumblrBlogPost struct {
	ID        int64  `json:"id"`
	Url       string `json:"post_url"`
	Slug      string `json:"slug"`
	Summary   string `json:"summary"`
	Body      string `json:"body"`
	Timestamp int64  `json:"timestamp"`
}

type BlogMetadata struct {
	Name  string
	Posts int
}

type Post struct {
	Body  string
	Title string
	ID    int64
	Url   string // orig tumblr.com url
	Date  time.Time
	Slug  string
	/// TODO
}

var postTempl string = `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8">
		<title>{{.Title}}</title>
		<!-- this is an archived tumblr post published on {{.Date}} at {{.Url}} -->
  </head>
  <body>
	{{.Body}}
  </body>
</html>
`

// setUpDir creates the dir if it doesn't exist
// if dir is "", blogname-archive will be used
// it returns the name of the created / selected dir
func setUpDir(dir, blogName string) (string, error) {
	if dir == "" {
		dir = blogName + "-archive"
	}

	stat, err := os.Stat(dir)
	if os.IsNotExist(err) || !stat.IsDir() {
		// create it
		return dir, os.MkdirAll(dir, 0777)
	}
	if err != nil {
		return "", err
	}

	// otherwise, it already exists and is a directory
	return dir, nil
}
