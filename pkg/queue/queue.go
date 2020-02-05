//go:generate go run github.com/golang/mock/mockgen -source=queue.go -package=queue -destination=mock.go Interface
package queue

import "context"

type Interface interface {
	SendMsg(ctx context.Context, queue string, body interface{}, delaySec int64) error
}
