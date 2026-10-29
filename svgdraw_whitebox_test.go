// Copyright 2018 The oksvg Authors. All rights reserved.
// created: 2018 by S.R.Wiley
package oksvg

import "testing"

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
