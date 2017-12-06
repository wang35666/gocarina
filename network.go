// Package gocarina uses a neural network to implement a very simple form of OCR (Optical Character Recognition).
package gocarina

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"strconv"
	"time"
)

const (
	NumOutputs            = 8    // number of output bits. This constrains the range of chars that are recognizable.
	MinBoundingBoxPercent = 0.25 // threshold width for imposing a bounding box on char width/height
	TileTargetWidth       = 12   // tiles get scaled down to these dimensions
	TileTargetHeight      = 12
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// Network implements a feed-forward neural network for detecting letters in bitmap images.
type Network struct {
	// TODO: much of the array allocations and math could be simplified by using matrices;
	// Consider using github.com/gonum/matrix/mat64

	NumInputs     int         // total of bits in the image
	NumOutputs    int         // number of bits of output; determines the range of chars we can detect
	HiddenCount   int         // number of hidden nodes
	InputValues   []uint8     // image bits
	InputWeights  [][]float64 // weights from inputs -> hidden nodes
	HiddenOutputs []float64   // after feed-forward, what the hidden nodes output
	OutputWeights [][]float64 // weights from hidden nodes -> output nodes
	OutputValues  []float64   // after feed-forward, what the output nodes output
	OutputErrors  []float64   // error from the output nodes
	HiddenErrors  []float64   // error from the hidden nodes

	tileWidth  int
	tileHeight int
}

// NewNetwork returns a new instance of a neural network, with the given number of input nodes.
func NewNetwork(w int, h int) *Network {
	numInputs := w * h
	hiddenCount := numInputs + NumOutputs // somewhat arbitrary; you should experiment with this value

	n := &Network{
		NumInputs:   numInputs,
		HiddenCount: hiddenCount,
		NumOutputs:  NumOutputs,
		tileWidth:   w,
		tileHeight:  h,
	}

	n.InputValues = make([]uint8, n.NumInputs)
	n.OutputValues = make([]float64, n.NumOutputs)
	n.OutputErrors = make([]float64, n.NumOutputs)
	n.HiddenOutputs = make([]float64, n.NumOutputs)
	n.HiddenErrors = make([]float64, n.HiddenCount)

	n.assignRandomWeights()

	return n
}

func (n *Network) String() string {
	return fmt.Sprintf("NumInputs: %d, NumOutputs: %d, HiddenCount: %d", n.NumInputs, n.NumOutputs, n.HiddenCount)
}

// Train trains the network by sending the given image through the network, expecting the output to be equal to r.
func (n *Network) Train(img image.Image, r rune) {
	// feed the image data forward through the network to obtain a result
	//
	n.assignInputs(img)
	n.calculateHiddenOutputs()
	n.calculateFinalOutputs()

	// propagate the error correction backward through the net
	//
	n.calculateOutputErrors(r)
	n.calculateHiddenErrors()
	n.adjustOutputWeights()
	n.adjustInputWeights()
}

// Attempt to recognize the character displayed on the given image.
func (n *Network) Recognize(img image.Image) rune {
	n.assignInputs(img)
	n.calculateHiddenOutputs()
	n.calculateFinalOutputs()

	// quantize output values
	bitstring := ""
	for _, v := range n.OutputValues {
		//log.Printf("v: %f", v)
		bitstring += strconv.Itoa(round(v))
	}

	asciiCode, err := strconv.ParseInt(bitstring, 2, 16)
	if err != nil {
		log.Fatalf("error in ParseInt for %s: ", err)
	}

	log.Printf("returning bitstring: %s", bitstring)
	return rune(asciiCode)
}

func (n *Network) Save(filePath string) error {
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)

	err := encoder.Encode(n)
	if err != nil {
		return fmt.Errorf("error encoding network: %s", err)
	}

	err = ioutil.WriteFile(filePath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("error writing network to file: %s", err)
	}

	return nil
}

func RestoreNetwork(filePath string) (*Network, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading network file: %s", err)
	}

	decoder := gob.NewDecoder(bytes.NewBuffer(b))

	var result Network
	err = decoder.Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("error decoding network: %s", err)
	}

	return &result, nil
}

// can't believe this isn't in the stdlib!
func round(f float64) int {
	if math.Abs(f) < 0.5 {
		return 0
	}
	return int(f + math.Copysign(0.5, f))
}

// feed the image into the network
func (n *Network) assignInputs(img image.Image) {
	if img.Bounds().Dx() > n.tileWidth || img.Bounds().Dy() > n.tileHeight {
		log.Fatalf("expected %d %d inputs, got %d %d",
			n.tileWidth,
			n.tileHeight,
			img.Bounds().Dx(),
			img.Bounds().Dy())
	}
	//log.Printf("numPixels: %d", numPixels)

	i := 0
	for row := img.Bounds().Min.Y; row < img.Bounds().Min.Y + n.tileHeight; row++ {
		for col := img.Bounds().Min.X; col < img.Bounds().Min.X + n.tileWidth; col++ {
			pixel := pixelToBit(img.At(col, row))
			n.InputValues[i] = pixel
			i++
		}
	}

	if i != n.NumInputs {
		log.Fatalf("expected i to be: %d, was: %d", n.NumInputs, i)
	}
}

