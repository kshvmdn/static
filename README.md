## static

> Host static content from Dropbox or Google Drive locally.

Static is a command line tool to serve static files from Dropbox or Google Drive.

### Design

Static indefinitely caches each file and is [constantly polling](https://www.pubnub.com/blog/2014-12-01-http-long-polling/) each root directory to detect for file changes.

The program takes a path (or multiple) to a single directory and serves each file from that directory.

Example:

  - Assume you have a `/site` folder on your Dropbox account with the following contents: `index.html`, `style.css`, `image.png`.

  - You run the following to start Static:

    ```sh
    $ static -service=dropbox -path=/site -endpoint=/
    Listening on port 3030.
    ```

  - You can now access each of the files at: [`http://localhost:3030/index.html`](http://localhost:3030/index.html), [`.../style.css`](http://localhost:3030/style.css), or [`.../image.png`](http://localhost:3030/image.png).

  - Relative references also work as expected. So if `index.html` imported from `style.css`, the CSS would appear as you'd expect.

  - In this example, we're constantly pinging the Dropbox server to detect for changes in the `/site` directory.

### Installation / setup

  - Go should be [installed](https://golang.org/doc/install) and [configured](https://golang.org/doc/install#testing).

  - Install with Go:

    ```sh
    $ go get -v github.com/kshvmdn/static
    $ static # binary file is located in $GOPATH/bin
    ```

  - Or, install directly via source:

    ```sh
    $ git clone https://github.com/kshvmdn/static.git $GOPATH/src/github.com/kshvmdn/static
    $ cd $_ # $GOPATH/src/github.com/kshvmdn/static
    $ make install && make
    $ ./static
    ```

### Usage

  - View the help dialogue by running the program with `-help`.

    ```
    $ static -help
    Usage of ./static:
      -config string
          Optional config. file for multiple paths & endpoints.
      -endpoint string
          Endpoint to serve content at. (default "/")
      -path string
          Dropbox path to serve content from.
      -port int
          Server port. (default 3030)
      -service string
          Service to be used ("dropbox" or "drive").
      -sleep int
          Time to wait between polls (in seconds).
    ```

  - The program takes one (and only one) of the following:

    + Command line flags for both `-path` and `-endpoint` to define where to find the content (path) and which endpoint to serve them at (endpoint).

    + A JSON or YAML config. file with one or multiple paths and endpoints.

      * Example JSON file:

        ```json
        [
          { "path": "/personal-site", "endpoint": "/" },
          { "path": "/test", "endpoint": "test" }
        ]
        ```

      * And the same contents in a YAML file:

        ```yaml
        - path: /personal-site
          endpoint: /
        - path: /test
          endpoint: test
        ```

#### Dropbox

  - Head over to [this](https://www.dropbox.com/developers/apps) page and create a new Dropbox app with the following settings:

    1. API: Dropbox API
    2. Access: Full Dropbox
    3. Name: anything

  - You'll need to export the **app key**, **app secret**, and **access token** as environment variables.

    ```sh
    DROPBOX_APP_KEY=<app key>
    DROPBOX_APP_SECRET=<app secret>
    DROPBOX_ACCESS_TOKEN=<access token>
    ```

#### Google Drive

_Not yet implemented._

### Contribute

This project is completely open source, feel free to [open an issue](https://github.com/kshvmdn/static/issues) for questions/features/bugs or [submit a pull request](https://github.com/kshvmdn/static/pulls).

Prior to submitting work, please ensure your changes comply with [Golint](https://github.com/golang/lint). You can use `make lint` to test this.

#### TODO

  - [ ] Add Google Drive integration, refer to [this](https://developers.google.com/drive/v3/web/quickstart/go) to get started. Requires OAuth2, might be worth it to use OAuth2 for Dropbox as well to be consistent.
  - [ ] Rework the long poll model. Currently polls each directory path (not file!), which means **every** file in a given path polls the same root directory. This can be fixed by starting the polling when a file is first accessed (since that's when the associated key is created). We'll just need to track which keys are being polled, which shouldn't be a problem.
