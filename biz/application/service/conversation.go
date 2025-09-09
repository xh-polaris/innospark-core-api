package service

import (
	"context"
	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/conversation"
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
	if err := s.ConversationMapper.UpdateConversationBrief(ctx, uid, req.GetConversationId(), req.GetNewName()); err != nil {
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
	return &core_api.ListConversationResp{Response: util.Success(), Conversations: items}, nil
}

func (s *ConversationService) GetConversation(ctx context.Context, req *core_api.GetConversationReq) (*core_api.GetConversationResp, error) {
	// TODO 实现GetConversation接口
	return nil, nil
}
