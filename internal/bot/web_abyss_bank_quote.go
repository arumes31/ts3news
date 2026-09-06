package bot

import "math"

// abyssGoldAdd keeps nonnegative currency within the database BIGINT range.
func abyssGoldAdd(current, gain int64) int64 {
	current = max(current, 0)
	gain = max(gain, 0)
	if gain > math.MaxInt64-current {
		return math.MaxInt64
	}
	return current + gain
}

// abyssGoldScale preserves exact integer multipliers and saturates before a
// floating-point conversion can wrap a valid cache into a negative balance.
func abyssGoldScale(amount int64, multiplier float64) int64 {
	if amount <= 0 || multiplier <= 0 || math.IsNaN(multiplier) {
		return 0
	}
	if multiplier < float64(math.MaxInt64) && math.Trunc(multiplier) == multiplier {
		factor := int64(multiplier)
		if amount > math.MaxInt64/factor {
			return math.MaxInt64
		}
		return amount * factor
	}
	scaled := float64(amount) * multiplier
	if scaled >= float64(math.MaxInt64) {
		return math.MaxInt64
	}
	return int64(scaled)
}

// abyssGoldPercent divides before multiplying so intermediate products do not
// overflow. A percentage above 100 also saturates its final result.
func abyssGoldPercent(amount int64, percent int) int64 {
	if amount <= 0 || percent <= 0 {
		return 0
	}
	whole := amount / 100
	rate := int64(percent)
	if whole > math.MaxInt64/rate {
		return math.MaxInt64
	}
	// Split the percentage too: even amount's two-digit remainder may overflow
	// if a caller supplies an unusually large percentage.
	remainder := amount % 100
	fraction := remainder*(rate/100) + remainder*(rate%100)/100
	return abyssGoldAdd(whole*rate, fraction)
}
