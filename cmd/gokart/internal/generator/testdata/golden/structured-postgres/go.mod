module github.com/example/demo

go GOVERSION

require (
	github.com/alecthomas/kong v1.15.0
	github.com/spf13/viper v1.21.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/pressly/goose/v3 v3.27.2
)

tool github.com/pressly/goose/v3/cmd/goose
