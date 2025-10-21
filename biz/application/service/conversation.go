package service

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	dm "github.com/xh-polaris/innospark-core-api/biz/domain/msg"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/conversation"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/errorx"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
)

type IConversationService interface {
	CreateConversation(ctx context.Context, req *core_api.CreateConversationReq) (*core_api.CreateConversationResp, error)
	GenerateBrief(ctx context.Context, req *core_api.GenerateBriefReq) (*core_api.GenerateBriefResp, error)
	RenameConversation(ctx context.Context, req *core_api.RenameConversationReq) (*core_api.RenameConversationResp, error)
	ListConversation(ctx context.Context, req *core_api.ListConversationReq) (*core_api.ListConversationResp, error)
	GetConversation(ctx context.Context, req *core_api.GetConversationReq) (*core_api.GetConversationResp, error)
	DeleteConversation(ctx context.Context, req *core_api.DeleteConversationReq) (*core_api.DeleteConversationResp, error)
	SearchConversation(ctx context.Context, req *core_api.SearchConversationReq) (*core_api.SearchConversationResp, error)
}

type ConversationService struct {
	ConversationMapper conversation.MongoMapper
	MessageMapper      mmsg.MongoMapper
}

var ConversationServiceSet = wire.NewSet(
	wire.Struct(new(ConversationService), "*"),
	wire.Bind(new(IConversationService), new(*ConversationService)),
)

func (s *ConversationService) CreateConversation(ctx context.Context, req *core_api.CreateConversationReq) (*core_api.CreateConversationResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logx.Error("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.UnAuthErrCode)
	}

	// 调用mapper创建对话
	newConversation, err := s.ConversationMapper.CreateNewConversation(ctx, uid, req.BotId)
	if err != nil {
		logx.Error("create conversation error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.ConversationCreateErrCode)
	}

	// 返回conversationID
	return &core_api.CreateConversationResp{Resp: util.Success(), ConversationId: newConversation.ConversationId.Hex()}, nil
}

func (s *ConversationService) GenerateBrief(ctx context.Context, req *core_api.GenerateBriefReq) (*core_api.GenerateBriefResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logx.Error("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.UnAuthErrCode)
	}
	// 生成标题
	m, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		BaseURL: config.GetConfig().InnoSpark.DefaultBaseURL,
		APIKey:  config.GetConfig().InnoSpark.DefaultAPIKey,
		Model:   "InnoSpark",
	})
	//m, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
	//	BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
	//	Region:  "cn-beijing",
	//	APIKey:  config.GetConfig().ARK.APIKey,
	//	Model:   "doubao-1-5-pro-32k-250115",
	//})
	if err != nil {
		return nil, errorx.WrapByCode(err, cst.ConversationGenerateBriefErrCode)
	}
	in := []*schema.Message{schema.UserMessage("你是标题生成器, 不要回答, 而是根据用户输入概括[" + req.Messages[0].Content + "],不超过10个字, 简洁正式, 无额外内容")}
	out, err := m.Generate(ctx, in)
	if err != nil {
		return nil, errorx.WrapByCode(err, cst.ConversationGenerateBriefErrCode)
	}
	// 更新标题
	if err = s.ConversationMapper.UpdateConversationBrief(ctx, uid, req.ConversationId, out.Content); err != nil {
		return nil, errorx.WrapByCode(err, cst.ConversationGenerateBriefErrCode)
	}
	return &core_api.GenerateBriefResp{Resp: util.Success(), Brief: out.Content}, nil
}

