module github.com/cloudfoundry/conda-cnb

go 1.12

require (
	github.com/buildpack/libbuildpack v1.12.0
	github.com/burntsushi/toml v0.3.1 // indirect
	github.com/cloudfoundry/dagger v0.0.0-20190404183716-29cf9dd0dbf3
	github.com/cloudfoundry/libcfbuildpack v1.48.0
	github.com/golang/mock v1.2.0
	github.com/google/go-cmp v0.2.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1
	github.com/sclevine/spec v1.2.0
	golang.org/x/crypto v0.0.0-20190411191339-88737f569e3a // indirect
	golang.org/x/net v0.0.0-20190404232315-eb5bcb51f2a3 // indirect
	golang.org/x/sys v0.0.0-20190411185658-b44545bcd369 // indirect
	golang.org/x/text v0.3.1-0.20180807135948-17ff2d5776d2 // indirect
	golang.org/x/tools v0.0.0-20190411180116-681f9ce8ac52 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/cloudfoundry/dagger => /Users/pivotal/workspace/dagger
