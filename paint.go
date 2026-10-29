package oksvg

import (
	"image/color"
	"math"
	"sort"

	"github.com/golang-gui/rasterx"
)

// Paint sampling is in gradient space, independently of the device-space
// axis-aligned extent. Parsed gradients and their stop slices remain read-only.
func pathPaint(paint any, opacity float64, m rasterx.Matrix2D, b pathBounds) any {
	switch v := paint.(type) {
	case color.Color:
		return multiplyOpacity(v, opacity)
	case rasterx.Gradient:
		if len(v.Stops) == 0 {
			return color.RGBA{}
		}
		if len(v.Stops) == 1 {
			return multiplyOpacity(v.Stops[0].StopColor, v.Stops[0].Opacity*opacity)
		}
		mapping := m
		if v.Units == rasterx.ObjectBoundingBox {
			if !b.valid || b.minX == b.maxX || b.minY == b.maxY {
				return color.RGBA{}
			}
			mapping = mapping.Translate(b.minX, b.minY).Scale(b.maxX-b.minX, b.maxY-b.minY)
		}
		mapping = mapping.Mult(v.Matrix)
		if mapping.A*mapping.D-mapping.B*mapping.C == 0 {
			return color.RGBA{}
		}
		inverse := mapping.Invert()
		stops := v.Stops
		cx, cy, fx, fy, radius := v.Points[0], v.Points[1], v.Points[2], v.Points[3], v.Points[4]
		dx, dy := cx-fx, cy-fy
		radialA := dx*dx + dy*dy - radius*radius
		return rasterx.ColorFunc(func(x, y int) color.Color {
			px, py := inverse.Transform(float64(x)+.5, float64(y)+.5)
			t := 0.0
			if v.IsRadial {
				if radius <= 0 {
					return multiplyOpacity(stops[len(stops)-1].StopColor, stops[len(stops)-1].Opacity*opacity)
				}
				var covered bool
				t, covered = radialOffset(px-fx, py-fy, dx, dy, radialA)
				if !covered {
					return color.RGBA{}
				}
			} else {
				dx, dy := v.Points[2]-v.Points[0], v.Points[3]-v.Points[1]
				length := dx*dx + dy*dy
				if length == 0 {
					return multiplyOpacity(stops[len(stops)-1].StopColor, stops[len(stops)-1].Opacity*opacity)
				}
				t = ((px-v.Points[0])*dx + (py-v.Points[1])*dy) / length
			}
			switch v.Spread {
			case rasterx.RepeatSpread:
				t -= math.Floor(t)
			case rasterx.ReflectSpread:
				t = math.Mod(t, 2)
				if t < 0 {
					t += 2
				}
				if t > 1 {
					t = 2 - t
				}
			default:
				// Pad uses the first/last stop outside the gradient vector. Do not
				// collapse t<0 to t=0: coincident initial stops may have different
				// colors before and at the start of the vector.
				if t < 0 {
					return multiplyOpacity(stops[0].StopColor, stops[0].Opacity*opacity)
				}
				if t > 1 {
					last := stops[len(stops)-1]
					return multiplyOpacity(last.StopColor, last.Opacity*opacity)
				}
			}
			i := sort.Search(len(stops), func(i int) bool { return stops[i].Offset > t })
			if i == 0 {
				return multiplyOpacity(stops[0].StopColor, stops[0].Opacity*opacity)
			}
			if i == len(stops) {
				return multiplyOpacity(stops[i-1].StopColor, stops[i-1].Opacity*opacity)
			}
			left, right := stops[i-1], stops[i]
			fraction := (t - left.Offset) / (right.Offset - left.Offset)
			return interpolateStops(left, right, fraction, opacity)
		})
	}
	return color.RGBA{}
}

// radialOffset solves |Q-tD|² = (tr)² for the largest nonnegative t. SVG 2
// permits an external focus: points outside its cone have no valid circle and
// remain transparent. Interior roots use a stable form to avoid cancellation.
func radialOffset(qx, qy, dx, dy, a float64) (float64, bool) {
	b := qx*dx + qy*dy
	c := qx*qx + qy*qy
	if a == 0 {
		if b <= 0 {
			return 0, false
		}
		return c / (2 * b), true
	}
	if c == 0 {
		return 0, true
	}
	discriminant := math.FMA(-a, c, b*b)
	if discriminant < 0 || a > 0 && b <= 0 {
		return 0, false
	}
	root := math.Sqrt(discriminant)
	if a > 0 {
		return (b + root) / a, true
	}
	if b >= 0 {
		return c / (b + root), true
	}
	return (b - root) / a, true
}

func multiplyOpacity(c color.Color, opacity float64) color.RGBA {
	if c == nil {
		return color.RGBA{}
	}
	r, g, b, a := c.RGBA()
	factor := clampUnit(opacity) / 257
	return color.RGBA{uint8(math.Round(float64(r) * factor)), uint8(math.Round(float64(g) * factor)), uint8(math.Round(float64(b) * factor)), uint8(math.Round(float64(a) * factor))}
}

func interpolateStops(a, b rasterx.GradStop, t, opacity float64) color.Color {
	ar, ag, ab, aa := straightRGBA(a.StopColor)
	br, bg, bb, ba := straightRGBA(b.StopColor)
	u := 1 - t
	// SVG interpolates unassociated sRGB and alpha separately. Premultiplying
	// stops first discards the hue of transparent stops and changes the ramp.
	alpha := (aa*clampUnit(a.Opacity)*u + ba*clampUnit(b.Opacity)*t) * clampUnit(opacity)
	return color.RGBA{
		uint8(math.Round((ar*u + br*t) * alpha)),
		uint8(math.Round((ag*u + bg*t) * alpha)),
		uint8(math.Round((ab*u + bb*t) * alpha)),
		uint8(math.Round(255 * alpha)),
	}
}

// Keep the RGB channels of transparent NRGBA stops; Color.RGBA cannot do so.
func straightRGBA(c color.Color) (r, g, b, a float64) {
	switch v := c.(type) {
	case color.NRGBA:
		return float64(v.R), float64(v.G), float64(v.B), float64(v.A) / 255
	case color.NRGBA64:
		return float64(v.R) / 257, float64(v.G) / 257, float64(v.B) / 257, float64(v.A) / 65535
	}
	if c == nil {
		return
	}
	rr, gg, bb, aa := c.RGBA()
	if aa == 0 {
		return
	}
	return float64(rr) * 255 / float64(aa), float64(gg) * 255 / float64(aa), float64(bb) * 255 / float64(aa), float64(aa) / 65535
}
