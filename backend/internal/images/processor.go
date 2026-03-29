package images

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif" // register GIF decoder
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"mime/multipart"
	"net/http"
)

// Constraints
const (
	MaxImageBytes = 5 * 1024 * 1024 // 5 MB per image
	MaxImages     = 10              // images per upload request
	JPEGQuality   = 85              // JPEG encoding quality

	MaxOriginalDim  = 4096 // cap very large images at this px
	MaxLargeDim     = 1200
	MaxMediumDim    = 600
	MaxThumbnailDim = 200
)

// allowedMIME is the set of accepted Content-Types.
var allowedMIME = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": false, // webp decoding not bundled — reject
}

// ProcessedVariant holds one size-variant of an uploaded image.
type ProcessedVariant struct {
	Size   ImageSize
	Data   []byte
	Width  int
	Height int
}

// ValidateImageFile checks the file header for allowed MIME type and size.
func ValidateImageFile(fh *multipart.FileHeader) error {
	if fh.Size > MaxImageBytes {
		return fmt.Errorf("file %q is too large (max 5 MB, got %.1f MB)",
			fh.Filename, float64(fh.Size)/1024/1024)
	}

	// Open file to check magic bytes (more secure than extension/Content-Type)
	f, err := fh.Open()
	if err != nil {
		return fmt.Errorf("cannot open %q: %w", fh.Filename, err)
	}
	defer f.Close()

	// Read first 512 bytes for magic byte detection
	buffer := make([]byte, 512)
	n, err := f.Read(buffer)
	if err != nil {
		return fmt.Errorf("cannot read %q: %w", fh.Filename, err)
	}
	buffer = buffer[:n]

	// Detect actual MIME type from magic bytes
	detectedType := http.DetectContentType(buffer)

	// Check against allowed MIME types
	if allowed, known := allowedMIME[detectedType]; !known || !allowed {
		return fmt.Errorf("file %q: invalid type %q (magic byte check failed, accepted: JPEG, PNG, GIF)",
			fh.Filename, detectedType)
	}

	return nil
}

// ProcessImageFile decodes the multipart file, then generates 4 JPEG variants:
// original (capped at MaxOriginalDim), large, medium, thumbnail.
func ProcessImageFile(fh *multipart.FileHeader) ([]ProcessedVariant, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", fh.Filename, err)
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %q: %w", fh.Filename, err)
	}

	type spec struct {
		size ImageSize
		max  int
	}
	specs := []spec{
		{SizeOriginal, MaxOriginalDim},
		{SizeLarge, MaxLargeDim},
		{SizeMedium, MaxMediumDim},
		{SizeThumbnail, MaxThumbnailDim},
	}

	variants := make([]ProcessedVariant, 0, len(specs))
	for _, sp := range specs {
		resized := resizeFit(src, sp.max)
		data, encErr := encodeJPEG(resized)
		if encErr != nil {
			return nil, fmt.Errorf("encode %q/%s: %w", fh.Filename, sp.size, encErr)
		}
		b := resized.Bounds()
		variants = append(variants, ProcessedVariant{
			Size:   sp.size,
			Data:   data,
			Width:  b.Dx(),
			Height: b.Dy(),
		})
	}
	return variants, nil
}

// ════════════════════════════════════════════════════════════════════════════
// Internal helpers
// ════════════════════════════════════════════════════════════════════════════

// resizeFit scales src so that its longest side is at most maxDim pixels,
// preserving the aspect ratio. Uses bilinear sampling.
// If the image already fits, it is returned unchanged.
func resizeFit(src image.Image, maxDim int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxDim && h <= maxDim {
		return src
	}

	var newW, newH int
	if w >= h {
		newW = maxDim
		newH = int(float64(h) * float64(maxDim) / float64(w))
	} else {
		newH = maxDim
		newW = int(float64(w) * float64(maxDim) / float64(h))
	}
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewNRGBA(image.Rect(0, 0, newW, newH))
	// Bilinear interpolation
	for dy := 0; dy < newH; dy++ {
		for dx := 0; dx < newW; dx++ {
			// Map destination pixel to source position (fractional)
			sx := float64(dx) * float64(w-1) / float64(newW-1)
			sy := float64(dy) * float64(h-1) / float64(newH-1)

			x0, y0 := int(sx), int(sy)
			x1, y1 := x0+1, y0+1
			if x1 >= w {
				x1 = w - 1
			}
			if y1 >= h {
				y1 = h - 1
			}

			xf, yf := sx-float64(x0), sy-float64(y0)

			c00 := toNRGBA(src.At(b.Min.X+x0, b.Min.Y+y0))
			c10 := toNRGBA(src.At(b.Min.X+x1, b.Min.Y+y0))
			c01 := toNRGBA(src.At(b.Min.X+x0, b.Min.Y+y1))
			c11 := toNRGBA(src.At(b.Min.X+x1, b.Min.Y+y1))

			dst.SetNRGBA(dx, dy, color.NRGBA{
				R: lerp4(c00.R, c10.R, c01.R, c11.R, xf, yf),
				G: lerp4(c00.G, c10.G, c01.G, c11.G, xf, yf),
				B: lerp4(c00.B, c10.B, c01.B, c11.B, xf, yf),
				A: lerp4(c00.A, c10.A, c01.A, c11.A, xf, yf),
			})
		}
	}
	// Flatten alpha onto white background before JPEG encoding
	flat := image.NewRGBA(dst.Bounds())
	draw.Draw(flat, flat.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(flat, flat.Bounds(), dst, image.Point{}, draw.Over)
	return flat
}

func toNRGBA(c color.Color) color.NRGBA {
	r, g, bv, a := c.RGBA()
	if a == 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{
		R: uint8(r * 255 / a),
		G: uint8(g * 255 / a),
		B: uint8(bv * 255 / a),
		A: uint8(a >> 8),
	}
}

func lerp4(a, b, c, d uint8, xf, yf float64) uint8 {
	top := float64(a)*(1-xf) + float64(b)*xf
	bot := float64(c)*(1-xf) + float64(d)*xf
	return uint8(top*(1-yf) + bot*yf)
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
