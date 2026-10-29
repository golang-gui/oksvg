// Copyright 2017 The oksvg Authors. All rights reserved.
// created: 2/12/2017 by S.R.Wiley
//
// utils.go implements translation of an SVG2.0 path into a rasterx Path.

package oksvg

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/golang-gui/rasterx"
	"golang.org/x/image/colornames"
)

func parseColorValue(v string) (uint8, error) {
	v = strings.TrimSpace(v)
	divisor := 255.0
	if strings.HasSuffix(v, "%") {
		v = strings.TrimSuffix(v, "%")
		divisor = 100
	}
	n, err := parseFloat(v, 64)
	if err != nil {
		return 0, err
	}
	return uint8(math.Round(clampUnit(n/divisor) * 255)), nil
}

// parseFloat reads a finite, unitless SVG number.
func parseFloat(s string, bitSize int) (float64, error) {
	n, err := strconv.ParseFloat(strings.TrimSpace(s), bitSize)
	if err != nil {
		return 0, err
	}
	if !isFinite(n) {
		return 0, fmt.Errorf("non-finite SVG number %q", s)
	}
	return n, nil
}

// parseLength resolves absolute lengths in CSS pixels (96 pixels per inch).
// Font-relative units need a font context and are deliberately not guessed.
func parseLength(value string) (float64, error) {
	value = strings.TrimSpace(value)
	token := svgNumber.FindString(value)
	if token == "" {
		return 0, errParamMismatch
	}
	n, err := parseFloat(token, 64)
	if err != nil {
		return 0, err
	}
	factor := 1.0
	switch strings.ToLower(value[len(token):]) {
	case "", "px":
	case "in":
		factor = 96
	case "cm":
		factor = 96 / 2.54
	case "mm":
		factor = 96 / 25.4
	case "q":
		factor = 96 / 101.6
	case "pt":
		factor = 96.0 / 72
	case "pc":
		factor = 16
	default:
		return 0, fmt.Errorf("unsupported SVG length unit %q", value[len(token):])
	}
	n *= factor
	if !isFinite(n) {
		return 0, fmt.Errorf("non-finite SVG length %q", value)
	}
	return n, nil
}

// Relative intrinsic dimensions do not need a font context when an explicit
// viewBox already supplies the user space and Rasterize supplies the viewport.
func intrinsicLength(value string, hasViewBox bool) (float64, error) {
	if hasViewBox {
		value = strings.TrimSpace(value)
		for _, unit := range []string{"rem", "em", "ex"} {
			if strings.HasSuffix(value, unit) {
				if _, err := parseFloat(strings.TrimSuffix(value, unit), 64); err != nil {
					return 0, err
				}
				return 0, nil
			}
		}
	}
	return parseLength(value)
}

// Percentages refer to the viewport's dimensions, not its viewBox origin.
func (c *IconCursor) length(value string, axis int) (float64, error) {
	return viewportLength(value, axis, c.icon.ViewBox.W, c.icon.ViewBox.H)
}

func viewportLength(value string, axis int, width, height float64) (float64, error) {
	if !strings.HasSuffix(strings.TrimSpace(value), "%") {
		return parseLength(value)
	}
	n, err := readFraction(value)
	if err != nil {
		return 0, err
	}
	base := math.Hypot(width, height) / math.Sqrt2
	if axis == 0 {
		base = width
	} else if axis == 1 {
		base = height
	}
	n *= base
	if !isFinite(n) {
		return 0, fmt.Errorf("non-finite SVG percentage length %q", value)
	}
	return n, nil
}

func lengthAxis(name string) int {
	switch name {
	case "x", "x1", "x2", "cx", "rx", "width":
		return 0
	case "y", "y1", "y2", "cy", "ry", "height":
		return 1
	default:
		return 2
	}
}

// splitOnCommaOrSpace returns a list of strings after splitting the input on comma and space delimiters
func splitOnCommaOrSpace(s string) []string {
	return strings.FieldsFunc(s,
		func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
		})
}

