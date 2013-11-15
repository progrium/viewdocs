package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

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

func parseRequest(r *http.Request) (user, repo, ref, doc string, err error) {
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
		doc = "index"
	} else {
		doc = strings.Join(path[2:], "/")
		if strings.HasSuffix(doc, "/") {
			doc = doc[:len(doc)-1]
		}
	}
	return
}

func fetchAndRenderDoc(user, repo, ref, doc string) (string, error) {
	template := make(chan string)
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
	resp, err := http.Get("https://raw.github.com/" + user + "/" + repo + "/" + ref + "/docs/" + doc + ".md")
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
	// Ajust relative links for specific git revisions
	// FIXME: This doesn't handle relative links on the template / layout,
	//        just on the markdown page itself
	if ref != "master" {
		reg := regexp.MustCompile(`(\[[^\]]+\]\()(/` + repo + `)([^~)])`)
		bodyStr = reg.ReplaceAllString(bodyStr, "$1$2~"+ref+"$3")
	}

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
	output = strings.Replace(output, "{{NAME}}", repo, -1)
	output = strings.Replace(output, "{{USER}}", user, -1)
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
			user, repo, ref, doc, err := parseRequest(r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
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
