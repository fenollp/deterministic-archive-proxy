# deterministic-archive-proxy
Always get the same archive from GitHub (and others).

Save cache misses that are solely due to
* filenames being in a different order
* timestamps of identical files being different
* users, groups or permissions changing

## Usage

Start the proxy
```shell
go build cmd/proxy/proxy.go && PORT=8888 ./proxy
```

Update your `http_archive`s
```python
http_archive(
    name = "bazel_skylib",
    type = "tar.gz",
    url = "https://localhost:8080/github.com/bazelbuild/bazel-skylib/archive/1.0.3.tar.gz",
    sha256 = "868350c41188cda1b4923c0df62b06d7dbd977527b576eb1d2515da631709c40",
)
```
