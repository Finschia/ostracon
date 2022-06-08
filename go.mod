module github.com/line/ostracon

go 1.15

require (
	github.com/BurntSushi/toml v1.1.0
	github.com/ChainSafe/go-schnorrkel v0.0.0-20200405005733-88cbf1b4c40d
	github.com/Workiva/go-datastructures v1.0.52
	github.com/adlio/schema v1.3.0
	github.com/btcsuite/btcd v0.22.1
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce
	github.com/coniks-sys/coniks-go v0.0.0-20180722014011-11acf4819b71
	github.com/fortytw2/leaktest v1.3.0
	github.com/go-kit/kit v0.12.0
	github.com/go-kit/log v0.2.1
	github.com/go-logfmt/logfmt v0.5.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/google/orderedcode v0.0.1
	github.com/gorilla/websocket v1.5.0
	github.com/gtank/merlin v0.1.1
	github.com/hdevalence/ed25519consensus v0.0.0-20200813231810-1694d75e712a
	github.com/herumi/bls-eth-go-binary v0.0.0-20200923072303-32b29e5d8cbf
	github.com/lib/pq v1.10.6
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/line/tm-db/v2 v2.0.0-init.1.0.20220121012851-61d2bc1d9486
	github.com/minio/highwayhash v1.0.2
	github.com/ory/dockertest v3.3.5+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.2
	github.com/r2ishiguro/vrf v0.0.0-20180716233122-192de52975eb
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0
	github.com/rs/cors v1.8.2
	github.com/sasha-s/go-deadlock v0.2.1-0.20190427202633-1595213edefa
	github.com/snikch/goodman v0.0.0-20171125024755-10e37e294daa
	github.com/spf13/cobra v1.4.0
	github.com/spf13/viper v1.12.0
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.7.1
	github.com/tendermint/go-amino v0.16.0
	github.com/yahoo/coname v0.0.0-20170609175141-84592ddf8673 // indirect
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4
	golang.org/x/net v0.0.0-20220520000938-2e3eb7b945c2
	gonum.org/v1/gonum v0.11.0
	google.golang.org/grpc v1.46.2
	gopkg.in/yaml.v3 v3.0.1
)
// `runc` is referenced by `github.com/adlio/schema`.
// This is a temporary fix for a security vulnerability of `runc`.
// So, remove this `replace` when `github.com/adlio/schema` releases a version that references `runc 1.1.2`.
// For details, see here https://nvd.nist.gov/vuln/detail/CVE-2022-29162.
replace github.com/opencontainers/runc => github.com/opencontainers/runc v1.1.2
