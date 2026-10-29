package oksvg

import (
	"fmt"
	"image"
	"math"
	"strings"

	"github.com/srwiley/rasterx"
	"github.com/srwiley/scanFT"
)

type aspectRatio struct {
	none, slice bool
	x, y        float64
	set         bool
}

func parseAspectRatio(value string) (aspectRatio, error) {
	a := aspectRatio{x: .5, y: .5, set: true}
	f := strings.Fields(value)
	if len(f) == 0 {
		return a, nil
	}
	if f[0] == "defer" {
		f = f[1:]
		if len(f) == 0 {
			return a, fmt.Errorf("invalid preserveAspectRatio")
		}
	}
	if f[0] == "none" {
		a.none = true
	} else {
		align := f[0]
		if len(align) != 8 || align[0] != 'x' || align[4] != 'Y' {
			return a, fmt.Errorf("invalid preserveAspectRatio %q", value)
		}
		values := map[string]float64{"Min": 0, "Mid": .5, "Max": 1}
		var ok bool
		a.x, ok = values[align[1:4]]
		if !ok {
			return a, fmt.Errorf("invalid preserveAspectRatio %q", value)
		}
		a.y, ok = values[align[5:]]
		if !ok {
			return a, fmt.Errorf("invalid preserveAspectRatio %q", value)
		}
	}
	if len(f) > 2 {
		return a, fmt.Errorf("invalid preserveAspectRatio %q", value)
	}
	if len(f) == 2 {
		switch f[1] {
		case "meet":
		case "slice":
			a.slice = true
		default:
			return a, fmt.Errorf("invalid preserveAspectRatio %q", value)
		}
	}
	return a, nil
}

// Rasterize renders the SVG into a new, transparent-initialized premultiplied
// RGBA image. It uses the document's viewBox and preserveAspectRatio, independently
// of the legacy SetTarget/Transform state. Invalid sizes or an invalid zero-value
// icon return nil. Images are caller-owned and are never cached by the icon.
// Concurrent calls are safe while callers do not modify the parsed icon.
func (s *SvgIcon) Rasterize(width, height int) *image.RGBA {
	if s == nil || width <= 0 || height <= 0 {
		return nil
	}
	maxInt := int(^uint(0) >> 1)
	if width > maxInt/4 || height > maxInt/(width*4) {
		return nil
	}
	v := s.ViewBox
	if !isFinite(v.X) || !isFinite(v.Y) || !isFinite(v.W) || !isFinite(v.H) || v.W <= 0 || v.H <= 0 {
		return nil
	}
	a := s.aspect
	if !a.set {
		a = aspectRatio{x: .5, y: .5}
	}
	sx, sy := float64(width)/v.W, float64(height)/v.H
	x, y := 0.0, 0.0
	if !a.none {
		scale := math.Min(sx, sy)
		if a.slice {
			scale = math.Max(sx, sy)
		}
		sx, sy = scale, scale
		x = (float64(width) - v.W*scale) * a.x
		y = (float64(height) - v.H*scale) * a.y
	}
	target := rasterx.Identity.Translate(x, y).Scale(sx, sy).Translate(-v.X, -v.Y)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	scanner := scanFT.NewScannerFT(width, height, scanFT.NewRGBAPainter(img))
	raster := rasterx.NewDasher(width, height, scanner)
	for _, path := range s.SVGPaths {
		path.DrawTransformed(raster, 1, target)
	}
	return img
}
