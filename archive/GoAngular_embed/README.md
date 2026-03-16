# AngularFront
## Summary
This is a Single Page Web-App in Angular, embedded as a static file into Go, then served using gin-gonic. Routing falls back onto a new embedded file system to handle browser refresh.

## Running the Program
The Angular must be built first into a static file with 'ng build'.

Then build and run the Go, which serves it on localhost:5000

    $ cd client && ng build && go build && go run ../.

