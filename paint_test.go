package oksvg

import (
	"image/color"
	"testing"

	"github.com/golang-gui/rasterx"
)

// 在已知 t 的等值圆上采样：圆心 F+t(C-F)，半径 tr。
// 颜色为预乘 sRGB，像素中心恰好位于指定坐标；不依赖其他渲染器。
func TestRadialGradientCircles(t *testing.T) {
	for _, test := range []struct {
		name   string
		points [5]float64
		x, y   int
		want   color.RGBA
	}{
		{"同心", [5]float64{20.5, 40.5, 20.5, 40.5, 10}, 25, 40, color.RGBA{128, 128, 128, 255}},
		{"内部焦点", [5]float64{20.5, 40.5, 25.5, 40.5, 10}, 20, 40, color.RGBA{170, 170, 170, 255}},
		{"焦点自身", [5]float64{20.5, 40.5, 25.5, 40.5, 10}, 25, 40, color.RGBA{255, 255, 255, 255}},
		{"外部焦点取较大根", [5]float64{20.5, 40.5, 40.5, 40.5, 10}, 35, 40, color.RGBA{128, 128, 128, 255}},
		{"外部焦点第二等值圆", [5]float64{20.5, 40.5, 40.5, 40.5, 10}, 38, 40, color.RGBA{204, 204, 204, 255}},
		{"圆锥外无实根", [5]float64{20.5, 40.5, 40.5, 40.5, 10}, 40, 45, color.RGBA{}},
		{"所有根在反方向", [5]float64{20.5, 40.5, 40.5, 40.5, 10}, 45, 40, color.RGBA{}},
		{"外部焦点末端延伸", [5]float64{20.5, 40.5, 40.5, 40.5, 10}, 25, 40, color.RGBA{0, 0, 0, 255}},
		{"边界焦点一次方程", [5]float64{20.5, 40.5, 30.5, 40.5, 10}, 25, 40, color.RGBA{191, 191, 191, 255}},
		{"边界焦点中间值", [5]float64{20.5, 40.5, 30.5, 40.5, 10}, 20, 40, color.RGBA{128, 128, 128, 255}},
		{"边界焦点反方向", [5]float64{20.5, 40.5, 30.5, 40.5, 10}, 35, 40, color.RGBA{}},
		{"边界焦点侧向", [5]float64{20.5, 40.5, 30.5, 40.5, 10}, 30, 41, color.RGBA{}},
		{"零半径", [5]float64{20.5, 40.5, 40.5, 40.5, 0}, 35, 40, color.RGBA{0, 0, 0, 255}},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := rasterx.Gradient{Points: test.points, IsRadial: true, Units: rasterx.UserSpaceOnUse, Matrix: rasterx.Identity,
				Stops: []rasterx.GradStop{{StopColor: color.White, Opacity: 1}, {StopColor: color.Black, Offset: 1, Opacity: 1}}}
			paint := pathPaint(g, 1, rasterx.Identity, pathBounds{})
			got := paint.(rasterx.ColorFunc)(test.x, test.y)
			if got != test.want {
				t.Fatalf("像素 (%d,%d)=%v，期望 %v", test.x, test.y, got, test.want)
			}
		})
	}
}

func TestRadialGradientSpreadAndMapping(t *testing.T) {
	for _, test := range []struct {
		name   string
		spread rasterx.SpreadMethod
		matrix rasterx.Matrix2D
		x, y   int
		want   color.RGBA
	}{
		// x=28.5 位于 t=1.2 的等值圆上；repeat 为 .2，reflect 为 .8。
		{"repeat", rasterx.RepeatSpread, rasterx.Identity, 28, 40, color.RGBA{204, 204, 204, 255}},
		{"reflect", rasterx.ReflectSpread, rasterx.Identity, 28, 40, color.RGBA{51, 51, 51, 255}},
		{"平移", rasterx.PadSpread, rasterx.Identity.Translate(7, 9), 42, 49, color.RGBA{128, 128, 128, 255}},
		{"旋转", rasterx.PadSpread, rasterx.Matrix2D{B: 1, C: -1, E: 100}, 59, 35, color.RGBA{128, 128, 128, 255}},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := rasterx.Gradient{Points: [5]float64{20.5, 40.5, 40.5, 40.5, 10}, IsRadial: true,
				Units: rasterx.UserSpaceOnUse, Spread: test.spread, Matrix: rasterx.Identity,
				Stops: []rasterx.GradStop{{StopColor: color.White, Opacity: 1}, {StopColor: color.Black, Offset: 1, Opacity: 1}}}
			got := pathPaint(g, 1, test.matrix, pathBounds{}).(rasterx.ColorFunc)(test.x, test.y)
			if got != test.want {
				t.Fatalf("变换/扩展采样=%v，期望 %v", got, test.want)
			}
		})
	}
	// objectBoundingBox 映射后与上面的用户空间等值圆一致。
	g := rasterx.Gradient{Points: [5]float64{20.5 / 128, 40.5 / 128, 40.5 / 128, 40.5 / 128, 10.0 / 128}, IsRadial: true, Matrix: rasterx.Identity,
		Stops: []rasterx.GradStop{{StopColor: color.White, Opacity: 1}, {StopColor: color.Black, Offset: 1, Opacity: 1}}}
	got := pathPaint(g, 1, rasterx.Identity, pathBounds{valid: true, maxX: 128, maxY: 128}).(rasterx.ColorFunc)(35, 40)
	if got != (color.RGBA{128, 128, 128, 255}) {
		t.Fatalf("包围盒映射采样=%v", got)
	}
	// 不透明度与颜色分别插值，最后预乘；不能把透明区填成末端颜色。
	g.Units, g.Points = rasterx.UserSpaceOnUse, [5]float64{20.5, 40.5, 40.5, 40.5, 10}
	g.Stops[0].StopColor = color.NRGBA{R: 255, G: 255, B: 255, A: 128}
	g.Stops[1].StopColor = color.NRGBA{}
	paint := pathPaint(g, 1, rasterx.Identity, pathBounds{}).(rasterx.ColorFunc)
	if got := paint(35, 40); got != (color.RGBA{32, 32, 32, 64}) {
		t.Fatalf("半透明渐变=%v，期望预乘 RGBA(32,32,32,64)", got)
	}
	if got := paint(40, 45); got != (color.RGBA{}) {
		t.Fatalf("圆锥外像素=%v，期望透明", got)
	}
}
