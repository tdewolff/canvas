package canvas

import "testing"

func BenchmarkScanner(b *testing.B) {
	p := RandomPath(1000, true)
	b.Run("Manual", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for j := 0; j < len(p.d); {
				j += cmdLen(p.d[j])
				_, _ = p.d[j-3], p.d[j-2]
			}
		}
	})

	b.Run("Scanner", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for s := p.Scanner(); s.Scan(); {
				_ = s.End()
			}
		}
	})
}
