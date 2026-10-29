// Copyright 2017 The oksvg Authors. All rights reserved.
// created: 2/12/2017 by S.R.Wiley
//
// utils.go implements translation of an SVG2.0 path into a rasterx Path.

package oksvg

import (
	"image/color"

	"github.com/golang-gui/rasterx"
)

// PathStyle holds the state of the SVG style.
type PathStyle struct {
	color                             any
	stopColor                         any
	stopOpacity                       float64
	display, visible                  bool
	FillOpacity, LineOpacity          float64
	LineWidth, DashOffset, MiterLimit float64
	Dash                              []float64
	UseNonZeroWinding                 bool
	fillerColor, linerColor           any // either color.Color or rasterx.Gradient
	LineGap                           rasterx.GapFunc
	LeadLineCap                       rasterx.CapFunc // This is used if different than LineCap
	LineCap                           rasterx.CapFunc
	LineJoin                          rasterx.JoinMode
	mAdder                            rasterx.MatrixAdder // current transform
}

// styleAttribute describes draw options, such as {"fill":"black"; "stroke":"white"}.
type styleAttribute = map[string]string

// DefaultStyle sets the default PathStyle to fill black, winding rule,
// full opacity, no stroke, ButtCap line end and Miter line connect.
var DefaultStyle = PathStyle{
	FillOpacity: 1, LineOpacity: 1, LineWidth: 1, MiterLimit: 4, UseNonZeroWinding: true,
	fillerColor: color.NRGBA{A: 255}, color: color.NRGBA{A: 255}, stopColor: color.NRGBA{A: 255}, stopOpacity: 1,
	LineCap: rasterx.ButtCap, LineJoin: rasterx.Miter,
	mAdder: rasterx.MatrixAdder{M: rasterx.Identity}, display: true, visible: true,
}

type currentColorPaint struct{}
type paintReference struct {
	id          string
	fallback    any
	hasFallback bool
}
