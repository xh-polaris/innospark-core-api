package util

import (
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/basic"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func BuildFindOption(p *basic.Page) (opts *options.FindOptionsBuilder) {
	opts = options.Find()

	var page, size int64 = 1, 10
	if p != nil {
		if p.Page != nil {
			page = *p.Page
		}
		if p.Size != nil {
			size = *p.Size
		}
	}
	opts.SetSkip((page - 1) * size)
	return
}

func ObjectIDsFromHex(ids ...string) ([]primitive.ObjectID, error) {
	var objectIDs []primitive.ObjectID
	for _, id := range ids {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return nil, err
		}
		objectIDs = append(objectIDs, oid)
	}
	return objectIDs, nil
}

func HasMore(total int64, page *basic.Page) bool {
	return total > page.GetPage()*page.GetSize()
}

func SplitAndHasMore[T any](slice []T, page *basic.Page) (ans []T, hasMore bool) {
	size, length := page.GetSize(), int64(len(slice))
	hasMore = length > size
	if size > length {
		ans = slice[:length]
	} else {
		ans = slice[:size]
	}
	return
}
