package proxy

/*
HTTPErrs maps HTTP error codes to error pages.
*/
var HTTPErrs = map[int]string{
	502: `<!DOCTYPE html>
<html>
	<head>
		<title>Bad Gateway</title>
		<style>
			body {
				font-family: courier;
				margin: 5em 25%%;
				background-color: #f1f6f8;
			}
		</style>
	</head>
	<body>
		<h1>502 Bad Gateway</h1>
		<p>
			The requested service could not be reached<br>
		</p>
		<p style="font-size: 0.8em">
			<a href="https://github.com/mkenney/k8s-proxy/" target="_blank">k8s-proxy</a>: No %s proxy exists for service '%s'.
		</p>
	</body>
</html>`,

	503: `<!DOCTYPE html>
<html>
	<head>
		<title>Service Unavailable</title>
		<style>
			body {
				font-family: courier;
				margin: 5em 25%%;
				background-color: #f1f6f8;
			}
		</style>
	</head>
	<body>
		<h1>503 Service Unavailable</h1>
		<p>
			The requested service is currently unavailable.
		</p>
		<p>
			<a href="https://github.com/mkenney/k8s-proxy/" target="_blank">k8s-proxy</a>
		</p>
	</body>
</html>`,
}
