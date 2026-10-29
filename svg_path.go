// Copyright 2017 The oksvg Authors. All rights reserved.
// created: 2/12/2017 by S.R.Wiley
//
// utils.go implements translation of an SVG2.0 path into a rasterx Path.

package oksvg

import (
	"image/color"
	"math"

	"github.com/srwiley/rasterx"
	"golang.org/x/image/math/fixed"
)

// SvgPath binds a style to a path.
type SvgPath struct {
	PathStyle
	Path rasterx.Path
}

// Draw the compiled SvgPath into the Dasher.
func (svgp *SvgPath) Draw(r *rasterx.Dasher, opacity float64) {
	svgp.DrawTransformed(r, opacity, rasterx.Identity)
}

// DrawTransformed draws the compiled SvgPath into the Dasher while applying
// transform t. The path/style remain unchanged; the Dasher is call-local state.
func (svgp *SvgPath) DrawTransformed(r *rasterx.Dasher, opacity float64, t rasterx.Matrix2D) {
	m := t.Mult(svgp.mAdder.M)
	if m.A*m.D-m.B*m.C == 0 {
		return
	}
	bounds := boundsOf(svgp.Path)
	if svgp.fillerColor != nil {
		r.Clear()
		r.SetWinding(svgp.UseNonZeroWinding)
		svgp.Path.AddTo(&rasterx.MatrixAdder{Adder: &r.Filler, M: m})
		r.SetColor(pathPaint(svgp.fillerColor, svgp.FillOpacity*opacity, m, bounds))
		r.Draw()
		r.SetWinding(true)
	}
	if svgp.linerColor == nil || svgp.LineWidth <= 0 {
		return
	}
	// Stroke in local coordinates, then transform the emitted outline. This
	// preserves widths, joins, caps and dash metrics under shear/nonuniform scale.
	// Upscale local work where necessary to retain fixed-point device precision.
	scale := math.Max(1, math.Hypot(math.Hypot(m.A, m.B), math.Hypot(m.C, m.D)))
	outline := &outlineScanner{Scanner: r.Scanner, matrix: m.Scale(1/scale, 1/scale)}
	local := rasterx.NewDasher(0, 0, outline)
	cap := svgp.LineCap
	if cap == nil {
		cap = rasterx.ButtCap
	}
	lead := svgp.LeadLineCap
	if lead == nil {
		lead = cap
	}
	dash := make([]float64, len(svgp.Dash))
	for i, d := range svgp.Dash {
		dash[i] = d * scale
	}
	if len(dash)%2 != 0 {
		dash = append(dash, dash...)
	}
	offset := svgp.DashOffset * scale
	total := 0.0
	for _, d := range dash {
		total += d
	}
	if total > 0 {
		offset = math.Mod(offset, total)
		if offset < 0 {
			offset += total
		}
	}
	local.SetStroke(fixed.Int26_6(svgp.LineWidth*scale*64), fixed.Int26_6(svgp.MiterLimit*64), lead, cap, svgp.LineGap, svgp.LineJoin, dash, offset)
	local.SetColor(pathPaint(svgp.linerColor, svgp.LineOpacity*opacity, m, bounds))
	svgp.Path.AddTo(&rasterx.MatrixAdder{Adder: local, M: rasterx.Identity.Scale(scale, scale)})
	local.Draw()
}

// GetFillColor returns the fill color of the SvgPath if one is defined and otherwise returns colornames.Black
func (svgp *SvgPath) GetFillColor() color.Color {
	return getColor(svgp.fillerColor)
}

// GetLineColor returns the stroke color of the SvgPath if one is defined and otherwise returns colornames.Black
func (svgp *SvgPath) GetLineColor() color.Color {
	return getColor(svgp.linerColor)
}

// SetFillColor sets the fill color of the SvgPath
func (svgp *SvgPath) SetFillColor(clr color.Color) {
	svgp.fillerColor = clr
}

// SetLineColor sets the line color of the SvgPath
func (svgp *SvgPath) SetLineColor(clr color.Color) {
	svgp.linerColor = clr
}

// outlineScanner keeps the caller's raster bounds and paint coordinates while
// receiving local-space stroke outline segments from rasterx.
type outlineScanner struct {
	rasterx.Scanner
	matrix rasterx.Matrix2D
}

func (s *outlineScanner) Start(p fixed.Point26_6) { s.Scanner.Start(s.matrix.TFixed(p)) }
func (s *outlineScanner) Line(p fixed.Point26_6)  { s.Scanner.Line(s.matrix.TFixed(p)) }
func (s *outlineScanner) SetBounds(int, int)      {}
