package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type MSetting struct {
	Concurrency int `json:"concurrency"`
}

var middleware MSetting

// middleware 入口
func main() {
	flag.IntVar(&middleware.Concurrency, "concurrency", 1, "Number of concurrent requests to replay per input request. Default is 1")
	flag.Parse()

	_, _ = fmt.Fprintf(os.Stderr, "Middleware started, concurrency=%d\n", middleware.Concurrency)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		go processLine(line)
	}

	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Middleware error reading input: %v\n", err)
	}
}

func processLine(line string) {
	// 解码十六进制数据
	decoded, err := hex.DecodeString(line)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Middleware error decoding hex: %v\n", err)
		return
	}

	// 分割元数据和负载
	parts := strings.SplitN(string(decoded), "\n", 2)
	if len(parts) < 2 {
		_, _ = fmt.Fprintf(os.Stderr, "Middleware invalid message format\n")
		return
	}

	metadata := parts[0]
	payload := parts[1]

	// 解析元数据
	metaParts := strings.Split(metadata, " ")
	if len(metaParts) < 3 {
		_, _ = fmt.Fprintf(os.Stderr, "Middleware invalid metadata format\n")
		return
	}

	// 只处理请求类型 (类型1)
	if metaParts[0] != "1" {
		// 非请求数据直接转发
		outputLine(line)
		return
	}

	// 并发复制请求
	replicateRequest(metaParts, payload)
}

// 复制请求
func replicateRequest(originalMeta []string, payload string) {
	var wg sync.WaitGroup
	originalID := originalMeta[1] // 保存原始ID

	for i := 0; i < middleware.Concurrency; i++ {
		wg.Add(1)
		go func(copyNum int) {
			defer wg.Done()

			// 为每个副本生成新的唯一ID
			newID := generateNewID(originalID, copyNum)

			// 构建新的元数据
			newMeta := make([]string, len(originalMeta))
			copy(newMeta, originalMeta)
			newMeta[1] = newID                                    // 替换ID
			newMeta[2] = fmt.Sprintf("%d", time.Now().UnixNano()) // 更新时间戳

			newMetadata := strings.Join(newMeta, " ")

			// 重新编码消息
			message := newMetadata + "\n" + payload
			encoded := hex.EncodeToString([]byte(message))

			outputLine(encoded)
		}(i)
	}
	wg.Wait()
}

// 根据原始gor请求id生成新的请id
func generateNewID(originalID string, copyNum int) string {
	return fmt.Sprintf("%s_%d", originalID, copyNum)
}

// 标准输出请求内容，todo 后期如果有性能问题要考虑改造成writer := bufio.NewWriter(os.Stdout);defer writer.Flush()批量+定期刷新的方式替代fmt.Println
func outputLine(encoded string) {
	fmt.Println(encoded)
}
