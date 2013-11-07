package main

import (
	"encoding/json"
	"bytes"
	"log"
	"net/http"
	"io/ioutil"
	"os"
	"strings"

	"code.google.com/p/vitess/go/cache"
)

const cacheCapacity = 256*1024*1024 // 256MB

var defaultTemplate string

type CacheValue struct {
	Value string
}

func (cv *CacheValue) Size() int {
	return len(cv.Value)
}

func parseRequest(r *http.Request) (user, repo, doc string, err error) {
	hostname := strings.Split(r.Host, ".")
	user = hostname[0]
	path := strings.Split(r.RequestURI, "/")
	repo = path[1]
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

func fetchAndRenderDoc(user, repo, doc string) (string, error) {
	template := make(chan string)
	go func() {
		resp, err := http.Get("https://raw.github.com/"+user+"/"+repo+"/master/docs/template.html")
		if err != nil || resp.StatusCode == 404 {
			template <- defaultTemplate
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			template <- defaultTemplate
			return
		}
		template <- string(body)
	}()
	resp, err := http.Get("https://raw.github.com/"+user+"/"+repo+"/master/docs/"+doc+".md")
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
	payload, err := json.Marshal(map[string]string{"text": string(body)})
	if err != nil {
		return "", err
	}
	resp, err = http.Post("https://api.github.com/markdown?access_token="+os.Getenv("ACCESS_TOKEN"), "application/json", bytes.NewBuffer(payload))
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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	lru := cache.NewLRUCache(cacheCapacity)

	resp, err := http.Get("https://raw.github.com/progrium/viewdocs/master/docs/template.html")
	if err != nil || resp.StatusCode == 404 {
		log.Fatal("Unable to fetch default template")
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	defaultTemplate = string(body)

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
			user, repo, doc, err := parseRequest(r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			key := user + ":" + repo + ":" + doc
			value, ok := lru.Get(key)
			var output string
			if !ok {
				output, err = fetchAndRenderDoc(user, repo, doc)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				lru.Set(key, &CacheValue{output})
				log.Println("CACHE MISS:", key, lru.StatsJSON())
			} else {
				output = value.(*CacheValue).Value
			}
			w.Write([]byte(output))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	log.Println("Listening on port "+port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
