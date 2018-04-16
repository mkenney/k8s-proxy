# go-log-fmt

A simple https://github.com/sirupsen/logrus log formatter. Includes

* [iso 8601](https://en.wikipedia.org/wiki/ISO_8601) formatted date string - `"2006-01-02T15:04:05.000Z07:00"`
* OS hostname
* Log level
* Caller - "file:line func"
* Log message
* Additional data


## Usage

```go
import logfmt "github.com/mkenney/go-log-fmt"
```

#### Text format

Set the formatter:
```go
log.SetFormatter(&logfmt.TextFormat{})
```

Produces:
```js
time="2018-04-16T05:14:07.559Z" host="k8s-proxy-688fb8b57d-4rzt4" level="info" caller="proxy.go:252 github.com/mkenney/k8s-proxy/pkg/proxy.(*Proxy).Start" msg="starting kubernetes proxy" port="80"
```

#### JSON format

Set the formatter
```go
log.SetFormatter(&logfmt.JSONFormat{})
```

Produces:
```json
{"time":"2018-04-16T06:23:37.133Z","level":"info","host":"k8s-proxy-7b77bfd8bd-7xcvn","caller":"proxy.go:258 github.com/mkenney/k8s-proxy/pkg/proxy.(*Proxy).Start","msg":"starting kubernetes proxy","data":[{"Key":"port","Msg":"80"}]}
```
