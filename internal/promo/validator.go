package promo

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

type Validator struct {
	rdb *redis.Client
}

func NewValidator(rdb *redis.Client) *Validator {
	return &Validator{rdb: rdb}
}

func (v *Validator) IsReady(ctx context.Context) bool {
	val, err := v.rdb.Get(ctx, "promo:ready").Result()
	return err == nil && val == "1"
}

func (v *Validator) IsValid(ctx context.Context, code string) bool {
	if len(code) < 8 || len(code) > 10 {
		return false
	}
	code = strings.ToUpper(code)

	found := 0
	for i := 1; i <= 3; i++ {
		key := fmt.Sprintf("promo:set:%d", i)
		ok, err := v.rdb.SIsMember(ctx, key, code).Result()
		if err == nil && ok {
			found++
		}
	}
	return found >= 1
}
