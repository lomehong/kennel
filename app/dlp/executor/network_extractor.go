package executor

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/lomehong/kennel/app/dlp/engine"
	"github.com/lomehong/kennel/app/dlp/parser"
	"github.com/lomehong/kennel/pkg/logging"
)

// NetworkInfoExtractor 网络信息提取器
type NetworkInfoExtractor struct {
	logger    logging.Logger
	dnsCache  map[string]string // IP到域名的缓存
	cacheTime map[string]time.Time
}

// NewNetworkInfoExtractor 创建网络信息提取器
func NewNetworkInfoExtractor(logger logging.Logger) *NetworkInfoExtractor {
	return &NetworkInfoExtractor{
		logger:    logger,
		dnsCache:  make(map[string]string),
		cacheTime: make(map[string]time.Time),
	}
}

// ExtractNetworkInfo 从决策上下文中提取网络信息
func (nie *NetworkInfoExtractor) ExtractNetworkInfo(decision *engine.PolicyDecision) *NetworkInfo {
	networkInfo := &NetworkInfo{}

	// 从PacketInfo提取基础网络信息
	if decision.Context != nil && decision.Context.PacketInfo != nil {
		packetInfo := decision.Context.PacketInfo
		networkInfo.SourcePort = packetInfo.SourcePort
		networkInfo.DestPort = packetInfo.DestPort

		// 尝试解析目标域名
		if packetInfo.DestIP != nil {
			networkInfo.DestDomain = nie.resolveDomain(packetInfo.DestIP.String())
		}
	}

	// 从ParsedData提取HTTP/HTTPS信息
	if decision.Context != nil && decision.Context.ParsedData != nil {
		parsedData := decision.Context.ParsedData
		networkInfo.RequestURL = nie.extractRequestURL(parsedData)
		networkInfo.RequestData = nie.extractRequestData(parsedData)
	}

	return networkInfo
}

// NetworkInfo 网络信息结构体
type NetworkInfo struct {
	SourcePort  uint16
	DestPort    uint16
	DestDomain  string
	RequestURL  string
	RequestData string
}

// extractRequestURL 提取完整的请求URL
func (nie *NetworkInfoExtractor) extractRequestURL(parsedData *parser.ParsedData) string {
	if parsedData == nil {
		return ""
	}

	// 如果ParsedData中已经有URL，直接使用
	if parsedData.URL != "" {
		// 如果URL不包含协议，需要补充
		if !strings.Contains(parsedData.URL, "://") {
			// 从Headers或Metadata中获取Host信息
			host := ""
			if parsedData.Headers != nil {
				host = parsedData.Headers["Host"]
				if host == "" {
					host = parsedData.Headers["host"]
				}
			}

			if host == "" && parsedData.Metadata != nil {
				if h, ok := parsedData.Metadata["host"].(string); ok {
					host = h
				}
			}

			if host != "" {
				// 确定协议
				protocol := "http"
				if parsedData.Protocol == "HTTPS" || parsedData.Protocol == "https" {
					protocol = "https"
				} else if parsedData.Headers != nil {
					if parsedData.Headers["X-Forwarded-Proto"] == "https" {
						protocol = "https"
					}
				}

				// 构建完整URL
				if strings.HasPrefix(parsedData.URL, "/") {
					return fmt.Sprintf("%s://%s%s", protocol, host, parsedData.URL)
				} else {
					return fmt.Sprintf("%s://%s/%s", protocol, host, parsedData.URL)
				}
			}
		}
		return parsedData.URL
	}

	// 尝试从Headers和Metadata中构建URL
	if parsedData.Headers != nil {
		host := parsedData.Headers["Host"]
		if host == "" {
			host = parsedData.Headers["host"]
		}

		if host != "" {
			// 确定协议
			protocol := "http"
			if parsedData.Protocol == "HTTPS" || parsedData.Protocol == "https" {
				protocol = "https"
			} else if parsedData.Headers["X-Forwarded-Proto"] == "https" {
				protocol = "https"
			}

			// 构建基础URL
			baseURL := fmt.Sprintf("%s://%s", protocol, host)

			// 添加路径和查询参数（如果有的话）
			if parsedData.Metadata != nil {
				if requestURI, ok := parsedData.Metadata["request_uri"].(string); ok && requestURI != "" {
					// 使用完整的请求URI
					if strings.HasPrefix(requestURI, "/") {
						baseURL += requestURI
					} else {
						baseURL += "/" + requestURI
					}
				} else {
					// 分别处理路径和查询参数
					if path, ok := parsedData.Metadata["path"].(string); ok && path != "" {
						if strings.HasPrefix(path, "/") {
							baseURL += path
						} else {
							baseURL += "/" + path
						}
					}
					if query, ok := parsedData.Metadata["query"].(string); ok && query != "" {
						baseURL += "?" + query
					}
				}
			}

			return baseURL
		}
	}

	return ""
}

