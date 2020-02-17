package model

import (
	"sort"
)

const (
	CallbackTypeRefresh     = "Refresh"
	CallbackTypePassThrough = "PassThrough"
	CallbackTypeKick        = "Kick"

	CallbackTypeDonateWX     = "DonateWX"
	CallbackTypeDonateAlipay = "DonateAlipay"
)

const (
	UserLinkTemplate = `<a href="tg://user?id=%d">%s</a>`
	EnterRoomMsg     = ` 你好，欢迎加入 %s，本群已启用新成员验证模式，请发送以上 <b>【四字】</b> 验证码内容。
在验证通过之前，你所发送的所有消息都将会被删除。
本消息将在 %%d 秒后失效，届时若未通过验证，你将被移出群组，且一分钟之内无法再加入本群。`
	HelpMsg = `欢迎使用进群验证码机器人
本机器人使用姿势：
将本机器人加入需要启用验证的群组，设置为管理员，并授予 Delete messages，Ban users 权限即可
本项目开源于：https://github.com/jqs7/drei
若本项目对你有所帮助，可点击 /donate 为本项目捐款`
	DonateMsg = `所捐款项将用于：
1. 作者的续命咖啡 ☕️
2. 支付服务器等设施费用`
)

const (
	CaptchaRefreshSecond = 15
)

type DonateKV struct {
	Key string
	URL string
}

var Donates = map[string]DonateKV{
	CallbackTypeDonateWX: {
		Key: "微信",
		URL: "wxp://f2f0OWfabxt-G2eVGJuF9psyiEvqiL3u3gxB",
	},
	CallbackTypeDonateAlipay: {
		Key: "支付宝",
		URL: "https://qr.alipay.com/fkx00824kg0dc3tf1sf4c2e",
	},
}

type KVArr []KV

func (K KVArr) Len() int {
	return len(K)
}

func (K KVArr) Less(i, j int) bool {
	return K[i].K < K[j].K
}

func (K KVArr) Swap(i, j int) {
	K[i], K[j] = K[j], K[i]
}

func DonatesKeyboard(donateType string) [][]KV {
	var kv []KV
	for k, v := range Donates {
		if k == donateType {
			v.Key = v.Key + " ❤️ "
		}
		kv = append(kv, KV{K: v.Key, V: k})
	}
	sort.Sort(KVArr(kv))
	return [][]KV{kv}
}
