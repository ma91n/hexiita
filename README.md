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

