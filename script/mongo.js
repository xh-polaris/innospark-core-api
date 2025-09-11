// 为 Conversation 搜索创建检索 brief 的 text 索引
db.conversation.createIndex({ brief: "text" })

// 查看索引
db.conversation.getIndexes()