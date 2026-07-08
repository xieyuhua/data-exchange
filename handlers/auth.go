package handlers

import (
	"net/http"
	"strings"
	"time"

	"data-exchange/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// jwtSecret JWT 签名密钥（演示用；生产环境应通过配置/环境变量注入并定期轮换）
var jwtSecret = []byte("data-exchange-secret-key-change-me")

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login 账号密码登录，校验成功后返回 JWT
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if req.Username == "" || req.Password == "" {
		fail(c, "用户名和密码不能为空")
		return
	}

	user, err := h.App.UserRepo().GetByUsername(req.Username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			fail(c, "用户名或密码错误")
			return
		}
		fail(c, "登录失败: "+err.Error())
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		fail(c, "用户名或密码错误")
		return
	}

	token, err := generateToken(user.ID, user.Username, user.Role)
	if err != nil {
		fail(c, "生成令牌失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"token":    token,
		"username": user.Username,
		"nickname": user.Nickname,
		"role":     user.Role,
	})
}

// Me 返回当前登录用户信息
func (h *Handler) Me(c *gin.Context) {
	uid, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")
	success(c, gin.H{
		"user_id":  uid,
		"username": username,
		"role":     role,
	})
}

// generateToken 生成有效期 7 天的 JWT
func generateToken(userID int64, username, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// AuthMiddleware JWT 鉴权中间件：从 Authorization: Bearer <token> 解析并校验
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusOK, models.APIResponse{Code: 401, Message: "未登录或登录已过期"})
			c.Abort()
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusOK, models.APIResponse{Code: 401, Message: "鉴权头格式错误"})
			c.Abort()
			return
		}
		tokenStr := parts[1]

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusOK, models.APIResponse{Code: 401, Message: "登录已过期，请重新登录"})
			c.Abort()
			return
		}

		if uid, ok := claims["user_id"].(float64); ok {
			c.Set("user_id", int64(uid))
		}
		c.Set("username", claims["username"])
		c.Set("role", claims["role"])
		c.Next()
	}
}
