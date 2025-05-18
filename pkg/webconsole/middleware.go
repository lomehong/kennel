package webconsole

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// corsMiddleware 创建CORS中间件
func (c *Console) corsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("Origin")

		// 如果没有Origin头，使用通配符
		if origin == "" {
			origin = "*"
		}

		// 始终允许所有来源（开发环境）
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", origin)

		// 允许凭证
		ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		// 允许的方法
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")

		// 允许的头
		ctx.Writer.Header().Set("Access-Control-Allow-Headers",
			"Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, "+
				"Authorization, X-Requested-With, Origin, Accept, Access-Control-Request-Method, "+
				"Access-Control-Request-Headers")

		// 允许浏览器缓存预检请求结果
		ctx.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24小时

		// 处理预检请求
		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		// 记录请求信息
		c.logger.Debug("收到请求",
			"method", ctx.Request.Method,
			"path", ctx.Request.URL.Path,
			"origin", origin)

		ctx.Next()
	}
}

// 会话存储
var (
	sessions    = make(map[string]sessionInfo)
	sessionLock sync.RWMutex
)

// 会话信息
type sessionInfo struct {
	username  string
	expiresAt time.Time
}

// authMiddleware 创建认证中间件
func (c *Console) authMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 跳过静态文件和登录API
		if strings.HasPrefix(ctx.Request.URL.Path, "/assets/") ||
			ctx.Request.URL.Path == c.config.APIPrefix+"/login" {
			ctx.Next()
			return
		}

		// 检查会话Cookie
		sessionID, err := ctx.Cookie("session_id")
		if err == nil {
			// 验证会话
			sessionLock.RLock()
			session, exists := sessions[sessionID]
			sessionLock.RUnlock()

			if exists && time.Now().Before(session.expiresAt) {
				// 会话有效，更新过期时间
				sessionLock.Lock()
				session.expiresAt = time.Now().Add(c.config.SessionTimeout)
				sessions[sessionID] = session
				sessionLock.Unlock()

				// 设置用户信息
				ctx.Set("username", session.username)
				ctx.Next()
				return
			}
		}

		// 检查基本认证
		username, password, hasAuth := ctx.Request.BasicAuth()
		if hasAuth && username == c.config.Username && password == c.config.Password {
			// 创建新会话
			sessionID := uuid.New().String()
			expiresAt := time.Now().Add(c.config.SessionTimeout)

			sessionLock.Lock()
			sessions[sessionID] = sessionInfo{
				username:  username,
				expiresAt: expiresAt,
			}
			sessionLock.Unlock()

			// 设置会话Cookie
			ctx.SetCookie("session_id", sessionID, int(c.config.SessionTimeout.Seconds()), "/", "", c.config.EnableHTTPS, true)

			// 设置用户信息
			ctx.Set("username", username)
			ctx.Next()
			return
		}

		// 认证失败
		ctx.Header("WWW-Authenticate", `Basic realm="Web Console"`)
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "未授权访问",
		})
	}
}

// csrfToken 存储
var (
	csrfTokens    = make(map[string]time.Time)
	csrfTokenLock sync.RWMutex
)

// csrfMiddleware 创建CSRF中间件
func (c *Console) csrfMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 跳过GET和OPTIONS请求
		if ctx.Request.Method == "GET" || ctx.Request.Method == "OPTIONS" {
			// 生成CSRF令牌
			token := uuid.New().String()

			// 存储令牌
			csrfTokenLock.Lock()
			csrfTokens[token] = time.Now().Add(24 * time.Hour)
			csrfTokenLock.Unlock()

			// 设置令牌头
			ctx.Header("X-CSRF-Token", token)
			ctx.Next()
			return
		}

		// 验证CSRF令牌
		token := ctx.GetHeader("X-CSRF-Token")
		if token == "" {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "缺少CSRF令牌",
			})
			return
		}

		// 检查令牌是否有效
		csrfTokenLock.RLock()
		expiresAt, exists := csrfTokens[token]
		csrfTokenLock.RUnlock()

		if !exists || time.Now().After(expiresAt) {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "无效的CSRF令牌",
			})
			return
		}

		// 删除已使用的令牌
		csrfTokenLock.Lock()
		delete(csrfTokens, token)
		csrfTokenLock.Unlock()

		ctx.Next()
	}
}

// 请求限制存储
var (
	rateLimits    = make(map[string][]time.Time)
	rateLimitLock sync.RWMutex
)

// rateLimitMiddleware 创建请求限制中间件
func (c *Console) rateLimitMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 获取客户端IP
		clientIP := ctx.ClientIP()

		// 清理过期的请求记录
		now := time.Now()
		rateLimitLock.Lock()

		// 获取该IP的请求记录
		times, exists := rateLimits[clientIP]
		if !exists {
			times = make([]time.Time, 0)
		}

		// 只保留最近1分钟的请求记录
		validTimes := make([]time.Time, 0)
		for _, t := range times {
			if now.Sub(t) < time.Minute {
				validTimes = append(validTimes, t)
			}
		}

		// 检查是否超过限制
		if len(validTimes) >= c.config.RateLimit {
			rateLimitLock.Unlock()
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
			})
			return
		}

		// 添加当前请求
		validTimes = append(validTimes, now)
		rateLimits[clientIP] = validTimes
		rateLimitLock.Unlock()

		ctx.Next()
	}
}
