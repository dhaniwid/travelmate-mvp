package domain

import (
	"strconv"
	"strings"
)

type FlexibleInt64 int64

func (fi *FlexibleInt64) UnmarshalJSON(data []byte) error {
	s := string(data)
	s = strings.TrimSpace(s)

	// A. Handle "null"
	if s == "null" {
		*fi = 0
		return nil
	}

	// B. HANDLE OBJECT/ARRAY (Penyebab Error Anda)
	// Jika data dimulai dengan '{' atau '[', berarti AI memberikan object detail.
	// Untuk MVP, kita ignore saja detailnya dan set 0 agar tidak error.
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		// Opsional: Jika Anda ingin canggih, Anda bisa parse objectnya dan cari field "total".
		// Tapi untuk stability sekarang, kita set 0.
		*fi = 0
		return nil
	}

	// C. Bersihkan Tanda Kutip & Koma (Logic Lama)
	s = strings.Trim(s, "\"")
	s = strings.ReplaceAll(s, ",", "")

	// D. Handle Desimal (misal: 100.00)
	if idx := strings.Index(s, "."); idx != -1 {
		s = s[:idx]
	}

	// E. Parse ke Int
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		// Fallback terakhir
		*fi = 0
		return nil
	}

	*fi = FlexibleInt64(val)
	return nil
}
