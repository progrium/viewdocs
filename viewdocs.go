package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/progrium/viewdocs/Godeps/_workspace/src/github.com/youtube/vitess/go/cache"
	"github.com/progrium/viewdocs/Godeps/_workspace/src/golang.org/x/net/html"
)

// CacheCapacity is an integer of memory in megabytes
// allocated for in-memory caching of processed documents
const CacheCapacity = 256 * 1024 * 1024

// CacheTTL is an integer of seconds that is used to
// configure the length of time a document stays in the cache
// raw.github.com cache TTL is ~120
const CacheTTL = 60

// DefaultTemplate is a string that contains the default template
// when a given repository does not have it's own template
var DefaultTemplate string

// CacheValue is a struct that contains a processed document
// and some metadata as to when that document was created
type CacheValue struct {
	Value     string
	CreatedAt int64
}

// Size is a method attached to the CacheValue struct which
// returns the length of a cache entry
func (cv *CacheValue) Size() int {
	return len(cv.Value)
}

func getenv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return value
}

func pathPrefix() string {
	return getenv("PATH_PREFIX", "")
}

func parseRequest(r *http.Request) (user, repo, ref, doc string) {
	hostname := strings.Split(r.Host, ".")
	user = getenv("GITHUB_USER", hostname[0])
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

func fixRelativeLinks(user, repo, doc, ref, body string) (string, error) {
	hostname := getenv("HOSTNAME", "viewdocs.io")
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
					hrefValue := strings.TrimPrefix(a.Val, "http://"+user+"."+hostname+"")
					if strings.Index(hrefValue, "/"+repo+"/") == 0 {
						n.Attr[i].Val = "/" + repoAndRef + "/" + strings.TrimPrefix(hrefValue, "/"+repo+"/")
						continue
					}
					fs := strings.Index(hrefValue, "/")
					fc := strings.Index(hrefValue, ":")
					fh := strings.Index(hrefValue, "#")
					if fs == 0 || fh == 0 ||
						(fc >= 0 && fc < fs) ||
						(fh >= 0 && fh < fs) {
						continue
					}
					n.Attr[i].Val = "/" + repoAndRef + "/" + hrefValue
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

func fetchTemplate(template chan string, user string, repo string, ref string, name string) {
	if getenv("DEBUG", "0") == "1" {
		pathPrefix := pathPrefix()
		bodyStr, err := readFile(pathPrefix + "docs/" + name + ".html")
		if err == nil {
			template <- bodyStr
			return
		}
		if name != "template" {
			bodyStr, err := readFile(pathPrefix + "docs/template.html")
			if err == nil {
				template <- bodyStr
				return
			}
		}
	}

	fetched := fetchURL(template, "https://raw.github.com/"+user+"/"+repo+"/"+ref+"/docs/"+name+".html")
	if !fetched && name != "template" {
		fetched = fetchURL(template, "https://raw.github.com/"+user+"/"+repo+"/"+ref+"/docs/template.html")
	}

	if !fetched {
		template <- DefaultTemplate
	}
}

func fetchURL(channel chan string, url string) bool {
	resp, err := http.Get(url)
	if err == nil && resp.StatusCode != 404 {
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err == nil {
			channel <- string(body)
			return true
		}
	}

	return false
}

func fetchDoc(user, repo, ref, doc string) (string, error) {
	if getenv("DEBUG", "0") == "1" {
		pathPrefix := pathPrefix()
		bodyStr, err := readFile(pathPrefix + doc)
		if err == nil {
			return bodyStr, err
		}
	}
	log.Println("FETCH: https://raw.github.com/" + user + "/" + repo + "/" + ref + "/" + doc)
	resp, err := http.Get("https://raw.github.com/" + user + "/" + repo + "/" + ref + "/" + doc)
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
		if strings.HasPrefix(doc, "docs/") {
			newDoc := strings.TrimPrefix(doc, "docs/")
			// special-case the index page
			if doc == "docs/index.md" {
				for ext := range markdownExtensions() {
					bodyStr, err := fetchDoc(user, repo, ref, "README"+ext)
					return cleanupDocLinks(bodyStr, err)
				}
			}
			bodyStr, err := fetchDoc(user, repo, ref, newDoc)
			return cleanupDocLinks(bodyStr, err)
		}
		body = []byte("# Page not found")
	}
	return string(body), nil
}

func cleanupDocLinks(bodyStr string, err error) (string, error) {
	if err == nil {
		bodyStr = strings.Replace(bodyStr, "](docs/", "](", -1)
	}
	return bodyStr, err
}

func fetchAndRenderDoc(user, repo, ref, doc string) (string, error) {
	template := make(chan string)
	templateName := "template"
	templateRecv := false
	defer func() {
		if !templateRecv {
			<-template
		}
	}()

	if doc == "index.md" {
		templateName = "home"
	}

	go fetchTemplate(template, user, repo, ref, templateName)

	if !isAsset(doc) {
		// https://github.com/github/markup/blob/master/lib/github/markups.rb#L1
		mdExts := markdownExtensions()
		if ok, _ := mdExts[path.Ext(doc)]; !ok {
			doc += ".md"
		}
	}

	bodyStr, err := fetchDoc(user, repo, ref, "docs/"+doc)
	if err != nil {
		return "", err
	}

	if isAsset(doc) {
		return bodyStr, nil
	}

	resp, err := http.Post("https://api.github.com/markdown/raw?access_token="+os.Getenv("ACCESS_TOKEN"), "text/x-markdown", strings.NewReader(bodyStr))
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}

	pagesClass := strings.Replace(doc, "/", "-", -1)
	pagesClass = pagesClass[:len(pagesClass)-len(path.Ext(pagesClass))]

	output := strings.Replace(<-template, "{{CONTENT}}", string(body), 1)
	templateRecv = true
	output = strings.Replace(output, "{{NAME}}", repo, -1)
	output = strings.Replace(output, "{{USER}}", user, -1)
	output = strings.Replace(output, "{{PAGE_CLASS}}", pagesClass, -1)
	output = strings.Replace(output, "{{REF}}", ref, -1)
	output = strings.Replace(output, "{{DOC}}", doc, -1)

	// Fix relative links
	output, err = fixRelativeLinks(user, repo, doc, ref, output)
	if err != nil {
		return "", err
	}

	return output, nil
}

