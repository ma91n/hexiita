package main

import (
	"encoding/binary"
	"testing"
)

func box(typ string, payload ...[]byte) []byte {
	var body []byte
	for _, p := range payload {
		body = append(body, p...)
	}
	b := make([]byte, 8, 8+len(body))
	binary.BigEndian.PutUint32(b[0:4], uint32(8+len(body)))
	copy(b[4:8], typ)
	return append(b, body...)
}

func fullBoxPrefix() []byte {
	return []byte{0, 0, 0, 0} // version + flags
}

func avifFile(width, height uint32) []byte {
	ispe := make([]byte, 0, 12)
	ispe = append(ispe, fullBoxPrefix()...)
	ispe = binary.BigEndian.AppendUint32(ispe, width)
	ispe = binary.BigEndian.AppendUint32(ispe, height)

	ftyp := box("ftyp", []byte("avif"), []byte("\x00\x00\x00\x00"), []byte("avifmif1"))
	meta := box("meta", fullBoxPrefix(), box("hdlr", fullBoxPrefix()), box("iprp", box("ipco", box("ispe", ispe))))
	return append(ftyp, meta...)
}

func TestSniffExt(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"png", []byte("\x89PNG\r\n\x1a\n----"), ".png"},
		{"jpeg", []byte("\xff\xd8\xff\xe0----"), ".jpg"},
		{"gif", []byte("GIF89a--------"), ".gif"},
		{"webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), ".webp"},
		{"avif", avifFile(100, 50), ".avif"},
		// アニメーション AVIF。major brand が avis になる
		{"avis", box("ftyp", []byte("avis"), []byte("\x00\x00\x00\x00"), []byte("avifavismsf1")), ".avif"},
		// HEIC は AVIF ではない。ブランドで見分けられることの確認
		{"heic", box("ftyp", []byte("heic"), []byte("\x00\x00\x00\x00"), []byte("mif1heic")), ""},
		// テキスト形式と未知の形式は拡張子に触らせない
		{"svg", []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg">`), ""},
		{"empty", nil, ""},
	}
	for _, tt := range tests {
		if got := sniffExt(tt.in); got != tt.want {
			t.Errorf("%s: sniffExt = %q, 期待 %q", tt.name, got, tt.want)
		}
	}
}

// 中身が AVIF なのに名前が .png のファイルができると、GitHub Pages が
// 拡張子から Content-Type を決めるので AVIF を image/png として配ることになる
func TestCorrectFileNameAvifNamedPng(t *testing.T) {
	if got := correctFileName("arc.png", avifFile(1920, 1080)); got != "arc.avif" {
		t.Errorf("correctFileName = %q, 期待 arc.avif", got)
	}
}

// .jpeg と .jpg は同じ形式なので、表記の違いだけで名前を書き換えない
func TestCorrectFileNameKeepsJpegSpelling(t *testing.T) {
	if got := correctFileName("photo.jpeg", []byte("\xff\xd8\xff\xe0----")); got != "photo.jpeg" {
		t.Errorf("correctFileName = %q, 期待 photo.jpeg", got)
	}
}

// 判定できない形式は元の名前のまま（SVG の拡張子を落とさない）
func TestCorrectFileNameKeepsUnknown(t *testing.T) {
	if got := correctFileName("logo.svg", []byte("<svg></svg>")); got != "logo.svg" {
		t.Errorf("correctFileName = %q, 期待 logo.svg", got)
	}
}

func TestAvifSize(t *testing.T) {
	w, h, ok := avifSize(avifFile(1400, 764))
	if !ok || w != 1400 || h != 764 {
		t.Errorf("avifSize = %dx%d (ok=%v), 期待 1400x764", w, h, ok)
	}
}

// 寸法が読めないときは 0 のままにして、呼び出し側が width / height を書かない
func TestAvifSizeWithoutIspe(t *testing.T) {
	file := append(box("ftyp", []byte("avif")), box("meta", fullBoxPrefix())...)
	if _, _, ok := avifSize(file); ok {
		t.Error("ispe が無いのに寸法を返した")
	}
}
