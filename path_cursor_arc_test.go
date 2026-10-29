package oksvg

import (
	"image"
	"math"
	reflectpkg "reflect"
	"strings"
	"testing"

	"github.com/golang-gui/rasterx"
)

// SVG 规定 Z 后的非 M 命令以原起点建立新子路径，不能继续已闭合的轮廓。
func TestClosePathStartsNewSubpath(t *testing.T) {
	for _, next := range []string{"l-8 -6", "h-8", "v-8", "c0 -8 -8 -8 -8 0", "q-8 -8 -8 0", "t-8 0", "s-8 -8 -8 0", "a8 8 0 0 1 -8 -8"} {
		t.Run(next, func(t *testing.T) {
			var implicit, explicit PathCursor
			const closed = "M20 20H40V40H20z"
			if err := implicit.CompilePath(closed + next + "z"); err != nil {
				t.Fatal(err)
			}
			if err := explicit.CompilePath(closed + "M20 20" + next + "z"); err != nil {
				t.Fatal(err)
			}
			if !reflectpkg.DeepEqual(implicit.Path, explicit.Path) {
				t.Fatal("Z 后没有从 (20,20) 开始新的闭合子路径")
			}
			if implicit.placeX != 20 || implicit.placeY != 20 {
				t.Fatalf("再次 Z 后终点 = (%g,%g)，期望 (20,20)", implicit.placeX, implicit.placeY)
			}
		})
	}
}

// Each seven-parameter group following A/a is a separate arc. In particular,
// relative endpoints must be based on the preceding arc's endpoint.
func TestRepeatedArcParameters(t *testing.T) {
	for _, tt := range []struct {
		name, grouped, explicit string
		x, y                    float64
	}{
		{"relative circle", "M10 4a6 6 0 1 1 0 12 6 6 0 0 1 0-12", "M10 4a6 6 0 1 1 0 12a6 6 0 0 1 0-12", 10, 4},
		{"absolute circle", "M10 4A6 6 0 1 1 10 16 6 6 0 0 1 10 4", "M10 4A6 6 0 1 1 10 16A6 6 0 0 1 10 4", 10, 4},
		{"relative varied", "M20 30a12 8 30 0 1 10 5 4 9 -45 1 0 -8 12 7 3 60 1 1 9 -4l2 3", "M20 30a12 8 30 0 1 10 5a4 9 -45 1 0 -8 12a7 3 60 1 1 9 -4l2 3", 33, 46},
		{"absolute varied", "M20 30A12 8 30 0 1 30 35 4 9 -45 1 0 22 47 7 3 60 1 1 31 43l2 3", "M20 30A12 8 30 0 1 30 35A4 9 -45 1 0 22 47A7 3 60 1 1 31 43l2 3", 33, 46},
	} {
		t.Run(tt.name, func(t *testing.T) {
			grouped, explicit := new(PathCursor), new(PathCursor)
			if err := grouped.CompilePath(tt.grouped); err != nil {
				t.Fatal(err)
			}
			if err := explicit.CompilePath(tt.explicit); err != nil {
				t.Fatal(err)
			}
			if !reflectpkg.DeepEqual(grouped.Path, explicit.Path) {
				t.Error("grouped arcs differ from explicit commands")
			}
			// Independently check the SVG-specified endpoint, including the line
			// following the arcs; equality alone could hide a shared parser bug.
			if math.Abs(grouped.placeX-tt.x) > 1e-9 || math.Abs(grouped.placeY-tt.y) > 1e-9 {
				t.Errorf("endpoint = (%g, %g), want (%g, %g)", grouped.placeX, grouped.placeY, tt.x, tt.y)
			}
		})
	}
}

func TestRepeatedArcsCircleCoverage(t *testing.T) {
	for _, path := range []string{
		"M10 4a6 6 0 1 1 0 12 6 6 0 0 1 0-12z",
		"M10 4A6 6 0 1 1 10 16 6 6 0 0 1 10 4z",
	} {
		icon, err := ReadIconStream(strings.NewReader(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><path d="`+path+`"/></svg>`), StrictErrorMode)
		if err != nil {
			t.Fatal(err)
		}
		const size = 80
		img := image.NewRGBA(image.Rect(0, 0, size, size))
		icon.SetTarget(0, 0, size, size)
		icon.Draw(rasterx.NewDasher(size, size, rasterx.NewScannerGV(size, size, img, img.Bounds())), 1)
		// Two semicircles enclose a radius-24 circle centered at (40,40).
		// Only test pixels at least two physical pixels from its boundary:
		// the analytic expectation is independent of parser and antialiaser.
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				d := math.Hypot(float64(x)+0.5-40, float64(y)+0.5-40)
				if math.Abs(d-24) < 2 {
					continue
				}
				want := uint8(0)
				if d < 24 {
					want = 255
				}
				if got := img.RGBAAt(x, y).A; got != want {
					t.Fatalf("%s: alpha at (%d,%d) = %d, want %d", path, x, y, got, want)
				}
			}
		}
	}
}

// SVG 的两个 flag 是单字符，可与后续坐标连续书写。
func TestCompactArcFlags(t *testing.T) {
	for _, tt := range []struct {
		compact, spaced string
		x, y            float64
	}{
		{"M1 1a5 5 0 0110 0", "M1 1a5 5 0 0 1 10 0", 11, 1},
		{"M1 1A5 5 0 0111 1", "M1 1A5 5 0 0 1 11 1", 11, 1},
		{"M10 4a6 6 0 110 12 6 6 0 110-12", "M10 4a6 6 0 1 1 0 12 6 6 0 1 1 0-12", 10, 4},
		{"M1 1A5 5 0 00.5.6", "M1 1A5 5 0 0 0 .5 .6", .5, .6},
	} {
		var a, b PathCursor
		if err := a.CompilePath(tt.compact); err != nil {
			t.Fatal(tt.compact, err)
		}
		if err := b.CompilePath(tt.spaced); err != nil {
			t.Fatal(err)
		}
		if !reflectpkg.DeepEqual(a.Path, b.Path) {
			t.Fatal("compact flags changed geometry", tt.compact)
		}
		if math.Abs(a.placeX-tt.x) > 1e-9 || math.Abs(a.placeY-tt.y) > 1e-9 {
			t.Fatalf("endpoint=(%g,%g), want=(%g,%g)", a.placeX, a.placeY, tt.x, tt.y)
		}
	}
	for _, d := range []string{"M0 0A5 5 0 2 0 10 10", "M0 0A5 5 0 -1 0 10 10", "M0 0A5 5 0 1.0 0 10 10"} {
		var c PathCursor
		if err := c.CompilePath(d); err == nil {
			t.Fatal("invalid flag accepted", d)
		}
	}
}