func pixelToBit(c color.Color) uint8 {
	if IsBlack(c) {
		return 0
	}

	return 1
}

func (n *Network) assignRandomWeights() {
	// input -> hidden weights
	//
	for i := 0; i < n.NumInputs; i++ {
		weights := make([]float64, n.HiddenCount)

		for j := 0; j < len(weights); j++ {

			// we want the overall sum of weights to be < 1
			weights[j] = rand.Float64() / float64(n.NumInputs*n.HiddenCount)
		}

		n.InputWeights = append(n.InputWeights, weights)
	}

	// hidden -> output weights
	//
	for i := 0; i < n.HiddenCount; i++ {
		weights := make([]float64, n.NumOutputs)

		for j := 0; j < len(weights); j++ {

			// we want the overall sum of weights to be < 1
			weights[j] = rand.Float64() / float64(n.HiddenCount*n.NumOutputs)
		}

		n.OutputWeights = append(n.OutputWeights, weights)
	}
}

func (n *Network) calculateOutputErrors(r rune) {
	accumError := 0.0
	arrayOfInts := n.runeToArrayOfInts(r)

	// NB: binaryString[i] will return bytes, not a rune. range does the right thing
	for i, digit := range arrayOfInts {
		//log.Printf("digit: %d", digit)

		digitAsFloat := float64(digit)
		err := (digitAsFloat - n.OutputValues[i]) * (1.0 - n.OutputValues[i]) * n.OutputValues[i]
		n.OutputErrors[i] = err
		accumError += err * err
		//log.Printf("accumError: %.10f", accumError)
	}
}

func (n *Network) calculateHiddenErrors() {
	for i := 0; i < len(n.HiddenOutputs); i++ {
		sum := float64(0)

		for j := 0; j < len(n.OutputErrors); j++ {
			sum += n.OutputErrors[j] * n.OutputWeights[i][j]
		}

		n.HiddenErrors[i] = n.HiddenOutputs[i] * (1 - n.HiddenOutputs[i]) * sum
	}
}

func (n *Network) adjustOutputWeights() {
	for i := 0; i < len(n.HiddenOutputs); i++ {
		for j := 0; j < n.NumOutputs; j++ {
			n.OutputWeights[i][j] += n.OutputErrors[j] * n.HiddenOutputs[i]
		}
	}
}

func (n *Network) adjustInputWeights() {
	for i := 0; i < n.NumInputs; i++ {
		for j := 0; j < n.HiddenCount; j++ {
			//fmt.Printf("i: %d, j: %d, len(n.InputWeights): %d, len(n.HiddenErrors): %d, len(n.InputValues): %d\n", i, j, len(n.InputWeights), len(n.HiddenErrors), len(n.InputValues))
			n.InputWeights[i][j] += n.HiddenErrors[j] * float64(n.InputValues[i])
		}
	}
}

func (n *Network) calculateHiddenOutputs() {
	for i := 0; i < len(n.HiddenOutputs); i++ {
		sum := float64(0)

		for j := 0; j < len(n.InputValues); j++ {
			sum += float64(n.InputValues[j]) * n.InputWeights[j][i]
		}

		n.HiddenOutputs[i] = sigmoid(sum)
	}
}

func (n *Network) calculateFinalOutputs() {
	for i := 0; i < n.NumOutputs; i++ {
		sum := float64(0)

		for j := 0; j < len(n.HiddenOutputs); j++ {
			val := n.HiddenOutputs[j] * n.OutputWeights[j][i]
			sum += val
			//log.Printf("val: %f", val)
		}

		//log.Printf("sum: %f", sum)
		n.OutputValues[i] = sigmoid(sum)
	}
}

// function that maps its input to a range between 0..1
// mathematically it's supposed to be asymptotic, but large values of x may round up to 1
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// map a rune char to an array of int, representing its unicode codepoint in binary
// 'A' => 65 => []int {0, 1, 0, 0, 0, 0, 0, 1}
// result is zero-padded to n.NumOutputs
//
func (n *Network) runeToArrayOfInts(r rune) []int {
	var result []int = make([]int, n.NumOutputs)

	codePoint := int64(r) // e.g. 65

	// we want to pad with n.NumOutputs number of zeroes, so create a dynamic format for Sprintf
	format := fmt.Sprintf("%%0%db", n.NumOutputs)
	binaryString := fmt.Sprintf(format, codePoint) // e.g. "01000001"

	// must use range: array indexing of strings returns bytes
	for i, v := range binaryString {
		if v == '0' {
			result[i] = 0
		} else {
			result[i] = 1
		}
	}
	return result
}
