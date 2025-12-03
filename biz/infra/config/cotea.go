package config

type AgentPrompts struct {
	Template string
	Key      []string
}

// CoTea 相关配置
type CoTea struct {
	AgentPrompts map[string]*AgentPrompts
}
