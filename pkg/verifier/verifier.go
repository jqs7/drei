package verifier

import "context"

type Interface interface {
	OnNewMember(ctx context.Context, chatID int64, chatName string, newMemberID int, firstName, lastName string)
	Verify(ctx context.Context, chatID int64, userID, msgID int, msg string)
	OnLeftMember(ctx context.Context, chatID int64, leftMemberID int)
	OnCallbackQuery(ctx context.Context, chatID int64, msgID, fromUser int, callbackID, data string)
}
