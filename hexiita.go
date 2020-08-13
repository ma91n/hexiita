package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type HexoMeta struct {
	Title     string
	Date      string // like: 2020/07/16 10:49:27
	Tags      []string
	Category  string
	Thumbnail string
	Author    string
	Featured  bool
	Lede      string
}

func (m HexoMeta) WriteHeader(w io.Writer) error {
	_, _ = fmt.Fprintf(w, "title: %s\n", m.Title)
	_, _ = fmt.Fprintf(w, "date: %s\n", m.Date)
	_, _ = fmt.Fprintf(w, "tag:\n")
	for _, tag := range m.Tags {
		_, _ = fmt.Fprintf(w, "  - %s\n", tag)
	}
	_, _ = fmt.Fprintf(w, "category:\n")
	_, _ = fmt.Fprintf(w, "  - %s\n", m.Category)
	_, _ = fmt.Fprintf(w, "thumbnail: %s\n", m.Thumbnail)
	_, _ = fmt.Fprintf(w, "author: %s\n", m.Author)
	_, _ = fmt.Fprintf(w, "featured: %v\n", m.Featured)
	_, _ = fmt.Fprintf(w, "lede: %s\n", m.Lede)
	_, _ = fmt.Fprintln(w, "---")
	return nil
}

func main() {

	if len(os.Args) == 1 {
		log.Fatal("url argument is required")
	}

	url := os.Args[1]
	if url == "" {
		log.Fatal("URL is required variable")
	}

	var ymd string
	if len(os.Args) == 2 {
		ymd = time.Now().Format("20060102")
		fmt.Println("ymd is ", ymd)
	} else {
		ymd = os.Args[2]
	}

	if len(ymd) != 8 {
		log.Fatal("ymd must be YYYYMMDD format")
	}
	ymdTime, err := time.Parse("20060102", ymd)
	if err != nil {
		log.Fatal("ymd is invalid format")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("cannot get user home", err)
	}
	blogRoot := filepath.Join(home, "tech-blog", "source")
	postRoot := filepath.Join(blogRoot, "_posts")
	imageRoot := filepath.Join(blogRoot, "images", ymd)
	if err := os.MkdirAll(imageRoot, 0777); err != nil {
		fmt.Println("mkdir", err)
	}

	var articleFileName string // 本文をパースしてから決定
	var title string
	var author string
	var tags []string
	var lede string
	thumbnail := true       // 初回の画像をサムネイルにする
	var thumbnailExt string // png, jpeg

	if !strings.HasSuffix(url, ".md") {
		url = url + ".md"
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("cannot access qiita url page", err)
	}
	defer resp.Body.Close()

	lineNo := 0
	scanner := bufio.NewScanner(resp.Body)
	var hexoArticleContents []string

	for scanner.Scan() {
		line := scanner.Text()
		lineNo++

		if 1 <= lineNo && lineNo <= 6 {
			if strings.HasPrefix(line, "title") {
				title = line[len("title: "):]
				articleFileName = filepath.Join(postRoot, ymd+"_"+strings.ReplaceAll(title, " ", "_")+".md")
			}
			if strings.HasPrefix(line, "tags") {
				tags = strings.Split(line[len("tags: "):], " ")
			}
			if strings.HasPrefix(line, "author") {
				author = line[len("author: "):]
			}
			continue
		}

		if 6 < lineNo && lineNo < 14 {
			if strings.Trim(line, " ") != "" && !strings.HasPrefix(line, "#") {
				regex := regexp.MustCompile(`\(.*\)`) // リンクがLedeに出ると可読性に欠けるので削除
				simpleLine := regex.ReplaceAllString(line, "")
				lede = lede + strings.ReplaceAll(simpleLine, "**", "") // BOLDは削除しておく
			}
		}

		articleImage, err := ExtractImageURL(line)
		if err != nil {
			log.Fatalf("image link parse: %v: %s", err, line)
		}

		if articleImage.HasImage {
			if err := download(imageRoot, articleImage); err != nil {
				log.Fatal("download image", err)
			}

			if thumbnail {
				if err := downloadWithThumbnail(imageRoot, articleImage); err != nil {
					log.Fatal("download image", err)
				}
				thumbnailExt = filepath.Ext(articleImage.FileName)
				thumbnail = false
			}

			// Alt textはHexoで表示されてしまうのでなしにする。本当はあった方が良いとは思う
			imgLine := fmt.Sprintf("![](%s)", path.Join("/images", ymd, articleImage.FileName))
			hexoArticleContents = append(hexoArticleContents, imgLine)
			continue
		}

		// 名前付きのCode Block
		if strings.HasPrefix(line, "```") {
			line = strings.Replace(line, ":", " ", 1)
		}

		hexoArticleContents = append(hexoArticleContents, line)
	}

	if lineNo == 0 {
		// URL不正？
		log.Fatal("Specific URL page does not exists payload: ", url)
	}

	if len(tags) == 1 {
		tags = append([]string{"Programming"}, tags...)
	}

	var category string
	updateTags := tags
	for i, tag := range tags {
		if strings.ToLower(tag) == "infrastructure" {
			category = "Infrastructure"
			updateTags = remove(tags, i)
		} else if strings.ToLower(tag) == "programming" || IsProgrammingCategory(tag) {
			category = "Programming"
			updateTags = remove(tags, i)
		} else if IsDBCategory(tag) {
			category = "DB"
			updateTags = remove(tags, i)
		} else if strings.ToLower(tag) == "culture" {
			category = "culture"
			updateTags = remove(tags, i)
		} else if strings.ToLower(tag) == "datascience" {
			category = "DataScience"
			updateTags = remove(tags, i)
		} else if strings.ToLower(tag) == "iot" {
			category = "IoT"
			updateTags = remove(tags, i)
		}
	}
	if category == "" {
		category = "Infrastructure"
	}

	metaTitle := title
	if !strings.HasPrefix(title, `"`) {
		metaTitle = `"` + metaTitle + `"`
	}

	hexoMeta := &HexoMeta{
		Title:     metaTitle,
		Date:      ymdTime.Format("2006/01/02 15:04:05"),
		Tags:      updateTags,
		Category:  category,
		Thumbnail: path.Join("/images", ymd, "thumbnail"+thumbnailExt),
		Author:    author,
		Featured:  true,
		Lede:      "\"" + lede + "\"",
	}

	file, err := os.OpenFile(articleFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal("create post file: ", articleFileName, err)
	}
	defer file.Close()

	if err = hexoMeta.WriteHeader(file); err != nil {
		log.Fatal("write hexo header: ", err)
	}

	// 本文の出力
	for _, line := range hexoArticleContents {
		_, err := file.Write([]byte(line))
		if err != nil {
			log.Fatal("save post file: ", err)
		}
		fmt.Fprintln(file) // 改行
	}

	fmt.Println("finished")
}

