package identity

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkL1Cache_GetCanonicalID(b *testing.B) {
	cache := NewL1Cache()
	// Pre-populate with 1,000 realistic fixtures
	for i := 0; i < 1000; i++ {
		providerID := fmt.Sprintf("sr:match:%08d", i)
		canonicalID := fmt.Sprintf("SPD-FB-%06d", i)
		cache.SetAlias("SPORTYBET", providerID, canonicalID)
	}

	targetProvider := "SPORTYBET"
	targetID := "sr:match:00000500"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cid, ok := cache.GetCanonicalID(targetProvider, targetID)
		if !ok || cid == "" {
			b.Fatalf("expected cache hit for %s:%s", targetProvider, targetID)
		}
	}
}

func BenchmarkL1Cache_GetCanonicalID_Parallel(b *testing.B) {
	cache := NewL1Cache()
	for i := 0; i < 1000; i++ {
		providerID := fmt.Sprintf("sr:match:%08d", i)
		canonicalID := fmt.Sprintf("SPD-FB-%06d", i)
		cache.SetAlias("SPORTYBET", providerID, canonicalID)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		targetProvider := "SPORTYBET"
		targetID := "sr:match:00000500"
		for pb.Next() {
			cid, ok := cache.GetCanonicalID(targetProvider, targetID)
			if !ok || cid == "" {
				b.Fatalf("expected cache hit")
			}
		}
	})
}

func BenchmarkEngine_ResolveCanonicalID_L1Hit(b *testing.B) {
	tieredCache := NewTieredIdentityCache(nil, 0)
	engine := NewEngine(nil, tieredCache)

	for i := 0; i < 1000; i++ {
		providerID := fmt.Sprintf("sr:match:%08d", i)
		canonicalID := fmt.Sprintf("SPD-FB-%06d", i)
		tieredCache.PutAlias(context.Background(), "SPORTYBET", providerID, canonicalID)
	}

	ctx := context.Background()
	targetProvider := "SPORTYBET"
	targetID := "sr:match:00000500"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cid, err := engine.ResolveCanonicalID(ctx, targetProvider, targetID)
		if err != nil || cid == "" {
			b.Fatalf("expected resolve hit")
		}
	}
}

func BenchmarkEngine_ResolveCanonicalID_Parallel(b *testing.B) {
	tieredCache := NewTieredIdentityCache(nil, 0)
	engine := NewEngine(nil, tieredCache)

	for i := 0; i < 1000; i++ {
		providerID := fmt.Sprintf("sr:match:%08d", i)
		canonicalID := fmt.Sprintf("SPD-FB-%06d", i)
		tieredCache.PutAlias(context.Background(), "SPORTYBET", providerID, canonicalID)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		targetProvider := "SPORTYBET"
		targetID := "sr:match:00000500"
		for pb.Next() {
			cid, err := engine.ResolveCanonicalID(ctx, targetProvider, targetID)
			if err != nil || cid == "" {
				b.Fatalf("expected resolve hit")
			}
		}
	})
}
