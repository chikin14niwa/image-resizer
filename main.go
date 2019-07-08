package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

const (
	TYPE_JPG = "jpeg"
	TYPE_PNG = "png"
)

func ResizeImage(srcPath string, w, h int, outputDir, suffix string) error {
	// 画像ファイルを開く
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	// image.Decodeのunexpected EOF対策
	imgHeader := bytes.NewBuffer(nil)
	r := io.TeeReader(src, imgHeader)

	_, t, err := image.DecodeConfig(r)
	if err != nil {
		return err
	}

	if t != TYPE_JPG && t != TYPE_PNG {
		return errors.New("This method only run jpeg and png")
	}

	var imgSrc image.Image
	mReader := io.MultiReader(imgHeader, src)
	if t == TYPE_JPG {
		imgSrc, err = jpeg.Decode(mReader)
	} else {
		imgSrc, err = png.Decode(mReader)
	}
	if err != nil {
		return err
	}

	// rectange of image
	rctSrc := imgSrc.Bounds()
	var newW, newH int
	if w > 0 && h > 0 {
		newH = h
		newW = w
	} else if h > 0 {
		newH = h
		newW = rctSrc.Dx() * (newH * 100 / rctSrc.Dy()) / 100
	} else if w > 0 {
		newW = w
		newH = rctSrc.Dy() * (newW * 100 / rctSrc.Dx()) / 100
	}

	imgDst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(imgDst, imgDst.Bounds(), imgSrc, rctSrc, draw.Over, nil)

	if _, err := os.Stat(outputDir); err != nil {
		// 出力用ディレクトリが存在しないため、作成する。
		if dirErr := os.Mkdir(outputDir, os.ModeDir); dirErr != nil {
			return dirErr
		}
	}

	_, fileName := filepath.Split(srcPath)
	outFile := fileName
	if suffix != "" {
		outFile = fmt.Sprintf("%[1]s%[3]s.%[2]s", strings.Split(fileName, "."), suffix)
	}
	outPath := filepath.Join(outputDir, outFile)

	if _, err := os.Stat(outPath); err == nil {
		// 出力用ファイルが存在する場合消す。
		if rmErr := os.Remove(outPath); rmErr != nil {
			return rmErr
		}
	}
	dst, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if t == TYPE_JPG {
		if err := jpeg.Encode(dst, imgDst, &jpeg.Options{Quality: 100}); err != nil {
			return err
		}
	} else if t == TYPE_PNG {
		if err := png.Encode(dst, imgDst); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// コマンドライン引数の設定
	var (
		outputDir  = flag.String("outputDir", "output", "リサイズ後の出力先を指定します。ない場合は作ります。")
		width      = flag.Int("width", 0, "リサイズ後の画像サイズです。-1を指定した場合、高さから自動で計算されます。")
		height     = flag.Int("height", 0, "リサイズ後の画像サイズです。-1を指定した場合、幅から自動で計算されます。")
		inputFiles = flag.String("inputFiles", "", "画像変換するファイルです。,区切りで複数ファイルを指定できます。baseDirオプションを使用することで、相対位置を変更することができます。")
		baseDir    = flag.String("baseDir", "", "入力ファイルの基準となるディレクトリ位置です。デフォルトは実行ファイルを実行した位置です。")
		suffix     = flag.String("suffix", "", "変換後の画像名にsuffixで指定した文字列を付与します。例: -sufix _resized A01.jpg -> A01_resized.jpg")
	)
	flag.Parse()

	// 引数チェック。必須はinputFilesとheight, widthのいずれか。
	if *inputFiles == "" {
		fmt.Println("inputFilesの指定は必須です。")
		os.Exit(-1)
	}

	if *width < 1 && *height < 1 {
		fmt.Println("width, heightのいずれかは1以上の整数を指定する必要があります。")
		os.Exit(-1)
	}

	fileList := strings.Split(*inputFiles, ",")
	for i, v := range fileList {
		// baseDirが設定されていても絶対パスで指定されていれば、baseDirの設定を適用しない。
		if *baseDir != "" {
			if !filepath.IsAbs(v) {
				fileList[i] = filepath.Join(*baseDir, v)
			}
		}

		if err := ResizeImage(fileList[i], *width, *height, *outputDir, *suffix); err != nil {
			fmt.Printf("[ERROR] %s: %s\n", v, err.Error())
		}
	}
}
