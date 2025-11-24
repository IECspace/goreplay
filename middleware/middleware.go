package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/buger/goreplay/proto"
)

type MSetting struct {
	Concurrency int  `json:"concurrency"`
	Placeholder bool `json:"placeholder"`
}

var middleware MSetting

// middleware main function
func main() {
	flag.IntVar(&middleware.Concurrency, "concurrency", 1, "Number of concurrent requests to replay per input request. Default is 1")
	flag.BoolVar(&middleware.Placeholder, "placeholder", false, "Do you want to enable placeholder replacement in requests. Default is false")
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
	// decode hex string
	decoded, err := hex.DecodeString(line)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Middleware error decoding hex: %v\n", err)
		return
	}

	// split metadata and payload
	parts := strings.SplitN(string(decoded), "\n", 2)
	if len(parts) < 2 {
		_, _ = fmt.Fprintf(os.Stderr, "Middleware invalid message format\n")
		return
	}

	metadata := parts[0]
	payload := parts[1]

	// analyze metadata
	metaParts := strings.Split(metadata, " ")
	if len(metaParts) < 3 {
		_, _ = fmt.Fprintf(os.Stderr, "Middleware invalid metadata format\n")
		return
	}

	// only process request data (type "1")
	if metaParts[0] != "1" {
		// if it is not request data, just output original line
		outputLine(line)
		return
	}

	// replicate the request
	replicateRequest(metaParts, payload)
}

// replicateRequest replicates the request based on concurrency setting
func replicateRequest(originalMeta []string, payload string) {
	var wg sync.WaitGroup
	originalID := originalMeta[1] // assuming the second field is the request ID, keep it same for all copies

	for i := 0; i < middleware.Concurrency; i++ {
		wg.Add(1)
		go func(copyNum int) {
			defer wg.Done()

			// generate new ID for the replicated request
			newID := generateNewID(originalID, copyNum)

			// generate new metadata
			newMeta := make([]string, len(originalMeta))
			copy(newMeta, originalMeta)
			// update ID
			newMeta[1] = newID
			// update timestamp
			newMeta[2] = fmt.Sprintf("%d", time.Now().UnixNano())

			newMetadata := strings.Join(newMeta, " ")

			// if enabled placeholder replacement, do it here
			if middleware.Placeholder {
				payload = string(replacePlaceholders([]byte(payload)))
			}

			// encode and output
			message := newMetadata + "\n" + payload
			encoded := hex.EncodeToString([]byte(message))

			outputLine(encoded)
		}(i)
	}
	wg.Wait()
}

// generate new ID by appending copy number
func generateNewID(originalID string, copyNum int) string {
	return fmt.Sprintf("%s_%d", originalID, copyNum)
}

// 标准输出请求内容，todo 后期如果有性能问题要考虑改造成writer := bufio.NewWriter(os.Stdout);defer writer.Flush()批量+定期刷新的方式替代fmt.Println
func outputLine(encoded string) {
	fmt.Println(encoded)
}

// replacePlaceholders replaces placeholders in the payload and updates Content-Length header if necessary
func replacePlaceholders(payload []byte) []byte {
	payload = proto.ReplacePlaceholders(payload)

	// If chunked, do NOT modify Content-Length
	te := proto.Header(payload, []byte("Transfer-Encoding"))
	chunked := len(te) > 0 && bytes.Contains(te, []byte("chunked"))

	// update content-length only when NOT chunked
	if !chunked {
		newCL := []byte(strconv.Itoa(len(proto.Body(payload))))
		payload = proto.SetHeader(payload, []byte("Content-Length"), newCL)
	}
	return payload
}
