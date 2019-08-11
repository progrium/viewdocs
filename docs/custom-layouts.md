# Custom Layouts

Viewdocs.io requires a layout template in order to render your markdown pages.  The layout file, template.html, must be located at the root of your docs directory.  A default layout is included - and was used to render the page you are currently reading.  Developing a custom layout is simple, as long as certain constraints are accomodated.

## Template Tags

Template tags are case sensitive upper case identifiers.  The following tags are supported:

1. {{USER}} - the Github user name of the repository
2. {{NAME}} - the name of the repository
3. {{CONTENT}} - the contents of the markdown document when rendered
4. {{PAGE_CLASS}} - a css class representation of the current url
5. {{REF}} - the current repository reference, usually master
5. {{DOC}} - the name of the markdown file being viewed.

Here's a sample Bootstrap & Angularjs based template:

```html
<!doctype html>
<html lang="en" ng-app="myApp">
<head>
    <meta charset="utf-8">
    <title>{{NAME}} :: viewdocs.io</title>
    <link href="http://{{USER}}.github.io/{{NAME}}/docs/assets/bootstrap/bootstrap.min.css" rel="stylesheet" media="screen">
</head>
<body>
    <div ng-controller="NavbarCtrl">
        <navbar heading="Repo Human Name" name={{NAME}} user={{USER}} />
    </div>
    <div class="container">
        <div class="col-md-12 main-content">
            <section id="global">
              <div class="section">
                  {{CONTENT}}
              </div>
            </section>
        </div>
    </div>
    <script src="http://{{USER}}.github.io/{{NAME}}/docs/assets/js/jquery-1.10.1.min.js"></script>
    <script src="http://{{USER}}.github.io/{{NAME}}/docs/assets/js/bootstrap.min.js"></script>
    <script src="https://ajax.googleapis.com/ajax/libs/angularjs/1.2.0/angular.min.js"></script>
    <script src="https://ajax.googleapis.com/ajax/libs/angularjs/1.2.0/angular-route.js"></script>
    <script src="//cdnjs.cloudflare.com/ajax/libs/angular-ui-bootstrap/0.7.0/ui-bootstrap-tpls.js"></script>
    <script src="http://{{USER}}.github.io/{{NAME}}/docs/assets/js/app.js"></script>
</body>
</html>
```

## Static Assets

The viewdocs.io server only serves markdown files.  Static files such as javascript, image, css, etc. must be served from an alternate location.  Google and cloudflare serve many popular libraries, for example:

    <script src="https://ajax.googleapis.com/ajax/libs/angularjs/1.2.0/angular-route.js"></script>
    <script src="//cdnjs.cloudflare.com/ajax/libs/angular-ui-bootstrap/0.7.0/ui-bootstrap-tpls.js"></script>

Ideally, you'll want to serve assets from your Github repository. Github has instituted anti-hotlinking provision on their `raw.github.com` domain.  However, if you enable Github Pages for your repo Github will serve your files from their github.io domain.

A principal motivation for the development of viewdocs.io was to avoid having to maintain a separate gh-pages branch in order to serve a library documentation site.  It may seem contradictory to maintain a gh-pages branch to serve assets for a viewdocs.io site.  The following setup makes serving static assets from Github Pages painless.

### Painless Github Pages Setup

OK - it's a simple one-time setup, with all work local to your repository.  Don't follow
any other `gh-pages` tutorials, as we're not publishing a site, just serving our static
assets from our `/docs` directory.  (Repeat steps 2 & 3 on each workspace - i.e. If you work on your project at home & at work, repeat steps 2 & 3 each location.)

1. Start with an current clean repository, on the master branch.  It's easiest to work from
a new clone.  Create a `gh-pages` branch that is identical to your `master` branch:

        git clone git@github.com:your-user/your-repo.git
        cd your-repo
        git branch gh-pages master

2. Add a file, `.git/hooks/post-commit` with the following contents

        #!/bin/sh
        git branch -f gh-pages master

   make it executable

        chmod u+x .git/hooks/post-commit

   This `post-commit` hook automatically forces the gh-pages branch to mirror the master branch.

