## Cheropatilla http server

This repo contains the implementation of the http server for the Cheropatilla website.

#### Installation

1. run `go get github.com/luisguve/cherosite` and `go get github.com/luisguve/cheroapi`. Then run `go install github.com/luisguve/cherosite/cmd/cherosite` and `go install github.com/luisguve/cheroapi/cmd/...`. The required binaries will be installed in your $GOBIN.
1. You will need to write a .toml file in order to set the configuration for the site to  in the project root, containing most of the required configuration variables. Read it and set the locations of the .env files.
1. cookie_hash.env must contain two key/value pairs: SESSION_KEY, a value generated `func GenerateRandomKey(length int) []byte` by [gorilla/securecookie](https://github.com/gorilla/securecookie) and SESS_DIR, the path of a directory where all the sessions will be stored.
1. Follow the installation instructions in the [cheroapi project](https://github.com/luisguve/cheroapi#Installation).

To run the server, run both cherosite.exe and cheroapi.exe, then visit localhost:8000 from your browser.
