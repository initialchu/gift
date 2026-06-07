package utils

//生成验证码

import (
	"github.com/mojocn/base64Captcha"
)

// 内存存储，验证码存储在内存中，适用于单机部署，自带过期时间，线程安全
var captchaStore = base64Captcha.DefaultMemStore

// 配置数字验证码
var captchaDriver = base64Captcha.NewDriverDigit(80, 240, 4, 0.7, 80)

// 生成验证码
func GenerateCaptcha() (id string, b64s string, answer string, err error) {
	c := base64Captcha.NewCaptcha(captchaDriver, captchaStore)
	return c.Generate()
}

// 验证验证码
func VerifyCaptcha(id, answer string) bool {
	return captchaStore.Verify(id, answer, true)
}
