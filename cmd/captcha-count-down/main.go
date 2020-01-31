package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/jqs7/drei/pkg/bot"
	"github.com/jqs7/drei/pkg/db"
	"github.com/jqs7/drei/pkg/model"
	"github.com/jqs7/drei/pkg/verifier"
)

func main() {
	botAPI, err := bot.NewAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatalf("%+v", err)
	}

	sess, err := session.NewSession()
	if err != nil {
		log.Fatalln("init aws session: ", err)
	}
	svc := sqs.New(sess)
	queueName := aws.String(os.Getenv("CAPTCHA_COUNTDOWN_QUEUE"))
	blacklist := db.NewBlacklist(sess, os.Getenv("USERS_TABLE_NAME"))

	lambda.Start(func(ctx context.Context, req events.SQSEvent) error {
		for _, v := range req.Records {
			msg := &model.CountdownMsg{}
			if err := json.Unmarshal([]byte(v.Body), msg); err != nil {
				log.Println(err)
				continue
			}
			if _, err := svc.DeleteMessageWithContext(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      queueName,
				ReceiptHandle: &v.ReceiptHandle,
			}); err != nil {
				log.Println(err)
				continue
			}
			item, err := blacklist.GetItem(ctx, msg.ChatID, msg.UserID)
			if err != nil {
				log.Println(err)
				continue
			}
			if botAPI.HasLeft(msg.ChatID, msg.UserID) {
				botAPI.DeleteMsg(msg.ChatID, item.MsgID)
				blacklist.DeleteItem(ctx, msg.ChatID, msg.UserID)
				continue
			}
			if item.ExpireAt.Before(time.Now()) {
				botAPI.DeleteMsg(msg.ChatID, item.MsgID)
				botAPI.Kick(msg.ChatID, msg.UserID, time.Now().Add(time.Minute))
				blacklist.DeleteItem(ctx, msg.ChatID, msg.UserID)
				continue
			}
			botAPI.UpdateCaption(msg.ChatID, item.MsgID,
				fmt.Sprintf(item.UserLink+" "+item.MsgTemplate, time.Until(item.ExpireAt)/time.Second),
				verifier.InlineKeyboard,
			)
			_, err = svc.SendMessageWithContext(ctx, &sqs.SendMessageInput{
				DelaySeconds: aws.Int64(5),
				MessageBody:  aws.String(v.Body),
				QueueUrl:     queueName,
			})
			if err != nil {
				log.Println("send count down msg failed: ", err)
			}
		}
		log.Printf("%+v", req)
		return nil
	})
}
