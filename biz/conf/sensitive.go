package conf

type Sensitive struct {
	Sensitive          []string
	SensitiveStreamGap int
	Pre                bool // 用户输入检测
	Post               bool // 模型输出检测
}
