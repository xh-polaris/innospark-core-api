package ac

import (
	"bytes"
	"strings"

	ahocorasick "github.com/anknown/ahocorasick"
)

var m *ahocorasick.Machine

// readRunes 将字符串字典转换为rune切片数组, 用于Aho-Corasick算法的输入格式要求
func readRunes(dict []string) (runes [][]rune) {
	for _, word := range dict {
		word = strings.ToLower(word)          // 转换为小写，实现大小写不敏感匹配
		l := bytes.TrimSpace([]byte(word))    // 去除前后空白字符
		runes = append(runes, bytes.Runes(l)) // 将字符串转换为rune切片，支持中文等多字节字符
	}
	return runes
}

// InitAc 根据关键词字典初始化Aho-Corasick自动机
func InitAc(dict []string) error {
	m = new(ahocorasick.Machine)
	runes := readRunes(dict)               // 将字符串字典转换为rune格式
	if err := m.Build(runes); err != nil { // 构建AC自动机的Trie树结构
		return err
	}
	return nil
}

// AcSearch 使用Aho-Corasick算法进行多模式串搜索
// 参数: findText: 待搜索的文本内容  dict: 关键词字典列表 stopImmediately: 是否找到第一个匹配就停止搜索
// 返回值: bool: 是否找到匹配的关键词 []string: 匹配到的关键词列表
func AcSearch(findText string, dict []string, stopImmediately bool) (bool, []string) {
	// 空字典或空文本的边界情况处理
	if len(dict) == 0 || len(findText) == 0 {
		return false, nil
	}

	// 执行多模式串搜索
	hits := m.MultiPatternSearch([]rune(findText), stopImmediately)
	// 处理搜索结果
	if len(hits) > 0 {
		words := make([]string, 0)
		for _, hit := range hits {
			words = append(words, string(hit.Word)) // 将匹配到的rune切片转换回字符串
		}
		return true, words
	}
	return false, nil
}
