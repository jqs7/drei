package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jqs7/drei/pkg/bot"
	"github.com/jqs7/drei/pkg/captcha"
	"github.com/jqs7/drei/pkg/db"
	"github.com/jqs7/drei/pkg/model"
	"github.com/jqs7/drei/pkg/queue"
	"github.com/jqs7/drei/pkg/verifier"
	"github.com/skip2/go-qrcode"
)

var RespOK = &events.APIGatewayProxyResponse{
	Headers:    map[string]string{},
	StatusCode: http.StatusOK,
	Body:       "True",
}

func main() {
	botAPI, err := bot.NewAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatalf("%+v", err)
	}

	sess, err := session.NewSession()
	if err != nil {
		log.Fatalln("init aws session: ", err)
	}

	idiomCaptcha, err := captcha.NewRandIdiomCaptcha("/opt/idiom.json", "/opt/fonts")
	if err != nil {
		log.Fatalf("%+v", err)
	}

	idiomVerifier, err := verifier.NewIdiomVerifier(botAPI, queue.NewSQS(sess),
		db.NewBlacklist(sess, os.Getenv("USERS_TABLE_NAME")), idiomCaptcha,
	)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	lambda.Start(func(ctx context.Context, req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
		switch req.Path {
		case "/":
			update := &tgbotapi.Update{}
			if err := json.Unmarshal([]byte(req.Body), update); err != nil {
				return RespOK, nil
			}

			if update.CallbackQuery != nil {
				switch update.CallbackQuery.Message.Chat.Type {
				case "group", "supergroup":
					idiomVerifier.OnCallbackQuery(ctx,
						update.CallbackQuery.Message.Chat.ID,
						update.CallbackQuery.Message.MessageID,
						update.CallbackQuery.From.ID,
						update.CallbackQuery.ID,
						update.CallbackQuery.Data,
					)
				case "private":
					log.Println(update.CallbackQuery.Data)
					donateOpt, ok := model.Donates[update.CallbackQuery.Data]
					if !ok {
						break
					}
					log.Println(update.CallbackQuery.Data)
					b, err := qrcode.Encode(donateOpt.URL, qrcode.Medium, 256)
					if err != nil {
						log.Fatalln(err)
					}
					botAPI.UpdatePhoto(
						update.CallbackQuery.Message.Chat.ID,
						update.CallbackQuery.Message.MessageID,
						model.DonateMsg,
						model.DonatesKeyboard(update.CallbackQuery.Data), b,
					)
				}
				return RespOK, nil
			}

			if update.Message == nil {
				return RespOK, nil
			}

			if time.Since(update.Message.Time()) > time.Hour {
				return RespOK, nil
			}

			if update.Message.NewChatMembers != nil {
				botAPI.DeleteMsg(update.Message.Chat.ID, update.Message.MessageID)
				for _, v := range *update.Message.NewChatMembers {
					if v.IsBot {
						continue
					}
					idiomVerifier.OnNewMember(ctx,
						update.Message.Chat.ID,
						update.Message.Chat.Title,
						v.ID, v.FirstName, v.LastName,
					)
				}
			}

			if update.Message.LeftChatMember != nil {
				botAPI.DeleteMsg(update.Message.Chat.ID, update.Message.MessageID)
				idiomVerifier.OnLeftMember(ctx, update.Message.Chat.ID, update.Message.LeftChatMember.ID)
			}

			switch update.Message.Chat.Type {
			case "group", "supergroup":
				idiomVerifier.Verify(ctx,
					update.Message.Chat.ID,
					update.Message.From.ID,
					update.Message.MessageID,
					update.Message.Text,
				)
			case "private":
				switch update.Message.Text {
				case "/help", "/start":
					_, _ = botAPI.SendMsg(update.Message.Chat.ID, model.HelpMsg)
				case "/donate":
					donateOpt, ok := model.Donates[model.CallbackTypeDonateWX]
					if !ok {
						break
					}
					b, err := qrcode.Encode(donateOpt.URL, qrcode.Medium, 256)
					if err != nil {
						log.Fatalln(err)
					}
					_, _ = botAPI.SendImg(update.Message.Chat.ID,
						b, model.DonateMsg,
						model.DonatesKeyboard(model.CallbackTypeDonateWX),
					)
				}
			}
			return RespOK, nil
		case "/hook":
			hookAddr := "https://" + req.Headers["Host"] + "/" + req.RequestContext.Stage
			if err := botAPI.SetWebhook(hookAddr); err != nil {
				return &events.APIGatewayProxyResponse{
					Headers:    map[string]string{},
					StatusCode: http.StatusInternalServerError,
					Body:       fmt.Sprintf("set webhook %s failed: %v", hookAddr, err),
				}, nil
			}
			return RespOK, nil
		default:
			return nil, errors.New("path not found")
		}
	})
}