func download(dir string, articleImage *ArticleImage) error {
	imgResp, err := http.Get(articleImage.URL)
	if err != nil {
		return fmt.Errorf("cannot access image url: %w", err)
	}
	defer imgResp.Body.Close()

	file, err := os.OpenFile(filepath.Join(dir, articleImage.FileName), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("create image file: %w", err)
	}
	if _, err = io.Copy(file, imgResp.Body); err != nil {
		return fmt.Errorf("write image file to local: %w", err)
	}
	return nil
}

func downloadWithThumbnail(dir string, articleImage *ArticleImage) error {
	thumbnailImage := &ArticleImage{
		URL:      articleImage.URL,
		FileName: "thumbnail" + filepath.Ext(articleImage.FileName), // パスを書き換え
		HasImage: articleImage.HasImage,
	}

	return download(dir, thumbnailImage)
}

type ArticleImage struct {
	URL      string
	FileName string
	HasImage bool
}

type XMLImage struct {
	URL      string `xml:"src,attr"`
	FileName string `xml:"alt,attr"`
}

func ExtractImageURL(line string) (*ArticleImage, error) {
	// 2パターン
	// ![alt text](https:///qiita-image-store.s3.ap-northeast-1.amazonaws.com/hogehoge)
	// <img src="https:///qiita-image-store.s3.ap-northeast-1.amazonaws.com/hogehoge" width="800" height="300" alt="alt text">

	if strings.Contains(line, "![") && strings.Contains(line, "](") {

		nameStart := strings.Index(line, "![") + 2
		nameEnd := strings.Index(line[nameStart:], "]") + nameStart
		fileName := line[nameStart:nameEnd]

		// ファイル名に空白が含まれている場合は、アンダースコアに変換
		fileName = strings.ReplaceAll(fileName, " ", "_")

		if len(fileName) > 100 {
			// クリップボードから直接貼り付けたときに、意味のない長いファイル名になるので短くする
			fileName = fileName[:15] + filepath.Ext(fileName)
		}

		start := strings.Index(line, "https")
		end := strings.Index(line[start:], ")") + start
		url := line[start:end]


		return &ArticleImage{
			URL:      url,
			FileName: fileName,
			HasImage: true,
		}, nil
	}

	if strings.Contains(line, "<img") && strings.Contains(line, "src=") {

		tagStart := strings.Index(line, "<img")
		tagEnd := strings.Index(line, ">") + 1
		xmlElm := line[tagStart:tagEnd] + "</img>" // XMLパースのためにタグを追加

		var xi XMLImage
		if err := xml.Unmarshal([]byte(xmlElm), &xi); err != nil {
			return nil, fmt.Errorf("image xml tag parse: %w", err)
		}

		// ファイル名に空白が含まれている場合は、アンダースコアに変換
		fileName := strings.ReplaceAll(xi.FileName, " ", "_")

		return &ArticleImage{
			URL:      xi.URL,
			FileName: fileName,
			HasImage: true,
		}, nil
	}

	return &ArticleImage{
		URL:      "",
		FileName: "",
		HasImage: false,
	}, nil
}

func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

// pgm list
var pgm = map[string]bool{
	"go":     true,
	"golang": true,
	"python": true,
	"ruby":   true,
	"r":      true,
	"scala":  true,
	"java":   true,
	"c":      true,
	"clang":  true,
	"c++":    true,
	"shell":  true,
}

func IsProgrammingCategory(tag string) bool {
	return pgm[strings.ToLower(tag)]
}

var db = map[string]bool{
	"sql": true,
	"db":  true,
	"rdb": true,
}

func IsDBCategory(tag string) bool {
	return db[strings.ToLower(tag)]
}
