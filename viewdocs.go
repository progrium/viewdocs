package main

import (
	"flag"
	"encoding/json"
	"fmt"
	"bytes"
	"log"
	"net/http"
	"io/ioutil"
	"os"
	"strings"
)

var port = flag.String("p", "8888", "Port to listen on")
const template = `<!doctype html>
<head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge,chrome=1">
    <title>viewdocs.io</title>
    <meta name="viewport" content="width=device-width,initial-scale=1">
    <script src="//polyfill.io"></script>
    <link href="//polyfill.io/normalize.css" rel="stylesheet">
    <link rel="stylesheet" type="text/css" href="//cloud.typography.com/678416/735422/css/fonts.css" />
    <link href='http://fonts.googleapis.com/css?family=Source+Code+Pro:300,600' rel='stylesheet' type='text/css'>
    <link rel="stylesheet" href="http://static.gist.io/css/screen.css">
</head>
<body>
    <section class="content">
        <header>
            <h1 id="gistid"><a href="#">viewdocs</a></h1>
            <!--h2 id="description" class="instapaper_title entry-title"></h2-->
        </header>
        <div id="gistbody" class="instapaper_body entry-content">
            {{CONTENT}}
        </div>
        <footer>
            <p>Inspired by <a href="http://gist.io">gist.io</a> &middot; documentation for hackers &middot; zero setup &middot; publish in seconds</p>
        </footer>
    </section>
</body>
</html>`

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:	%v -p <port>\n\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func errorResponse(w http.ResponseWriter, e string) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(e))
}

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hostname := strings.Split(r.Host, ".")
		if len(hostname) < 2 {
			errorResponse(w, "Huh?")
			return
		}
		username := hostname[0]
		path := strings.Split(r.RequestURI, "/")
		if len(path) < 2 {
			errorResponse(w, "Huh?")	
			return
		}
		reponame := path[1]
		var docpath string
		if len(path) < 3 {
			docpath = "index"
		} else {
			docpath = strings.Join(path[2:], "/")
		}
		switch r.Method {
		case "GET":
			resp, err := http.Get("https://raw.github.com/"+username+"/"+reponame+"/master/docs/"+docpath+".md")
			if err != nil {
				errorResponse(w, err.Error())
				return
			}
			body, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				errorResponse(w, err.Error())
				return	
			}
			payload, _ := json.Marshal(map[string]string{"text": string(body)})
			resp, err = http.Post("https://api.github.com/markdown", "application/json", bytes.NewBuffer(payload))
			if err != nil {
				errorResponse(w, err.Error())
				return
			}
			body, err = ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				errorResponse(w, err.Error())
				return	
			}
			w.Write([]byte(strings.Replace(template, "{{CONTENT}}", string(body), 1)))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	log.Println("Listening on port "+*port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
