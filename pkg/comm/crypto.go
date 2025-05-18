package comm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// deriveKey 从密码派生密钥
func deriveKey(password []byte) ([]byte, error) {
	// 使用SHA-256哈希密码，得到32字节的密钥
	// 这是一个简单的密钥派生方法，生产环境应该使用更安全的方法，如PBKDF2、bcrypt或Argon2
	key := sha256.Sum256(password)
	return key[:], nil
}

// encryptAES 使用AES-GCM加密数据
func encryptAES(plaintext, password []byte) ([]byte, error) {
	// 派生密钥
	key, err := deriveKey(password)
	if err != nil {
		return nil, fmt.Errorf("派生密钥失败: %w", err)
	}

	// 创建AES密码块
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES密码块失败: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM模式失败: %w", err)
	}

	// 创建随机数
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("创建随机数失败: %w", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptAES 使用AES-GCM解密数据
func decryptAES(ciphertext, password []byte) ([]byte, error) {
	// 派生密钥
	key, err := deriveKey(password)
	if err != nil {
		return nil, fmt.Errorf("派生密钥失败: %w", err)
	}

	// 创建AES密码块
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES密码块失败: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM模式失败: %w", err)
	}

	// 检查密文长度
	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("密文太短")
	}

	// 分离随机数和密文
	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密数据失败: %w", err)
	}

	return plaintext, nil
}

// EncryptAES 使用AES-GCM加密数据（公共方法）
func EncryptAES(plaintext, password []byte) ([]byte, error) {
	return encryptAES(plaintext, password)
}

// DecryptAES 使用AES-GCM解密数据（公共方法）
func DecryptAES(ciphertext, password []byte) ([]byte, error) {
	return decryptAES(ciphertext, password)
}
