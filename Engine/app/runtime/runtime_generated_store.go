// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package runtime

import (
	"context"
	"strconv"
	"strings"
	"time"
)

const (
	generatedDomainQueueKey = "dns:generated:queue"
	generatedMetaKeyPrefix  = "meta:"

	generatedMetaTTL     = 48 * time.Hour
	generatedQueueWait   = 1 * time.Second
	generatedStoreIOTime = 400 * time.Millisecond
)

const storeGeneratedDomainScript = `
redis.call("HSET", KEYS[1],
  "risk_score", ARGV[2],
  "confidence", ARGV[3],
  "generated_by", ARGV[4])
redis.call("EXPIRE", KEYS[1], tonumber(ARGV[5]))
redis.call("LPUSH", KEYS[2], ARGV[1])
return 1
`

const readGeneratedMetaScript = `
return redis.call("HMGET", KEYS[1], "risk_score", "confidence", "generated_by")
`

func streamGeneratedDomainsToRedis(
	ctx context.Context,
	path string,
	modules ModuleFactory,
	store CacheStore,
) (int64, error) {
	if err := resetGeneratedDomainQueue(store); err != nil {
		return 0, err
	}

	var count int64
	err := modules.StreamGeneratedDomains(path, func(item GeneratedDomain) error {
		if err := storeGeneratedDomain(ctx, store, item); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

func storeGeneratedDomain(ctx context.Context, store CacheStore, item GeneratedDomain) error {
	domain := normalizeGeneratedDomainName(item.Domain)
	if domain == "" {
		return nil
	}
	meta := normalizeGeneratedMeta(generatedDomainMeta{
		RiskScore:   item.RiskScore,
		Confidence:  item.Confidence,
		GeneratedBy: item.GeneratedBy,
	})

	ioCtx, cancel := context.WithTimeout(ctx, generatedStoreIOTime)
	defer cancel()

	_, err := store.Eval(ioCtx, storeGeneratedDomainScript,
		[]string{generatedMetaKey(domain), generatedDomainQueueKey},
		domain,
		formatRiskScore(meta.RiskScore),
		meta.Confidence,
		meta.GeneratedBy,
		int64(generatedMetaTTL/time.Second),
	)
	return err
}

func resetGeneratedDomainQueue(store CacheStore) error {
	ctx, cancel := context.WithTimeout(context.Background(), generatedStoreIOTime)
	defer cancel()
	return store.Delete(ctx, generatedDomainQueueKey)
}

func popGeneratedDomain(ctx context.Context, store CacheStore) (string, bool, error) {
	ioCtx, cancel := context.WithTimeout(ctx, generatedQueueWait+generatedStoreIOTime)
	defer cancel()
	value, ok, err := store.BLPop(ioCtx, generatedQueueWait, generatedDomainQueueKey)
	if err != nil {
		return "", false, err
	}
	return normalizeGeneratedDomainName(value), ok, nil
}

func loadGeneratedMeta(ctx context.Context, store intelQueueStore, domain string) (generatedDomainMeta, error) {
	domain = normalizeGeneratedDomainName(domain)
	if domain == "" {
		return defaultGeneratedMeta(), nil
	}

	ioCtx, cancel := context.WithTimeout(ctx, generatedStoreIOTime)
	defer cancel()

	raw, err := store.Eval(ioCtx, readGeneratedMetaScript, []string{generatedMetaKey(domain)})
	if err != nil {
		return defaultGeneratedMeta(), err
	}
	return parseGeneratedMeta(raw), nil
}

func parseGeneratedMeta(raw interface{}) generatedDomainMeta {
	meta := defaultGeneratedMeta()
	values, ok := raw.([]interface{})
	if !ok || len(values) < 3 {
		return meta
	}

	if score, ok := parseFloatString(stringValue(values[0])); ok {
		meta.RiskScore = score
	}
	if value := stringValue(values[1]); value != "" {
		meta.Confidence = value
	}
	if value := stringValue(values[2]); value != "" {
		meta.GeneratedBy = value
	}
	return normalizeGeneratedMeta(meta)
}

func generatedMetaKey(domain string) string {
	return generatedMetaKeyPrefix + normalizeGeneratedDomainName(domain)
}

func normalizeGeneratedDomainName(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

func formatRiskScore(score float64) string {
	return strconv.FormatFloat(score, 'f', 2, 64)
}

func parseFloatString(value string) (float64, bool) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func stringValue(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	default:
		return ""
	}
}
