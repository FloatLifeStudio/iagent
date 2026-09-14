package normalize

import "testing"

func TestNormalizeCapacity(t *testing.T) {
	cases := []struct {
		bytes    int64
		wantSize int64
		wantUnit string
	}{
		{1024 * 1024 * 1024, 1, "GB"},       // 1024MB → 1GB
		{16384 * 1024 * 1024, 16, "GB"},     // 16384MB → 16GB
		{81920 * 1024 * 1024, 80, "GB"},     // H100 显存
		{64 * 1024 * 1024 * 1024, 64, "GB"}, // 64GB 内存条
		{8 * 1024 * 1024 * 1024 * 1024, 8, "TB"},
		{2 * 1024 * 1024 * 1024 * 1024, 2, "TB"},  // 2TB = 2048GB ≥ 1024 → TB
		{512 * 1024 * 1024 * 1024, 512, "GB"},     // 512GB < 1024GB → GB
	}
	for _, c := range cases {
		size, unit := NormalizeCapacity(c.bytes)
		if size != c.wantSize || unit != c.wantUnit {
			t.Errorf("NormalizeCapacity(%d) = %d %s, want %d %s", c.bytes, size, unit, c.wantSize, c.wantUnit)
		}
	}
}

func TestNormalizeCapacityNonPositive(t *testing.T) {
	size, unit := NormalizeCapacity(0)
	if size != 1 || unit != "GB" {
		t.Errorf("NormalizeCapacity(0) = %d %s, want 1 GB", size, unit)
	}
}

func TestParseSizeToBytes(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"64 GB", 64 * 1024 * 1024 * 1024},
		{"16384 MB", 16384 * 1024 * 1024},
		{"8 TB", 8 * 1024 * 1024 * 1024 * 1024},
		{"81920", 81920 * 1024 * 1024}, // 无单位按 MiB(nvidia-smi nounits)
	}
	for _, c := range cases {
		got, err := ParseSizeToBytes(c.in)
		if err != nil {
			t.Errorf("ParseSizeToBytes(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseSizeToBytes(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
