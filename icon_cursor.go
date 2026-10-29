// Copyright 2017 The oksvg Authors. All rights reserved.
// created: 2/12/2017 by S.R.Wiley
//
// utils.go implements translation of an SVG2.0 path into a rasterx Path.

package oksvg

import (
	"encoding/xml"
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/srwiley/rasterx"
)

// IconCursor is used while parsing SVG files.
type IconCursor struct {
	PathCursor
	icon       *SvgIcon
	StyleStack []PathStyle
	grad       *rasterx.Gradient
}

// ReadGradURL reads an SVG format gradient url
// Since the context of the gradient can affect the colors
// the current fill or line color is passed in and used in
// the case of a nil stopClor value
func (c *IconCursor) ReadGradURL(v string, defaultColor any) (grad rasterx.Gradient, ok bool) {
	if strings.HasPrefix(v, "url(") && strings.HasSuffix(v, ")") {
		urlStr := strings.TrimSpace(v[4 : len(v)-1])
		if strings.HasPrefix(urlStr, "#") {
			var g *rasterx.Gradient
			g, ok = c.icon.Grads[urlStr[1:]]
			if ok {
				grad = localizeGradIfStopClrNil(g, defaultColor)
			}
		}
	}
	return
}

// ReadGradAttr reads an SVG gradient attribute
func (c *IconCursor) ReadGradAttr(attr xml.Attr) (err error) {
	switch attr.Name.Local {
	case "gradientTransform":
		c.grad.Matrix, err = c.parseTransform(attr.Value, rasterx.Identity)
	case "gradientUnits":
		switch strings.TrimSpace(attr.Value) {
		case "userSpaceOnUse":
			c.grad.Units = rasterx.UserSpaceOnUse
		case "objectBoundingBox":
			c.grad.Units = rasterx.ObjectBoundingBox
		default:
			return fmt.Errorf("invalid gradientUnits %q", attr.Value)
		}
	case "spreadMethod":
		switch strings.TrimSpace(attr.Value) {
		case "pad":
			c.grad.Spread = rasterx.PadSpread
		case "reflect":
			c.grad.Spread = rasterx.ReflectSpread
		case "repeat":
			c.grad.Spread = rasterx.RepeatSpread
		default:
			return fmt.Errorf("invalid spreadMethod %q", attr.Value)
		}
	}
	return
}

