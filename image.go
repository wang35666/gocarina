package gocarina

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand"
	"os"
	"fmt"
	"image/png"
	"time"
)

// BoundingBox returns the minimum rectangle containing all non-white pixels in the source image.
func BoundingBox(src image.Image, border int) image.Rectangle {
	min := src.Bounds().Min
	max := src.Bounds().Max

	leftX := func() int {
		for x := min.X; x < max.X; x++ {
			for y := min.Y; y < max.Y; y++ {
				c := src.At(x, y)
				if IsBlack(c) {
					return x - border
				}
			}
		}

		// no non-white pixels found
		return min.X
	}

	rightX := func() int {
		for x := max.X - 1; x >= min.X; x-- {
			for y := min.Y; y < max.Y; y++ {
				c := src.At(x, y)
				if IsBlack(c) {
					return x + border
				}
			}
		}

		// no non-white pixels found
		return max.X
	}

	topY := func() int {
		for y := min.Y; y < max.Y; y++ {
			for x := min.X; x < max.X; x++ {
				c := src.At(x, y)
				if IsBlack(c) {
					return y - border
				}
			}
		}

		// no non-white pixels found
		return max.Y
	}

	bottomY := func() int {
		for y := max.Y - 1; y >= min.Y; y-- {
			for x := min.X; x < max.X; x++ {
				c := src.At(x, y)
				if IsBlack(c) {
					return y + border
				}
			}
		}

		// no non-white pixels found
		return max.Y
	}

	// TODO: decide if +1 is correct or not
	return image.Rect(leftX(), topY(), rightX()+1, bottomY()+1)
}

// Scale scales the src image to the given rectangle using Nearest Neighbor
func Scale(src image.Image, r image.Rectangle) image.Image {
	dst := image.NewRGBA(r)

	sb := src.Bounds()
	db := dst.Bounds()

	for y := db.Min.Y; y < db.Max.Y; y++ {
		percentDownDest := float64(y) / float64(db.Dy())

		for x := db.Min.X; x < db.Max.X; x++ {
			percentAcrossDest := float64(x) / float64(db.Dx())

			srcX := int(math.Floor(percentAcrossDest * float64(sb.Dx())))
			srcY := int(math.Floor(percentDownDest * float64(sb.Dy())))

			pix := src.At(sb.Min.X+srcX, sb.Min.Y+srcY)
			dst.Set(x, y, pix)
		}
	}

	return dst
}

// NoiseImage randomly alters the pixels of the given image.
// Originally this used randomColor(), but that result in some black pixels, which totally defeats the
// bounding box algorithm. A better BBox algorithm would be nice...
func AddNoise(img *image.RGBA) {
	for row := img.Bounds().Min.Y; row < img.Bounds().Max.Y; row++ {
		for col := img.Bounds().Min.X; col < img.Bounds().Max.X; col++ {
			if rand.Float64() > 0.90 {
				//img.Set(col, row, randomColor())
				img.Set(col, row, color.White)
			}
		}
	}
}

// from http://blog.golang.org/go-imagedraw-package ("Converting an Image to RGBA"),
// modified slightly to be a no-op if the src image is already RGBA
//
func ConvertToRGBA(img image.Image) (result *image.RGBA) {
	result, ok := img.(*image.RGBA)
	if ok {
		return result
	}

	b := img.Bounds()
	result = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(result, result.Bounds(), img, b.Min, draw.Src)

	return
}

// randomColor returns a color with completely random values for RGBA.
func randomColor() color.Color {
	// start with non-premultiplied RGBA
	c := color.NRGBA{R: uint8(rand.Intn(256)), G: uint8(rand.Intn(256)), B: uint8(rand.Intn(256)), A: uint8(rand.Intn(256))}
	return color.RGBAModel.Convert(c)
}

// ImageToString returns a textual approximation of a black & white image for debugging purposes.
func ImageToString(img image.Image) (result string) {
	for row := img.Bounds().Min.Y; row < img.Bounds().Max.Y; row++ {
		for col := img.Bounds().Min.X; col < img.Bounds().Max.X; col++ {
			if IsBlack(img.At(col, row)) {
				result += "."
			} else {
				result += "O"
			}
		}

		result += "\n"
	}

	return
}



func IsYAxisBlank(src image.Image, x int) bool {
	max := src.Bounds().Max
	for y := 0; y < max.Y; y++ {
		c := src.At(x, y)
		if IsBlackX(c) {
			return false
		}
	}

	return true
}

func ImageSegmentX(src image.Image, x int) (int,int) {
	max := src.Bounds().Max

	s := x
	find := false
	for ; x < max.X; x++ {
		if !IsYAxisBlank(src, x) {
			if !find {
				s = x
				find = true
			}
		} else {
			if find {
				break
			}
		}
	}

	return s, x
}

// SaveToPNG create a png file with the img
func SaveToPNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	png.Encode(f, img)
}

// SaveToPNG create a png file, which name is 'time'
func SaveToTimePNG(img image.Image) {
	path := fmt.Sprintf("./res/%s.png", time.Now().Format("20170101-171513"))
	SaveToPNG(path, img)
}

// NewSubRGBA create sub image
func NewSubRGBA(rgba image.Image, b image.Rectangle) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(result, result.Bounds(), rgba, b.Min, draw.Src)
	return result
}

func ImageSplit(rgba image.Image) []*image.RGBA {
	var ret []*image.RGBA
	start := 0
	end := 0
	for {
		start, end = ImageSegmentX(rgba, start)

		fmt.Println(start, end, rgba.Bounds().Max.X)
		if end == rgba.Bounds().Max.X {
			break
		}

		newRgba := NewSubRGBA(rgba, image.Rect(start, 0, end, rgba.Bounds().Max.Y))
		start = end
		ret = append(ret, newRgba)
	}

	return ret
}
