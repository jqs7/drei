package captcha

import "github.com/jqs7/drei/pkg/model"

//go:generate go run github.com/golang/mock/mockgen -source=captcha.go -package=captcha -destination=captcha_mock.go Interface
type Interface interface {
	GenRandImg() (model.Answer, []byte)
	VerifyAnswer(answer, request model.Answer) bool
}
