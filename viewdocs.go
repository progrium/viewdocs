package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"code.google.com/p/go.net/html"
	"code.google.com/p/vitess/go/cache"
)

const CacheCapacity = 256 * 1024 * 1024 // 256MB
const CacheTTL = 60                     // raw.github.com cache TTL is ~120

var DefaultTemplate string

type CacheValue struct {
	Value     string
	CreatedAt int64
}

func (cv *CacheValue) Size() int {
	return len(cv.Value)
}

func parseRequest(r *http.Request) (user, repo, ref, doc string) {
	hostname := strings.Split(r.Host, ".")
	user = hostname[0]
	path := strings.Split(r.RequestURI, "/")

	repoAndRef := strings.Split(path[1], "~")
	repo = repoAndRef[0]
	if len(repoAndRef) == 1 {
		ref = "master"
	} else {
		ref = repoAndRef[1]
	}

	if len(path) < 3 || (len(path) == 3 && strings.HasSuffix(r.RequestURI, "/")) {
		doc = "index.md"
	} else {
		doc = strings.Join(path[2:], "/")
		if strings.HasSuffix(doc, "/") {
			doc = doc[:len(doc)-1]
		}
	}
	return
}

func fixRelativeLinks(doc, repo, ref, body string) (string, error) {
	repoAndRef := repo
	if ref != "master" {
		repoAndRef += "~" + ref
	}
	n, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for i, a := range n.Attr {
				if a.Key == "href" {
					fs := strings.Index(a.Val, "/")
					fc := strings.Index(a.Val, ":")
					fh := strings.Index(a.Val, "#")
					if fs == 0 || fh == 0 ||
						(fc >= 0 && fc < fs) ||
						(fh >= 0 && fh < fs) {
						continue
					}
					n.Attr[i].Val = "/" + repoAndRef + "/" + a.Val
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	b := new(bytes.Buffer)
	if err := html.Render(b, n); err != nil {
		return "", err
	}
	return b.String(), nil
}

func fetchAndRenderDoc(user, repo, ref, doc string) (string, error) {
	template := make(chan string)
	templateRecv := false
	defer func() {
		if !templateRecv {
			<-template
		}
	}()
	go func() {
		resp, err := http.Get("https://raw.github.com/" + user + "/" + repo + "/" + ref + "/docs/template.html")
		if err != nil || resp.StatusCode == 404 {
			template <- DefaultTemplate
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			template <- DefaultTemplate
			return
		}
		template <- string(body)
	}()
	// https://github.com/github/markup/blob/master/lib/github/markups.rb#L1
	mdExts := map[string]bool{
		".md":        true,
		".mkdn":      true,
		".mdwn":      true,
		".mdown":     true,
		".markdown":  true,
		".litcoffee": true,
	}
	if ok, _ := mdExts[path.Ext(doc)]; !ok {
		doc += ".md"
	}
	resp, err := http.Get("https://raw.github.com/" + user + "/" + repo + "/" + ref + "/docs/" + doc)
	if err != nil {
		return "", err
	}
	var body []byte
	if resp.StatusCode == 200 {
		body, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", err
		}
	} else {
		resp.Body.Close()
		body = []byte("# Page not found")
	}

	bodyStr := string(body)

	resp, err = http.Post("https://api.github.com/markdown/raw?access_token="+os.Getenv("ACCESS_TOKEN"), "text/x-markdown", strings.NewReader(bodyStr))
	if err != nil {
		return "", err
	}
	body, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}

	output := strings.Replace(<-template, "{{CONTENT}}", string(body), 1)
	templateRecv = true
	output = strings.Replace(output, "{{NAME}}", repo, -1)
	output = strings.Replace(output, "{{USER}}", user, -1)

	// Fix relative links
	output, err = fixRelativeLinks(doc, repo, ref, output)
	if err != nil {
		return "", err
	}

	return output, nil
}

func main() {
	if os.Getenv("ACCESS_TOKEN") == "" {
		// TODO: Add direct link to Development section of the README
		log.Fatal("ACCESS_TOKEN was not found!")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	lru := cache.NewLRUCache(CacheCapacity)

	resp, err := http.Get("https://raw.github.com/progrium/viewdocs/master/docs/template.html")
	if err != nil || resp.StatusCode == 404 {
		log.Fatal("Unable to fetch default template")
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	DefaultTemplate = string(body)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/" {
			http.Redirect(w, r, "http://progrium.viewdocs.io/viewdocs", 301)
			return
		}
		if r.RequestURI == "/favicon.ico" {
			return
		}
		switch r.Method {
		case "GET":
			user, repo, ref, doc := parseRequest(r)
			log.Printf("Building docs for '%s/%s' (ref: %s)", user, repo, ref)
			key := user + ":" + repo + ":" + doc + ":" + ref
			value, ok := lru.Get(key)
			var output string
			if !ok {
				output, err = fetchAndRenderDoc(user, repo, ref, doc)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				lru.Set(key, &CacheValue{output, time.Now().Unix()})
				log.Println("CACHE MISS:", key, lru.StatsJSON())
			} else {
				output = value.(*CacheValue).Value
				if time.Now().Unix()-value.(*CacheValue).CreatedAt > CacheTTL {
					lru.Delete(key)
				}
			}
			w.Write([]byte(output))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	log.Println("Listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
