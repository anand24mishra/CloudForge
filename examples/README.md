# Examples

The Node and Python examples contain Dockerfiles so the Docker path is easy to run locally. The Go example intentionally has no Dockerfile: it is used to demonstrate the Cloud Native Buildpacks fallback.

Run detection from the repository root:

```sh
./bin/cloudforge detect examples/node-app
./bin/cloudforge detect examples/python-app
./bin/cloudforge detect examples/go-app
```
