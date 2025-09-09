package service

import (
	"context"

	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	dm "github.com/xh-polaris/innospark-core-api/biz/domain/model"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/conversation"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
)

type IConversationService interface {
	CreateConversation(ctx context.Context, req *core_api.CreateConversationReq) (*core_api.CreateConversationResp, error)
	RenameConversation(ctx context.Context, req *core_api.RenameConversationReq) (*core_api.RenameConversationResp, error)
	ListConversation(ctx context.Context, req *core_api.ListConversationReq) (*core_api.ListConversationResp, error)
	GetConversation(ctx context.Context, req *core_api.GetConversationReq) (*core_api.GetConversationResp, error)
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
		logx.Error("extract user id error: %v", err)
		return nil, cst.UnAuthErr
	}

	// 调用mapper创建对话
	newConversation, err := s.ConversationMapper.CreateNewConversation(ctx, uid)
	if err != nil {
		logx.Error("create conversation error: %v", err)
		return nil, cst.ConversationCreationErr
	}

	// 返回conversationID
	return &core_api.CreateConversationResp{Resp: util.Success(), ConversationId: newConversation.ConversationId.Hex()}, nil
}

func (s *ConversationService) RenameConversation(ctx context.Context, req *core_api.RenameConversationReq) (*core_api.RenameConversationResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logx.Error("extract user id error: %v", err)
		return nil, cst.UnAuthErr
	}

	// 更新对话描述
	if err = s.ConversationMapper.UpdateConversationBrief(ctx, uid, req.GetConversationId(), req.GetBrief()); err != nil {
		logx.Error("update conversation brief error: %v", err)
		return nil, cst.ConversationRenameErr
	}

	// 返回响应
	return &core_api.RenameConversationResp{Resp: util.Success()}, nil
}

func (s *ConversationService) ListConversation(ctx context.Context, req *core_api.ListConversationReq) (*core_api.ListConversationResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logx.Error("extract user id error: %v", err)
		return nil, cst.UnAuthErr
	}

	// 分页获取Conversation列表，并转化为ListConversationResp_ConversationItem
	conversations, err := s.ConversationMapper.ListConversations(ctx, uid, req.GetPage())
	if err != nil {
		logx.Error("list conversation error: %v", err)
		return nil, cst.ConversationListErr
	}
	items := make([]*core_api.ListConversationResp_ConversationItem, len(conversations))
	for i, conv := range conversations {
		items[i] = &core_api.ListConversationResp_ConversationItem{
			ConversationId: conv.ConversationId.Hex(),
			Brief:          conv.Brief,
			CreateTime:     conv.CreateTime.Unix(),
			UpdateTime:     conv.UpdateTime.Unix(),
		}
	}

	// 返回响应
	return &core_api.ListConversationResp{Resp: util.Success(), Conversations: items}, nil
}

func (s *ConversationService) GetConversation(ctx context.Context, req *core_api.GetConversationReq) (*core_api.GetConversationResp, error) {
	// 鉴权 optimize
	_, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logx.Error("extract user id error: %v", err)
		return nil, cst.UnAuthErr
	}

	msgs, hasMore, err := s.MessageMapper.ListMessage(ctx, req.GetConversationId(), req.GetPage())
	if err != nil {
		logx.Error("get conversation messages error: %v", err)
		return nil, cst.ConversationGetErr
	}
	// 判断是否有regen
	var regen []*mmsg.Message
	if len(msgs) > 0 {
		replyId := msgs[0].ReplyId
		for _, msg := range msgs[1:] {
			if msg.ReplyId == replyId {
				if regen == nil {
					regen = []*mmsg.Message{msgs[0]}
				}
				regen = append(regen, msg)
			}
		}
	}
	return &core_api.GetConversationResp{
		Resp:        util.Success(),
		MessageList: dm.MMsgToFMsgList(msgs),
		RegenList:   dm.MMsgToFMsgList(regen),
		HasMore:     hasMore,
	}, nil
}
