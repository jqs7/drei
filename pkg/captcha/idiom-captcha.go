package captcha

import (
	"bytes"
	"encoding/json"
	"image/png"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/hanguofeng/gocaptcha"
	"github.com/jqs7/drei/pkg/model"
	"golang.org/x/xerrors"
)

type RandIdiomCaptcha struct {
	idioms            []model.Idiom
	captchaImgCfg     *gocaptcha.ImageConfig
	captchaImgFilters *gocaptcha.ImageFilterManager
}

func NewRandIdiomCaptcha(idiomPath, fontPath string) (Interface, error) {
	f, err := os.Open(idiomPath)
	if err != nil {
		return nil, xerrors.Errorf("读取 %s 文件失败: %w", idiomPath, err)
	}
	var idioms []model.Idiom
	if err := json.NewDecoder(f).Decode(&idioms); err != nil {
		return nil, xerrors.Errorf("解码 idiom 文件失败: %w", err)
	}
	tmp := idioms[:0]
	for _, p := range idioms {
		if len([]rune(p.Word)) == 4 {
			tmp = append(tmp, p)
		}
	}
	idioms = tmp

	filterConfig := new(gocaptcha.FilterConfig)
	filterConfig.Init()
	filterConfig.Filters = []string{
		gocaptcha.IMAGE_FILTER_NOISE_LINE,
		gocaptcha.IMAGE_FILTER_NOISE_POINT,
		gocaptcha.IMAGE_FILTER_STRIKE,
	}
	for _, v := range filterConfig.Filters {
		filterConfigGroup := new(gocaptcha.FilterConfigGroup)
		filterConfigGroup.Init()
		filterConfigGroup.SetItem("Num", "180")
		filterConfig.SetGroup(v, filterConfigGroup)
	}

	return &RandIdiomCaptcha{
		idioms: idioms,
		captchaImgCfg: &gocaptcha.ImageConfig{
			Width:    320,
			Height:   100,
			FontSize: 80,
			FontFiles: []string{
				filepath.Join(fontPath, "STFANGSO.ttf"),
				filepath.Join(fontPath, "STHEITI.ttf"),
				filepath.Join(fontPath, "STXIHEI.ttf"),
			},
		},
		captchaImgFilters: gocaptcha.CreateImageFilterManagerByConfig(filterConfig),
	}, nil
}

func (r RandIdiomCaptcha) GenRandImg() (model.Answer, []byte) {
	rIdx := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(r.idioms))

	cImg := gocaptcha.CreateCImage(r.captchaImgCfg)
	cImg.DrawString(r.idioms[rIdx].Word)
	for _, f := range r.captchaImgFilters.GetFilters() {
		f.Proc(cImg)
	}
	captchaBuffer := bytes.NewBuffer([]byte{})
	if err := png.Encode(captchaBuffer, cImg); err != nil {
		log.Fatalln("encode png img failed: ", err)
	}
	return model.Answer{Number: rIdx}, captchaBuffer.Bytes()
}

func (r RandIdiomCaptcha) VerifyAnswer(answer, request model.Answer) bool {
	return r.idioms[answer.Number].Word == request.String
}
