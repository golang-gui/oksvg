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
	v := strings.ToLower(strings.TrimSpace(colorStr))
	switch v {
	case "none", "":
		return nil, nil
	case "transparent":
		return color.NRGBA{}, nil
	}
	if strings.HasPrefix(v, "url(") {
		return nil, fmt.Errorf("paint references require a document")
	}
	if strings.HasPrefix(v, "#") {
		if len(v) == 5 || len(v) == 9 {
			hex := v[1:]
			if len(hex) == 4 {
				var b strings.Builder
				for _, r := range hex {
					b.WriteRune(r)
					b.WriteRune(r)
				}
				hex = b.String()
			}
			n, err := strconv.ParseUint(hex, 16, 32)
			if err != nil {
				return nil, err
			}
			return color.NRGBA{R: uint8(n >> 24), G: uint8(n >> 16), B: uint8(n >> 8), A: uint8(n)}, nil
		}
		r, g, b, err := ParseSVGColorNum(v)
		if err != nil {
			return nil, err
		}
		return color.NRGBA{R: r, G: g, B: b, A: 255}, nil
	}
	if c, ok := colornames.Map[v]; ok {
		return color.NRGBA(c), nil
	}
	open := strings.IndexByte(v, '(')
	if open < 0 || !strings.HasSuffix(v, ")") {
		return nil, errParamMismatch
	}
	name := v[:open]
	if name != "rgb" && name != "rgba" && name != "hsl" && name != "hsla" {
		return nil, errParamMismatch
	}
	body := v[open+1 : len(v)-1]
	args, comma, err := colorArguments(body)
	if err != nil {
		return nil, err
	}
	alpha := 1.0
	if len(args) == 4 {
		alpha, err = readFraction(args[3])
		if err != nil {
			return nil, err
		}
	}
	result := color.NRGBA{A: uint8(math.Round(clampUnit(alpha) * 255))}
	if name == "rgb" || name == "rgba" {
		percent := strings.HasSuffix(args[0], "%")
		for _, v := range args[:3] {
			if comma && strings.HasSuffix(v, "%") != percent {
				return nil, fmt.Errorf("mixed RGB number/percentage channels")
			}
		}
		result.R, err = parseColorValue(args[0])
		if err != nil {
			return nil, err
		}
		result.G, err = parseColorValue(args[1])
		if err != nil {
			return nil, err
		}
		result.B, err = parseColorValue(args[2])
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	hue, err := parseHue(args[0])
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(args[1], "%") || !strings.HasSuffix(args[2], "%") {
		return nil, errParamMismatch
	}
	saturation, err := readFraction(args[1])
	if err != nil {
		return nil, err
	}
	lightness, err := readFraction(args[2])
	if err != nil {
		return nil, err
	}
	saturation, lightness = clampUnit(saturation), clampUnit(lightness)
	chroma := (1 - math.Abs(2*lightness-1)) * saturation
	x := chroma * (1 - math.Abs(math.Mod(hue/60, 2)-1))
	m := lightness - chroma/2
	var r, g, b float64
	switch {
	case hue < 60:
		r, g = chroma, x
	case hue < 120:
		r, g = x, chroma
	case hue < 180:
		g, b = chroma, x
	case hue < 240:
		g, b = x, chroma
	case hue < 300:
		r, b = x, chroma
	default:
		r, b = chroma, x
	}
	result.R = uint8(math.Round(clampUnit(r+m) * 255))
	result.G = uint8(math.Round(clampUnit(g+m) * 255))
	result.B = uint8(math.Round(clampUnit(b+m) * 255))
	return result, nil
}

func colorArguments(body string) ([]string, bool, error) {
	comma := strings.Contains(body, ",")
	var args []string
	if comma {
		if strings.Contains(body, "/") {
			return nil, comma, errParamMismatch
		}
		args = strings.Split(body, ",")
	} else {
		parts := strings.Split(body, "/")
		if len(parts) > 2 {
			return nil, comma, errParamMismatch
		}
		args = strings.Fields(parts[0])
		if len(args) != 3 {
			return nil, comma, errParamMismatch
		}
		if len(parts) == 2 {
			args = append(args, parts[1])
		}
	}
	if len(args) != 3 && len(args) != 4 {
		return nil, comma, errParamMismatch
	}
	for i := range args {
		args[i] = strings.TrimSpace(args[i])
		if args[i] == "" {
			return nil, comma, errParamMismatch
		}
	}
	return args, comma, nil
}

func parseHue(value string) (float64, error) {
	factor := 1.0
	for _, unit := range []struct {
		name   string
		factor float64
	}{{"grad", .9}, {"turn", 360}, {"deg", 1}, {"rad", 180 / math.Pi}} {
		if strings.HasSuffix(value, unit.name) {
			value = strings.TrimSuffix(value, unit.name)
			factor = unit.factor
			break
		}
	}
	n, err := parseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	n *= factor
	if !isFinite(n) {
		return 0, errParamMismatch
	}
	n = math.Mod(n, 360)
	if n < 0 {
		n += 360
	}
	return n, nil
}
