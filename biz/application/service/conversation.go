package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	dm "github.com/xh-polaris/innospark-core-api/biz/domain/msg"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/conversation"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"github.com/xh-polaris/innospark-core-api/types/errno"
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
		logs.Errorf("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.UnAuthErrCode)
	}

	// 调用mapper创建对话
	newConversation, err := s.ConversationMapper.CreateNewConversation(ctx, uid, req.BotId)
	if err != nil {
		logs.Errorf("create conversation error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.ConversationCreateErrCode)
	}

	// 返回conversationID
	return &core_api.CreateConversationResp{Resp: util.Success(), ConversationId: newConversation.ConversationId.Hex()}, nil
}

func (s *ConversationService) GenerateBrief(ctx context.Context, req *core_api.GenerateBriefReq) (*core_api.GenerateBriefResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logs.Errorf("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.UnAuthErrCode)
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
		return nil, errorx.WrapByCode(err, errno.ConversationGenerateBriefErrCode)
	}
	in := []*schema.Message{schema.UserMessage(fmt.Sprintf(config.GetConfig().TitleGen, req.Messages[0].Content))}
	out, err := m.Generate(ctx, in)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ConversationGenerateBriefErrCode)
	}
	out.Content = strings.Trim(out.Content, "\"")
	re := regexp.MustCompile(`[\[(（][^]）)]*[]）)]`)
	out.Content = re.ReplaceAllString(out.Content, "")
	// 更新标题
	if err = s.ConversationMapper.UpdateConversationBrief(ctx, uid, req.ConversationId, out.Content); err != nil {
		return nil, errorx.WrapByCode(err, errno.ConversationGenerateBriefErrCode)
	}
	return &core_api.GenerateBriefResp{Resp: util.Success(), Brief: out.Content}, nil
}

func (s *ConversationService) RenameConversation(ctx context.Context, req *core_api.RenameConversationReq) (*core_api.RenameConversationResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logs.Errorf("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.UnAuthErrCode)
	}

	// 更新对话描述
	if err = s.ConversationMapper.UpdateConversationBrief(ctx, uid, req.GetConversationId(), req.GetBrief()); err != nil {
		logs.Errorf("update conversation brief error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.ConversationRenameErrCode)
	}

	// 返回响应
	return &core_api.RenameConversationResp{Resp: util.Success()}, nil
}

func (s *ConversationService) ListConversation(ctx context.Context, req *core_api.ListConversationReq) (*core_api.ListConversationResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logs.Errorf("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.UnAuthErrCode)
	}

	// 分页获取Conversation列表，并转化为ListConversationResp_ConversationItem
	conversations, hasMore, err := s.ConversationMapper.ListConversations(ctx, uid, req.GetPage())
	if err != nil {
		logs.Errorf("list conversation error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.ConversationListErrCode)
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
		logs.Errorf("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.UnAuthErrCode)
	}

	msgs, hasMore, err := s.MessageMapper.ListMessage(ctx, req.GetConversationId(), req.GetPage())
	if err != nil {
		logs.Errorf("get conversation messages error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.ConversationGetErrCode)
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
		logs.Errorf("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.UnAuthErrCode)
	}
	if err = s.ConversationMapper.DeleteConversation(ctx, uid, req.ConversationId); err != nil {
		logs.Errorf("delete conversation error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.ConversationDeleteErrCode)
	}
	return &core_api.DeleteConversationResp{Resp: util.Success()}, nil
}

func (s *ConversationService) SearchConversation(ctx context.Context, req *core_api.SearchConversationReq) (*core_api.SearchConversationResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logs.Errorf("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.UnAuthErrCode)
	}

	// 分页获取存储域Conversation列表，并转化为交互域中Conversation
	conversations, hasMore, err := s.ConversationMapper.SearchConversations(ctx, uid, req.GetKey(), req.GetPage())
	if err != nil {
		logs.Errorf("list conversation error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.ConversationSearchErrCode)
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
