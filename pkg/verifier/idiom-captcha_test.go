package verifier

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/jqs7/drei/pkg/bot"
	"github.com/jqs7/drei/pkg/captcha"
	"github.com/jqs7/drei/pkg/db"
	"github.com/jqs7/drei/pkg/model"
	"github.com/jqs7/drei/pkg/queue"
	"github.com/stretchr/testify/assert"
)

func TestIdiomCaptcha(t *testing.T) {
	ctx := context.Background()
	delMsgQueue := "DelMsg"
	countdownQueue := "CountDown"
	_ = os.Setenv("DELETE_MSG_QUEUE", delMsgQueue)
	_ = os.Setenv("CAPTCHA_COUNTDOWN_QUEUE", countdownQueue)

	type mockRst struct {
		bot         *bot.MockInterface
		blacklist   *db.MockIBlacklist
		queue       *queue.MockInterface
		imgVerifier *captcha.MockInterface
		verifier    Interface
	}

	userEnterGroup := func(t *testing.T, ctrl *gomock.Controller) mockRst {
		mockBot := bot.NewMockInterface(ctrl)
		mockBot.EXPECT().SendImg(int64(1), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

		mockBlacklist := db.NewMockIBlacklist(ctrl)
		mockBlacklist.EXPECT().CreateItem(ctx, gomock.Any()).Times(1)

		mockQueue := queue.NewMockInterface(ctrl)
		mockQueue.EXPECT().SendMsg(ctx, countdownQueue, gomock.Any(), int64(model.CaptchaRefreshSecond)).Times(1)

		imgVerifier := captcha.NewMockInterface(ctrl)
		imgVerifier.EXPECT().GenRandImg().Times(1)

		verifier, err := NewIdiomVerifier(mockBot, mockQueue, mockBlacklist, imgVerifier)
		assert.NoError(t, err)
		verifier.OnNewMember(ctx, int64(1), "ChatName", 1, "FirstName", "LastName")
		return mockRst{
			bot:         mockBot,
			blacklist:   mockBlacklist,
			queue:       mockQueue,
			verifier:    verifier,
			imgVerifier: imgVerifier,
		}
	}

	t.Run("用户进群", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userEnterGroup(t, ctrl)
	})

	t.Run("用户发送验证失败信息", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mock := userEnterGroup(t, ctrl)
		mock.bot.EXPECT().DeleteMsg(int64(1), 2).Times(1)
		mock.blacklist.EXPECT().GetItem(ctx, int64(1), 1).Return(&model.Blacklist{
			ChatID: int64(1),
			UserID: 1,
			MsgID:  2,
		}, nil).Times(1)
		mock.imgVerifier.EXPECT().VerifyAnswer(model.Answer{Number: 0}, model.Answer{String: "WTF"}).Return(false)
		mock.verifier.Verify(ctx, int64(1), 1, 2, "WTF")
	})

	t.Run("用户发送验证成功信息", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mock := userEnterGroup(t, ctrl)
		mock.bot.EXPECT().SendMsg(int64(1), gomock.Any())
		mock.bot.EXPECT().DeleteMsg(int64(1), 2).Times(1)
		mock.bot.EXPECT().DeleteMsg(int64(1), 3).Times(1)
		mock.blacklist.EXPECT().GetItem(ctx, int64(1), 1).Return(&model.Blacklist{
			ChatID: int64(1),
			UserID: 1,
			MsgID:  2,
		}, nil).Times(1)
		mock.blacklist.EXPECT().DeleteItem(ctx, int64(1), 1).Times(1)
		mock.queue.EXPECT().SendMsg(ctx, delMsgQueue, gomock.Any(), int64(10)).Times(1)
		mock.imgVerifier.EXPECT().VerifyAnswer(model.Answer{Number: 0}, model.Answer{String: "OK"}).Return(true)
		mock.verifier.Verify(ctx, int64(1), 1, 3, "OK")
	})

	t.Run("用户进群后退群", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mock := userEnterGroup(t, ctrl)

		mock.bot.EXPECT().DeleteMsg(int64(1), 2).Times(1)
		mock.blacklist.EXPECT().GetItem(ctx, int64(1), 1).Return(&model.Blacklist{
			ChatID: int64(1),
			UserID: 1,
			MsgID:  2,
		}, nil).Times(1)
		mock.blacklist.EXPECT().DeleteItem(ctx, int64(1), 1).Times(1)
		mock.verifier.OnLeftMember(ctx, int64(1), 1)
	})

	t.Run("管理员直接令用户通过验证", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mock := userEnterGroup(t, ctrl)
		mock.bot.EXPECT().SendMsg(int64(1), gomock.Any())
		mock.bot.EXPECT().DeleteMsg(int64(1), 2).Times(1)
		mock.bot.EXPECT().IsAdmin(int64(1), 3).Return(true).Times(1)
		mock.blacklist.EXPECT().DeleteItem(ctx, int64(1), 1).Times(1)
		mock.blacklist.EXPECT().GetItemByMsgID(ctx, int64(1), 2).Return(&model.Blacklist{
			ChatID: 1,
			UserID: 1,
			MsgID:  2,
		}, nil).Times(1)
		mock.queue.EXPECT().SendMsg(ctx, delMsgQueue, gomock.Any(), int64(10)).Times(1)
		mock.verifier.OnCallbackQuery(ctx, int64(1), 2, 3, "callbackID", model.CallbackTypePassThrough)
	})

	t.Run("管理员直接踢出用户", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mock := userEnterGroup(t, ctrl)
		mock.bot.EXPECT().DeleteMsg(int64(1), 2).Times(1)
		mock.bot.EXPECT().IsAdmin(int64(1), 3).Return(true).Times(1)
		mock.bot.EXPECT().Kick(int64(1), 1, time.Unix(0, 0)).Times(1)
		mock.blacklist.EXPECT().DeleteItem(ctx, int64(1), 1).Times(1)
		mock.blacklist.EXPECT().GetItemByMsgID(ctx, int64(1), 2).Return(&model.Blacklist{
			ChatID: 1,
			UserID: 1,
			MsgID:  2,
		}, nil).Times(1)
		mock.verifier.OnCallbackQuery(ctx, int64(1), 2, 3, "callbackID", model.CallbackTypeKick)
	})

	t.Run("非管理员令用户通过验证", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mock := userEnterGroup(t, ctrl)
		mock.bot.EXPECT().IsAdmin(int64(1), 3).Return(false).Times(1)
		mock.bot.EXPECT().AnswerCallback("callbackID", "无权限")
		mock.verifier.OnCallbackQuery(ctx, int64(1), 2, 3, "callbackID", model.CallbackTypePassThrough)
	})

	t.Run("非管理员踢出用户", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mock := userEnterGroup(t, ctrl)
		mock.bot.EXPECT().IsAdmin(int64(1), 3).Return(false).Times(1)
		mock.bot.EXPECT().AnswerCallback("callbackID", "无权限")
		mock.verifier.OnCallbackQuery(ctx, int64(1), 2, 3, "callbackID", model.CallbackTypeKick)
	})

	t.Run("其他用户刷新验证码", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mock := userEnterGroup(t, ctrl)
		mock.bot.EXPECT().AnswerCallback("callbackID", "无权限")
		mock.blacklist.EXPECT().GetItem(ctx, int64(1), 3).Return(nil, db.ErrNotFound).Times(1)
		mock.verifier.OnCallbackQuery(ctx, int64(1), 2, 3, "callbackID", model.CallbackTypeRefresh)
	})

	t.Run("用户刷新验证码", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mock := userEnterGroup(t, ctrl)
		mock.bot.EXPECT().UpdatePhoto(int64(1), 2, gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
		mock.bot.EXPECT().AnswerCallback("callbackID", "刷新成功")
		mock.blacklist.EXPECT().GetItem(ctx, int64(1), 1).Return(&model.Blacklist{
			MsgID:    2,
			ExpireAt: time.Now().Add(time.Second),
		}, nil).Times(1)
		mock.blacklist.EXPECT().UpdateIdx(ctx, int64(1), 1, gomock.Any()).Times(1)
		mock.imgVerifier.EXPECT().GenRandImg().Times(1)
		mock.verifier.OnCallbackQuery(ctx, int64(1), 2, 1, "callbackID", model.CallbackTypeRefresh)
	})
}
