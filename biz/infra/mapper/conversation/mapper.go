package conversation

import (
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/zeromicro/go-zero/core/stores/monc"
)

var _ MongoMapper = (*mongoMapper)(nil)

const (
	collection     = "conversation"
	cacheKeyPrefix = "cache:conversation:"
)

type MongoMapper interface{}

type mongoMapper struct {
	conn *monc.Model
}

func NewConversationMongoMapper(config *config.Config) MongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, collection, config.Cache)
	return &mongoMapper{conn: conn}
}
