# hexiita

Hexiita is a migration tool that convert Qiita to Hexo format.

## Installation

```
go install github.com/ma91n/hexiita@latest
```

## Rquirements

```sh
cd <user-home> # Win:%USER_PROFILE% Mac:~
git clone --depth 1 https://github.com/future-architect/tech-blog.git
cd tech-blog
npm install
cd ../
git clone --depth 1  https://github.com/future-architect/future-architect.github.io.git
cd tech-blog

# [Windowsの場合]hexoの生成先を future-architect.github.ioにする
mklink /J public %USER_PROFILE%/future-architect.github.io.git
```

## Usage

```
hexiita <qiita url>
```

成功すると、tech-blogフォルダにファイルが生成されている。

## Options

日付も指定可能。デフォルトでは現在日付になる。

```sh
hexiita <qiita url> 20201231
```

### 連載

連載に属する記事は `-series` で連載名を指定する。値がそのまま連載名として表示され、
記事末に前 / 次 と索引へのリンクが出る。

```sh
hexiita -series "Go1.27リリース" <qiita url> 20260804a
```

連載の索引記事は `-index` も付ける。`インデックス` タグが足され、
連載の他の記事から索引へ戻れるようになる。

```sh
hexiita -series "Go1.27リリース" -index <qiita url> 20260728a
```

フラグは URL より **前** に置くこと（Go の `flag` は最初の非フラグ引数で解釈を止める）。

連載名の決め方:

- 表示は「連載：&lt;値&gt;」になるので、**末尾に「連載」「企画」は付けない**（`Go1.27リリース`）
- 年ごとに続く企画は年を付けて区別する。括弧は使わない（`春の入門祭り2025`）
- バージョン番号はタグの表記に合わせる（`Go1.27リリース`。`Go 1.27` ではない）
- 連載を束ねる年間企画（「2026年 フューチャー技術ブログリレー企画」など）には付けない

