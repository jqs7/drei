module github.com/jqs7/drei

go 1.13

require (
	github.com/aws/aws-lambda-go v1.13.3
	github.com/aws/aws-sdk-go v1.28.4
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible
	github.com/golang/mock v1.4.0
	github.com/hanguofeng/gocaptcha v1.0.7
	github.com/skip2/go-qrcode v0.0.0-20191027152451-9434209cb086
	github.com/stretchr/testify v1.4.0
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
)

replace github.com/hanguofeng/gocaptcha v1.0.7 => github.com/jqs7/gocaptcha v1.0.8-0.20181014100812-c7bcbe23fde4
