package system

import (
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
	"github.com/xh-polaris/innospark-core-api/biz/infra/storage"
)

func InitAttachSVC(cos storage.COS, user user.MongoMapper) {
	AttachSVC = &AttachService{
		Cos:        cos,
		UserMapper: user,
	}
}
