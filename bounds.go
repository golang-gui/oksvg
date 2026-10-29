package oksvg

import (
	"math"

	"github.com/golang-gui/rasterx"
	"golang.org/x/image/math/fixed"
)

type pathBounds struct {
	x, y, startX, startY, minX, minY, maxX, maxY float64
	valid                                        bool
}

func (b *pathBounds) include(x, y float64) {
	if !b.valid {
		b.minX, b.maxX, b.minY, b.maxY = x, x, y, y
		b.valid = true
		return
	}
	b.minX = math.Min(b.minX, x)
	b.maxX = math.Max(b.maxX, x)
	b.minY = math.Min(b.minY, y)
	b.maxY = math.Max(b.maxY, y)
}
func (b *pathBounds) Start(p fixed.Point26_6) {
	b.x, b.y = float64(p.X)/64, float64(p.Y)/64
	b.startX, b.startY = b.x, b.y
}
func (b *pathBounds) Line(p fixed.Point26_6) {
	b.include(b.x, b.y)
	b.x, b.y = float64(p.X)/64, float64(p.Y)/64
	b.include(b.x, b.y)
}
func (b *pathBounds) Stop(closed bool) {
	if closed {
		b.include(b.x, b.y)
		b.x, b.y = b.startX, b.startY
		b.include(b.x, b.y)
	}
}
func (b *pathBounds) QuadBezier(cp, end fixed.Point26_6) {
	x0, y0 := b.x, b.y
	x1, y1 := float64(cp.X)/64, float64(cp.Y)/64
	x2, y2 := float64(end.X)/64, float64(end.Y)/64
	b.include(x0, y0)
	b.include(x2, y2)
	for _, p := range [][3]float64{{x0, x1, x2}, {y0, y1, y2}} {
		d := p[0] - 2*p[1] + p[2]
		if d == 0 {
			continue
		}
		t := (p[0] - p[1]) / d
		if t > 0 && t < 1 {
			u := 1 - t
			b.include(u*u*x0+2*u*t*x1+t*t*x2, u*u*y0+2*u*t*y1+t*t*y2)
		}
	}
	b.x, b.y = x2, y2
}
func (b *pathBounds) CubeBezier(cp1, cp2, end fixed.Point26_6) {
	x0, y0 := b.x, b.y
	x1, y1 := float64(cp1.X)/64, float64(cp1.Y)/64
	x2, y2 := float64(cp2.X)/64, float64(cp2.Y)/64
	x3, y3 := float64(end.X)/64, float64(end.Y)/64
	b.include(x0, y0)
	b.include(x3, y3)
	for _, p := range [][4]float64{{x0, x1, x2, x3}, {y0, y1, y2, y3}} {
		a := -p[0] + 3*p[1] - 3*p[2] + p[3]
		bb := 2 * (p[0] - 2*p[1] + p[2])
		c := p[1] - p[0]
		roots := []float64{}
		if a == 0 {
			if bb != 0 {
				roots = append(roots, -c/bb)
			}
		} else {
			d := bb*bb - 4*a*c
			if d >= 0 {
				d = math.Sqrt(d)
				roots = append(roots, (-bb+d)/(2*a), (-bb-d)/(2*a))
			}
		}
		for _, t := range roots {
			if t > 0 && t < 1 {
				u := 1 - t
				b.include(u*u*u*x0+3*u*u*t*x1+3*u*t*t*x2+t*t*t*x3, u*u*u*y0+3*u*u*t*y1+3*u*t*t*y2+t*t*t*y3)
			}
		}
	}
	b.x, b.y = x3, y3
}
func boundsOf(path rasterx.Path) pathBounds { var b pathBounds; path.AddTo(&b); return b }
