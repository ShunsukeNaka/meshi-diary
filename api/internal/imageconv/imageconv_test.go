package imageconv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
	"time"

	"github.com/gen2brain/webp"
)

// testImage は左半分が赤、右半分が青の画像を作る。向きを確かめるのに使う。
func testImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{R: 255, A: 255}
			if x >= w/2 {
				c = color.RGBA{B: 255, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func encode(t *testing.T, format string, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	var err error
	switch format {
	case "jpeg":
		err = jpeg.Encode(&buf, img, nil)
	case "png":
		err = png.Encode(&buf, img)
	case "webp":
		err = webp.Encode(&buf, img)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func decodeWebP(t *testing.T, data []byte) image.Image {
	t.Helper()
	if !bytes.HasPrefix(data, []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WEBP")) {
		t.Fatalf("output is not webp: % x", data[:12])
	}
	img, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestToWebP(t *testing.T) {
	for _, format := range []string{"jpeg", "png", "webp"} {
		t.Run(format+" keeps the size", func(t *testing.T) {
			out, err := ToWebP(encode(t, format, testImage(64, 48)))
			if err != nil {
				t.Fatal(err)
			}
			if b := decodeWebP(t, out).Bounds(); b.Dx() != 64 || b.Dy() != 48 {
				t.Fatalf("size = %v, want 64x48", b)
			}
		})
	}

	t.Run("plain WebP is returned as is", func(t *testing.T) {
		data := encode(t, "webp", testImage(64, 48))
		if out := mustToWebP(t, data); !bytes.Equal(out, data) {
			t.Fatalf("webp was re-encoded (%d bytes -> %d bytes)", len(data), len(out))
		}
	})

	for _, chunk := range []string{"EXIF", "XMP "} {
		t.Run("WebP with "+chunk+" is re-encoded without it", func(t *testing.T) {
			data := withMetadata(t, encode(t, "webp", testImage(64, 48)), chunk, 64, 48)
			out := mustToWebP(t, data)
			if bytes.Equal(out, data) || bytes.Contains(out, []byte(chunk)) {
				t.Fatalf("%s chunk is kept", chunk)
			}
			if b := decodeWebP(t, out).Bounds(); b.Dx() != 64 || b.Dy() != 48 {
				t.Fatalf("size = %v, want 64x48", b)
			}
		})
	}

	t.Run("broken WebP", func(t *testing.T) {
		data := encode(t, "webp", testImage(64, 48))
		if _, err := ToWebP(data[:len(data)/2]); !errors.Is(err, ErrUnsupportedFormat) {
			t.Fatalf("err = %v, want ErrUnsupportedFormat", err)
		}
	})

	t.Run("JPEG EXIF orientation is applied", func(t *testing.T) {
		data := withOrientation(t, encode(t, "jpeg", testImage(64, 48)), 6) // 6 = 時計回りに90度回して表示する
		img := decodeWebP(t, mustToWebP(t, data))
		if b := img.Bounds(); b.Dx() != 48 || b.Dy() != 64 {
			t.Fatalf("size = %v, want 48x64 (rotated)", b)
		}
		// 時計回りに90度回すと、左にあった赤は上に来る
		if r, _, b, _ := img.At(24, 4).RGBA(); r < b {
			t.Fatalf("top is not red: r=%d b=%d", r, b)
		}
	})

	t.Run("unsupported formats", func(t *testing.T) {
		gif := []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")
		for name, data := range map[string][]byte{
			"text":        []byte("hello"),
			"gif":         gif,
			"broken jpeg": []byte("\xff\xd8\xff\xe0garbage"),
			"empty":       {},
		} {
			if _, err := ToWebP(data); !errors.Is(err, ErrUnsupportedFormat) {
				t.Fatalf("%s: err = %v, want ErrUnsupportedFormat", name, err)
			}
		}
	})
}

func mustToWebP(t *testing.T, data []byte) []byte {
	t.Helper()
	out, err := ToWebP(data)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// withOrientation は JPEG の先頭に、Orientation だけを持つ EXIF（APP1）を差し込む。
func withOrientation(t *testing.T, jpg []byte, orientation byte) []byte {
	t.Helper()
	tiff := []byte{
		'M', 'M', 0, 42, 0, 0, 0, 8, // ビッグエンディアン、IFD0 は 8 バイト目から
		0, 1, // エントリは1つ
		0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, orientation, 0, 0, // Orientation (0x0112), SHORT, 1個
		0, 0, 0, 0, // 次の IFD はない
	}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	n := len(payload) + 2
	app1 := append([]byte{0xFF, 0xE1, byte(n >> 8), byte(n)}, payload...)
	return append(append([]byte{0xFF, 0xD8}, app1...), jpg[2:]...)
}

// withMetadata は、拡張形式（VP8X）にした WebP の最後に、chunk（"EXIF" か "XMP "）のチャンクを足す。
func withMetadata(t *testing.T, simple []byte, chunk string, w, h int) []byte {
	t.Helper()
	flag := map[string]byte{"EXIF": 0x08, "XMP ": 0x04}[chunk]
	vp8x := []byte{flag, 0, 0, 0,
		byte(w - 1), byte((w - 1) >> 8), byte((w - 1) >> 16),
		byte(h - 1), byte((h - 1) >> 8), byte((h - 1) >> 16)}
	var body bytes.Buffer
	body.WriteString("WEBP")
	writeChunk(&body, "VP8X", vp8x)
	body.Write(simple[12:]) // もとの画像のチャンク（VP8 か VP8L）
	writeChunk(&body, chunk, []byte("GPS 35.0,135.0"))
	var out bytes.Buffer
	out.WriteString("RIFF")
	binary.Write(&out, binary.LittleEndian, uint32(body.Len()))
	out.Write(body.Bytes())
	return out.Bytes()
}

func writeChunk(buf *bytes.Buffer, name string, data []byte) {
	buf.WriteString(name)
	binary.Write(buf, binary.LittleEndian, uint32(len(data)))
	buf.Write(data)
	if len(data)%2 == 1 {
		buf.WriteByte(0)
	}
}

// TestToWebPSpeed は、スマホの写真くらいの大きさ（4000x3000）の変換にかかる時間を記録する。
func TestToWebPSpeed(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	data := encode(t, "jpeg", testImage(4000, 3000))
	start := time.Now()
	out := mustToWebP(t, data)
	t.Logf("4000x3000 jpeg %d bytes -> webp %d bytes in %v", len(data), len(out), time.Since(start))
}