func parseClasses(data string) (map[string]styleAttribute, error) {
	res := map[string]styleAttribute{}
	arr := strings.Split(data, "}")
	for _, v := range arr {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		valueIndex := strings.Index(v, "{")
		if valueIndex == -1 || valueIndex == len(v)-1 {
			return res, errors.New(v + "}: invalid map format in class definitions")
		}
		classesStr := v[:valueIndex]
		attrStr := v[valueIndex+1:]
		attrMap, err := parseAttrs(attrStr)
		if err != nil {
			return res, err
		}
		classes := strings.Split(classesStr, ",")
		for _, class := range classes {
			class = strings.TrimSpace(class)
			if len(class) > 0 && class[0] == '.' {
				class = class[1:]
			}
			for attrKey, attrVal := range attrMap {
				if res[class] == nil {
					res[class] = make(styleAttribute, len(attrMap))
				}
				res[class][attrKey] = attrVal
			}
		}
	}
	return res, nil
}

func simpleClassSelector(selector string) bool {
	if len(selector) < 2 || selector[0] != '.' {
		return false
	}
	for _, c := range selector[1:] {
		if c != '-' && c != '_' && !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c < 128 {
			return false
		}
	}
	return true
}

// stripCSSComments leaves quoted strings intact, including literal /* text.
func stripCSSComments(text string) (string, error) {
	var out strings.Builder
	var quote byte
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if quote != 0 {
			out.WriteByte(ch)
			if ch == '\\' && i+1 < len(text) {
				i++
				out.WriteByte(text[i])
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			out.WriteByte(ch)
			continue
		}
		if ch == '/' && i+1 < len(text) && text[i+1] == '*' {
			end := strings.Index(text[i+2:], "*/")
			if end < 0 {
				return "", fmt.Errorf("unterminated CSS comment")
			}
			i += end + 3
			out.WriteByte(' ')
			continue
		}
		out.WriteByte(ch)
	}
	return out.String(), nil
}

func parseAttrs(attrStr string) (styleAttribute, error) {
	var err error
	attrStr, err = stripCSSComments(attrStr)
	if err != nil {
		return nil, err
	}
	arr := strings.Split(attrStr, ";")
	res := make(styleAttribute, len(arr))
	for _, kv := range arr {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		tmp := strings.SplitN(kv, ":", 2)
		if len(tmp) != 2 {
			return res, errors.New(kv + ": invalid attribute format")
		}
		k := strings.ToLower(strings.TrimSpace(tmp[0]))
		v := strings.TrimSpace(tmp[1])
		res[k] = v
	}
	return res, nil
}

func readFraction(v string) (f float64, err error) {
	v = strings.TrimSpace(v)
	d := 1.0
	if strings.HasSuffix(v, "%") {
		d = 100
		v = strings.TrimSuffix(v, "%")
	}
	f, err = parseFloat(v, 64)
	f /= d
	// Is this is an unnecessary restriction? For now fractions can be all values not just in the range [0,1]
	// if f > 1 {
	// 	f = 1
	// } else if f < 0 {
	// 	f = 0
	// }
	return
}

// getColor is a helper function to get the background color
// if ReadGradUrl needs it.
func getColor(clr any) color.Color {
	switch c := clr.(type) {
	case rasterx.Gradient: // This is a bit lazy but oh well
		for _, s := range c.Stops {
			if s.StopColor != nil {
				return s.StopColor
			}
		}
	case color.NRGBA:
		return c
	}
	return colornames.Black
}

func localizeGradIfStopClrNil(g *rasterx.Gradient, defaultColor any) (grad rasterx.Gradient) {
	grad = *g
	for _, s := range grad.Stops {
		if s.StopColor == nil { // This means we need copy the gradient's Stop slice
			// and fill in the default color

			// Copy the stops
			stops := make([]rasterx.GradStop, len(grad.Stops))
			copy(stops, grad.Stops)
			grad.Stops = stops
			// Use the background color when a stop color is nil
			clr := getColor(defaultColor)
			for i, s := range stops {
				if s.StopColor == nil {
					grad.Stops[i].StopColor = clr
				}
			}
			break // Only need to do this once
		}
	}
	return
}
