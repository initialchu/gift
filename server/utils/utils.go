package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// JWT 配置（由 config 包在启动时注入）
var (
	jwtSecret      string
	jwtExpireHours int
)

// SetJWTConfig 由 config.InitConfig 调用，注入 JWT 配置
func SetJWTConfig(secret string, expireHours int) {
	jwtSecret = secret
	if expireHours <= 0 {
		expireHours = 24
	}
	jwtExpireHours = expireHours
}

// 使用bcrypt加密密码
func HashPassword(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), 12)
	return string(hash), err
}

// 创建JWT
func GenerateJWT(username string, role string) (string, error) {
	expireHours := jwtExpireHours
	if expireHours == 0 {
		expireHours = 24 // 兜底默认值
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(time.Hour * time.Duration(expireHours)).Unix(),
	})
	signedToken, err := token.SignedString([]byte(jwtSecret))

	return signedToken, err
}

// 验证密码
func CheckPWD(pwd string, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd))

	return err
}

// 验证JWT
func ParseJWT(tokenstr string) (string, string, error) {
	if len(tokenstr) > 7 && tokenstr[:7] == "Bearer " {
		tokenstr = tokenstr[7:]
	}
	token, err := jwt.Parse(tokenstr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return "", "", err
	}
	// 验证token有效性
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username, ok := claims["username"].(string)
		if !ok {
			return "", "", errors.New("invalid token claims")
		}
		role, ok := claims["role"].(string)
		if !ok {
			return "", "", errors.New("invalid token claims")
		}
		return username, role, nil
	}
	return "", "", errors.New("invalid token")
}
