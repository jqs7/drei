package bot

import (
	"log"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jqs7/drei/pkg/model"
	"github.com/jqs7/drei/pkg/utils"
	"golang.org/x/xerrors"
)

type TGBotAPI struct {
	bot *tgbotapi.BotAPI
}

func (b TGBotAPI) SendImg(chatID int64, img []byte, caption string, keyboard [][]model.KV) (int, error) {
	captchaMsg := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{
		Name:  strconv.FormatInt(time.Now().UnixNano(), 10),
		Bytes: img,
	})
	captchaMsg.Caption = caption
	captchaMsg.ParseMode = tgbotapi.ModeHTML
	captchaMsg.ReplyMarkup = TransformKeyboard(keyboard)
	resp, err := b.bot.Send(captchaMsg)
	if err != nil {
		return -1, err
	}
	return resp.MessageID, nil
}

func TransformKeyboard(keyboard [][]model.KV) tgbotapi.InlineKeyboardMarkup {
	inlineKeyboard := make([][]tgbotapi.InlineKeyboardButton, len(keyboard))
	for i, v := range keyboard {
		line := make([]tgbotapi.InlineKeyboardButton, len(v))
		for j, w := range v {
			line[j] = tgbotapi.NewInlineKeyboardButtonData(w.K, w.V)
		}
		inlineKeyboard[i] = line
	}
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: inlineKeyboard}
}

func NewAPI(botToken string) (Interface, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, xerrors.Errorf("初始化机器人失败: %w", err)
	}
	return &TGBotAPI{
		bot: bot,
	}, nil
}

func (b TGBotAPI) SendMsg(chatID int64, msg string) (int, error) {
	m := tgbotapi.NewMessage(chatID, msg)
	m.ParseMode = tgbotapi.ModeHTML
	msgRst, err := b.bot.Send(m)
	if err != nil {
		return -1, xerrors.Errorf("发送消息 %s 至 %d 失败: %w", msg, chatID, err)
	}
	return msgRst.MessageID, nil
}

func (b TGBotAPI) SetWebhook(addr string) error {
	_, err := b.bot.SetWebhook(tgbotapi.NewWebhook(addr))
	if err != nil {
		return xerrors.Errorf("设置 webhook: %s 失败: %w", addr, err)
	}
	return nil
}

func (b TGBotAPI) DeleteMsg(chatID int64, msgID int) {
	_, err := b.bot.DeleteMessage(tgbotapi.NewDeleteMessage(chatID, msgID))
	if err != nil {
		log.Printf("删除消息: %d %d 失败: %+v", chatID, msgID, err)
	}
}

func (b TGBotAPI) Kick(chatID int64, userID int, until time.Time) {
	_, err := b.bot.KickChatMember(tgbotapi.KickChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: chatID,
			UserID: userID,
		},
		UntilDate: until.Unix(),
	})
	if err != nil {
		log.Printf("删除成员: %d %d 失败: %+v", chatID, userID, err)
	}
}

func (b TGBotAPI) IsAdmin(chatID int64, userID int) bool {
	member, err := b.bot.GetChatMember(tgbotapi.ChatConfigWithUser{
		ChatID: chatID,
		UserID: userID,
	})
	if err != nil {
		return false
	}
	if !member.IsCreator() && !member.IsAdministrator() {
		log.Println("not admin")
		return false
	}
	return true
}

func (b TGBotAPI) HasLeft(chatID int64, userID int) bool {
	member, err := b.bot.GetChatMember(tgbotapi.ChatConfigWithUser{
		ChatID: chatID,
		UserID: userID,
	})
	if err != nil {
		return false
	}
	if member.HasLeft() || member.WasKicked() {
		return true
	}
	return false
}

func (b TGBotAPI) UpdateCaption(chatID int64, msgID int, caption string, keyboard [][]model.KV) {
	editor := tgbotapi.NewEditMessageCaption(chatID, msgID, caption)
	editor.ParseMode = tgbotapi.ModeHTML
	markup := TransformKeyboard(keyboard)
	editor.ReplyMarkup = &markup
	_, err := b.bot.Send(editor)
	if err != nil {
		log.Printf("编辑消息: %d %d 失败: %+v", chatID, msgID, err)
	}
}

func (b TGBotAPI) UpdatePhoto(chatID int64, msgID int, caption string, keyboard [][]model.KV, img []byte) {
	_, err := utils.UpdateMsgPhoto(b.bot, chatID, msgID, caption, tgbotapi.ModeHTML, TransformKeyboard(keyboard), tgbotapi.FileBytes{
		Name:  strconv.FormatInt(time.Now().UnixNano(), 10),
		Bytes: img,
	})
	if err != nil {
		log.Println("图片更新失败: ", err)
	}
}

func (b TGBotAPI) AnswerCallback(callbackID, text string) {
	_, err := b.bot.AnswerCallbackQuery(tgbotapi.NewCallback(callbackID, text))
	if err != nil {
		log.Println("发送回调响应失败: ", err)
	}
}