func markdownExtensions() map[string]bool {
	return map[string]bool{
		".md":        true,
		".mkdn":      true,
		".mdwn":      true,
		".mdown":     true,
		".markdown":  true,
		".litcoffee": true,
	}
}

func isAsset(name string) bool {
	assetExts := map[string]bool{
		".appcache": true,
		".bmp":      true,
		".css":      true,
		".jpg":      true,
		".jpeg":     true,
		".js":       true,
		".json":     true,
		".png":      true,
		".ico":      true,
	}

	if ok, _ := assetExts[path.Ext(name)]; ok {
		return true
	}

	return false
}

func readFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return strings.Join(lines, "\n"), scanner.Err()
}

func handleRedirects(w http.ResponseWriter, r *http.Request, user string, repo string, ref string, doc string) bool {
	redirectTo := ""
	if r.RequestURI == "/" {
		redirectTo = "http://progrium.viewdocs.io/viewdocs/"
	}
	if strings.Contains(r.Host, "progrium") && strings.HasPrefix(r.RequestURI, "/dokku") {
		redirectTo = "http://dokku.viewdocs.io" + r.RequestURI
	}
	if isAsset(doc) {
		redirectTo = "https://cdn.rawgit.com/" + user + "/" + repo + "/" + ref + "/docs/" + doc
	}
	if !strings.HasSuffix(r.RequestURI, "/") {
		for ext := range markdownExtensions() {
			if strings.HasSuffix(r.RequestURI, ext) {
				redirectTo = strings.TrimSuffix(r.RequestURI, ext) + "/"
				break
			}
		}
		if redirectTo == "" && !isAsset(r.RequestURI) {
			redirectTo = r.RequestURI + "/"
		}
	}
	for ext := range markdownExtensions() {
		if strings.HasSuffix(r.RequestURI, ext+"/") {
			redirectTo = strings.TrimSuffix(r.RequestURI, ext+"/") + "/"
			break
		}
	}
	if redirectTo != "" {
		log.Println("REDIRECT: ", redirectTo)
		http.Redirect(w, r, redirectTo, 301)
		return true
	}
	return false
}

func main() {
	if getenv("ACCESS_TOKEN", "") == "" {
		log.Fatal("ACCESS_TOKEN was not found. Read http://progrium.viewdocs.io/viewdocs/development/ for more info")
	}

	port := getenv("PORT", "8888")
	doNotCache := getenv("USE_CACHE", "true") == "false"
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
		if r.RequestURI == "/favicon.ico" {
			return
		}
		switch r.Method {
		case "GET":
			user, repo, ref, doc := parseRequest(r)
			redirected := handleRedirects(w, r, user, repo, ref, doc)
			if redirected {
				return
			}
			log.Printf("Building docs for '%s/%s' (ref: %s)", user, repo, ref)
			key := user + ":" + repo + ":" + doc + ":" + ref
			value, ok := lru.Get(key)
			var output string
			if !ok || doNotCache {
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
