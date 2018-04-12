package proxy

import (
	"html/template"

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

	HTTPErrs[502], err = template.New("502").Parse(`<!DOCTYPE html>
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
		</style>
	</head>
	<body>
		<h1>502 Bad Gateway</h1>
		<p>
			The requested service could not be reached<br>
		</p>
		<p class="sub">
			<a href="https://github.com/mkenney/k8s-proxy/" target="_blank">k8s-proxy</a>: No {{.Scheme}} service could be matched to host '{{.Host}}'.
		</p>
		<p class="sub">
			Routable services:
			<ul>
				{{range .Services}}<li>{{ .Name }}</li>{{end}}
			</ul>
		</p>
	</body>
</html>`)
	check(err)

	HTTPErrs[503], err = template.New("503").Parse(`<!DOCTYPE html>
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
		<h1>503 Service Unavailable</h1>
		<p>
			The proxy service is currently unavailable.
		</p>
		<p class="sub">
			<a href="https://github.com/mkenney/k8s-proxy/" target="_blank">k8s-proxy</a>: {{.Reason}} {{.Host}}
		</p>
	</body>
</html>`)
	check(err)
}
