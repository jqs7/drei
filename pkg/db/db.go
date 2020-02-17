package db

import (
	"context"

	"github.com/jqs7/drei/pkg/model"
	"golang.org/x/xerrors"
)

var ErrNotFound = xerrors.New("Record Not Found")

//go:generate go run github.com/golang/mock/mockgen -source=db.go -package=db -destination=mock.go IBlacklist
type IBlacklist interface {
	GetItem(ctx context.Context, chatID int64, userID int) (*model.Blacklist, error)
	UpdateIdx(ctx context.Context, chatID int64, userID, idx int)
	DeleteItem(ctx context.Context, chatID int64, userID int)
	CreateItem(ctx context.Context, item model.Blacklist)
	GetItemByMsgID(ctx context.Context, chatID int64, msgID int) (*model.Blacklist, error)
}
