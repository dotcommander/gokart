module github.com/example/demo

go GOVERSION

require (
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.21.0
	github.com/jackc/pgx/v5 v5.8.0
	github.com/pressly/goose/v3 v3.27.0
)

tool github.com/pressly/goose/v3/cmd/goose
