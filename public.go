// Copyright 2017 The oksvg Authors. All rights reserved.
// created: 2/12/2017 by S.R.Wiley
//
// utils.go implements translation of an SVG2.0 path into a rasterx Path.

package oksvg

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"golang.org/x/image/colornames"
)

// ReadIconStream reads the Icon from the given io.Reader.
// This only supports a sub-set of SVG, but
// is enough to draw many icons. If errMode is provided,
// the first value determines if the icon ignores, errors out, or logs a warning
// if it does not handle an element found in the icon file. Ignore warnings is
// the default if no ErrorMode value is provided.
func ReadIconStream(stream io.Reader, errMode ...ErrorMode) (*SvgIcon, error) {
	mode := IgnoreErrorMode
	if len(errMode) > 0 {
		mode = errMode[0]
	}
	return parseDocument(stream, mode)
}

// ReadReplacingCurrentColor replaces currentColor value with specified value and loads SvgIcon as ReadIconStream do.
// currentColor value should be valid hex, rgb or named color value.
func ReadReplacingCurrentColor(stream io.Reader, currentColor string, errMode ...ErrorMode) (icon *SvgIcon, err error) {
	var data []byte

	if data, err = io.ReadAll(stream); err != nil {
		return nil, fmt.Errorf("%w: read data: %v", errParamMismatch, err)
	}

	if currentColor != "" && strings.Contains(string(data), "currentColor") {
		data = []byte(strings.ReplaceAll(string(data), "currentColor", currentColor))
	}

	if icon, err = ReadIconStream(bytes.NewBuffer(data), errMode...); err != nil {
		return nil, fmt.Errorf("%w: load: %v", errParamMismatch, err)
	}

	return icon, nil
}

// ReadIcon reads the Icon from the named file.
// This only supports a sub-set of SVG, but is enough to draw many icons.
// If errMode is provided, the first value determines if the icon ignores, errors out, or logs a warning
// if it does not handle an element found in the icon file.
// Ignore warnings is the default if no ErrorMode value is provided.
func ReadIcon(iconFile string, errMode ...ErrorMode) (*SvgIcon, error) {
	fin, errf := os.Open(iconFile)
	if errf != nil {
		return nil, errf
	}
	defer fin.Close()
	return ReadIconStream(fin, errMode...)
}

