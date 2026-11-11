// Package imageconv はアップロードされた写真を WebP に変換する。
package imageconv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"net/http"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp" // WebP の書き出し。読み込み（image.Decode）にも登録される
)

// ErrUnsupportedFormat は JPEG・PNG・WebP 以外のファイルのときに返す。
var ErrUnsupportedFormat = errors.New("unsupported image format")

// ErrTooLarge は WebP にできない大きさ（縦か横が maxDimension を超える）のときに返す。
var ErrTooLarge = errors.New("image is too large")

// maxDimension は WebP で扱える縦横の最大（libwebp の上限）。
const maxDimension = 16383

// quality は WebP の画質。写真を見るには十分で、ファイルは小さくなる。
const quality = 80

// method は圧縮の手間（0=速い〜6=小さい）。4000x3000 の写真で 0 は約2秒、既定の 4 は約10秒かかり、
// ファイルの大きさはほとんど変わらなかったので、いちばん速い 0 にする。
const method = 0

// supported は受け付ける形式（http.DetectContentType の結果）。
var supported = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// ThumbnailShortSide はサムネイルの短辺（px）。カレンダーのマス（スマホで約 43×38 px、PC で約 100×70 px）と
// 一覧（56 px）に object-cover で表示するので、画素の密度が 2〜3 倍の画面でもぼやけない大きさにする。
const ThumbnailShortSide = 192

// Converted は保存する写真と、そのサムネイル。どちらも WebP。
type Converted struct {
	Photo     []byte // 送られた写真の大きさのまま
	Thumbnail []byte // 短辺を ThumbnailShortSide に縮めたもの
}

// Convert は JPEG・PNG・WebP の画像を WebP にし、サムネイルも作る。写真の大きさは変えない。
// JPEG の EXIF の向き（スマホで縦に撮った写真など）は、画素を回転させて反映する（WebP には EXIF を残さないため）。
// WebP は、EXIF・XMP・アニメーションがなければ、変換し直さずにそのまま返す（圧縮を二度すると画質が落ちるため）。
func Convert(data []byte) (Converted, error) {
	contentType := http.DetectContentType(data)
	if !supported[contentType] {
		return Converted{}, ErrUnsupportedFormat
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Converted{}, fmt.Errorf("%w: %v", ErrUnsupportedFormat, err)
	}
	if cfg.Width > maxDimension || cfg.Height > maxDimension {
		return Converted{}, ErrTooLarge
	}

	if contentType == "image/webp" && isPlainWebP(data) {
		// 最後まで読んで、壊れていないかを確かめる。読んだ画像はサムネイルに使う
		img, err := webp.Decode(bytes.NewReader(data))
		if err != nil {
			return Converted{}, fmt.Errorf("%w: %v", ErrUnsupportedFormat, err)
		}
		thumb, err := encodeWebP(shrink(img))
		if err != nil {
			return Converted{}, err
		}
		return Converted{Photo: data, Thumbnail: thumb}, nil
	}

	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return Converted{}, fmt.Errorf("%w: %v", ErrUnsupportedFormat, err)
	}
	photo, err := encodeWebP(img)
	if err != nil {
		return Converted{}, err
	}
	thumb, err := encodeWebP(shrink(img))
	if err != nil {
		return Converted{}, err
	}
	return Converted{Photo: photo, Thumbnail: thumb}, nil
}

// Thumbnail は保存してある写真（Convert で作った WebP）から、サムネイルを作る。前からある写真のサムネイルを作るのに使う。
// Convert と同じく、JPEG の EXIF の向きは画素を回転させて反映する（保存してある WebP には EXIF がないので、ふつうは何もしない）。
func Thumbnail(data []byte) ([]byte, error) {
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedFormat, err)
	}
	return encodeWebP(shrink(img))
}

// shrink は短辺が ThumbnailShortSide になるように縮める。もとから小さい画像は、大きくせずにそのまま返す。
func shrink(img image.Image) image.Image {
	b := img.Bounds()
	if min(b.Dx(), b.Dy()) <= ThumbnailShortSide {
		return img
	}
	if b.Dx() < b.Dy() {
		return imaging.Resize(img, ThumbnailShortSide, 0, imaging.Lanczos)
	}
	return imaging.Resize(img, 0, ThumbnailShortSide, imaging.Lanczos)
}

func encodeWebP(img image.Image) ([]byte, error) {
	var out bytes.Buffer
	if err := webp.Encode(&out, img, webp.Options{Quality: quality, Method: method}); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return out.Bytes(), nil
}

// isPlainWebP は、WebP（RIFF）のチャンクに EXIF・XMP（撮影場所などが入りうる）と
// アニメーションがないかを確かめる。ブラウザの canvas で書き出した WebP はこれに当てはまる。
func isPlainWebP(data []byte) bool {
	// 先頭 12 バイトは "RIFF"、ファイルの大きさ、"WEBP"。そのあとにチャンク（4 文字の名前、大きさ、中身）が続く
	for p := 12; p+8 <= len(data); {
		switch string(data[p : p+4]) {
		case "EXIF", "XMP ", "ANIM", "ANMF":
			return false
		}
		size := int(binary.LittleEndian.Uint32(data[p+4 : p+8]))
		p += 8 + size + size%2 // 中身が奇数バイトのときは、1 バイト詰め物がある
	}
	return true
}
