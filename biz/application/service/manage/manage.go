package manage

import (
	"context"
	"sort"
	"time"

	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/basic"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/manage"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/feedback"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/types/errno"
)

var ManageSVC *ManageService

type ManageService struct {
	UserMapper     user.MongoMapper
	FeedbackMapper feedback.MongoMapper
	MessageMapper  message.MongoMapper
}

func (m *ManageService) AdminLogin(ctx context.Context, req *manage.AdminLoginReq) (resp *manage.AdminLoginResp, err error) {
	if req.Account != conf.GetConfig().Admin.Account || req.Password != conf.GetConfig().Admin.Password {
		return nil, errorx.New(errno.ErrLogin)
	}
	return &manage.AdminLoginResp{Resp: util.Success(), Token: conf.GetConfig().Admin.Token}, nil
}

func (m *ManageService) ListUser(ctx context.Context, req *manage.ListUserReq) (resp *manage.ListUserResp, err error) {
	if err = checkAdmin(ctx); err != nil {
		return
	}
	total, us, err := m.UserMapper.ListUser(ctx, req.Page, req.Status, req.SortedBy, req.Reverse)
	if err != nil {
		return
	}
	var users []*manage.User
	for _, u := range us {
		var expire int64
		if !u.Expire.IsZero() {
			expire = u.Expire.Unix()
		}
		users = append(users, &manage.User{
			Id:         u.ID.Hex(),
			Phone:      u.Phone,
			Name:       u.Name,
			Avatar:     u.Avatar,
			Warnings:   u.Warnings,
			Status:     u.Status,
			Expire:     expire,
			LoginTime:  u.LoginTime.Unix(),
			CreateTime: u.CreateTime.Unix(),
			UpdateTime: u.UpdateTime.Unix(),
		})
	}
	return &manage.ListUserResp{
		Resp:  util.Success(),
		Total: total,
		User:  users,
	}, nil
}

func (m *ManageService) Forbidden(ctx context.Context, req *manage.ForbiddenUserReq) (resp *manage.ForbiddenUserResp, err error) {
	if err = checkAdmin(ctx); err != nil {
		return
	}
	if req.Status == user.StatusForbidden && req.Expire != nil {
		err = m.UserMapper.Forbidden(ctx, req.Id, time.Unix(*req.Expire, 0))
	} else if req.Status == user.StatusNormal {
		err = m.UserMapper.UnForbidden(ctx, req.Id)
	}
	if err != nil {
		return
	}
	return &manage.ForbiddenUserResp{Resp: util.Success()}, nil
}

func (m *ManageService) ListFeedback(ctx context.Context, req *manage.ListFeedBackReq) (resp *manage.ListFeedBackResp, err error) {
	if err = checkAdmin(ctx); err != nil {
		return
	}
	total, fbs, err := m.FeedbackMapper.ListFeedback(ctx, req.Page, req.MessageId, req.UserId, req.Action, req.Type)
	if err != nil {
		return
	}
	var feedbacks []*manage.ListFeedBackResp_FeedBack
	for _, fb := range fbs {
		feedbacks = append(feedbacks, &manage.ListFeedBackResp_FeedBack{
			MessageId:  fb.MessageId.Hex(),
			UserId:     fb.UserId.Hex(),
			Action:     fb.Action,
			Type:       fb.Type,
			Content:    fb.Content,
			CreateTime: fb.UpdateTime.Unix(),
		})
	}
	return &manage.ListFeedBackResp{
		Resp:      util.Success(),
		Feedbacks: feedbacks,
		Total:     total,
	}, nil
}

func checkAdmin(ctx context.Context) error {
	if c, err := adaptor.ExtractContext(ctx); err != nil {
		return err
	} else if string(c.GetHeader("Authorization")) != conf.GetConfig().Admin.Token {
		return errorx.New(errno.UnAuthErrCode)
	}
	return nil
}

