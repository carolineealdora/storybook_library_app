package models

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"codebase/config"

	" logger/implementations/logrus"
	" logger/logger"
	" persistent/sql/implementations/gorm/postgres"
	" persistent/sql/sql"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

const (
	CommonTimeStampFormat = "2006-01-02 15:04:05.000Z07:00"
	CommonDateFormat      = "2006-01-02"
	DDMMYYFormat          = "020106"
	LogPattern            = "[%s] %v"
)

type (
	CommonDatabaseModel struct {
		ID        uint64    `json:"id" gorm:"column:id;primary_key"`
		CreatedAt time.Time `json:"created_at" gorm:"created_at"`
		UpdatedAt time.Time `json:"updated_at" gorm:"updated_at"`
	}

	SharedHolder struct {
		Config *config.SystemConfig
		Log    logger.Logger
		Echo   *echo.Echo
		Sql    sql.ORM
		Redis  *redis.Client
		Mongo  *mongo.Client
	}

	Model interface {
		TableName() string
		Validate() error
	}
)

func (h *SharedHolder) Init() {
	h.Config = config.LoadEnv()
	h.Log = initLog(*h.Config)
	h.Echo = initEcho(*h.Config)
	h.Sql = initSql(h.Log, *h.Config)
	h.Redis = initRedis(h.Log, *h.Config)
	h.Mongo = initMongo(h.Log, *h.Config)
}

func initLog(cfg config.SystemConfig) logger.Logger {
	option := logrus.Option{
		Level:       logrus.Level(cfg.LogLevel),
		LogFilePath: "logs/",
		Formatter:   logrus.JSONFormatter,
		MaxSize:     1,
		MaxBackups:  3,
		MaxAge:      -1,
		Compress:    false,
	}
	logs, _ := logrus.New(&option)
	return logs
}

func initEcho(cfg config.SystemConfig) *echo.Echo {
	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)
	if cfg.Env == "development" {
		e.Debug = true
	}
	return e
}

func initSql(log logger.Logger, sysConfig config.SystemConfig) sql.ORM {
	connection := sql.Connection{
		Host:     sysConfig.DBHost,
		Username: sysConfig.DBUsername,
		Password: sysConfig.DBPassword,
		DBName:   sysConfig.DBName,
		Port:     sysConfig.DBPort,
		SSLMode:  sysConfig.DBSSLMode,
		Timezone: sysConfig.DBTimezone,
		SQLDebug: sysConfig.DBDebugMode,
		ORMDebug: true,
	}

	sql, err := postgres.NewPostgresConnection(connection, log)
	if err != nil {
		log.Fatalf("failed to start connection to database : %v", connection)
		return nil
	}

	return sql
}

func SetRequestID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		UUID, _ := uuid.NewRandom()
		c.Set("RequestID", UUID.String())
		return next(c)
	}
}

type Caches struct {
	Redis *redis.Client
}

func initRedis(log logger.Logger, sysConfig config.SystemConfig) *redis.Client {
	redisCache := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", sysConfig.RedisHost, sysConfig.RedisPort),
		Password: sysConfig.RedisPassword,
		DB:       0,
	})

	log.Info(fmt.Sprintf("check redis connection %v", redisCache.Ping(context.Background())))
	return redisCache
}

func initMongo(log logger.Logger, sysConfig config.SystemConfig) *mongo.Client {
	opts := options.Client().ApplyURI(sysConfig.MongoUri)
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		log.Fatalf("failed to start connection to database : %v", err)
		return nil
	}
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		log.Fatalf("failed to start connection to database : %v", err)
		return nil
	}

	log.Info("mongo connection establish")
	return client
}
