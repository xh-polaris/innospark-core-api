package conf

type AgentPrompts struct {
	ExtractInfo   bool
	ExtractPrompt string
	Template      string
	Key           []string
}

// CoTea 相关配置
type CoTea struct {
	AgentPrompts map[string]*AgentPrompts
}
