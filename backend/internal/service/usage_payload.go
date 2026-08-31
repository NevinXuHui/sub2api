package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"regexp"
)

// UsagePayload 采集的请求/响应正文
type UsagePayload struct {
	RequestBody       string
	ResponseBody      string
	RequestTruncated  bool
	ResponseTruncated bool
	RequestBytes      int
	ResponseBytes     int
}

// UsagePayloadCaptureResult 单次采集结果
type UsagePayloadCaptureResult struct {
	Body      string
	Truncated bool
	Bytes     int
}

var (
	// base64DataURLPattern 匹配 data: URL（图片/视频/音频）
	base64DataURLPattern = regexp.MustCompile(`(?i)"data:([^;,]+);base64,([A-Za-z0-9+/=]{100,})"`)

	// jsonBase64FieldPattern 匹配 JSON 中的 base64 字段（image、audio、video）
	jsonBase64FieldPattern = regexp.MustCompile(`(?i)"(image|audio|video|content_bytes)":\s*"([A-Za-z0-9+/=_-]{100,})"`)
)

// CaptureUsagePayload 采集并截断请求或响应正文，替换大型 base64 为占位符
func CaptureUsagePayload(src []byte, maxBytes int) UsagePayloadCaptureResult {
	if src == nil {
		return UsagePayloadCaptureResult{Body: "", Truncated: false, Bytes: 0}
	}

	originalBytes := len(src)

	// 先替换 base64 占位，避免一张图吃光额度
	processed := replaceLargeBase64WithPlaceholder(src)

	// 再按上限截断
	if len(processed) <= maxBytes {
		return UsagePayloadCaptureResult{
			Body:      string(processed),
			Truncated: false,
			Bytes:     originalBytes,
		}
	}

	return UsagePayloadCaptureResult{
		Body:      string(processed[:maxBytes]),
		Truncated: true,
		Bytes:     originalBytes,
	}
}

// replaceLargeBase64WithPlaceholder 替换超过 1KB 的 base64 数据为简短占位符
func replaceLargeBase64WithPlaceholder(src []byte) []byte {
	const minBase64Size = 1024 // 小于 1KB 的 base64 保留

	// 替换 data: URL 中的 base64
	src = base64DataURLPattern.ReplaceAllFunc(src, func(match []byte) []byte {
		parts := base64DataURLPattern.FindSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		mimeType := string(parts[1])
		b64Data := parts[2]

		if len(b64Data) < minBase64Size {
			return match
		}

		// 解码计算原始大小
		decoded, err := base64.StdEncoding.DecodeString(string(b64Data))
		var sizeHint string
		if err == nil {
			sizeHint = fmt.Sprintf(" %d bytes", len(decoded))
		} else {
			sizeHint = fmt.Sprintf(" ~%d chars", len(b64Data))
		}

		placeholder := fmt.Sprintf(`"data:%s;base64,[omitted binary%s]"`, mimeType, sizeHint)
		return []byte(placeholder)
	})

	// 替换 JSON 字段中的 base64
	src = jsonBase64FieldPattern.ReplaceAllFunc(src, func(match []byte) []byte {
		parts := jsonBase64FieldPattern.FindSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		fieldName := string(parts[1])
		b64Data := parts[2]

		if len(b64Data) < minBase64Size {
			return match
		}

		// 尝试解码计算原始大小
		decoded, err := base64.StdEncoding.DecodeString(string(b64Data))
		var sizeHint string
		if err == nil {
			sizeHint = fmt.Sprintf("%d bytes", len(decoded))
		} else {
			// 可能是 URL-safe base64
			decoded, err = base64.URLEncoding.DecodeString(string(b64Data))
			if err == nil {
				sizeHint = fmt.Sprintf("%d bytes", len(decoded))
			} else {
				sizeHint = fmt.Sprintf("~%d chars", len(b64Data))
			}
		}

		placeholder := fmt.Sprintf(`"%s":"[omitted binary %s]"`, fieldName, sizeHint)
		return []byte(placeholder)
	})

	return src
}

// ClampPayloadMaxBytes 钳制 payload 上限到合理范围 [4KB, 1MB]
func ClampPayloadMaxBytes(requested int) int {
	const (
		minPayloadBytes = 4096    // 4KB
		maxPayloadBytes = 1048576 // 1MB
	)

	if requested < minPayloadBytes {
		return minPayloadBytes
	}
	if requested > maxPayloadBytes {
		return maxPayloadBytes
	}
	return requested
}

// LimitedBuffer 有界缓冲区，超出上限后停止追加但继续计数
type LimitedBuffer struct {
	buf       bytes.Buffer
	maxBytes  int
	totalRead int
}

// NewLimitedBuffer 创建有界缓冲区
func NewLimitedBuffer(maxBytes int) *LimitedBuffer {
	return &LimitedBuffer{maxBytes: maxBytes}
}

// Write 实现 io.Writer
func (lb *LimitedBuffer) Write(p []byte) (n int, err error) {
	n = len(p)
	lb.totalRead += n

	if lb.buf.Len() >= lb.maxBytes {
		// 已达上限，不再追加，但返回成功
		return n, nil
	}

	remaining := lb.maxBytes - lb.buf.Len()
	if len(p) <= remaining {
		lb.buf.Write(p)
	} else {
		lb.buf.Write(p[:remaining])
	}

	return n, nil
}

// Bytes 返回缓冲的字节（最多 maxBytes）
func (lb *LimitedBuffer) Bytes() []byte {
	return lb.buf.Bytes()
}

// TotalRead 返回写入的总字节数（包括被截断部分）
func (lb *LimitedBuffer) TotalRead() int {
	return lb.totalRead
}

// Truncated 返回是否发生截断
func (lb *LimitedBuffer) Truncated() bool {
	return lb.totalRead > lb.maxBytes
}

// ExtractPayloadIfEnabled 如果采集开关打开且上限合法，返回有界拷贝；否则返回 nil
func ExtractPayloadIfEnabled(enabled bool, maxBytes int, src []byte) []byte {
	if !enabled || maxBytes <= 0 || src == nil {
		return nil
	}

	clamped := ClampPayloadMaxBytes(maxBytes)
	result := CaptureUsagePayload(src, clamped)
	return []byte(result.Body)
}
