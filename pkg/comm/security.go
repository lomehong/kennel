package comm

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"
)

// createTLSConfig 创建TLS配置
func (c *Client) createTLSConfig() (*tls.Config, error) {
	// 创建基本的TLS配置
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// 如果不验证服务器证书，设置InsecureSkipVerify为true
	if !c.config.Security.VerifyServerCert {
		tlsConfig.InsecureSkipVerify = true
		c.logger.Warn("TLS配置不验证服务器证书，这可能存在安全风险")
	}

	// 如果提供了CA证书，加载CA证书
	if c.config.Security.CACertFile != "" {
		// 加载CA证书
		caCert, err := ioutil.ReadFile(c.config.Security.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("读取CA证书失败: %w", err)
		}

		// 创建CA证书池
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("解析CA证书失败")
		}

		// 设置CA证书池
		tlsConfig.RootCAs = caCertPool
	}

	// 如果提供了客户端证书和私钥，加载客户端证书
	if c.config.Security.ClientCertFile != "" && c.config.Security.ClientKeyFile != "" {
		// 加载客户端证书和私钥
		cert, err := tls.LoadX509KeyPair(c.config.Security.ClientCertFile, c.config.Security.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("加载客户端证书和私钥失败: %w", err)
		}

		// 设置客户端证书
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// createAuthHeader 创建认证头
func (c *Client) createAuthHeader() (map[string]string, error) {
	// 创建认证头
	headers := make(map[string]string)

	// 根据认证类型创建认证头
	switch strings.ToLower(c.config.Security.AuthType) {
	case "basic":
		// 检查用户名和密码
		if c.config.Security.Username == "" || c.config.Security.Password == "" {
			return nil, fmt.Errorf("basic认证需要用户名和密码")
		}

		// 创建basic认证头
		auth := c.config.Security.Username + ":" + c.config.Security.Password
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		headers["Authorization"] = "Basic " + encodedAuth

	case "token":
		// 检查令牌
		if c.config.Security.AuthToken == "" {
			return nil, fmt.Errorf("token认证需要令牌")
		}

		// 创建token认证头
		headers["Authorization"] = "Bearer " + c.config.Security.AuthToken

	case "jwt":
		// 检查令牌
		if c.config.Security.AuthToken == "" {
			return nil, fmt.Errorf("jwt认证需要令牌")
		}

		// 创建JWT认证头
		headers["Authorization"] = "Bearer " + c.config.Security.AuthToken

	default:
		return nil, fmt.Errorf("不支持的认证类型: %s", c.config.Security.AuthType)
	}

	return headers, nil
}

// encryptMessage 加密消息
func (c *Client) encryptMessage(data []byte) ([]byte, error) {
	// 如果未启用加密，直接返回原始数据
	if !c.config.Security.EnableEncryption {
		return data, nil
	}

	// 检查加密密钥
	if c.config.Security.EncryptionKey == "" {
		return nil, fmt.Errorf("加密需要密钥")
	}

	// 使用AES加密
	encryptedData, err := encryptAES(data, []byte(c.config.Security.EncryptionKey))
	if err != nil {
		return nil, fmt.Errorf("加密消息失败: %w", err)
	}

	return encryptedData, nil
}

// decryptMessage 解密消息
func (c *Client) decryptMessage(data []byte) ([]byte, error) {
	// 如果未启用加密，直接返回原始数据
	if !c.config.Security.EnableEncryption {
		return data, nil
	}

	// 检查加密密钥
	if c.config.Security.EncryptionKey == "" {
		return nil, fmt.Errorf("解密需要密钥")
	}

	// 使用AES解密
	decryptedData, err := decryptAES(data, []byte(c.config.Security.EncryptionKey))
	if err != nil {
		return nil, fmt.Errorf("解密消息失败: %w", err)
	}

	return decryptedData, nil
}

// SetServerURL 设置服务器URL
func (c *Client) SetServerURL(url string) {
	c.config.ServerURL = url
}
