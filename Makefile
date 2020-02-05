.PHONY: build clean deploy bot
all: clean deploy

build: build-bot build-delete-msg build-captcha-count-down

build-bot: gen
	env GOOS=linux go build -ldflags="-s -w" -o bin/bot github.com/jqs7/drei/cmd/bot

build-delete-msg: gen
	env GOOS=linux go build -ldflags="-s -w" -o bin/delete-msg github.com/jqs7/drei/cmd/delete-msg

build-captcha-count-down: gen
	env GOOS=linux go build -ldflags="-s -w" -o bin/captcha-count-down github.com/jqs7/drei/cmd/captcha-count-down

gen:
	go generate ./...

clean:
	rm -rf ./bin ./vendor Gopkg.lock

deploy-prod: clean build
	sls deploy --stage prod

deploy: clean build
	sls deploy --verbose

bot: clean build-bot
	sls deploy -f bot --verbose

delete-msg: clean build-delete-msg
	sls deploy -f deleteMsg

captcha-count-down: clean build-captcha-count-down
	sls deploy -f captchaCountDown

