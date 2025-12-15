package user

import "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"

func InitUserSVC(user user.MongoMapper) {
	UserSVC = &UserService{
		UserMapper: user,
	}
}
