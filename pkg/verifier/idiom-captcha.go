package verifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"image/png"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/hanguofeng/gocaptcha"
	"github.com/jqs7/drei/pkg/bot"
	"github.com/jqs7/drei/pkg/db"
	"github.com/jqs7/drei/pkg/model"
	"github.com/jqs7/drei/pkg/queue"
	"github.com/jqs7/drei/pkg/utils"
	"golang.org/x/xerrors"
)

type IdiomCaptcha struct {
	bot               bot.Interface
	queue             queue.Interface
	delMsgQueue       string
	countDownQueue    string
	blacklist         db.IBlacklist
	idioms            []model.Idiom
	captchaImgCfg     *gocaptcha.ImageConfig
	captchaImgFilters *gocaptcha.ImageFilterManager
}

func (ic IdiomCaptcha) OnLeftMember(ctx context.Context, chatID int64, leftMemberID int) {
	blacklist, err := ic.blacklist.GetItem(ctx, chatID, leftMemberID)
	if err != nil {
		return
	}
	ic.bot.DeleteMsg(chatID, blacklist.MsgID)
	ic.blacklist.DeleteItem(ctx, chatID, leftMemberID)
}

func (ic IdiomCaptcha) Verify(ctx context.Context, chatID int64, userID, msgID int, msg string) {
	blacklist, err := ic.blacklist.GetItem(ctx, chatID, userID)
	if err != nil {
		return
	}
	ic.bot.DeleteMsg(chatID, msgID)
	if ic.idioms[blacklist.Index].Word == msg {
		ic.bot.DeleteMsg(blacklist.ChatID, blacklist.MsgID)
		ic.verifyOK(ctx, *blacklist)
		return
	}
}

func (ic IdiomCaptcha) verifyOK(ctx context.Context, blacklist model.Blacklist) {
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

func NewIdiomCaptcha(bot bot.Interface, queue queue.Interface, blacklist db.IBlacklist, idiomPath, fontPath string) (Interface, error) {
	f, err := os.Open(idiomPath)
	if err != nil {
		return nil, xerrors.Errorf("读取 %s 文件失败: %w", idiomPath, err)
	}
	var idioms []model.Idiom
	if err := json.NewDecoder(f).Decode(&idioms); err != nil {
		return nil, xerrors.Errorf("解码 idiom 文件失败: %w", err)
	}
	tmp := idioms[:0]
	for _, p := range idioms {
		if len([]rune(p.Word)) == 4 {
			tmp = append(tmp, p)
		}
	}
	idioms = tmp

	filterConfig := new(gocaptcha.FilterConfig)
	filterConfig.Init()
	filterConfig.Filters = []string{
		gocaptcha.IMAGE_FILTER_NOISE_LINE,
		gocaptcha.IMAGE_FILTER_NOISE_POINT,
		gocaptcha.IMAGE_FILTER_STRIKE,
	}
	for _, v := range filterConfig.Filters {
		filterConfigGroup := new(gocaptcha.FilterConfigGroup)
		filterConfigGroup.Init()
		filterConfigGroup.SetItem("Num", "180")
		filterConfig.SetGroup(v, filterConfigGroup)
	}
	return &IdiomCaptcha{
		bot:            bot,
		blacklist:      blacklist,
		idioms:         idioms,
		queue:          queue,
		delMsgQueue:    os.Getenv("DELETE_MSG_QUEUE"),
		countDownQueue: os.Getenv("CAPTCHA_COUNTDOWN_QUEUE"),
		captchaImgCfg: &gocaptcha.ImageConfig{
			Width:    320,
			Height:   100,
			FontSize: 80,
			FontFiles: []string{
				filepath.Join(fontPath, "STFANGSO.ttf"),
				filepath.Join(fontPath, "STHEITI.ttf"),
				filepath.Join(fontPath, "STXIHEI.ttf"),
			},
		},
		captchaImgFilters: gocaptcha.CreateImageFilterManagerByConfig(filterConfig),
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

func (ic IdiomCaptcha) OnNewMember(ctx context.Context, chatID int64, chatName string, newMemberID int, firstName, lastName string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	rIdx := r.Intn(len(ic.idioms))

	cImg := gocaptcha.CreateCImage(ic.captchaImgCfg)
	cImg.DrawString(ic.idioms[rIdx].Word)
	for _, f := range ic.captchaImgFilters.GetFilters() {
		f.Proc(cImg)
	}
	captchaBuffer := bytes.NewBuffer([]byte{})
	if err := png.Encode(captchaBuffer, cImg); err != nil {
		return
	}
	userLink := fmt.Sprintf(model.UserLinkTemplate, newMemberID, html.EscapeString(utils.GetFullName(firstName, lastName)))
	msgTemplate := fmt.Sprintf(model.EnterRoomMsg, chatName)
	msgID, err := ic.bot.SendImg(chatID, captchaBuffer.Bytes(), fmt.Sprintf(userLink+" "+msgTemplate, 300), InlineKeyboard)
	if err != nil {
		return
	}
	ic.blacklist.CreateItem(ctx, model.Blacklist{
		UserID:      newMemberID,
		ChatID:      chatID,
		Index:       rIdx,
		MsgID:       msgID,
		ExpireAt:    time.Now().Add(time.Second * 300),
		UserLink:    userLink,
		MsgTemplate: msgTemplate,
	})
	err = ic.queue.SendMsg(ctx, ic.countDownQueue, model.CountdownMsg{
		ChatID: chatID,
		UserID: newMemberID,
	}, 5)
	if err != nil {
		log.Println("send count down msg failed: ", err)
	}
}

func (ic IdiomCaptcha) OnCallbackQuery(ctx context.Context, chatID int64, msgID, fromUser int, callbackID, data string) {
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
		idx, img := ic.getRandImg()
		ic.blacklist.UpdateIdx(ctx, chatID, fromUser, idx)
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

func (ic IdiomCaptcha) getRandImg() (int, []byte) {
	rIdx := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(ic.idioms))

	cImg := gocaptcha.CreateCImage(ic.captchaImgCfg)
	cImg.DrawString(ic.idioms[rIdx].Word)
	for _, f := range ic.captchaImgFilters.GetFilters() {
		f.Proc(cImg)
	}
	captchaBuffer := bytes.NewBuffer([]byte{})
	if err := png.Encode(captchaBuffer, cImg); err != nil {
		log.Fatalln("encode png img failed: ", err)
	}
	return rIdx, captchaBuffer.Bytes()
}
