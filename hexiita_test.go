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

func TestExtractImageURLInTableCell(t *testing.T) {
	line := `| <img src="https://qiita-image-store.s3.ap-northeast-1.amazonaws.com/0/1/abc.png" alt="APM認定バッジ" width="150"> | APM | 11月上旬 | 日本語 |`
	got, err := ExtractImageURL(line)
	if err != nil {
		t.Fatal(err)
	}
	if got.Prefix != "| " || got.Suffix != " | APM | 11月上旬 | 日本語 |" {
		t.Errorf("prefix=%q suffix=%q", got.Prefix, got.Suffix)
	}
	if got.DisplayWidth != 150 {
		t.Errorf("DisplayWidth=%d", got.DisplayWidth)
	}

	got.Width, got.Height = 1200, 600
	out := got.Prefix + ImageTag("/images/2026/20261117a/APM認定バッジ.png", got) + got.Suffix
	want := `| <img src="/images/2026/20261117a/APM認定バッジ.png" alt="APM認定バッジ" width="150" height="75" loading="lazy"> | APM | 11月上旬 | 日本語 |`
	if out != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}
}

func TestExtractImageURLMarkdownInline(t *testing.T) {
	line := `| ![a.png](https://qiita-image-store.s3.ap-northeast-1.amazonaws.com/0/1/abc.png) | x |`
	got, err := ExtractImageURL(line)
	if err != nil {
		t.Fatal(err)
	}
	if got.Prefix != "| " || got.Suffix != " | x |" || got.URL != "https://qiita-image-store.s3.ap-northeast-1.amazonaws.com/0/1/abc.png" {
		t.Errorf("prefix=%q suffix=%q url=%q", got.Prefix, got.Suffix, got.URL)
	}
}
