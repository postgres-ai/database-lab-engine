module gitlab.com/postgres-ai/database-lab/v2

go 1.16

require (
	github.com/AlekSi/pointer v1.1.0
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/araddon/dateparse v0.0.0-20210207001429-0eec95c9db7e
	github.com/aws/aws-sdk-go v1.33.8
	github.com/containerd/containerd v1.5.2 // indirect
	github.com/docker/cli v0.0.0-20200721130541-80fd48bcb7e7
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/dustin/go-humanize v1.0.0
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/google/go-github/v34 v34.0.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/jackc/pgtype v1.5.0
	github.com/jackc/pgx/v4 v4.9.0
	github.com/lib/pq v1.8.0
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/xid v1.2.1
	github.com/sergi/go-diff v1.1.0
	github.com/sethvargo/go-password v0.2.0
	github.com/shirou/gopsutil v2.20.9+incompatible
	github.com/stretchr/testify v1.7.0
	github.com/testcontainers/testcontainers-go v0.8.0
	github.com/urfave/cli/v2 v2.1.1
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/grpc v1.38.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/docker/docker v1.13.1 => github.com/docker/engine v0.0.0-20200618181300-9dc6525e6118
