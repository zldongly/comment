package data

import (
	"github.com/zldongly/comment/app/comment/service/internal/conf"

	"github.com/Shopify/sarama"
	"github.com/coocood/freecache"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/gomodule/redigo/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	_ "github.com/go-sql-driver/mysql"
)

// ProviderSet is data providers.
//var ProviderSet = wire.NewSet(NewData, NewDB, NewOrderRepo)

// Data .
type Data struct {
	db    *gorm.DB
	redis *redis.Pool
	cache *freecache.Cache
	kafka sarama.SyncProducer
	log   *log.Helper
}

func NewDB(conf *conf.Data, logger log.Logger) *gorm.DB {
	log := log.NewHelper(log.With(logger, "module", "comment-service/data/gorm"))

	db, err := gorm.Open(mysql.Open(conf.Database.Source), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed opening connection to mysql: %v", err)
	}

	err = db.AutoMigrate(
		&CommentSubject{},
		&CommentIndex{},
		&CommentContent{})
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func NewRedis(conf *conf.Data, logger log.Logger) {

	log := log.NewHelper(log.With(logger, "module", "comment-service/data/redis"))
	_ = log
	// TODO: 2021/8/15
}

func NewKafka(conf *conf.Data, logger log.Logger) {
	log := log.NewHelper(log.With(logger, "module", "comment-service/data/kafka"))
	_ = log
	//sarama.NewSyncProducer()
}

// NewData .
func NewData(db *gorm.DB, logger log.Logger) (*Data, func(), error) {
	log := log.NewHelper(log.With(logger, "module", "comment-service/data"))

	d := &Data{
		db:    db,
		log:   log,
		cache: freecache.NewCache(128 * 1024 * 1024),
	}
	return d, func() {

	}, nil
}
