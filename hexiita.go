package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nfnt/resize"
)

// サムネイル画像のMax width [px]
const (
	MaxThumbnailWidthPx = 300
	MaxImageWidth       = 1200
)

var fileNameMap = map[string]int{}

type HexoMeta struct {
	Title     string
	Date      string // like: 2020/07/16 10:49:27
	PostID    string
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
	_, _ = fmt.Fprintf(w, "postid: %s\n", m.PostID)
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

	var postID = "a"

	var ymd string
	if len(os.Args) == 2 {
		ymd = time.Now().Format("20060102")
		fmt.Println("ymd is", ymd)
	} else {
		arg := os.Args[2]
		if len(arg) == 9 {
			ymd = arg[0:8]
			postID = arg[8:]
		} else {
			ymd = os.Args[2]
		}
	}

	fmt.Println(ymd, postID)

	if len(ymd) != 8 {
		log.Fatal("ymd must be YYYYMMDD format")
	}
	ymd = ymd + postID

	ymdTime, err := time.Parse("20060102", ymd[0:8])
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

	var (
		articleFileName string // 本文をパースしてから決定
		title           string
		author          string
		tags            []string
		lede            string
		thumbnailExt    string // png, jpeg
		thumbnail       = true // 初回の画像をサムネイルにする
	)

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

				// https://github.com/laqiiz/hexiita/issues/18
				escapedTitle := strings.ReplaceAll(title, "/", "／")

				articleFileName = filepath.Join(postRoot, ymd+"_"+strings.ReplaceAll(escapedTitle, " ", "_")+".md")
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
			articleImage.MaxWidthPx = MaxImageWidth
			count := fileNameMap[articleImage.FileName]
			count++
			fileNameMap[articleImage.FileName] = count
			if count > 1 {
				// #16
				ext := filepath.Ext(articleImage.FileName)
				fileName := strings.Replace(articleImage.FileName, ext, "", 1)
				articleImage.FileName = fileName + "_" + strconv.Itoa(count) + ext
			}

			a, err := download(imageRoot, articleImage)
			if err != nil {
				log.Fatal("download image", err)
			}
			articleImage = a

			if thumbnail {
				if err := downloadWithThumbnail(imageRoot, articleImage); err != nil {
					log.Fatal("download image", err)
				}
				thumbnailExt = filepath.Ext(articleImage.FileName)
				thumbnail = false
			}

			// Alt textはHexoで表示されてしまうのでなしにする。本当はあった方が良いとは思う
			imgLine := fmt.Sprintf(`<img src="%s" alt="%s" width="%d" height="%d" loading="lazy"`, path.Join("/images", ymd, "", articleImage.FileName), articleImage.AltText, articleImage.Width, articleImage.Height)
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
		PostID:    postID,
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

func download(dir string, articleImage *ArticleImage) (*ArticleImage, error) {
	imgResp, err := http.Get(articleImage.URL)
	if err != nil {
		return nil, fmt.Errorf("cannot access image url: %w", err)
	}
	defer imgResp.Body.Close()

	file, err := os.OpenFile(filepath.Join(dir, articleImage.FileName), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, fmt.Errorf("create image file: %w", err)
	}
	defer file.Close()
	img, err := ioutil.ReadAll(imgResp.Body)
	if err != nil {
		return nil, err
	}

	if articleImage.MaxWidthPx != 0 {
		// resize target
		imgSrc, _, err := image.Decode(bytes.NewBuffer(img))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%+v\n", articleImage)
			return nil, err
		}

		width := imgSrc.Bounds().Max.X - imgSrc.Bounds().Min.X

		articleImage.Width = width
		articleImage.Height = imgSrc.Bounds().Max.Y - -imgSrc.Bounds().Min.Y

		if width > articleImage.MaxWidthPx {
			resizedImg := resize.Resize(uint(articleImage.MaxWidthPx), 0, imgSrc, resize.Lanczos3)

			articleImage.Width = resizedImg.Bounds().Max.X - resizedImg.Bounds().Min.X
			articleImage.Height = resizedImg.Bounds().Max.Y - resizedImg.Bounds().Min.Y

			ext := strings.Replace(strings.ToLower(filepath.Ext(articleImage.FileName)), ".jpeg", ".jpg", 1)

			if ext == ".jpg" {
				return articleImage, jpeg.Encode(file, resizedImg, &jpeg.Options{Quality: 100})
			} else if ext == ".png" {
				return articleImage, png.Encode(file, resizedImg)
			} else if ext == ".gif" {
				return articleImage, gif.Encode(file, resizedImg, &gif.Options{})
			} else {
				fmt.Fprintf(os.Stderr, "unknown extention: %v", ext)
			}
		}
	}

	// Originalをそのまま利用
	if _, err = io.Copy(file, bytes.NewBuffer(img)); err != nil {
		return nil, fmt.Errorf("write image file to local: %w", err)
	}
	return articleImage, nil
}

func downloadWithThumbnail(dir string, articleImage *ArticleImage) error {
	thumbnailImage := &ArticleImage{
		URL:        articleImage.URL,
		FileName:   "thumbnail" + filepath.Ext(articleImage.FileName), // パスを書き換え
		HasImage:   articleImage.HasImage,
		MaxWidthPx: MaxThumbnailWidthPx,
	}

	_, err := download(dir, thumbnailImage)
	return err
}

type ArticleImage struct {
	URL        string
	FileName   string
	AltText    string
	HasImage   bool
	MaxWidthPx int
	Width      int
	Height     int
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

		// https://github.com/laqiiz/hexiita/issues/22
		// ![2020-09-23_20h26_14.png](https://qiita-image-store.s3.ap-northeast-1.amazonaws.com/0/717995/8e698035-ea79-cf21-ce93-f70b770e0e15.png "実装した画面")
		url = strings.Split(url, " ")[0]

		return &ArticleImage{
			URL:      url,
			FileName: fileName,
			AltText:  fileName,
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

		if fileName == "" {
			fileName = strings.Replace(strings.Split(xi.URL, "/")[len(strings.Split(xi.URL, "/"))-1], "https://", "", 1)
		}

		return &ArticleImage{
			URL:      xi.URL,
			FileName: fileName,
			AltText:  xi.FileName,
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
