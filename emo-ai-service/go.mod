module emo-ai-service

go 1.25.7

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/wire v0.6.0
	github.com/redis/go-redis/v9 v9.21.0
	go.uber.org/automaxprocs v1.6.0
	golang.org/x/crypto v0.53.0
	google.golang.org/genproto/googleapis/api v0.0.0-20260519071638-aa98bba5eb94
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.11
	gorm.io/driver/mysql v1.6.0
	gorm.io/gorm v1.31.2
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)

require (
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-kratos/kratos/contrib/otel/v3 v3.0.0-20260515082355-1ddb58e407c5
	github.com/go-kratos/kratos/v3 v3.0.0
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/form/v4 v4.3.0 // indirect
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260511170946-3700d4141b60 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-kratos/kratos/v3 v3.0.0 => github.com/go-kratos/kratos/v3 v3.0.0-20260515082355-1ddb58e407c5
