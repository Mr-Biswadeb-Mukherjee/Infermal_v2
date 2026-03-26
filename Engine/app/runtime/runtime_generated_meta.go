// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package runtime

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

func loadGeneratedMeta(
	ctx context.Context,
	store intelQueueStore,
	spool *generatedDomainSpool,
	domain string,
) (generatedDomainMeta, error) {
	domain = normalizeGeneratedDomainName(domain)
	if domain == "" {
		return defaultGeneratedMeta(), nil
	}
	meta, found, cacheErr := loadGeneratedMetaFromCache(ctx, store, domain)
	if cacheErr == nil && found {
		return meta, nil
	}
	spoolMeta, spoolFound, spoolErr := loadGeneratedMetaFromSpool(ctx, spool, domain)
	if spoolErr != nil {
		return defaultGeneratedMeta(), errors.Join(cacheErr, spoolErr)
	}
	if spoolFound {
		return spoolMeta, nil
	}
	if cacheErr != nil {
		return defaultGeneratedMeta(), cacheErr
	}
	return defaultGeneratedMeta(), nil
}

func loadGeneratedMetaFromCache(
	ctx context.Context,
	store intelQueueStore,
	domain string,
) (generatedDomainMeta, bool, error) {
	ioCtx, cancel := context.WithTimeout(ctx, generatedStoreIOTime)
	defer cancel()
	raw, err := store.Eval(ioCtx, readGeneratedMetaScript, []string{generatedMetaKey(domain)})
	if err != nil {
		return defaultGeneratedMeta(), false, err
	}
	meta, found := parseGeneratedMeta(raw)
	return meta, found, nil
}

func loadGeneratedMetaFromSpool(
	ctx context.Context,
	spool *generatedDomainSpool,
	domain string,
) (generatedDomainMeta, bool, error) {
	if spool == nil {
		return defaultGeneratedMeta(), false, nil
	}
	ioCtx, cancel := context.WithTimeout(ctx, generatedStoreIOTime)
	defer cancel()
	return spool.metaByDomain(ioCtx, domain)
}

func parseGeneratedMeta(raw interface{}) (generatedDomainMeta, bool) {
	meta := defaultGeneratedMeta()
	values, ok := raw.([]interface{})
	if !ok || len(values) < 3 || metaFieldsMissing(values[:3]) {
		return meta, false
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
	return normalizeGeneratedMeta(meta), true
}

func metaFieldsMissing(values []interface{}) bool {
	for _, value := range values {
		if value != nil {
			return false
		}
	}
	return true
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
