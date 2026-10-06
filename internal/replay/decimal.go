package replay

import "errors"

// Scale is the fixed-point multiplier for prices and quantities.
// "100.5" becomes 100.5 * Scale.
const Scale int64 = 100_000_000

// ErrDecimal means a price or quantity string is not a fixed-point number.
var ErrDecimal = errors.New("bad decimal")

// ParseFixed converts a decimal string to an integer at Scale.
// More than 8 fractional digits is rejected.
func ParseFixed(s string) (int64, error) {
	if s == "" {
		return 0, ErrDecimal
	}
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
		if s == "" {
			return 0, ErrDecimal
		}
	}
	whole := s
	frac := ""
	if i := indexByte(s, '.'); i >= 0 {
		whole = s[:i]
		frac = s[i+1:]
	}
	if len(frac) > 8 {
		return 0, ErrDecimal
	}
	if whole == "" {
		whole = "0"
	}
	var n int64
	for i := 0; i < len(whole); i++ {
		c := whole[i]
		if c < '0' || c > '9' {
			return 0, ErrDecimal
		}
		n = n*10 + int64(c-'0')
	}
	n *= Scale
	mul := Scale / 10
	for i := 0; i < len(frac); i++ {
		c := frac[i]
		if c < '0' || c > '9' {
			return 0, ErrDecimal
		}
		n += int64(c-'0') * mul
		mul /= 10
	}
	if neg {
		n = -n
	}
	return n, nil
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
