package db

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/jqs7/drei/pkg/model"
)

type Blacklist struct {
	db        *dynamodb.DynamoDB
	tableName *string
}

func NewBlacklist(p client.ConfigProvider, tableName string) IBlacklist {
	return &Blacklist{
		db:        dynamodb.New(p),
		tableName: &tableName,
	}
}

func (bl Blacklist) GetItem(ctx context.Context, chatID int64, userID int) (*model.Blacklist, error) {
	result, err := bl.db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: bl.tableName,
		Key: map[string]*dynamodb.AttributeValue{
			"chatID": {
				N: aws.String(strconv.FormatInt(chatID, 10)),
			},
			"userID": {
				N: aws.String(strconv.Itoa(userID)),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(result.Item) == 0 {
		return nil, ErrNotFound
	}
	return bl.unmarshal(result.Item), nil
}

func (bl Blacklist) i64ToStr(i int64) *string {
	return aws.String(strconv.FormatInt(i, 10))
}

func (bl Blacklist) iToStr(i int) *string {
	return aws.String(strconv.Itoa(i))
}

func (bl Blacklist) indexKeys(chatID int64, userID int) map[string]*dynamodb.AttributeValue {
	return map[string]*dynamodb.AttributeValue{
		"chatID": {
			N: bl.i64ToStr(chatID),
		},
		"userID": {
			N: bl.iToStr(userID),
		},
	}
}

func (bl Blacklist) UpdateIdx(ctx context.Context, chatID int64, userID, idx int) {
	_, err := bl.db.UpdateItemWithContext(ctx, &dynamodb.UpdateItemInput{
		TableName: bl.tableName,
		Key:       bl.indexKeys(chatID, userID),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":idx": {N: bl.iToStr(idx)},
		},
		UpdateExpression: aws.String("SET idx = :idx"),
	})
	if err != nil {
		log.Fatalln("update item failed:  ", err)
	}
}

func (bl Blacklist) CreateItem(ctx context.Context, item model.Blacklist) {
	_, err := bl.db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		Item:      bl.marshalItem(item),
		TableName: bl.tableName,
	})
	if err != nil {
		log.Fatalln("create item failed: ", err)
	}
}

func (bl Blacklist) marshalItem(item model.Blacklist) map[string]*dynamodb.AttributeValue {
	return map[string]*dynamodb.AttributeValue{
		"userID": {
			N: aws.String(strconv.Itoa(item.UserID)),
		},
		"chatID": {
			N: aws.String(strconv.FormatInt(item.ChatID, 10)),
		},
		"idx": {
			N: aws.String(strconv.Itoa(item.Index)),
		},
		"msgID": {
			N: aws.String(strconv.Itoa(item.MsgID)),
		},
		"expireAt": {
			N: aws.String(strconv.FormatInt(item.ExpireAt.UnixNano(), 10)),
		},
		"userLink": {
			S: &item.UserLink,
		},
		"msgTemplate": {
			S: &item.MsgTemplate,
		},
	}
}

func (bl Blacklist) DeleteItem(ctx context.Context, chatID int64, userID int) {
	_, err := bl.db.DeleteItemWithContext(ctx, &dynamodb.DeleteItemInput{
		TableName: bl.tableName,
		Key:       bl.indexKeys(chatID, userID),
	})
	if err != nil {
		log.Fatalln("delete item failed: ", err)
	}
}

func (bl Blacklist) GetItemByMsgID(ctx context.Context, chatID int64, msgID int) (*model.Blacklist, error) {
	rst, err := bl.db.QueryWithContext(ctx, &dynamodb.QueryInput{
		TableName:              bl.tableName,
		KeyConditionExpression: aws.String("chatID = :chatID"),
		FilterExpression:       aws.String("msgID = :msgID"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":chatID": {N: bl.i64ToStr(chatID)},
			":msgID":  {N: bl.iToStr(msgID)},
		},
		Limit: aws.Int64(1),
	})
	if err != nil {
		return nil, err
	}
	if *rst.Count == 0 {
		return nil, ErrNotFound
	}
	return bl.unmarshal(rst.Items[0]), nil
}

func (bl Blacklist) unmarshal(item map[string]*dynamodb.AttributeValue) *model.Blacklist {
	chatID, err := strconv.ParseInt(*item["chatID"].N, 10, 64)
	if err != nil {
		log.Fatalf("convert chatID %s to int64 failed", *item["chatID"].N)
	}
	userID, err := strconv.Atoi(*item["userID"].N)
	if err != nil {
		log.Fatalf("convert userID %s to int failed", *item["userID"].N)
	}
	msgID, err := strconv.Atoi(*item["msgID"].N)
	if err != nil {
		log.Fatalf("convert msgID %s to int failed", *item["msgID"].N)
	}
	idx, err := strconv.Atoi(*item["idx"].N)
	if err != nil {
		log.Fatalf("convert idx %s to int failed", *item["idx"].N)
	}
	expireAt, err := strconv.ParseInt(*item["expireAt"].N, 10, 64)
	if err != nil {
		log.Fatalf("convert expireAt %s to int64 failed", *item["expireAt"].N)
	}
	return &model.Blacklist{
		ChatID:      chatID,
		UserID:      userID,
		MsgID:       msgID,
		Index:       idx,
		ExpireAt:    time.Unix(0, expireAt),
		MsgTemplate: *item["msgTemplate"].S,
		UserLink:    *item["userLink"].S,
	}
}
