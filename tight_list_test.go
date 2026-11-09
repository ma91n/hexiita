package main

import (
	"reflect"
	"testing"
)

func TestTightenLists(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "入れ子の項目の間の空行を消す",
			in:   []string{"- 親1", "    - 子", "", "- 親2", "    - 子"},
			want: []string{"- 親1", "    - 子", "- 親2", "    - 子"},
		},
		{
			name: "番号付きのリスト",
			in:   []string{"1. a", "", "2. b"},
			want: []string{"1. a", "2. b"},
		},
		{
			name: "リストの前後の空行は残す",
			in:   []string{"本文", "", "- a", "", "本文"},
			want: []string{"本文", "", "- a", "", "本文"},
		},
		{
			name: "項目の中の2段落目がある形は残す",
			in:   []string{"- a", "", "  続きの段落", "", "- b"},
			want: []string{"- a", "", "  続きの段落", "", "- b"},
		},
		{
			name: "コードフェンスの中は触らない",
			in:   []string{"```md", "- a", "", "- b", "```"},
			want: []string{"```md", "- a", "", "- b", "```"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tightenLists(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
