// Package imageconv はアップロードされた写真を WebP に変換する。
package imageconv

import (
	"bytes"
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

// ToWebP は JPEG・PNG・WebP の画像を WebP にする。大きさは変えない。
// JPEG の EXIF の向き（スマホで縦に撮った写真など）は、画素を回転させて反映する（WebP には EXIF を残さないため）。
func ToWebP(data []byte) ([]byte, error) {
	if !supported[http.DetectContentType(data)] {
		return nil, ErrUnsupportedFormat
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedFormat, err)
	}
	if cfg.Width > maxDimension || cfg.Height > maxDimension {
		return nil, ErrTooLarge
	}

	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedFormat, err)
	}
	var out bytes.Buffer
	if err := webp.Encode(&out, img, webp.Options{Quality: quality, Method: method}); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return out.Bytes(), nil
}
