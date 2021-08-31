package data

import (
	"github.com/zldongly/comment/app/comment/job/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gomodule/redigo/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	_ "github.com/go-sql-driver/mysql"
)

// ProviderSet is data providers.
//var ProviderSet = wire.NewSet(NewData, NewDB, NewRedis, NewSubjectRepo, NewCommentRepo)

// Data .
type Data struct {
	db    *gorm.DB
	redis *redis.Pool
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

func NewRedis(conf *conf.Data, logger log.Logger) *redis.Pool {

	log := log.NewHelper(log.With(logger, "module", "comment-service/data/redis"))
	_ = log
	// TODO: 2021/8/15
	return nil
}

// NewData .
func NewData(db *gorm.DB, redis *redis.Pool, logger log.Logger) (*Data, func(), error) {
	log := log.NewHelper(log.With(logger, "module", "comment-service/data"))

	d := &Data{
		db:    db,
		redis: redis,
		log:   log,
	}

	return d, func() {
		d.redis.Close()
	}, nil
}
