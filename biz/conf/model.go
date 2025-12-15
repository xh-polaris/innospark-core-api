package conf

// InnoSpark 启创配置
type InnoSpark struct {
	DefaultBaseURL   string
	DefaultAPIKey    string
	DeepThinkBaseURL string
	DeepThinkAPIKey  string
	VlmURL           string
	VlmAPIKey        string
}

// Bocha 博查搜索API
type Bocha struct {
	APIKey   string
	Template string
}

// ARK 火山配置
type ARK struct {
	APIKey          string
	CodeGenTemplate string
}

// Claude 配置
type Claude struct {
	BaseURL string
	APIKey  string
}
