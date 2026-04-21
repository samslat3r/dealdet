package pricing

import (
	"context"
	"errors"
	"testing"

	"dealdet/internal/domain"

	"github.com/google/uuid"
)

// stubLister implements SoldPriceLister for testing.
type stubLister struct {
	prices   map[string][]float64 // key: "productID|condition"
	fallback map[string][]float64 // key: "productID|" (empty condition = all)
	err      error
}

func (s *stubLister) ListSoldPrices(_ context.Context, productID uuid.UUID, condition domain.ConditionTier, _ int) ([]float64, error) {
	if s.err != nil {
		return nil, s.err
	}
	key := productID.String() + "|" + string(condition)
	if prices, ok := s.prices[key]; ok {
		return prices, nil
	}
	if prices, ok := s.fallback[key]; ok {
		return prices, nil
	}
	return nil, nil
}

func newProductID() uuid.UUID {
	return uuid.New()
}

func TestTrimmedMean_Basic(t *testing.T) {
	// 10 values, trim 10% = 1 from each end
	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	got := trimmedMean(values, 0.10)
	// After trim: [20,30,40,50,60,70,80,90] → mean = 55
	if got != 55 {
		t.Errorf("trimmedMean = %f, want 55", got)
	}
}

func TestTrimmedMean_Empty(t *testing.T) {
	got := trimmedMean(nil, 0.10)
	if got != 0 {
		t.Errorf("trimmedMean(nil) = %f, want 0", got)
	}
}

func TestTrimmedMean_SmallSlice(t *testing.T) {
	// 4 values: trim 10% = 0 from each end (since 4 - 2*0 >= 3)
	// But floor(4*0.10) = 0, so no trim, mean of all
	values := []float64{10, 20, 30, 40}
	got := trimmedMean(values, 0.10)
	want := 25.0
	if got != want {
		t.Errorf("trimmedMean = %f, want %f", got, want)
	}
}

func TestTrimmedMean_TooFewAfterTrim(t *testing.T) {
	// 5 values, trim 25% = 1 from each side → 3 remain, OK
	values := []float64{1, 2, 3, 4, 5}
	got := trimmedMean(values, 0.25)
	// After trim: [2, 3, 4] → mean = 3
	if got != 3 {
		t.Errorf("trimmedMean = %f, want 3", got)
	}
}

func TestTrimmedMean_DoesNotMutateInput(t *testing.T) {
	values := []float64{50, 10, 30, 20, 40}
	original := make([]float64, len(values))
	copy(original, values)
	trimmedMean(values, 0.10)
	for i, v := range values {
		if v != original[i] {
			t.Errorf("trimmedMean mutated input at index %d: got %f, want %f", i, v, original[i])
		}
	}
}

func TestComputeBaseline_ExactCondition(t *testing.T) {
	pid := newProductID()
	prices := make([]float64, 10)
	for i := range prices {
		prices[i] = float64(100 + i*10) // 100, 110, ..., 190
	}

	lister := &stubLister{
		prices: map[string][]float64{
			pid.String() + "|very_good": prices,
		},
	}

	result, err := ComputeBaseline(context.Background(), lister, pid, domain.ConditionVeryGood, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Fallback {
		t.Error("expected no fallback, got fallback=true")
	}
	if result.Snapshot.SampleSize != 10 {
		t.Errorf("sample size = %d, want 10", result.Snapshot.SampleSize)
	}
	if result.Snapshot.CanonicalProductID != pid {
		t.Error("product ID mismatch")
	}
	if result.Snapshot.ConditionCanonical != domain.ConditionVeryGood {
		t.Errorf("condition = %s, want very_good", result.Snapshot.ConditionCanonical)
	}
	if result.Snapshot.TrimmedMeanPriceUSD <= 0 {
		t.Error("trimmed mean should be positive")
	}
}

func TestComputeBaseline_ConditionFallback(t *testing.T) {
	pid := newProductID()
	// Exact condition has only 3 prices (< MinSampleSize)
	exactPrices := []float64{100, 200, 300}
	// All-condition fallback has 10
	fallbackPrices := make([]float64, 10)
	for i := range fallbackPrices {
		fallbackPrices[i] = float64(80 + i*5)
	}

	lister := &stubLister{
		prices: map[string][]float64{
			pid.String() + "|new": exactPrices,
			pid.String() + "|":    fallbackPrices,
		},
	}

	result, err := ComputeBaseline(context.Background(), lister, pid, domain.ConditionNew, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Fallback {
		t.Error("expected fallback=true")
	}
	if result.Snapshot.SampleSize != 10 {
		t.Errorf("sample size = %d, want 10", result.Snapshot.SampleSize)
	}
}

func TestComputeBaseline_InsufficientData(t *testing.T) {
	pid := newProductID()
	lister := &stubLister{
		prices: map[string][]float64{
			pid.String() + "|good": {100, 200},
			pid.String() + "|":     {100, 200},
		},
	}

	_, err := ComputeBaseline(context.Background(), lister, pid, domain.ConditionGood, 90)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInsufficientData) {
		t.Errorf("expected ErrInsufficientData, got: %v", err)
	}
}

func TestComputeBaseline_ListerError(t *testing.T) {
	pid := newProductID()
	lister := &stubLister{err: errors.New("db down")}

	_, err := ComputeBaseline(context.Background(), lister, pid, domain.ConditionGood, 90)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestComputeBaseline_DefaultWindowDays(t *testing.T) {
	pid := newProductID()
	prices := make([]float64, 10)
	for i := range prices {
		prices[i] = float64(50 + i)
	}
	lister := &stubLister{
		prices: map[string][]float64{
			pid.String() + "|good": prices,
		},
	}

	result, err := ComputeBaseline(context.Background(), lister, pid, domain.ConditionGood, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Snapshot.WindowDays != DefaultWindowDays {
		t.Errorf("window days = %d, want %d", result.Snapshot.WindowDays, DefaultWindowDays)
	}
}