// ParseSVGColorNum reads the SFG color string e.g. #FBD9BD
func ParseSVGColorNum(colorStr string) (r, g, b uint8, err error) {
	colorStr = strings.TrimPrefix(colorStr, "#")
	if len(colorStr) != 6 && len(colorStr) != 3 {
		return 0, 0, 0, fmt.Errorf("color string %s is not length 3 or 6 as required by SVG specification",
			colorStr)
	} else if len(colorStr) == 3 {
		// SVG specs say duplicate characters in case of 3 digit hex number
		colorStr = string([]byte{
			colorStr[0], colorStr[0],
			colorStr[1], colorStr[1], colorStr[2], colorStr[2],
		})
	}

	rc, err := strconv.ParseUint(colorStr[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, err
	}
	gc, err := strconv.ParseUint(colorStr[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, err
	}
	bc, err := strconv.ParseUint(colorStr[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, err
	}
	return uint8(rc), uint8(gc), uint8(bc), nil
}

// ParseSVGColor parses an SVG color string in all forms
// including all SVG1.1 names, obtained from the image.colornames package
func ParseSVGColor(colorStr string) (color.Color, error) {
	// _, _, _, a := curColor.RGBA()
	colorStr = strings.TrimSpace(colorStr)
	v := strings.ToLower(colorStr)
	if strings.HasPrefix(v, "url(") {
		return nil, fmt.Errorf("paint references require a document")
	}
	if v == "transparent" {
		return color.NRGBA{}, nil
	}
	if strings.HasPrefix(v, "#") && (len(v) == 5 || len(v) == 9) {
		hex := v[1:]
		if len(hex) == 4 {
			var expanded strings.Builder
			for _, r := range hex {
				expanded.WriteRune(r)
				expanded.WriteRune(r)
			}
			hex = expanded.String()
		}
		rgba, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return nil, err
		}
		return color.NRGBA{R: uint8(rgba >> 24), G: uint8(rgba >> 16), B: uint8(rgba >> 8), A: uint8(rgba)}, nil
	}
	if strings.HasPrefix(v, "rgba(") && strings.HasSuffix(v, ")") {
		args := strings.Split(v[5:len(v)-1], ",")
		if len(args) != 4 {
			return nil, errParamMismatch
		}
		r, err := parseColorValue(strings.TrimSpace(args[0]))
		if err != nil {
			return nil, err
		}
		g, err := parseColorValue(strings.TrimSpace(args[1]))
		if err != nil {
			return nil, err
		}
		b, err := parseColorValue(strings.TrimSpace(args[2]))
		if err != nil {
			return nil, err
		}
		a, err := readFraction(args[3])
		if err != nil {
			return nil, err
		}
		return color.NRGBA{R: r, G: g, B: b, A: uint8(math.Round(clampUnit(a) * 255))}, nil
	}
	switch v {
	case "none", "":
		// nil signals that the function (fill or stroke) is off;
		// not the same as black
		return nil, nil
	default:
		cn, ok := colornames.Map[v]
		if ok {
			r, g, b, a := cn.RGBA()
			return color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(a)}, nil
		}
	}
	cStr := strings.TrimPrefix(v, "rgb(")
	if cStr != v {
		cStr := strings.TrimSuffix(cStr, ")")
		vals := strings.Split(cStr, ",")
		if len(vals) != 3 {
			return color.NRGBA{}, errParamMismatch
		}
		var cvals [3]uint8
		var err error
		for i := range cvals {
			cvals[i], err = parseColorValue(vals[i])
			if err != nil {
				return nil, err
			}
		}
		return color.NRGBA{cvals[0], cvals[1], cvals[2], 0xFF}, nil
	}

	cStr = strings.TrimPrefix(v, "hsl(")
	if cStr != v {
		cStr := strings.TrimSuffix(cStr, ")")
		vals := strings.Split(cStr, ",")
		if len(vals) != 3 {
			return color.NRGBA{}, errParamMismatch
		}

		H, err := strconv.ParseInt(strings.TrimSpace(vals[0]), 10, 64)
		if err != nil {
			return color.NRGBA{}, fmt.Errorf("invalid hue in hsl: '%s' (%s)", vals[0], err)
		}

		if !strings.HasSuffix(strings.TrimSpace(vals[1]), "%") || !strings.HasSuffix(strings.TrimSpace(vals[2]), "%") {
			return nil, errParamMismatch
		}
		S, err := parseFloat(strings.TrimSuffix(strings.TrimSpace(vals[1]), "%"), 64)
		if err != nil {
			return color.NRGBA{}, fmt.Errorf("invalid saturation in hsl: '%s' (%s)", vals[1], err)
		}
		S = S / 100

		L, err := parseFloat(strings.TrimSuffix(strings.TrimSpace(vals[2]), "%"), 64)
		if err != nil {
			return color.NRGBA{}, fmt.Errorf("invalid lightness in hsl: '%s' (%s)", vals[2], err)
		}
		L = L / 100

		C := (1 - math.Abs((2*L)-1)) * S
		X := C * (1 - math.Abs(math.Mod((float64(H)/60), 2)-1))
		m := L - C/2

		var rp, gp, bp float64
		if H < 60 {
			rp, gp, bp = float64(C), float64(X), float64(0)
		} else if H < 120 {
			rp, gp, bp = float64(X), float64(C), float64(0)
		} else if H < 180 {
			rp, gp, bp = float64(0), float64(C), float64(X)
		} else if H < 240 {
			rp, gp, bp = float64(0), float64(X), float64(C)
		} else if H < 300 {
			rp, gp, bp = float64(X), float64(0), float64(C)
		} else {
			rp, gp, bp = float64(C), float64(0), float64(X)
		}

		r, g, b := math.Round((rp+m)*255), math.Round((gp+m)*255), math.Round((bp+m)*255)
		if r > 255 {
			r = 255
		}
		if g > 255 {
			g = 255
		}
		if b > 255 {
			b = 255
		}

		return color.NRGBA{
			uint8(r),
			uint8(g),
			uint8(b),
			0xFF,
		}, nil
	}

	if colorStr[0] == '#' {
		r, g, b, err := ParseSVGColorNum(colorStr)
		if err != nil {
			return nil, err
		}
		return color.NRGBA{r, g, b, 0xFF}, nil
	}
	return nil, errParamMismatch
}
