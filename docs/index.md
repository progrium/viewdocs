# Welcome to Viewdocs

Viewdocs is [Read the Docs](https://readthedocs.org/) meets [Gist.io](http://gist.io/) for simple Markdown project documentation.

**It renders Markdown in your repository's `docs` directory as simple static pages.**

Just make a `docs` directory in your Github project repository and put an `index.md` file in there to get started. Then browse to:

	http://<github-username>.viewdocs.io/<repository-name>

Any other Markdown files in your `docs` directory are available as a subpath, including files in directories. You can update pages by just pushing to your repository or editing directly on Github. It can take up to 1-2 minutes before changes will appear.

This page is an example of what documentation will look like by default. Here is [another example page](/viewdocs/example). The source for these pages are in the [docs directory](https://github.com/progrium/viewdocs/tree/master/docs) of the Viewdocs project.

For the adventurous, make your own `docs/template.html` based on the [default template](https://github.com/progrium/viewdocs/blob/master/docs/template.html) for custom layouts. I also highly recommend you [read the source](https://github.com/progrium/viewdocs/blob/master/viewdocs.go) to this app. It's only 150 lines of Go.

If you want to hack on it, please [check this out](/viewdocs/development).

Enjoy!<br />
[Jeff Lindsay](http://twitter.com/progrium)