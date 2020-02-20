package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/jqs7/drei/pkg/bot"
	"github.com/jqs7/drei/pkg/model"
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
	queue := aws.String(os.Getenv("DELETE_MSG_QUEUE"))

	lambda.Start(func(ctx context.Context, req events.SQSEvent) error {
		for _, v := range req.Records {
			msg := &model.MsgToDelete{}
			if err := json.Unmarshal([]byte(v.Body), msg); err != nil {
				log.Println(err)
				continue
			}
			if _, err := svc.DeleteMessageWithContext(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      queue,
				ReceiptHandle: &v.ReceiptHandle,
			}); err != nil {
				log.Println(err)
				return err
			}
		}
		log.Printf("%+v", req)
		return nil
	})
}
