package comm

import (
	"bytes"
	"testing"
)

// TestEncryptDecryptAES 测试AES加密和解密
func TestEncryptDecryptAES(t *testing.T) {
	// 测试数据
	plaintext := []byte("这是一条测试消息")
	password := []byte("test-password")

	// 加密
	ciphertext, err := encryptAES(plaintext, password)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 检查密文不等于明文
	if bytes.Equal(ciphertext, plaintext) {
		t.Error("密文等于明文，加密可能失败")
	}

	// 解密
	decrypted, err := decryptAES(ciphertext, password)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	// 检查解密后的数据等于原始数据
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("解密后的数据不等于原始数据: %s != %s", decrypted, plaintext)
	}
}

// TestEncryptDecryptAESWithWrongPassword 测试使用错误密码解密
func TestEncryptDecryptAESWithWrongPassword(t *testing.T) {
	// 测试数据
	plaintext := []byte("这是一条测试消息")
	password := []byte("test-password")
	wrongPassword := []byte("wrong-password")

	// 加密
	ciphertext, err := encryptAES(plaintext, password)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 使用错误密码解密
	_, err = decryptAES(ciphertext, wrongPassword)
	if err == nil {
		t.Error("使用错误密码解密应该失败，但成功了")
	}
}

// TestEncryptDecryptAESWithEmptyData 测试加密和解密空数据
func TestEncryptDecryptAESWithEmptyData(t *testing.T) {
	// 测试数据
	plaintext := []byte("")
	password := []byte("test-password")

	// 加密
	ciphertext, err := encryptAES(plaintext, password)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 解密
	decrypted, err := decryptAES(ciphertext, password)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	// 检查解密后的数据等于原始数据
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("解密后的数据不等于原始数据: %s != %s", decrypted, plaintext)
	}
}

// TestEncryptDecryptAESWithLargeData 测试加密和解密大数据
func TestEncryptDecryptAESWithLargeData(t *testing.T) {
	// 创建1MB的测试数据
	plaintext := make([]byte, 1024*1024)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}
	password := []byte("test-password")

	// 加密
	ciphertext, err := encryptAES(plaintext, password)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 解密
	decrypted, err := decryptAES(ciphertext, password)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	// 检查解密后的数据等于原始数据
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("解密后的数据不等于原始数据")
	}
}