// PushStyle computes presentation attributes, class rules and inline style
// against the current parent, then pushes the element style.
func (c *IconCursor) PushStyle(attrs []xml.Attr) error {
	parent := c.StyleStack[len(c.StyleStack)-1]
	curStyle := parent
	// display and stop properties do not inherit. color and visibility do.
	curStyle.display = true
	curStyle.stopColor = color.NRGBA{A: 255}
	curStyle.stopOpacity = 1
	presentation := make(styleAttribute)
	var inline, className string
	for _, attr := range attrs {
		if attr.Name.Space != "" {
			continue
		}
		switch attr.Name.Local {
		case "style":
			inline = attr.Value
		case "class":
			className = attr.Value
		default:
			presentation[attr.Name.Local] = attr.Value
		}
	}
	classSet := make(map[string]bool)
	for _, name := range strings.Fields(className) {
		classSet[name] = true
	}
	for _, rule := range c.icon.classRules {
		if classSet[rule.name] {
			for k, v := range rule.attributes {
				presentation[k] = v
			}
		}
	}
	if inline != "" {
		attrs, err := parseAttrs(inline)
		if err != nil {
			return err
		}
		for k, v := range attrs {
			presentation[k] = v
		}
	}
	keys := make([]string, 0, len(presentation))
	for k := range presentation {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// currentColor is resolved on leaves; color itself must be computed first.
	if v, ok := presentation["color"]; ok {
		if err := c.readStyleAttr(&curStyle, "color", strings.TrimSpace(v)); err != nil {
			return err
		}
	}
	for _, k := range keys {
		if k == "color" {
			continue
		}
		v := strings.TrimSpace(presentation[k])
		if v == "inherit" {
			switch k {
			case "display":
				curStyle.display = parent.display
			case "stop-color":
				curStyle.stopColor = parent.stopColor
			case "stop-opacity":
				curStyle.stopOpacity = parent.stopOpacity
			}
			continue
		}
		if err := c.readStyleAttr(&curStyle, k, v); err != nil {
			return err
		}
	}
	c.StyleStack = append(c.StyleStack, curStyle)
	return nil
}

func (c *IconCursor) readTransformAttr(m1 rasterx.Matrix2D, k string) (rasterx.Matrix2D, error) {
	ln := len(c.points)
	switch k {
	case "rotate":
		if ln == 1 {
			m1 = m1.Rotate(c.points[0] * math.Pi / 180)
		} else if ln == 3 {
			m1 = m1.Translate(c.points[1], c.points[2]).
				Rotate(c.points[0]*math.Pi/180).
				Translate(-c.points[1], -c.points[2])
		} else {
			return m1, errParamMismatch
		}
	case "translate":
		if ln == 1 {
			m1 = m1.Translate(c.points[0], 0)
		} else if ln == 2 {
			m1 = m1.Translate(c.points[0], c.points[1])
		} else {
			return m1, errParamMismatch
		}
	case "skewx":
		if ln == 1 {
			m1 = m1.SkewX(c.points[0] * math.Pi / 180)
		} else {
			return m1, errParamMismatch
		}
	case "skewy":
		if ln == 1 {
			m1 = m1.SkewY(c.points[0] * math.Pi / 180)
		} else {
			return m1, errParamMismatch
		}
	case "scale":
		if ln == 1 {
			m1 = m1.Scale(c.points[0], c.points[0])
		} else if ln == 2 {
			m1 = m1.Scale(c.points[0], c.points[1])
		} else {
			return m1, errParamMismatch
		}
	case "matrix":
		if ln == 6 {
			m1 = m1.Mult(rasterx.Matrix2D{
				A: c.points[0],
				B: c.points[1],
				C: c.points[2],
				D: c.points[3],
				E: c.points[4],
				F: c.points[5],
			})
		} else {
			return m1, errParamMismatch
		}
	default:
		return m1, errParamMismatch
	}
	return m1, nil
}

func (c *IconCursor) parseTransform(v string, base rasterx.Matrix2D) (rasterx.Matrix2D, error) {
	m1 := base
	for v = strings.TrimSpace(v); v != ""; v = strings.TrimLeft(v, " \t\r\n,") {
		open, close := strings.IndexByte(v, '('), strings.IndexByte(v, ')')
		if open <= 0 || close < open {
			return m1, errParamMismatch
		}
		err := c.GetPoints(v[open+1 : close])
		if err != nil {
			return m1, err
		}
		m1, err = c.readTransformAttr(m1, strings.ToLower(strings.TrimSpace(v[:open])))
		if err != nil {
			return m1, err
		}
		for _, n := range [...]float64{m1.A, m1.B, m1.C, m1.D, m1.E, m1.F} {
			if !isFinite(n) {
				return m1, fmt.Errorf("non-finite transform")
			}
		}
		v = v[close+1:]
	}
	return m1, nil
}

func (c *IconCursor) readStyleAttr(curStyle *PathStyle, k, v string) error {
	if v == "inherit" {
		return nil
	}
	switch k {
	case "color":
		if v == "currentColor" {
			return nil
		}
		clr, err := ParseSVGColor(v)
		if err != nil {
			return err
		}
		if clr == nil {
			return fmt.Errorf("invalid color %q", v)
		}
		curStyle.color = clr
	case "fill", "stroke", "stop-color":
		var paint any
		if v == "currentColor" {
			paint = currentColorPaint{}
		} else if strings.HasPrefix(v, "url(") {
			if !strings.HasSuffix(v, ")") {
				return fmt.Errorf("invalid paint reference %q", v)
			}
			href := strings.Trim(strings.TrimSpace(v[4:len(v)-1]), "\"'")
			if !strings.HasPrefix(href, "#") || len(href) == 1 {
				return c.unsupported("external paint reference")
			}
			paint = paintReference(href[1:])
		} else {
			var err error
			paint, err = ParseSVGColor(v)
			if err != nil {
				return err
			}
		}
		switch k {
		case "fill":
			curStyle.fillerColor = paint
		case "stroke":
			curStyle.linerColor = paint
		case "stop-color":
			curStyle.stopColor = paint
		}
	case "fill-rule":
		switch v {
		case "nonzero":
			curStyle.UseNonZeroWinding = true
		case "evenodd":
			curStyle.UseNonZeroWinding = false
		default:
			return fmt.Errorf("invalid fill-rule %q", v)
		}
	case "display":
		curStyle.display = v != "none"
	case "visibility":
		curStyle.visible = v != "hidden" && v != "collapse"
	case "clip-path", "mask", "filter":
		if v != "none" {
			return c.unsupported(k)
		}
	case "vector-effect":
		if v != "none" {
			return c.unsupported(k)
		}
	case "color-interpolation":
		if v != "sRGB" && v != "auto" {
			return c.unsupported(k + " " + v)
		}
	case "stroke-linegap":
		switch v {
		case "flat":
			curStyle.LineGap = rasterx.FlatGap
		case "round":
			curStyle.LineGap = rasterx.RoundGap
		case "cubic":
			curStyle.LineGap = rasterx.CubicGap
		case "quadratic":
			curStyle.LineGap = rasterx.QuadraticGap
		}
	case "stroke-leadlinecap":
		switch v {
		case "butt":
			curStyle.LeadLineCap = rasterx.ButtCap
		case "round":
			curStyle.LeadLineCap = rasterx.RoundCap
		case "square":
			curStyle.LeadLineCap = rasterx.SquareCap
		case "cubic":
			curStyle.LeadLineCap = rasterx.CubicCap
		case "quadratic":
			curStyle.LeadLineCap = rasterx.QuadraticCap
		}
	case "stroke-linecap":
		switch v {
		case "butt":
			curStyle.LineCap = rasterx.ButtCap
		case "round":
			curStyle.LineCap = rasterx.RoundCap
		case "square":
			curStyle.LineCap = rasterx.SquareCap
		case "cubic":
			curStyle.LineCap = rasterx.CubicCap
		case "quadratic":
			curStyle.LineCap = rasterx.QuadraticCap
		}
	case "stroke-linejoin":
		switch v {
		case "miter":
			curStyle.LineJoin = rasterx.Miter
		case "miter-clip":
			curStyle.LineJoin = rasterx.MiterClip
		case "arc-clip":
			curStyle.LineJoin = rasterx.ArcClip
		case "round":
			curStyle.LineJoin = rasterx.Round
		case "arc":
			curStyle.LineJoin = rasterx.Arc
		case "bevel":
			curStyle.LineJoin = rasterx.Bevel
		}
	case "stroke-miterlimit":
		mLimit, err := parseFloat(v, 64)
		if err != nil {
			return err
		}
		if mLimit < 1 {
			return fmt.Errorf("invalid miter limit")
		}
		curStyle.MiterLimit = mLimit
	case "stroke-width":
		width, err := parseFloat(v, 64)
		if err != nil {
			return err
		}
		if width < 0 {
			return fmt.Errorf("negative stroke width")
		}
		curStyle.LineWidth = width
	case "stroke-dashoffset":
		dashOffset, err := parseFloat(v, 64)
		if err != nil {
			return err
		}
		curStyle.DashOffset = dashOffset
	case "stroke-dasharray":
		curStyle.Dash = nil
		if v != "none" {
			dashes := splitOnCommaOrSpace(v)
			dList := make([]float64, len(dashes))
			for i, dstr := range dashes {
				d, err := parseFloat(strings.TrimSpace(dstr), 64)
				if err != nil {
					return err
				}
				if d < 0 {
					return fmt.Errorf("negative dash length")
				}
				dList[i] = d
			}
			curStyle.Dash = dList
			break
		}
	case "opacity", "stroke-opacity", "fill-opacity", "stop-opacity":
		op, err := readFraction(v)
		if err != nil {
			return err
		}
		op = clampUnit(op)
		switch k {
		case "opacity":
			if op != 1 {
				return c.unsupported("object/group opacity")
			}
		case "fill-opacity":
			curStyle.FillOpacity = op
		case "stroke-opacity":
			curStyle.LineOpacity = op
		case "stop-opacity":
			curStyle.stopOpacity = op
		}
	case "transform":
		m, err := c.parseTransform(v, curStyle.mAdder.M)
		if err != nil {
			return err
		}
		curStyle.mAdder.M = m
	}
	return nil
}
