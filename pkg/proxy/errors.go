package proxy

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/bdlm/log"
)

/*
HTTPErrs maps HTTP error codes to HTNL error pages.
*/
var HTTPErrs = map[int]*template.Template{}

func init() {
	var err error

	check := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
	}

	// Bad Gateway should be displayed when a request does not map to
	// any known service.
	statusText := http.StatusText(http.StatusBadGateway)
	HTTPErrs[http.StatusBadGateway], err = template.
		New(statusText).
		Funcs(funcMap).
		Parse(`<!DOCTYPE html>
<html>
	<head>
		<title>` + statusText + `</title>
		<style>
			body {
				font-family: courier;
				margin: 5em 25%;
				background-color: #f1f6f8;
			}
			p.sub, ul {
				font-size: 0.8em;
			}
			.arr {
				font-family: Helvetica, Arial, sans-serif;
			}
		</style>
	</head>
	<body>
		<h1>⎈ 502 Bad Gateway</h1>
		<p>
			The requested service could not be found.
		</p>
		<p class="sub">
			No routable services match the request '{{.Scheme|ToLower}}://{{.Host}}'.
		</p>
		<p class="sub">
			Available routes:
			<ul>
				{{$scheme := .Scheme}}
				{{range $k, $v := .Services}}<li>{{$scheme|ToLower}}://{{ $k }}.* <span class="arr">&rarr;</span> {{$v.Name}}</li>{{end}}
			</ul>
		</p>
		<p class="sub">
			<a href="https://github.com/mkenney/k8s-proxy/" target="_blank">k8s-proxy</a>
		</p>
	</body>
</html>`)
	check(err)

	// Service Unavailable should be displayed when a request is made
	// before the service becomes ready, or when an attempt to direct
	// a request to a known service that is malfunctioning in some way
	// and times out.
	statusText = http.StatusText(http.StatusServiceUnavailable)
	HTTPErrs[http.StatusServiceUnavailable], err = template.
		New(statusText).
		Funcs(funcMap).
		Parse(`<!DOCTYPE html>
<html>
	<head>
		<title>` + statusText + `</title>
		<style>
			body {
				font-family: courier;
				margin: 5em 25%;
				background-color: #f1f6f8;
			}
			p.sub, ul {
				font-size: 0.8em;
			}
		</style>
	</head>
	<body>
		<h1>⎈ 503 Service Unavailable</h1>
		<p>
			The requested service did not respond.
		</p>
		<p class="sub">
			Received <b>{{.Reason}}</b> from <i>{{.Host}}</i>. {{.Msg}}
		</p>
		<p class="sub">
			<a href="https://github.com/mkenney/k8s-proxy/" target="_blank">k8s-proxy</a>
		</p>
	</body>
</html>`)
	check(err)
}
