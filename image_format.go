package main

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"strings"
)

// URL の拡張子と中身が食い違うことがある（Qiita が中身だけ AVIF にして .png のまま返す）。
// 拡張子を信用すると AVIF を image/png として配ることになる（GitHub Pages は
// 拡張子で Content-Type を決める）ので、中身の側から拡張子を決める (#54)
func sniffExt(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return ".png"
	case bytes.HasPrefix(b, []byte("\xff\xd8\xff")):
		return ".jpg"
	case bytes.HasPrefix(b, []byte("GIF87a")), bytes.HasPrefix(b, []byte("GIF89a")):
		return ".gif"
	case len(b) >= 12 && bytes.Equal(b[0:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return ".webp"
	case isAVIF(b):
		return ".avif"
	}
	return "" // SVG のようなテキスト形式と未知の形式は拡張子に触らない
}

// .jpeg と .jpg のように同じ形式の別表記を1つに寄せる。既存記事に .jpeg の
// ファイルがあるので、表記の違いだけで名前を書き換えないため
func normalizeExt(ext string) string {
	ext = strings.ToLower(ext)
	if ext == ".jpeg" {
		return ".jpg"
	}
	return ext
}

// AVIF は ISOBMFF で、ftyp の major brand か compatible brands に avif（静止画）か
// avis（アニメーション）が入る
func isAVIF(b []byte) bool {
	payload, ok := findBox(b, "ftyp")
	if !ok || len(payload) < 4 {
		return false
	}
	for i := 0; i+4 <= len(payload); i += 4 {
		switch string(payload[i : i+4]) {
		case "avif", "avis":
			return true
		}
	}
	return false
}

// AVIF の寸法は meta > iprp > ipco > ispe（Image Spatial Extents）が持つ。
// Go 標準に AVIF のデコーダが無いので、寸法だけ自前で読む (#54)。
// アニメーション AVIF（avis）も同じ形で持っている
func avifSize(b []byte) (int, int, bool) {
	meta, ok := findBox(b, "meta")
	if !ok || len(meta) < 4 {
		return 0, 0, false
	}

	// meta は FullBox なので、先頭4バイトの version / flags を読み飛ばす
	iprp, ok := findBox(meta[4:], "iprp")
	if !ok {
		return 0, 0, false
	}
	ipco, ok := findBox(iprp, "ipco")
	if !ok {
		return 0, 0, false
	}
	// ipco はサムネイルや別解像度のぶんも ispe を持つが、実ファイルではいずれも
	// 先頭が主画像だったので最初のものを使う
	ispe, ok := findBox(ipco, "ispe")
	if !ok || len(ispe) < 12 {
		return 0, 0, false
	}

	// ispe も FullBox。version / flags の後ろに width / height が並ぶ
	w := binary.BigEndian.Uint32(ispe[4:8])
	h := binary.BigEndian.Uint32(ispe[8:12])
	if w == 0 || h == 0 {
		return 0, 0, false
	}
	return int(w), int(h), true
}

// 兄弟のボックスを順に見て、type が一致した最初のボックスの中身を返す。
// ISOBMFF のボックスは size(4) + type(4) の後ろに中身が続く
func findBox(b []byte, typ string) ([]byte, bool) {
	for offset := 0; offset+8 <= len(b); {
		size := int(binary.BigEndian.Uint32(b[offset : offset+4]))
		name := string(b[offset+4 : offset+8])
		headerLen := 8

		switch size {
		case 0:
			size = len(b) - offset // 最後のボックス。ファイル末尾までが中身
		case 1:
			// 64bit 長。size(4) + type(4) の後ろに largesize(8) が入る
			if offset+16 > len(b) {
				return nil, false
			}
			large := binary.BigEndian.Uint64(b[offset+8 : offset+16])
			if large > uint64(len(b)-offset) {
				return nil, false
			}
			size = int(large)
			headerLen = 16
		}

		if size < headerLen || offset+size > len(b) {
			return nil, false
		}
		if name == typ {
			return b[offset+headerLen : offset+size], true
		}
		offset += size
	}
	return nil, false
}

// 与えられたファイル名の拡張子を、中身から判定した拡張子に置き換える。
// 判定できない形式と、表記だけが違う場合（.jpeg と .jpg）は元の名前を返す
func correctFileName(fileName string, content []byte) string {
	sniffed := sniffExt(content)
	if sniffed == "" {
		return fileName
	}
	ext := filepath.Ext(fileName)
	if normalizeExt(ext) == sniffed {
		return fileName
	}
	return strings.TrimSuffix(fileName, ext) + sniffed
}