func (s *ConversationService) RenameConversation(ctx context.Context, req *core_api.RenameConversationReq) (*core_api.RenameConversationResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logx.Error("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.UnAuthErrCode)
	}

	// 更新对话描述
	if err = s.ConversationMapper.UpdateConversationBrief(ctx, uid, req.GetConversationId(), req.GetBrief()); err != nil {
		logx.Error("update conversation brief error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.ConversationRenameErrCode)
	}

	// 返回响应
	return &core_api.RenameConversationResp{Resp: util.Success()}, nil
}

func (s *ConversationService) ListConversation(ctx context.Context, req *core_api.ListConversationReq) (*core_api.ListConversationResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logx.Error("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.UnAuthErrCode)
	}

	// 分页获取Conversation列表，并转化为ListConversationResp_ConversationItem
	conversations, hasMore, err := s.ConversationMapper.ListConversations(ctx, uid, req.GetPage())
	if err != nil {
		logx.Error("list conversation error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.ConversationListErrCode)
	}
	items := make([]*core_api.Conversation, len(conversations))
	for i, conv := range conversations {
		items[i] = &core_api.Conversation{
			ConversationId: conv.ConversationId.Hex(),
			Brief:          conv.Brief,
			BotId:          conv.BotId,
			CreateTime:     conv.CreateTime.Unix(),
			UpdateTime:     conv.UpdateTime.Unix(),
		}
	}

	resp := &core_api.ListConversationResp{Resp: util.Success(), Conversations: items, HasMore: hasMore}
	if len(conversations) > 0 {
		resp.Cursor = conversations[len(conversations)-1].ConversationId.Hex()
	}
	// 返回响应
	return resp, nil
}

func (s *ConversationService) GetConversation(ctx context.Context, req *core_api.GetConversationReq) (*core_api.GetConversationResp, error) {
	// 鉴权 optimize
	_, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logx.Error("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.UnAuthErrCode)
	}

	msgs, hasMore, err := s.MessageMapper.ListMessage(ctx, req.GetConversationId(), req.GetPage())
	if err != nil {
		logx.Error("get conversation messages error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.ConversationGetErrCode)
	}
	// 判断是否有regen
	var regen []*mmsg.Message
	if len(msgs) > 0 {
		replyId := msgs[0].ReplyId.Hex()
		for _, msg := range msgs[1:] {
			if msg.ReplyId.Hex() == replyId {
				if regen == nil {
					regen = []*mmsg.Message{msgs[0]}
				}
				regen = append(regen, msg)
			}
		}
	}
	resp := &core_api.GetConversationResp{
		Resp:        util.Success(),
		MessageList: dm.MMsgToFMsgList(msgs),
		RegenList:   dm.MMsgToFMsgList(regen),
		HasMore:     hasMore,
	}
	if len(resp.MessageList) > 0 {
		resp.Cursor = msgs[len(msgs)-1].MessageId.Hex()
	}
	return resp, nil
}

func (s *ConversationService) DeleteConversation(ctx context.Context, req *core_api.DeleteConversationReq) (*core_api.DeleteConversationResp, error) {
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logx.Error("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.UnAuthErrCode)
	}
	if err = s.ConversationMapper.DeleteConversation(ctx, uid, req.ConversationId); err != nil {
		logx.Error("delete conversation error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.ConversationDeleteErrCode)
	}
	return &core_api.DeleteConversationResp{Resp: util.Success()}, nil
}

func (s *ConversationService) SearchConversation(ctx context.Context, req *core_api.SearchConversationReq) (*core_api.SearchConversationResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logx.Error("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.UnAuthErrCode)
	}

	// 分页获取存储域Conversation列表，并转化为交互域中Conversation
	conversations, hasMore, err := s.ConversationMapper.SearchConversations(ctx, uid, req.GetKey(), req.GetPage())
	if err != nil {
		logx.Error("list conversation error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, cst.ConversationSearchErrCode)
	}
	items := make([]*core_api.Conversation, len(conversations))
	for i, conv := range conversations {
		items[i] = &core_api.Conversation{
			ConversationId: conv.ConversationId.Hex(),
			Brief:          conv.Brief,
			CreateTime:     conv.CreateTime.Unix(),
			UpdateTime:     conv.UpdateTime.Unix(),
		}
	}

	resp := &core_api.SearchConversationResp{Resp: util.Success(), Conversations: items, HasMore: hasMore}
	if len(conversations) > 0 {
		resp.Cursor = conversations[len(conversations)-1].ConversationId.Hex()
	}
	// 返回响应
	return resp, nil
}
