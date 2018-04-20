package proxy

import (
	"html/template"
	"strings"

	log "github.com/sirupsen/logrus"
)

/*
HTTPErrs maps HTTP error codes to error pages.
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

	HTTPErrs[502], err = template.New("502").Funcs(funcMap).Parse(`<!DOCTYPE html>
<html>
	<head>
		<title>Bad Gateway</title>
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

	HTTPErrs[503], err = template.New("503").Funcs(funcMap).Parse(`<!DOCTYPE html>
<html>
	<head>
		<title>Service Unavailable</title>
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
