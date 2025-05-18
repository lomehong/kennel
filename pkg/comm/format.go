package comm

import (
	"fmt"
	"strconv"
	"time"
)

// formatUint64 格式化uint64数字
func formatUint64(n uint64) string {
	return strconv.FormatUint(n, 10)
}

// formatInt64 格式化int64数字
func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}

// formatFloat 格式化浮点数，保留2位小数
func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

// formatPercentage 格式化百分比，保留2位小数
func formatPercentage(f float64) string {
	return fmt.Sprintf("%.2f%%", f*100)
}

// formatBytes 格式化字节数，自动选择合适的单位
func formatBytes(bytes uint64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatTime 格式化时间戳
func formatTime(timestamp int64) string {
	t := time.Unix(0, timestamp*int64(time.Millisecond))
	return t.Format("2006-01-02 15:04:05.000")
}

// formatDuration 格式化持续时间
func formatDuration(duration int64) string {
	d := time.Duration(duration) * time.Millisecond
	
	// 如果小于1秒，直接显示毫秒
	if d < time.Second {
		return fmt.Sprintf("%d ms", d.Milliseconds())
	}
	
	// 如果小于1分钟，显示秒和毫秒
	if d < time.Minute {
		seconds := d.Seconds()
		return fmt.Sprintf("%.2f s", seconds)
	}
	
	// 如果小于1小时，显示分钟和秒
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%d m %d s", minutes, seconds)
	}
	
	// 如果小于1天，显示小时、分钟和秒
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%d h %d m %d s", hours, minutes, seconds)
	}
	
	// 如果大于等于1天，显示天、小时、分钟和秒
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%d d %d h %d m %d s", days, hours, minutes, seconds)
}
