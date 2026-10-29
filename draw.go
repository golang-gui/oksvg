// Copyright 2017 The oksvg Authors. All rights reserved.
// created: 2/12/2017 by S.R.Wiley
//
// utils.go implements translation of an SVG2.0 path into a rasterx Path.

package oksvg

import (
	"encoding/xml"
	"errors"
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/golang-gui/rasterx"
	"golang.org/x/image/math/fixed"
)

func draw(name string, c *IconCursor, attrs []xml.Attr) error {
	switch name {
	case "line":
		return drawLine(c, attrs)
	case "rect":
		return drawRect(c, attrs)
	case "circle":
		return drawCircle(c, attrs)
	case "ellipse":
		return drawCircle(c, attrs) // circleF handles ellipse also
	case "polyline":
		return drawPolyline(c, attrs)
	case "polygon":
		return drawPolygon(c, attrs)
	case "path":
		return drawPath(c, attrs)
	default:
		return fmt.Errorf("cannot process svg element %s", name)
	}
}

func drawSvg(c *IconCursor, attrs []xml.Attr) error {
	c.icon.ViewBox.X = 0
	c.icon.ViewBox.Y = 0
	c.icon.ViewBox.W = 0
	c.icon.ViewBox.H = 0
	var width, height float64
	var hasViewBox bool
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "viewBox":
			hasViewBox = true
			err = c.GetPoints(attr.Value)
			if len(c.points) != 4 {
				return errParamMismatch
			}
			c.icon.ViewBox.X = c.points[0]
			c.icon.ViewBox.Y = c.points[1]
			c.icon.ViewBox.W = c.points[2]
			c.icon.ViewBox.H = c.points[3]
		case "width":
			if strings.HasSuffix(strings.TrimSpace(attr.Value), "%") {
				_, err = readFraction(attr.Value)
			} else {
				width, err = parseFloat(attr.Value, 64)
			}
		case "height":
			if strings.HasSuffix(strings.TrimSpace(attr.Value), "%") {
				_, err = readFraction(attr.Value)
			} else {
				height, err = parseFloat(attr.Value, 64)
			}
		}
		if err != nil {
			return err
		}
	}
	if !hasViewBox {
		c.icon.ViewBox.W = width
		c.icon.ViewBox.H = height
	}
	return nil
}
func drawRect(c *IconCursor, attrs []xml.Attr) error {
	var x, y, w, h, rx, ry float64
	var hasRX, hasRY bool
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "x":
			x, err = parseFloat(attr.Value, 64)
		case "y":
			y, err = parseFloat(attr.Value, 64)
		case "width":
			w, err = parseFloat(attr.Value, 64)
		case "height":
			h, err = parseFloat(attr.Value, 64)
		case "rx":
			hasRX = true
			rx, err = parseFloat(attr.Value, 64)
		case "ry":
			hasRY = true
			ry, err = parseFloat(attr.Value, 64)
		}
		if err != nil {
			return err
		}
	}
	if w < 0 || h < 0 || rx < 0 || ry < 0 {
		return fmt.Errorf("negative rectangle extent")
	}
	if w == 0 || h == 0 {
		return nil
	}
	if !hasRX {
		rx = ry
	}
	if !hasRY {
		ry = rx
	}
	rx = math.Min(rx, w/2)
	ry = math.Min(ry, h/2)
	rasterx.AddRoundRect(x, y, w+x, h+y, rx, ry, 0, rasterx.RoundGap, &c.Path)
	return nil
}

func drawCircle(c *IconCursor, attrs []xml.Attr) error {
	var cx, cy, rx, ry float64
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "cx":
			cx, err = parseFloat(attr.Value, 64)
		case "cy":
			cy, err = parseFloat(attr.Value, 64)
		case "r":
			rx, err = parseFloat(attr.Value, 64)
			ry = rx
		case "rx":
			rx, err = parseFloat(attr.Value, 64)
		case "ry":
			ry, err = parseFloat(attr.Value, 64)
		}
		if err != nil {
			return err
		}
	}
	if rx < 0 || ry < 0 {
		return fmt.Errorf("negative ellipse radius")
	}
	if rx == 0 || ry == 0 { // not drawn, but not an error
		return nil
	}
	c.EllipseAt(cx, cy, rx, ry)
	return nil
}

func drawLine(c *IconCursor, attrs []xml.Attr) error {
	var x1, x2, y1, y2 float64
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "x1":
			x1, err = parseFloat(attr.Value, 64)
		case "x2":
			x2, err = parseFloat(attr.Value, 64)
		case "y1":
			y1, err = parseFloat(attr.Value, 64)
		case "y2":
			y2, err = parseFloat(attr.Value, 64)
		}
		if err != nil {
			return err
		}
	}
	c.Path.Start(fixed.Point26_6{
		X: fixed.Int26_6((x1) * 64),
		Y: fixed.Int26_6((y1) * 64),
	})
	c.Path.Line(fixed.Point26_6{
		X: fixed.Int26_6((x2) * 64),
		Y: fixed.Int26_6((y2) * 64),
	})
	return nil
}

func drawPolyline(c *IconCursor, attrs []xml.Attr) error {
	c.points = c.points[:0]
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "points":
			err = c.GetPoints(attr.Value)
			if len(c.points)%2 != 0 {
				return errors.New("polygon has odd number of points")
			}
		}
		if err != nil {
			return err
		}
	}
	if len(c.points) >= 4 {
		c.Path.Start(fixed.Point26_6{
			X: fixed.Int26_6((c.points[0]) * 64),
			Y: fixed.Int26_6((c.points[1]) * 64),
		})
		for i := 2; i < len(c.points)-1; i += 2 {
			c.Path.Line(fixed.Point26_6{
				X: fixed.Int26_6((c.points[i]) * 64),
				Y: fixed.Int26_6((c.points[i+1]) * 64),
			})
		}
	}
	return nil
}

func drawPolygon(c *IconCursor, attrs []xml.Attr) error {
	err := drawPolyline(c, attrs)
	if len(c.points) >= 4 {
		c.Path.Stop(true)
	}
	return err
}

func drawPath(c *IconCursor, attrs []xml.Attr) error {
	var err error
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "d":
			err = c.CompilePath(attr.Value)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func drawStop(c *IconCursor, attrs []xml.Attr) error {
	if c.grad == nil {
		return fmt.Errorf("stop outside gradient")
	}
	s := c.StyleStack[len(c.StyleStack)-1]
	clr := s.stopColor
	if _, ok := clr.(currentColorPaint); ok {
		clr = s.color
	}
	col, ok := clr.(color.Color)
	if !ok {
		return fmt.Errorf("invalid stop color")
	}
	stop := rasterx.GradStop{Opacity: s.stopOpacity, StopColor: col}
	for _, a := range attrs {
		if a.Name.Local == "offset" {
			var err error
			stop.Offset, err = readFraction(a.Value)
			if err != nil {
				return err
			}
		}
	}
	c.grad.Stops = append(c.grad.Stops, stop)
	return nil
}
