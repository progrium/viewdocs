package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

var port = flag.String("p", "8888", "Port to listen on")

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:	%v -p <port>\n\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func errorResponse(w http.ResponseWriter, e error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(e.Error()))
}

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			resp, err := http.Get("https://raw.github.com/idan/gistio/master/static/css/screen.css")
			if err != nil {
				errorResponse(w, err)
				return
			}
			defer resp.Body.Close()
			_, err = io.Copy(w, resp.Body)
			if err != nil {
				errorResponse(w, err)
				return
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
