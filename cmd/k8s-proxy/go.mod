module github.com/mkenney/k8s-proxy/cmd/k8s-proxy

require (
	github.com/bdlm/log v0.1.13
	github.com/mkenney/k8s-proxy v0.0.0
	github.com/pkg/errors v0.8.0 // indirect
	golang.org/x/crypto v0.0.0-20181012144002-a92615f3c490 // indirect
	golang.org/x/net v0.0.0-20181011144130-49bb7cea24b1 // indirect
	golang.org/x/sys v0.0.0-20181011152604-fa43e7bc11ba // indirect
	k8s.io/api v0.0.0-20181005203742-357ec6384fa7 // indirect
)

replace github.com/mkenney/k8s-proxy => ../..