func (m *ManageService) UserStatistics(ctx context.Context, req *manage.UserStatisticsReq) (resp *manage.UserStatisticsResp, err error) {
	if err = checkAdmin(ctx); err != nil {
		return
	}
	before, err := m.UserMapper.CountUserByCreateTime(ctx, time.Unix(req.Start, 0), false)
	if err != nil {
		return nil, err
	}

	var size int64 = 5000
	var page int64 = 1
	dailyGrowthMap := map[int64]int{}
	var isTooAfter bool
	for {
		_, list, err := m.UserMapper.ListUser(ctx, &basic.Page{Page: &page, Size: &size}, 0, 0, 1)
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			break // 数据拉完了
		}
		// 聚合新增
		for _, u := range list {
			if u.CreateTime.Unix() > req.End {
				isTooAfter = true
				break
			} else if u.CreateTime.Unix() < req.Start {
				continue
			}
			day := normalizeDay(u.CreateTime)
			dailyGrowthMap[day]++
		}
		if isTooAfter {
			break
		}
		page++
	}
	// 排序日期
	days := make([]int64, 0, len(dailyGrowthMap))
	for d := range dailyGrowthMap {
		days = append(days, d)
	}
	sort.Slice(days, func(i, j int) bool { return days[i] < days[j] })
	// 每日新增
	growth := make([]*manage.UserStatisticsResp_Item, 0, len(days))
	for _, day := range days {
		growth = append(growth, &manage.UserStatisticsResp_Item{
			Date:  day,
			Count: int64(dailyGrowthMap[day]),
		})
	}

	// 累积数据
	accumulate := make([]*manage.UserStatisticsResp_Item, 0, len(days))
	var sum = before
	for _, g := range growth {
		sum += g.Count
		accumulate = append(accumulate, &manage.UserStatisticsResp_Item{
			Date:  g.Date,
			Count: sum,
		})
	}
	// 趋势线
	trend := calcLinearRegression(accumulate)
	// 平均
	avgDaily := float64(sum) / float64(len(days))
	return &manage.UserStatisticsResp{
		Resp:               util.Success(),
		Growth:             growth,
		Accumulate:         accumulate,
		Trend:              trend,
		TotalNewUsers:      sum,
		AverageDailyGrowth: avgDaily,
	}, nil
}

// 将时间戳归零到当天 00:00:00
func normalizeDay(t time.Time) int64 {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
}

func calcLinearRegression(acc []*manage.UserStatisticsResp_Item) *manage.UserStatisticsResp_Trend {
	n := float64(len(acc))
	if n <= 1 {
		return &manage.UserStatisticsResp_Trend{W: 0, B: 0}
	}
	var sumX, sumY, sumXY, sumX2 float64
	for i := range acc {
		x := float64(i)
		y := float64(acc[i].Count)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	w := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	b := (sumY - w*sumX) / n
	return &manage.UserStatisticsResp_Trend{
		W: w,
		B: b,
	}
}

var shanghaiLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*60*60)
	}
	return loc
}()

func (m *ManageService) GetWeeklyStats(ctx context.Context, req *manage.GetWeeklyStatsReq) (*manage.GetWeeklyStatsResp, error) {
	if err := checkAdmin(ctx); err != nil {
		return nil, err
	}
	thisTuesday, lastTuesday, twoAgoTuesday := calcWeekBounds(time.Now())
	monthStart := calcMonthStart(time.Now())
	tomorrow := truncDay(time.Now().In(shanghaiLoc)).Add(24 * time.Hour)

	totalUsers, err := m.UserMapper.CountUserByCreateTime(ctx, time.Time{}, true)
	if err != nil {
		return nil, errorx.New(errno.ErrGetWeeklyStats)
	}
	newUsersThisWeek, err := m.UserMapper.CountUserByCreateTimeBetween(ctx, lastTuesday, thisTuesday)
	if err != nil {
		return nil, errorx.New(errno.ErrGetWeeklyStats)
	}
	newUsersLastWeek, err := m.UserMapper.CountUserByCreateTimeBetween(ctx, twoAgoTuesday, lastTuesday)
	if err != nil {
		return nil, errorx.New(errno.ErrGetWeeklyStats)
	}
	weeklyActiveUsers, err := m.MessageMapper.CountActiveUsers(ctx, lastTuesday, thisTuesday)
	if err != nil {
		return nil, errorx.New(errno.ErrGetWeeklyStats)
	}
	monthlyActiveUsers, err := m.MessageMapper.CountActiveUsers(ctx, monthStart, tomorrow)
	if err != nil {
		return nil, errorx.New(errno.ErrGetWeeklyStats)
	}
	return &manage.GetWeeklyStatsResp{
		Resp:             util.Success(),
		TotalUsers:       totalUsers,
		NewUsersThisWeek: newUsersThisWeek,
		NewUsersLastWeek: newUsersLastWeek,
		WeeklyAvgDAU:     weeklyActiveUsers,
		MonthlyAvgDAU:    monthlyActiveUsers,
		WeekStart:        lastTuesday.Format(time.RFC3339),
		WeekEnd:          thisTuesday.Format(time.RFC3339),
	}, nil
}

func calcWeekBounds(now time.Time) (thisTuesday, lastTuesday, twoAgoTuesday time.Time) {
	now = now.In(shanghaiLoc)
	daysBack := (int(now.Weekday()) - 2 + 7) % 7
	thisTuesday = truncDay(now.AddDate(0, 0, -daysBack))
	lastTuesday = thisTuesday.AddDate(0, 0, -7)
	twoAgoTuesday = lastTuesday.AddDate(0, 0, -7)
	return
}

func calcMonthStart(now time.Time) time.Time {
	now = now.In(shanghaiLoc)
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, shanghaiLoc)
}

func truncDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
