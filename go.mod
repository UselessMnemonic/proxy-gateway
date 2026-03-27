module proxy-gateway

go 1.25.0

require github.com/alecthomas/kingpin/v2 v2.4.0

require github.com/google/nftables v0.3.0

require proxy-gateway/api v0.0.0

require github.com/ti-mo/conntrack v0.6.0

require golang.org/x/sys v0.42.0

replace proxy-gateway/api => ./api

require (
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/mdlayher/netlink v1.9.0 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/ti-mo/netfilter v0.5.3 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
