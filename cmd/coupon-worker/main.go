package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

const batchSize = 500

var couponFileURLs = []string{
	"https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase1.gz",
	"https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase2.gz",
	"https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase3.gz",
}

func getMaxTokens() int {
	v := os.Getenv("MAX_TOKENS_PER_FILE")
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func main() {
	godotenv.Load()

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		log.Fatal("REDIS_URL is required")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}

	maxTokens := getMaxTokens()
	if maxTokens > 0 {
		log.Printf("capping at %d tokens per file", maxTokens)
	}

	for idx, url := range couponFileURLs {
		setKey := fmt.Sprintf("promo:set:%d", idx+1)
		log.Printf("[file %d] starting %s → %s", idx+1, url, setKey)
		if err := loadFile(ctx, rdb, setKey, url, maxTokens); err != nil {
			log.Fatalf("[file %d] failed: %v", idx+1, err)
		}
		log.Printf("[file %d] done", idx+1)
	}

	if err := rdb.Set(ctx, "promo:ready", "1", 0).Err(); err != nil {
		log.Fatalf("set promo:ready: %v", err)
	}
	log.Println("promo:ready = 1 — all files loaded")
}

func loadFile(ctx context.Context, rdb *redis.Client, setKey, url string, maxTokens int) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	br := bufio.NewReaderSize(gz, 64*1024)
	var sb strings.Builder
	var batch []interface{}
	total := 0

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := rdb.SAdd(ctx, setKey, batch...).Err(); err != nil {
			return err
		}
		total += len(batch)
		if total%100000 == 0 {
			log.Printf("[%s] %d tokens loaded so far...", setKey, total)
		}
		batch = batch[:0]
		return nil
	}

	for {
		if maxTokens > 0 && total >= maxTokens {
			log.Printf("[%s] reached %d token cap, stopping early", setKey, maxTokens)
			break
		}
		b, err := br.ReadByte()
		if err != nil {
			break
		}
		r := rune(b)
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(unicode.ToUpper(r))
			continue
		}
		if word := sb.String(); len(word) >= 8 && len(word) <= 10 {
			batch = append(batch, word)
			if len(batch) >= batchSize {
				if err := flush(); err != nil {
					return err
				}
			}
		}
		sb.Reset()
	}

	if word := sb.String(); len(word) >= 8 && len(word) <= 10 {
		batch = append(batch, word)
	}
	if err := flush(); err != nil {
		return err
	}

	log.Printf("[%s] %d tokens loaded (final)", setKey, total)
	return nil
}
