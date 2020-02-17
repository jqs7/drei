.PHONY: build clean deploy bot

STAGE := dev

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

deploy: clean build
	sls deploy --verbose --stage ${STAGE}

bot: clean build-bot
	sls deploy -f bot --verbose ${STAGE}

delete-msg: clean build-delete-msg
	sls deploy -f deleteMsg --stage ${STAGE}

captcha-count-down: clean build-captcha-count-down
	sls deploy -f captchaCountDown --stage ${STAGE}

