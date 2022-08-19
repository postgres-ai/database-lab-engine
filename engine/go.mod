module gitlab.com/postgres-ai/database-lab/v3

go 1.18

require (
	github.com/AlekSi/pointer v1.1.0
	github.com/ahmetalpbalkan/dlog v0.0.0-20170105205344-4fb5f8204f26
	github.com/araddon/dateparse v0.0.0-20210207001429-0eec95c9db7e
	github.com/aws/aws-sdk-go v1.33.8
	github.com/docker/cli v20.10.13+incompatible
	github.com/docker/docker v20.10.13+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/dustin/go-humanize v1.0.0
	github.com/golang-jwt/jwt/v4 v4.4.2
	github.com/google/go-github/v34 v34.0.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/jackc/pgtype v1.5.0
	github.com/jackc/pgx/v4 v4.9.0
	github.com/lib/pq v1.8.0
	github.com/pkg/errors v0.9.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/xid v1.2.1
	github.com/sergi/go-diff v1.1.0
	github.com/sethvargo/go-password v0.2.0
	github.com/shirou/gopsutil v2.20.9+incompatible
	github.com/stretchr/testify v1.7.0
	github.com/testcontainers/testcontainers-go v0.12.0
	github.com/urfave/cli/v2 v2.1.1
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/mod v0.5.1
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/containerd/containerd v1.6.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.8.0+incompatible // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.7.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.0.5 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jmespath/go-jmespath v0.3.0 // indirect
	github.com/klauspost/compress v1.11.13 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/moby/sys/mountinfo v0.5.0 // indirect
	github.com/moby/term v0.0.0-20210610120745-9d4ed1856297 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/opencontainers/runc v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/grpc v1.43.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)

// Include the single version of the dependency to clean up go.sum from old revisions.
// Since old and indirect dependencies are listed in the sum file and the vulnerability scanner flags the project as containing vulnerabilities.
replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.6.1 // mitigate CVE-2021-32760, CVE-2020-15257, CVE-2022-23648
	github.com/coreos/etcd => github.com/coreos/etcd v3.3.27+incompatible // mitigate CVE-2020-15113 and CVE-2020-15112
	github.com/docker/cli => github.com/docker/cli v20.10.3-0.20220214172424-cf8c4bab6477+incompatible // mitigate CVE-2018-20699
	github.com/docker/distribution v2.8.0+incompatible => github.com/agneum/distribution v2.8.1-0.20220215080619-a3a6b67e8f8d+incompatible
	github.com/docker/docker => github.com/docker/docker v20.10.3-0.20220207145910-4b3471ddc064+incompatible // mitigate CVE-2018-20699
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2 // mitigate CVE-2021-3121
	github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2 // mitigate CVE-2021-41190
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.3 // mitigate CVE-2021-30465
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.11.1 // mitigate CVE-2022-21698
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20220214200702-86341886e292 // mitigate CVE-2021-43565, CVE-2020-29652, and CVE-2018-16875
	k8s.io/kubernetes v1.13.0 => k8s.io/kubernetes v1.23.3 // mitigate CVE-2020-8559 and CVE-2020-8565
)
