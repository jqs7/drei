package queue

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/jqs7/drei/pkg/utils"
)

type SQS struct {
	sqs *sqs.SQS
}

func NewSQS(p client.ConfigProvider) Interface {
	return &SQS{
		sqs: sqs.New(p),
	}
}

func (s SQS) SendMsg(ctx context.Context, queue string, body interface{}, delaySec int64) error {
	_, err := s.sqs.SendMessageWithContext(ctx, &sqs.SendMessageInput{
		DelaySeconds: aws.Int64(delaySec),
		MessageBody:  aws.String(utils.EncodeToString(body)),
		QueueUrl:     &queue,
	})
	return err
}
