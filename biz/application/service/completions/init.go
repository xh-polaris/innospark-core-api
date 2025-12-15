package completions

import (
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/domain/graph"
	"github.com/xh-polaris/innospark-core-api/biz/domain/memory"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
)

func InitCompletionsSVC(memory *memory.MemoryManager) {
	CompletionsSVC = &CompletionsService{
		CompletionGraph: graph.DrawCompletionGraph(memory),
		UserMapper:      user.NewUserMongoMapper(conf.GetConfig()),
	}
}
