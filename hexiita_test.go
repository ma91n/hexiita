package main

import (
	"bytes"
	"strings"
	"testing"
)

func header(t *testing.T, m HexoMeta) string {
	t.Helper()
	var buf bytes.Buffer
	if err := m.WriteHeader(&buf); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	return buf.String()
}

func base() HexoMeta {
	return HexoMeta{
		Title:     `"Go 1.27 リリース連載： uuid"`,
		Date:      "2026/08/04 00:00:00",
		PostID:    "a",
		Tags:      []string{"Go", "Go1.27"},
		Category:  "Programming",
		Thumbnail: "/images/2026/20260804a/thumbnail.jpg",
		Author:    "武田大輝",
		Lede:      `"lede"`,
	}
}

// 連載に属さない記事の方が多い。series 行が増えていないことを確かめる
func TestWriteHeaderWithoutSeries(t *testing.T) {
	got := header(t, base())
	if strings.Contains(got, "series") {
		t.Errorf("series を指定していないのに出力された:\n%s", got)
	}
}

// series は categories の直後、thumbnail の前。既存620記事がこの並び
func TestWriteHeaderSeriesPosition(t *testing.T) {
	m := base()
	m.Series = `"Go1.27リリース"`
	got := header(t, m)

	want := "categories:\n  - Programming\nseries: \"Go1.27リリース\"\nthumbnail: "
	if !strings.Contains(got, want) {
		t.Errorf("series の位置が違う\n--- got ---\n%s", got)
	}
}

// 索引記事はタグで見分ける。連載ナビが索引へ戻るリンクを出す条件
func TestWriteHeaderIndexTag(t *testing.T) {
	m := base()
	m.Series = `"Go1.27リリース"`
	m.Tags = append(m.Tags, "インデックス")
	got := header(t, m)

	if !strings.Contains(got, "  - インデックス\n") {
		t.Errorf("インデックス タグが出ていない:\n%s", got)
	}
}

func TestWarnSeriesName(t *testing.T) {
	tests := []struct {
		name string
		warn bool
	}{
		// 表示は「連載：<値>」なので末尾の「連載」は重複する
		{"サーバレス連載", true},
		{"夏休み自由研究企画", true},
		{"業界ドメインシリーズ", true},
		// 既存71連載の実例
		{"Go1.27リリース", false},
		{"CI/CD", false},
		{"春の入門祭り2025", false},
		{"サービス間通信とIDL", false},
	}
	for _, tt := range tests {
		var got bool
		for _, s := range redundantSuffix {
			if strings.HasSuffix(tt.name, s) {
				got = true
			}
		}
		if got != tt.warn {
			t.Errorf("%q: 警告 %v, 期待 %v", tt.name, got, tt.warn)
		}
	}
}
