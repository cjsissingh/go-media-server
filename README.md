# GoLang Media Server
An experiment in learning GoLang. Port an implementation of realestate.co.nz's Media Server using the `bimg` library

# Running
Export the required Env Vars
- `SOURCE_BUCKETS`
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_SESSION_TOKEN`

Optionally get `ENABLE_WEBP` if you'd like to convert the images to webp format. // TODO - Check user-agent and conditionally convert to webp for supported browsers

Start the [Gin](https://github.com/gin-gonic/gin) webserver 

```
go run *.go
```
