// Copyright 2018 The oksvg Authors. All rights reserved.
// created: 2018 by S.R.Wiley
package oksvg

import (
	"math"
	"testing"
)

func TestReadFloat(t *testing.T) {
	c := new(PathCursor)

	fStr := "23.4.56"
	c.ReadFloat(fStr)
	if len(c.points) != 2 || c.points[0] != 23.4 || c.points[1] != 0.56 {
		t.Error("read float failed", fStr, c.points)
	}

	c.points = nil

	fStr = "23.4"
	c.ReadFloat(fStr)
	if len(c.points) != 1 || c.points[0] != 23.4 {
		t.Error("read float failed", fStr)
	}
	c.points = nil

	fStr = "23"
	c.ReadFloat(fStr)
	if len(c.points) != 1 || c.points[0] != 23.0 {
		t.Error("read float failed", fStr, c.points)
	}

	c.points = nil
	fStr = ".4"
	c.ReadFloat(fStr)
	if len(c.points) != 1 || c.points[0] != 0.4 {
		t.Error("read float failed", fStr, c.points)
	}

	c.points = nil
	fStr = "23.4.56.67.32"
	c.ReadFloat(fStr)
	if len(c.points) != 4 || c.points[0] != 23.4 || c.points[1] != 0.56 ||
		c.points[2] != 0.67 || c.points[3] != 0.32 {
		t.Error("read float failed", fStr, c.points)
	}
}

// 使用曲线解析极值作为独立预期，控制点和孤立 moveto 不能扩大绘制包围盒。
func TestGeometricBounds(t *testing.T) {
	for _, tt := range []struct {
		path string
		maxY float64
	}{
		{"M0 0 Q50 100 100 0 M500 500", 50},
		{"M0 0 C0 100 100 100 100 0 M500 500", 75},
	} {
		var cursor PathCursor
		if err := cursor.CompilePath(tt.path); err != nil {
			t.Fatal(err)
		}
		b := boundsOf(cursor.Path)
		if !b.valid || b.minX != 0 || b.minY != 0 || b.maxX != 100 || b.maxY != tt.maxY {
			t.Fatalf("%s: bounds=(%g,%g)-(%g,%g)", tt.path, b.minX, b.minY, b.maxX, b.maxY)
		}
	}
}

func TestAbsoluteLengths(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  float64
	}{
		{"1in", 96}, {"2.54cm", 96}, {"25.4mm", 96}, {"72pt", 96}, {"6pc", 96}, {"101.6q", 96}, {"4px", 4}, {"-2cm", -96 / 1.27}, {"1e2", 100},
	} {
		got, err := parseLength(tt.input)
		if err != nil || math.Abs(got-tt.want) > 1e-9 {
			t.Fatalf("%s=%g,%v want=%g", tt.input, got, err, tt.want)
		}
	}
	for _, v := range []string{"1em", "1ex", "NaN", "1e308in", "1 px"} {
		if _, err := parseLength(v); err == nil {
			t.Fatal("invalid/unresolved length", v)
		}
	}
	if _, err := parseFloat("1px", 64); err == nil {
		t.Fatal("unit accepted as a plain number")
	}
}
