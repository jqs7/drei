//go:generate go run github.com/golang/mock/mockgen -source=bot.go -package=bot -destination=mock.go Interface
package bot

import (
	"time"

	"github.com/jqs7/drei/pkg/model"
)

type Interface interface {
	SendMsg(chatID int64, msg string) (int, error)
	SetWebhook(addr string) error
	DeleteMsg(chatID int64, msgID int)
	SendImg(chatID int64, img []byte, caption string, keyboard [][]model.KV) (int, error)
	UpdateCaption(chatID int64, msgID int, caption string, keyboard [][]model.KV)
	UpdatePhoto(chatID int64, msgID int, caption string, keyboard [][]model.KV, img []byte)
	AnswerCallback(callbackID, text string)
	Kick(chatID int64, userID int, until time.Time)
	IsAdmin(chatID int64, userID int) bool
	HasLeft(chatID int64, userID int) bool
}
