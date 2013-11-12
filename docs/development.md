# Development

Viewdocs is written in [Go](http://golang.org/) and interacts with the [GitHub
Markdown API](http://developer.github.com/v3/markdown/), if you want to hack on
it [get your access token](https://help.github.com/articles/creating-an-access-token-for-command-line-use),
add a `127.0.0.1 <github-username>.viewdocs.dev` to your `/etc/hosts` or equivalent and sing
that same old song:

```sh
mkdir -p $GOPATH/src/github.com/progrium
git clone https://github.com/progrium/viewdocs.git $GOPATH/src/github.com/progrium/viewdocs
cd $GOPATH/src/github.com/progrium/viewdocs
export ACCESS_TOKEN='<your access token>'
go get
go run viewdocs.go
```

Then visit `http://<github-username>.viewdocs.dev:8888/<one of your repos>` on
your browser and enjoy!

