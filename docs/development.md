# Development

Viewdocs is written in [Go](http://golang.org/) and interacts with the [GitHub
Markdown API](http://developer.github.com/v3/markdown/).

If you want to hack on it, first you'll need to [get your GitHub access token](https://help.github.com/articles/creating-an-access-token-for-command-line-use)
and prepare your environment with something like the script below:

```sh
export ACCESS_TOKEN='<your access token>'
echo '127.0.0.1 <github-username>.viewdocs.dev' | sudo tee -a /etc/hosts
```

After that, just sing that same old song:

```sh
mkdir -p $GOPATH/src/github.com/progrium
cd $GOPATH/src/github.com/progrium
git clone https://github.com/progrium/viewdocs.git
cd viewdocs
glide install
go run viewdocs.go
```

Then visit `http://<github-username>.viewdocs.dev:8888/<one of your repos>` on
your browser and enjoy!
