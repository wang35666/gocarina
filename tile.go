package gocarina

import (
	"fmt"
	"image"
	"log"
)

// Tile represents a lettered square from a Letterpress game board.
type Tile struct {
	Letter  rune        // the letter this tile represents, if known
	img     image.Image // the original tile image, prior to any scaling/downsampling
	Reduced image.Image // the tile in black and white, bounding-boxed, and scaled down
	Bounded image.Image // the bounded tile (used only for debugging)
}

func NewTile(letter rune, img image.Image) (result *Tile) {
	result = &Tile{Letter: letter, img: img}
	result.reduce(0)

	return
}

// Reduce the tile by converting to monochrome, applying a bounding box, and scaling to match the given size.
// The resulting image will be stored in t.Reduced.
func (t *Tile) reduce(border int) {
	targetRect := image.Rect(0, 0, TileTargetWidth, TileTargetHeight)
	if targetRect.Dx() != TileTargetWidth {
		log.Fatalf("expected targetRect.Dx() to be %d, got: %d", TileTargetWidth, targetRect.Dx())
	}

	if targetRect.Dy() != TileTargetHeight {
		log.Fatalf("expected targetRect.Dy() to be %d, got: %d", TileTargetHeight, targetRect.Dy())
	}

	src := BlackWhiteImage(t.img)

	// find the bounding box for the character
	bbox := BoundingBox(src, border)

	// Only apply the bounding box if it's above some % of the width/height of original tile.
	// This is to avoid pathological cases for skinny letters like "I", which
	// would otherwise result in completely black tiles when bounded.

	if bbox.Bounds().Dx() >= int(MinBoundingBoxPercent*float64(t.img.Bounds().Dx())) &&
		bbox.Bounds().Dy() >= int(MinBoundingBoxPercent*float64(t.img.Bounds().Dy())) {
		src = src.(interface {
			SubImage(r image.Rectangle) image.Image
		}).SubImage(bbox)
	} else {
		// enable only for debugging
		//log.Printf("rune: %c: skipping boundingbox: orig width: %d, boundbox width: %d", t.Letter, t.img.Bounds().Dx(), bbox.Dx())
	}

	t.Bounded = src
	t.Reduced = Scale(src, targetRect)

	// it's sometimes helpful to see a textual version of the reduced tile
	//log.Printf("\n%s\n", ImageToString(t.Reduced))

	if t.Reduced.Bounds().Dx() != TileTargetWidth {
		log.Fatalf("expected t.Reduced.Bounds().Dx() to be %d, got: %d", TileTargetWidth, t.Reduced.Bounds().Dx())
	}

	if t.Reduced.Bounds().Dy() != TileTargetHeight {
		log.Fatalf("expected t.Reduced.Bounds().Dy() to be %d, got: %d", TileTargetHeight, t.Reduced.Bounds().Dy())
	}

}

// Save the bounded tile. Only for debugging.
func (t *Tile) SaveBoundedAndReduced() {
	SaveToPNG(fmt.Sprintf("debug_output/bounded_%c.png", t.Letter), t.Bounded)
	SaveToPNG(fmt.Sprintf("debug_output/reduced_%c.png", t.Letter), t.Reduced)
}
