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
	"sync"
	"time"
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

func getWorkerCount() int {
	v := os.Getenv("WORKER_COUNT")
	if v == "" {
		return len(couponFileURLs)
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return len(couponFileURLs)
	}
	return n
}

type fileJob struct {
	index  int
	url    string
	setKey string
}

type fileResult struct {
	index  int
	count  int
	err    error
	millis int64
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
	workers := getWorkerCount()

	if maxTokens > 0 {
		log.Printf("config: max_tokens_per_file=%d, workers=%d", maxTokens, workers)
	} else {
		log.Printf("config: no token cap (loading all), workers=%d", workers)
	}

	jobs := make(chan fileJob, len(couponFileURLs))
	results := make(chan fileResult, len(couponFileURLs))

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				log.Printf("[worker %d] processing file %d → %s", workerID, job.index+1, job.setKey)
				start := time.Now()
				count, err := loadFile(ctx, rdb, job.setKey, job.url, maxTokens)
				results <- fileResult{
					index:  job.index,
					count:  count,
					err:    err,
					millis: time.Since(start).Milliseconds(),
				}
			}
		}(w)
	}

	for idx, url := range couponFileURLs {
		jobs <- fileJob{
			index:  idx,
			url:    url,
			setKey: fmt.Sprintf("promo:set:%d", idx+1),
		}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var failed bool
	for res := range results {
		if res.err != nil {
			log.Printf("[file %d] FAILED after %dms: %v", res.index+1, res.millis, res.err)
			failed = true
		} else {
			log.Printf("[file %d] done — %d tokens in %dms", res.index+1, res.count, res.millis)
		}
	}

	if failed {
		log.Fatal("one or more files failed to load")
	}

	if err := rdb.Set(ctx, "promo:ready", "1", 0).Err(); err != nil {
		log.Fatalf("set promo:ready: %v", err)
	}
	log.Println("promo:ready = 1 — all files loaded")
}

func loadFile(ctx context.Context, rdb *redis.Client, setKey, url string, maxTokens int) (int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("gzip: %w", err)
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
					return total, err
				}
			}
		}
		sb.Reset()
	}

	if word := sb.String(); len(word) >= 8 && len(word) <= 10 {
		batch = append(batch, word)
	}
	if err := flush(); err != nil {
		return total, err
	}

	return total, nil
}
