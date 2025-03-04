module libdb.so/dnsmill

go 1.21.0

toolchain go1.21.7

// https://github.com/libdns/cloudflare/pull/14
replace github.com/libdns/cloudflare => github.com/diamondburned/libdns-cloudflare v0.1.4-0.20250304082825-4f76fad2b46b

require (
	github.com/hexops/autogold/v2 v2.2.1
	github.com/invopop/yaml v0.3.1
	github.com/libdns/cloudflare v0.1.0
	github.com/libdns/libdns v0.2.3
	github.com/libdns/namecheap v0.0.0-20211109042440-fc7440785c8e
	github.com/libdns/netlify v1.1.0
	github.com/libdns/porkbun v0.2.0
	github.com/libdns/vercel v0.0.2
	github.com/lmittmann/tint v1.0.4
	github.com/mattn/go-isatty v0.0.19
	github.com/spf13/pflag v1.0.5
)

require (
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/strfmt v0.23.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hexops/gotextdiff v1.0.3 // indirect
	github.com/hexops/valast v1.4.4 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/netlify/open-api/v2 v2.36.0 // indirect
	github.com/nightlyone/lockfile v1.0.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	go.mongodb.org/mongo-driver v1.17.3 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/tools v0.12.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	mvdan.cc/gofumpt v0.5.0 // indirect
)