// extractRequestData 提取请求数据摘要
func (nie *NetworkInfoExtractor) extractRequestData(parsedData *parser.ParsedData) string {
	if parsedData == nil || len(parsedData.Body) == 0 {
		return ""
	}

	// 限制数据大小，避免日志过大
	const maxDataSize = 1024
	bodyData := parsedData.Body
	if len(bodyData) > maxDataSize {
		bodyData = bodyData[:maxDataSize]
	}

	// 根据Content-Type处理不同类型的数据
	contentType := parsedData.ContentType
	if contentType == "" && parsedData.Headers != nil {
		contentType = parsedData.Headers["Content-Type"]
		if contentType == "" {
			contentType = parsedData.Headers["content-type"]
		}
	}

	switch {
	case strings.Contains(contentType, "application/json"):
		return nie.extractJSONData(bodyData)
	case strings.Contains(contentType, "application/x-www-form-urlencoded"):
		return nie.extractFormData(bodyData)
	case strings.Contains(contentType, "multipart/form-data"):
		return nie.extractMultipartData(bodyData)
	case strings.Contains(contentType, "text/"):
		return nie.extractTextData(bodyData)
	default:
		// 对于其他类型，返回数据摘要
		return nie.createDataSummary(bodyData, contentType)
	}
}

// extractJSONData 提取JSON数据的关键信息
func (nie *NetworkInfoExtractor) extractJSONData(data []byte) string {
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		// 如果解析失败，返回原始数据的前部分
		return nie.sanitizeData(string(data))
	}

	// 提取关键字段并脱敏
	result := make(map[string]interface{})
	sensitiveFields := []string{"password", "token", "secret", "key", "auth", "credential"}

	for key, value := range jsonData {
		lowerKey := strings.ToLower(key)
		isSensitive := false

		for _, sensitive := range sensitiveFields {
			if strings.Contains(lowerKey, sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			result[key] = "***REDACTED***"
		} else {
			// 限制值的长度
			if str, ok := value.(string); ok && len(str) > 100 {
				result[key] = str[:100] + "..."
			} else {
				result[key] = value
			}
		}
	}

	// 转换回JSON字符串
	if resultBytes, err := json.Marshal(result); err == nil {
		return string(resultBytes)
	}

	return nie.sanitizeData(string(data))
}

// extractFormData 提取表单数据
func (nie *NetworkInfoExtractor) extractFormData(data []byte) string {
	formData := string(data)

	// 解析表单数据
	values, err := url.ParseQuery(formData)
	if err != nil {
		return nie.sanitizeData(formData)
	}

	// 脱敏处理
	result := make(map[string]string)
	sensitiveFields := []string{"password", "token", "secret", "key", "auth", "credential"}

	for key, valueList := range values {
		lowerKey := strings.ToLower(key)
		isSensitive := false

		for _, sensitive := range sensitiveFields {
			if strings.Contains(lowerKey, sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			result[key] = "***REDACTED***"
		} else if len(valueList) > 0 {
			value := valueList[0]
			if len(value) > 100 {
				result[key] = value[:100] + "..."
			} else {
				result[key] = value
			}
		}
	}

	// 构建结果字符串
	var parts []string
	for key, value := range result {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}

	return strings.Join(parts, "&")
}

// extractMultipartData 提取多部分表单数据摘要
func (nie *NetworkInfoExtractor) extractMultipartData(data []byte) string {
	dataStr := string(data)

	// 查找文件上传信息
	filePattern := regexp.MustCompile(`filename="([^"]+)"`)
	matches := filePattern.FindAllStringSubmatch(dataStr, -1)

	if len(matches) > 0 {
		var filenames []string
		for _, match := range matches {
			if len(match) > 1 {
				filenames = append(filenames, match[1])
			}
		}
		return fmt.Sprintf("multipart/form-data with files: %s", strings.Join(filenames, ", "))
	}

	return "multipart/form-data"
}

// extractTextData 提取文本数据
func (nie *NetworkInfoExtractor) extractTextData(data []byte) string {
	text := string(data)
	return nie.sanitizeData(text)
}

// createDataSummary 创建数据摘要
func (nie *NetworkInfoExtractor) createDataSummary(data []byte, contentType string) string {
	return fmt.Sprintf("%s (%d bytes)", contentType, len(data))
}

// sanitizeData 数据脱敏处理
func (nie *NetworkInfoExtractor) sanitizeData(data string) string {
	// 限制长度
	if len(data) > 500 {
		data = data[:500] + "..."
	}

	// 脱敏敏感信息
	sensitivePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|pwd|secret|token|key|auth)["\s]*[:=]["\s]*([^"\s,}]+)`),
		regexp.MustCompile(`(?i)(email)["\s]*[:=]["\s]*([^"\s,}]+@[^"\s,}]+)`),
		regexp.MustCompile(`(?i)(phone|mobile)["\s]*[:=]["\s]*([0-9\-\+\(\)\s]+)`),
	}

	for _, pattern := range sensitivePatterns {
		data = pattern.ReplaceAllString(data, `$1: "***REDACTED***"`)
	}

	return data
}

// resolveDomain 解析IP地址对应的域名
func (nie *NetworkInfoExtractor) resolveDomain(ip string) string {
	// 检查缓存
	if domain, exists := nie.dnsCache[ip]; exists {
		// 检查缓存是否过期（5分钟）
		if time.Since(nie.cacheTime[ip]) < 5*time.Minute {
			return domain
		}
		// 清理过期缓存
		delete(nie.dnsCache, ip)
		delete(nie.cacheTime, ip)
	}

	// 尝试反向DNS查询（非阻塞）
	go func() {
		if names, err := net.LookupAddr(ip); err == nil && len(names) > 0 {
			domain := strings.TrimSuffix(names[0], ".")
			nie.dnsCache[ip] = domain
			nie.cacheTime[ip] = time.Now()
			nie.logger.Debug("DNS解析成功", "ip", ip, "domain", domain)
		}
	}()

	// 立即返回空字符串，不阻塞主流程
	return ""
}
