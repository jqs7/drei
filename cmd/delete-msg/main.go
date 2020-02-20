package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jqs7/drei/pkg/bot"
	"github.com/jqs7/drei/pkg/model"
)

func main() {
	botAPI, err := bot.NewAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatalf("%+v", err)
	}

	lambda.Start(func(ctx context.Context, req events.SQSEvent) error {
		for _, v := range req.Records {
			msg := &model.MsgToDelete{}
			if err := json.Unmarshal([]byte(v.Body), msg); err != nil {
				log.Println(err)
				continue
			}
			botAPI.DeleteMsg(msg.ChatID, msg.MsgID)
		}
		log.Printf("%+v", req)
		return nil
	})
}
