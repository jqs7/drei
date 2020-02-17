package verifier

import (
	"context"
	"fmt"
	"html"
	"log"
	"os"
	"time"

	"github.com/jqs7/drei/pkg/bot"
	"github.com/jqs7/drei/pkg/captcha"
	"github.com/jqs7/drei/pkg/db"
	"github.com/jqs7/drei/pkg/model"
	"github.com/jqs7/drei/pkg/queue"
	"github.com/jqs7/drei/pkg/utils"
)

type IdiomVerifier struct {
	bot            bot.Interface
	queue          queue.Interface
	delMsgQueue    string
	countDownQueue string
	blacklist      db.IBlacklist
	captcha        captcha.Interface
}

func (ic IdiomVerifier) OnLeftMember(ctx context.Context, chatID int64, leftMemberID int) {
	blacklist, err := ic.blacklist.GetItem(ctx, chatID, leftMemberID)
	if err != nil {
		return
	}
	ic.bot.DeleteMsg(chatID, blacklist.MsgID)
	ic.blacklist.DeleteItem(ctx, chatID, leftMemberID)
}

func (ic IdiomVerifier) Verify(ctx context.Context, chatID int64, userID, msgID int, msg string) {
	blacklist, err := ic.blacklist.GetItem(ctx, chatID, userID)
	if err != nil {
		return
	}
	ic.bot.DeleteMsg(chatID, msgID)
	if ic.captcha.VerifyAnswer(model.Answer{Number: blacklist.Index}, model.Answer{String: msg}) {
		ic.bot.DeleteMsg(blacklist.ChatID, blacklist.MsgID)
		ic.verifyOK(ctx, *blacklist)
		return
	}
}

func (ic IdiomVerifier) verifyOK(ctx context.Context, blacklist model.Blacklist) {
	ic.blacklist.DeleteItem(ctx, blacklist.ChatID, blacklist.UserID)
	msgID, err := ic.bot.SendMsg(blacklist.ChatID, blacklist.UserLink+" 恭喜，你已验证通过")
	if err != nil {
		return
	}
	err = ic.queue.SendMsg(ctx, ic.delMsgQueue, model.MsgToDelete{
		ChatID: blacklist.ChatID,
		MsgID:  msgID,
	}, 10)
	if err != nil {
		log.Println("send delete msg: ", err)
	}
}

func NewIdiomVerifier(bot bot.Interface, queue queue.Interface, blacklist db.IBlacklist, verifier captcha.Interface) (Interface, error) {
	return &IdiomVerifier{
		bot:            bot,
		blacklist:      blacklist,
		captcha:        verifier,
		queue:          queue,
		delMsgQueue:    os.Getenv("DELETE_MSG_QUEUE"),
		countDownQueue: os.Getenv("CAPTCHA_COUNTDOWN_QUEUE"),
	}, nil
}

var InlineKeyboard = [][]model.KV{
	{
		{K: "刷新验证码", V: model.CallbackTypeRefresh},
		{K: "通过验证[管理员]", V: model.CallbackTypePassThrough},
	},
	{
		{K: "踢出群组[管理员]", V: model.CallbackTypeKick},
	},
}

func (ic IdiomVerifier) OnNewMember(ctx context.Context, chatID int64, chatName string, newMemberID int, firstName, lastName string) {
	answer, img := ic.captcha.GenRandImg()
	userLink := fmt.Sprintf(model.UserLinkTemplate, newMemberID, html.EscapeString(utils.GetFullName(firstName, lastName)))
	msgTemplate := fmt.Sprintf(model.EnterRoomMsg, chatName)
	msgID, err := ic.bot.SendImg(chatID, img, fmt.Sprintf(userLink+" "+msgTemplate, 300), InlineKeyboard)
	if err != nil {
		return
	}
	ic.blacklist.CreateItem(ctx, model.Blacklist{
		UserID:      newMemberID,
		ChatID:      chatID,
		Index:       answer.Number,
		MsgID:       msgID,
		ExpireAt:    time.Now().Add(time.Second * 300),
		UserLink:    userLink,
		MsgTemplate: msgTemplate,
	})
	err = ic.queue.SendMsg(ctx, ic.countDownQueue, model.CountdownMsg{
		ChatID: chatID,
		UserID: newMemberID,
	}, model.CaptchaRefreshSecond)
	if err != nil {
		log.Println("send count down msg failed: ", err)
	}
}

func (ic IdiomVerifier) OnCallbackQuery(ctx context.Context, chatID int64, msgID, fromUser int, callbackID, data string) {
	switch data {
	case model.CallbackTypeRefresh:
		blacklist, err := ic.blacklist.GetItem(ctx, chatID, fromUser)
		if err != nil {
			if err == db.ErrNotFound {
				ic.bot.AnswerCallback(callbackID, "无权限")
			}
			return
		}
		if blacklist.ExpireAt.Before(time.Now()) {
			ic.bot.AnswerCallback(callbackID, "已过期")
			return
		}
		answer, img := ic.captcha.GenRandImg()
		ic.blacklist.UpdateIdx(ctx, chatID, fromUser, answer.Number)
		ic.bot.UpdatePhoto(chatID, blacklist.MsgID,
			fmt.Sprintf(blacklist.UserLink+" "+blacklist.MsgTemplate, time.Until(blacklist.ExpireAt)/time.Second),
			InlineKeyboard, img,
		)
		ic.bot.AnswerCallback(callbackID, "刷新成功")
	case model.CallbackTypeKick:
		if !ic.bot.IsAdmin(chatID, fromUser) {
			ic.bot.AnswerCallback(callbackID, "无权限")
			return
		}
		blacklist, err := ic.blacklist.GetItemByMsgID(ctx, chatID, msgID)
		if err != nil {
			return
		}
		ic.bot.DeleteMsg(chatID, blacklist.MsgID)
		ic.bot.Kick(chatID, blacklist.UserID, time.Unix(0, 0))
		ic.blacklist.DeleteItem(ctx, chatID, blacklist.UserID)
	case model.CallbackTypePassThrough:
		if !ic.bot.IsAdmin(chatID, fromUser) {
			ic.bot.AnswerCallback(callbackID, "无权限")
			return
		}
		blacklist, err := ic.blacklist.GetItemByMsgID(ctx, chatID, msgID)
		if err != nil {
			return
		}
		ic.bot.DeleteMsg(chatID, msgID)
		ic.verifyOK(ctx, *blacklist)
	}
}