3. Edit your `.git/config` file so that the `[remote "origin"]` section looks like this:

        [remote "origin"]
    	    url = git@github.com:your-user/your-repo.git
    	    fetch = +refs/heads/*:refs/remotes/origin/*
            push = refs/heads/master:refs/heads/master
            push = refs/heads/gh-pages:refs/heads/gh-pages

    now when you `git push origin` you automatically push both `master` and `gh-pages` branches

4. Tell Github that you are not using Jekyll

        touch .nojekyll

5. Check in your changes and push to origin, adding the gh-pages branch to Github:

        git add -A .
        git commit -m "now serving assets from github.io"
        git push origin

   Wait ten minutes for Github to notice your new gh-pages branch and start serving your files.  Now, in your templates you can reference:

        http://{{USER}}.github.io/{{NAME}}/docs/assets/css/demo.css

   assuming you created a file `demo.css` and put it at `assets/css` relative to your docs folder.  Note that you may experience up to a 10 minute delay before change pushed to Github are reflected at the github.io domain.



## JSON, HTLM Partials and Ajax

Since static assets are being served via github.io, and your primary domain is viewdocs.io, ajax requests will be considered 'cross-domain.'  Unfortunately, Github Pages doesn't support CORS.  Here are two methods that can be used to work around that limitation.

### HTML Partials

HTML partials for Angularjs templates can be embedded in your template.html, for example, the following embedded template:

    <script type="text/ng-template" id="navbar.html">
    <div xmlns="http://www.w3.org/1999/html">
          <div class="navbar navbar-default navbar-fixed-top" role="navigation">
              <div class="navbar-header">
                  <button type="button" class="navbar-toggle" data-toggle="collapse" data-target=".navbar-ex5-collapse">
                      <span class="sr-only">Toggle navigation</span>
                      <span class="icon-bar"></span>
                      <span class="icon-bar"></span>
                      <span class="icon-bar"></span>
                  </button>
                  <a class="navbar-brand" href="/{{NAME}}">{{heading}}</a>
              </div>

              <div class="collapse navbar-collapse navbar-ex5-collapse" id="navbar-main">
                  <ul class="nav navbar-nav">
                      <li ng-repeat="item in items" ng-class="{active: item.selected}">
                          <a href="/{{NAME}}/{{item.link}}">{{item.title}}</a>
                      </li>
                  </ul>

                  <ul class="nav navbar-nav navbar-right">
                      <li ng-class="{active: item.selected}">
                          <a href="http://github.com/{{USER}}/{{NAME}}">On Github</a>
                      </li>
                  </ul>

              </div>
          </div>
    </div>
    </script>

can be referenced in javascript as if it were remote:

    directive('navbar', ['$location', '$http',  function ($location, $http) {
        return {
            restrict: 'E',
            transclude: true,
            scope: { heading: '@'},
            controller: 'NavbarCtrl',
            templateUrl: 'navbar.html',
            replace: true,
            ...

### JSON

JSON can be embedded in a markdown file and then parsed out of the rendered page.  Given the following markdown content in a file nav.md:

    <div>
        [
            {"title": "Getting Started", "link": "getting-started"},
            {"title": "API", "link": "api"},
            {"title": "Validations", "link": "validations"},
            {"title": "Query Language", "link": "gql"},
            {"title": "Examples", "link": "examples"},
            {"title": "Annotated Code", "link": "annotated"}
        ]
    </div>


The following example javascript function will request and extract it.

    var itemsXpath = '//*[@id="global"]/div/div';
    var itemsUrl = 'http://'+ $scope.user + '.viewdocs.io/' + $scope.name + '/nav';
    $http.get(itemsUrl).success(function(data) {
        var parser = new DOMParser();
        var doc = parser.parseFromString(data, "text/html");

        $scope.items = angular.fromJson(getElementByXpath(doc,itemsXpath).innerText);
        navbarCtrl.selectByUrl($location.absUrl());
    });


Note:
 - The Github Markdown API strips id and class attributes from html tags.
 - Not all browsers' DOMParser implementations support 'text/html' format.  A shim can be found here: https://gist.github.com/1129031


