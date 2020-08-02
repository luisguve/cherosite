## Cheropatilla http server

This project contains the implementation of the http API for the Cheropatilla website.

#### Installation

1. run `go install github.com/luisguve/cherosite` and `go install github.com/luisguve/cheroapi`. The required executable files will be installed in $GOBIN.
1. There will be a cherosite.toml file in the project root, containing most of the required configuration variables. Read it and set the locations of the .env files.
1. cookie_hash.env must contain two key/value pairs: SESSION_KEY, a value generated `func GenerateRandomKey(length int) []byte` by [gorilla/securecookie](https://github.com/gorilla/securecookie) and SESS_DIR, the path of a directory where all the sessions will be stored.
1. Follow the installation instructions in the [cheroapi project](https://github.com/luisguve/cheroapi#Installation).

To run the server, run both cherosite.exe and cheroapi.exe, then visit localhost:8000 from your browser.
