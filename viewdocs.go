package main

import (
	"encoding/json"
	"bytes"
	"errors"
	"log"
	"net/http"
	"io/ioutil"
	"os"
	"strings"

	"code.google.com/p/vitess/go/cache"
)

const cacheCapacity = 246*1024*1024 // 256MB
const template = `<!doctype html>
<head>
    <meta charset="utf-8">
    <title>viewdocs.io</title>
    <link rel="stylesheet" type="text/css" href="//cloud.typography.com/678416/735422/css/fonts.css" />
    <link href='http://fonts.googleapis.com/css?family=Source+Code+Pro:300,600' rel='stylesheet' type='text/css'>
    <link rel="stylesheet" href="http://static.gist.io/css/screen.css">
</head>
<body>
    <section class="content">
        <header>
            <h1 id="gistid"><a href="http://github.com/{{USER}}/{{NAME}}">{{NAME}}</a></h1>
        </header>
        <div id="gistbody" class="instapaper_body entry-content">
            {{CONTENT}}
        </div>
    </section>
</body>
</html>`

type CacheValue struct {
	Value string
}

func (cv *CacheValue) Size() int {
	return len(cv.Value)
}

func parseRequest(r *http.Request) (user, repo, doc string, err error) {
	hostname := strings.Split(r.Host, ".")
	if len(hostname) < 2 {
		return "", "", "", errors.New("Bad hostname")
	}
	user = hostname[0]
	path := strings.Split(r.RequestURI, "/")
	repo = path[1]
	if len(path) < 3 {
		doc = "index"
	} else {
		doc = strings.Join(path[2:], "/")
	}
	return
}

func fetchAndRenderDoc(user, repo, doc string) (string, error) {
	resp, err := http.Get("https://raw.github.com/"+user+"/"+repo+"/master/docs/"+doc+".md")
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
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
	output := strings.Replace(template, "{{CONTENT}}", string(body), 1)
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
