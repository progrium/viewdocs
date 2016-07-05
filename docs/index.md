# Welcome to Viewdocs

Viewdocs is [Read the Docs](https://readthedocs.org/) meets [Gist.io](http://gist.io/) for simple project documentation. It renders Markdown from your repository's `docs` directory as simple static pages.

### Getting Started

Just make a `docs` directory in your Github project repository and put an `index.md` file in there to get started. Then browse to:

	http://<github-username>.viewdocs.io/<repository-name>

Any other Markdown files in your `docs` directory are available as a subpath, including files in directories. You can update pages by just pushing to your repository or editing directly on Github. It can take up to 1-2 minutes before changes will appear.

This page is an example of what documentation will look like by default. Here is [another example page](/viewdocs/example). The source for these pages are in the [docs directory](https://github.com/progrium/viewdocs/tree/master/docs) of the Viewdocs project.

### Preview changes before pushing documentation back the repository

If you want to find out how things look like locally before pushing your code back to the remote repository, you might want to try out [`previewdocs`](http://fgrehm.viewdocs.io/previewdocs).

### Advanced Usage

You can show documentation for different [branches](http://inconshreveable.viewdocs.io/ngrok~master/DEVELOPMENT) or [tags](http://discourse.viewdocs.io/discourse~v0.9.6/INSTALL-ubuntu) of a repository by including a reference name after a tilde in the repository part of the path. It would look like this:

	http://<github-username>.viewdocs.io/<repository-name>~<refname>

You can also customize the look and layout of your docs. Make your own `docs/template.html` based on the [default template](https://github.com/progrium/viewdocs/blob/master/docs/template.html) and your pages will be rendered with that template. If you create a `home.html` template, this will be used for your project's landing page.

Redirects are supported for users who wish to move around files in their documentation structure. Simply create a `docs/viewdocs.json` file and specify your redirects like so:

```json
{
    "redirects": {
        "configuration-management": "configuration/environment-variables/"
    }
}
```

This will result in the url `/REPOSITORY/configuration-management/` being redirected to `/REPOSITORY/configuration/environment-variables/` url.

I also highly recommend you [read the source](https://github.com/progrium/viewdocs/blob/master/viewdocs.go) to this app. It's less than 500 lines of Go. If you want to hack on Viewdocs, [check this out](/viewdocs/development).

<br />
Enjoy!<br />
[Jeff Lindsay](http://twitter.com/progrium)
