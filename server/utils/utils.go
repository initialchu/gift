package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// 使用bcrypt加密密码
func HashPassword(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), 12)
	return string(hash), err
}

// 创建JWT
func GenerateJWT(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // 设置过期时间为24小时
	})
	signedToken, err := token.SignedString([]byte("wushijiazu")) // 替换为你的密钥

	return signedToken, err
}

// 验证JWT
func CheckPWD(pwd string, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd))

	return err
}
