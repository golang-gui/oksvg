// Copyright 2018 The oksvg Authors. All rights reserved.
// created: 2018 by S.R.Wiley
package oksvg_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"sync"
	"testing"

	. "github.com/golang-gui/oksvg"
	. "github.com/golang-gui/rasterx"
	// . "github.com/golang-gui/scanFT"
)

func ReadIconSet(folder string, paths []string) (icons []*SvgIcon) {
	for _, p := range paths {
		icon, errSvg := ReadIcon(folder+p+".svg", IgnoreErrorMode)
		if errSvg == nil {
			icons = append(icons, icon)
		}
	}
	return
}

func BenchmarkLandscapeIcons(b *testing.B) {
	var (
		beachIconNames = []string{
			"beach", "cape", "iceberg", "island",
			"mountains", "sea", "trees", "village",
		}
		beachIcons = ReadIconSet("testdata/landscapeIcons/", beachIconNames)
		w, h       = int(beachIcons[0].ViewBox.W), int(beachIcons[0].ViewBox.H)
		img        = image.NewRGBA(image.Rect(0, 0, w, h))
		// source     = image.NewUniform(color.NRGBA{0, 0, 0, 255})
		scannerGV = NewScannerGV(w, h, img, img.Bounds())
		raster    = NewDasher(w, h, scannerGV)

		//	painter   = NewRGBAPainter(img)
		//	scannerFT = NewScannerFT(w, h, painter)
		//	raster    = NewDasher(w, h, scannerFT)
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ic := range beachIcons {
			ic.Draw(raster, 1.0)
		}
	}
}

func BenchmarkSportsIcons(b *testing.B) {
	var (
		sportsIconNames = []string{
			"archery", "fencing", "rugby_sevens",
			"artistic_gymnastics", "football", "sailing",
			"athletics", "golf", "shooting",
			"badminton", "handball", "swimming",
			"basketball", "hockey", "synchronised_swimming",
			"beach_volleyball", "judo", "table_tennis",
			"boxing", "marathon_swimming", "taekwondo",
			"canoe_slalom", "modern_pentathlon", "tennis",
			"canoe_sprint", "olympic_medal_bronze", "trampoline_gymnastics",
			"cycling_bmx", "olympic_medal_gold", "triathlon",
			"cycling_mountain_bike", "olympic_medal_silver", "trophy",
			"cycling_road", "olympic_torch", "volleyball",
			"cycling_track", "water_polo",
			"diving", "rhythmic_gymnastics", "weightlifting",
			"equestrian", "rowing", "wrestling",
		}
		sportsIcons = ReadIconSet("testdata/sportsIcons/", sportsIconNames)
		w2, h2      = int(sportsIcons[0].ViewBox.W), int(sportsIcons[0].ViewBox.H)
		img2        = image.NewRGBA(image.Rect(0, 0, w2, h2))
		scannerGV2  = NewScannerGV(w2, h2, img2, image.Rect(0, 0, w2, h2))
		raster2     = NewDasher(w2, h2, scannerGV2)

		//	painter2   = NewRGBAPainter(img2)
		//	scannerFT2 = NewScannerFT(w2, h2, painter2)
		//	raster2    = NewDasher(w2, h2, scannerFT2)
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ic := range sportsIcons {
			ic.Draw(raster2, 1.0)
		}
	}
}

// 使用明确的纯蓝矩形预期，检查元数据不会改变像素或泄漏其中的图形。
func TestXMLCompatibility(t *testing.T) {
	baseline := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><rect x="4" y="4" width="12" height="12" fill="blue"/></svg>`
	variants := []string{
		`<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">` + baseline,
		strings.Replace(baseline, `viewBox=`, `xml:space="preserve" viewBox=`, 1),
		strings.Replace(baseline, `fill="blue"`, `fill="blue" clip-rule="evenodd" style="clip-rule:nonzero"`, 1),
		strings.Replace(baseline, `viewBox=`, `xmlns:vendor="urn:test" vendor:transform="translate(100)" vendor:fill="red" viewBox=`, 1),
		strings.Replace(baseline, `<rect`, `<metadata><svg xmlns="urn:private"><rect width="20" height="20"/></svg></metadata><rect`, 1),
		strings.Replace(baseline, `<rect`, `<vendor:data xmlns:vendor="urn:private"><rect width="20" height="20" fill="red"/></vendor:data><rect`, 1),
	}
	icon, err := ReadIconStream(strings.NewReader(baseline), StrictErrorMode)
	if err != nil {
		t.Fatal(err)
	}
	want := icon.Rasterize(20, 20)
	if want.RGBAAt(8, 8) != (color.RGBA{B: 255, A: 255}) || want.RGBAAt(1, 1) != (color.RGBA{}) {
		t.Fatal("independent rectangle expectation failed")
	}
	for _, src := range variants {
		got, err := ReadIconStream(strings.NewReader(src), StrictErrorMode)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got.Rasterize(20, 20).Pix, want.Pix) {
			t.Fatalf("metadata changed pixels: %s", src)
		}
	}
}

func TestDocumentErrors(t *testing.T) {
	for _, src := range []string{
		``, `<g/>`, `<svg/>`, `<svg width="NaN" height="20"/>`,
		`<svg width="20" height="20"/><svg width="20" height="20"/>`,
		`<svg width="20" height="20" x="1" x="2"/>`,
		`<svg width="20" height="20" viewBox="0 0 0 20"/>`,
		`<!DOCTYPE svg [<!ENTITY item "text">]><svg width="20" height="20">&item;</svg>`,
		`<svg width="20" height="20"><use href="#missing"/></svg>`,
		`<svg width="20" height="20"><defs><g id="a"><use href="#a"/></g></defs><use href="#a"/></svg>`,
		`<svg width="20" height="20"><rect fill="url(#missing)" width="4" height="4"/></svg>`,
		`<svg width="20" height="20"><path d="M?"/></svg>`,
		`<svg width="20" height="20"><path d="M0 0 X2 2"/></svg>`,
		`<svg width="20" height="20"><rect transform="rotate(2"/></svg>`,
		`<svg width="20" height="20"><defs><radialGradient id="g" r="-1"/></defs></svg>`,
		`<svg width="20" height="20"><defs><linearGradient id="g" href="#h"/><linearGradient id="h" href="#g"/></defs></svg>`,
	} {
		if icon, err := ReadIconStream(strings.NewReader(src), StrictErrorMode); err == nil || icon != nil {
			t.Fatalf("malformed document accepted: %q, %v", src, err)
		}
	}
	if _, err := ReadIconStream(nil); err == nil {
		t.Fatal("nil reader accepted")
	}
}

func TestStyleAndResources(t *testing.T) {
	cases := []struct {
		name, src string
		want      color.RGBA
	}{
		{"opacity override", `<g fill-opacity="0.25"><rect width="20" height="20" fill="red" fill-opacity="0.5"/></g>`, color.RGBA{R: 127, A: 127}},
		{"inline before presentation", `<rect style="fill:blue" fill="red" width="20" height="20"/>`, color.RGBA{B: 255, A: 255}},
		{"currentColor override", `<g color="red" fill="currentColor"><rect color="blue" width="20" height="20"/></g>`, color.RGBA{B: 255, A: 255}},
		{"class order", `<rect class="second first" width="20" height="20"/><style>.first{fill:red}.second{fill:blue}</style>`, color.RGBA{B: 255, A: 255}},
		{"forward class", `<rect class="blue" fill="red" width="20" height="20"/><style>.blue{fill:blue}</style>`, color.RGBA{B: 255, A: 255}},
		{"forward use", `<use href="#r"/><defs><rect id="r" fill="blue" width="20" height="20"/></defs>`, color.RGBA{B: 255, A: 255}},
		{"forward gradient", `<rect fill="url(#g)" width="20" height="20"/><defs><linearGradient id="g"><stop offset="0" stop-color="blue"/><stop offset="1" stop-color="blue"/></linearGradient></defs>`, color.RGBA{B: 255, A: 255}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="0 0 20 20">`+tt.src+`</svg>`), StrictErrorMode)
			if err != nil {
				t.Fatal(err)
			}
			got := icon.Rasterize(20, 20).RGBAAt(10, 10)
			for _, pair := range [][2]uint8{{got.R, tt.want.R}, {got.G, tt.want.G}, {got.B, tt.want.B}, {got.A, tt.want.A}} {
				d := int(pair[0]) - int(pair[1])
				if d < -1 || d > 1 {
					t.Fatalf("pixel=%v want=%v", got, tt.want)
				}
			}
		})
	}
}

func TestRasterizeFillAndViewport(t *testing.T) {
	for _, rule := range []string{"nonzero", "evenodd"} {
		icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="10 20 20 20"><g fill-rule="`+rule+`"><path fill="blue" d="M11 21H29V39H11Z M15 25H25V35H15Z"/></g></svg>`), StrictErrorMode)
		if err != nil {
			t.Fatal(err)
		}
		img := icon.Rasterize(80, 40)
		if img.RGBAAt(2, 20).A != 0 || img.RGBAAt(24, 5).A != 255 {
			t.Fatal("viewBox origin/aspect padding is incorrect")
		}
		want := uint8(255)
		if rule == "evenodd" {
			want = 0
		}
		if got := img.RGBAAt(40, 20).A; got != want {
			t.Fatalf("%s center alpha=%d want=%d", rule, got, want)
		}
	}
	for _, aspect := range []string{"xMidYMid meet", "xMinYMin meet", "xMaxYMax meet", "none", "xMidYMid slice"} {
		icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="10 20 20 10" preserveAspectRatio="`+aspect+`"><rect x="10" y="20" width="20" height="10" fill="red"/></svg>`), StrictErrorMode)
		if err != nil {
			t.Fatal(err)
		}
		img := icon.Rasterize(40, 40)
		for y := 1; y < 39; y++ {
			want := uint8(255)
			switch aspect {
			case "xMidYMid meet":
				if y < 10 || y >= 30 {
					want = 0
				}
			case "xMinYMin meet":
				if y >= 20 {
					want = 0
				}
			case "xMaxYMax meet":
				if y < 20 {
					want = 0
				}
			}
			if img.RGBAAt(20, y).A != want {
				t.Fatalf("%s y=%d alpha=%d want=%d", aspect, y, img.RGBAAt(20, y).A, want)
			}
		}
	}
}

func TestRasterizeStrokeTransforms(t *testing.T) {
	for _, scale := range []int{1, 2, 4} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="0 0 40 30"><g transform="translate(2,2) scale(2,1)"><path fill="none" stroke="blue" stroke-width="4" stroke-linecap="round" d="M4 10H12"/></g></svg>`), StrictErrorMode)
			if err != nil {
				t.Fatal(err)
			}
			img := icon.Rasterize(40*scale, 30*scale)
			// 局部圆形线帽变为椭圆：x 半径 4，y 半径 2。
			for _, p := range []struct {
				x, y  int
				solid bool
			}{{10, 12, true}, {26, 12, true}, {7, 12, true}, {10, 11, true}, {10, 9, false}, {5, 12, false}, {31, 12, false}} {
				got := img.RGBAAt(p.x*scale, p.y*scale).A
				if p.solid && got < 250 || !p.solid && got != 0 {
					t.Fatalf("scale=%d at(%d,%d) alpha=%d", scale, p.x, p.y, got)
				}
			}
		})
	}
	// 虚线长度使用局部单位，随几何恰好缩放一次。
	icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="0 0 30 10"><path transform="scale(2)" fill="none" stroke="red" stroke-width="2" stroke-dasharray="3 2" d="M1 2H14"/></svg>`), StrictErrorMode)
	if err != nil {
		t.Fatal(err)
	}
	img := icon.Rasterize(60, 20)
	for _, p := range []struct {
		x int
		a uint8
	}{{6, 255}, {14, 255}, {18, 0}, {26, 255}, {34, 255}, {38, 0}} {
		if got := img.RGBAAt(p.x, 8).A; got != p.a {
			t.Fatalf("dash x=%d alpha=%d want=%d", p.x, got, p.a)
		}
	}
}

func TestRasterizeGradientMapping(t *testing.T) {
	for _, kind := range []string{"linearGradient", "radialGradient"} {
		icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="0 0 40 40"><defs><`+kind+` id="g"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue"/></`+kind+`></defs><rect transform="translate(30,10) rotate(90)" width="20" height="10" fill="url(#g)"/></svg>`), StrictErrorMode)
		if err != nil {
			t.Fatal(err)
		}
		img := icon.Rasterize(80, 80)
		if kind == "linearGradient" {
			top, bottom := img.RGBAAt(50, 22), img.RGBAAt(50, 58)
			if top.R < 220 || bottom.B < 220 {
				t.Fatalf("gradient did not follow local axis: %v %v", top, bottom)
			}
		} else {
			center, edge := img.RGBAAt(50, 40), img.RGBAAt(40, 40)
			if center.R < 210 || edge.B < 200 {
				t.Fatalf("radial gradient did not follow transformed bbox: %v %v", center, edge)
			}
		}
	}
}

// 样本处均远离几何边缘；预期为预乘 sRGB。允许 1/255 的量化误差。
// 仿射样本经 SVG 坐标公式和 Chrome 156 核对，不以轴对齐包围盒近似。
func TestRasterizeGradientSemantics(t *testing.T) {
	type sample struct {
		x, y int
		want color.RGBA
	}
	for _, tt := range []struct {
		name, src     string
		width, height int
		samples       []sample
	}{
		{"affine", `<svg viewBox="0 0 60 60"><defs><linearGradient id="g" gradientTransform="rotate(30 .5 .5)"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient></defs><rect x="5" y="5" width="30" height="20" transform="matrix(1 .4 .5 1 5 5)" fill="url(#g)"/></svg>`, 160, 160, []sample{{50, 60, color.RGBA{R: 227, B: 28, A: 255}}, {70, 60, color.RGBA{R: 182, B: 73, A: 255}}, {100, 80, color.RGBA{R: 89, B: 166, A: 255}}}},
		{"stop alpha", `<svg viewBox="0 0 1 1"><defs><linearGradient id="g"><stop offset="0" stop-color="red" stop-opacity="0"/><stop offset="1" stop-color="blue"/></linearGradient></defs><rect width="1" height="1" fill="url(#g)"/></svg>`, 101, 101, []sample{{50, 50, color.RGBA{R: 64, B: 64, A: 128}}}},
		{"intrinsic alpha", `<svg viewBox="0 0 1 1"><defs><linearGradient id="g"><stop offset="0" stop-color="#ff000000"/><stop offset="1" stop-color="blue"/></linearGradient></defs><rect width="1" height="1" fill="url(#g)" fill-opacity=".5"/></svg>`, 101, 101, []sample{{50, 50, color.RGBA{R: 32, B: 32, A: 64}}}},
		{"user percent", `<svg viewBox="10 20 40 20"><defs><linearGradient id="g" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient></defs><rect x="10" y="20" width="40" height="20" fill="url(#g)"/></svg>`, 101, 51, []sample{{50, 25, color.RGBA{R: 64, B: 191, A: 255}}}},
		{"inherited units", `<svg viewBox="10 20 40 20"><defs><linearGradient id="g" href="#base" gradientUnits="userSpaceOnUse"/><linearGradient id="base" x2="100%"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient></defs><rect x="10" y="20" width="40" height="20" fill="url(#g)"/></svg>`, 101, 51, []sample{{50, 25, color.RGBA{R: 64, B: 191, A: 255}}}},
		{"stop inherit", `<svg viewBox="0 0 1 1"><defs><linearGradient id="g" color="blue" stop-color="currentColor" stop-opacity=".5"><stop offset="0" stop-color="inherit" stop-opacity="inherit"/><stop offset="1" stop-color="inherit" stop-opacity="inherit"/></linearGradient></defs><rect width="1" height="1" fill="url(#g)"/></svg>`, 101, 101, []sample{{50, 50, color.RGBA{B: 128, A: 128}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			icon, err := ReadIconStream(strings.NewReader(tt.src), StrictErrorMode)
			if err != nil {
				t.Fatal(err)
			}
			img := icon.Rasterize(tt.width, tt.height)
			for _, p := range tt.samples {
				got := img.RGBAAt(p.x, p.y)
				for _, pair := range [][2]uint8{{got.R, p.want.R}, {got.G, p.want.G}, {got.B, p.want.B}, {got.A, p.want.A}} {
					if d := int(pair[0]) - int(pair[1]); d < -1 || d > 1 {
						t.Fatalf("(%d,%d)=%v want %v", p.x, p.y, got, p.want)
					}
				}
			}
		})
	}
}

func TestUnsupportedFeatures(t *testing.T) {
	for _, body := range []string{
		`<g opacity=".5"><rect width="20" height="20"/></g>`,
		`<rect width="20" height="20" clip-path="url(#clip)"/>`,
		`<rect width="20" height="20" color-interpolation="linearRGB"/>`,
		`<unknown><style>.blue{fill:red}</style><rect width="20" height="20"/></unknown>`,
	} {
		src := `<svg viewBox="0 0 20 20">` + body + `<rect class="blue" width="20" height="20" fill="blue"/></svg>`
		if icon, err := ReadIconStream(strings.NewReader(src), StrictErrorMode); err == nil || icon != nil {
			t.Fatalf("unsupported feature accepted: %s", body)
		}
		icon, err := ReadIconStream(strings.NewReader(src), IgnoreErrorMode)
		if err != nil {
			t.Fatal(err)
		}
		if got := icon.Rasterize(20, 20).RGBAAt(10, 10); got != (color.RGBA{B: 255, A: 255}) {
			t.Fatalf("ignored subtree leaked: %v", got)
		}
	}
}

func TestSetTargetViewBoxOrigin(t *testing.T) {
	icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="10 20 20 10"><rect x="10" y="20" width="20" height="10" fill="red"/></svg>`), StrictErrorMode)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 60, 40))
	raster := NewDasher(60, 40, NewScannerGV(60, 40, img, img.Bounds()))
	icon.SetTarget(7, 9, 40, 20)
	icon.Draw(raster, 1)
	for _, p := range []struct {
		x, y  int
		alpha uint8
	}{{8, 10, 255}, {46, 28, 255}, {6, 10, 0}, {48, 28, 0}, {20, 30, 0}} {
		if got := img.RGBAAt(p.x, p.y).A; got != p.alpha {
			t.Fatalf("(%d,%d) alpha=%d want=%d", p.x, p.y, got, p.alpha)
		}
	}
}

func TestDocumentLimits(t *testing.T) {
	for _, body := range []string{
		strings.Repeat(`<g>`, 70) + strings.Repeat(`</g>`, 70),
		func() string {
			var b strings.Builder
			b.WriteString(`<defs>`)
			for i := 0; i < 70; i++ {
				fmt.Fprintf(&b, `<linearGradient id="g%d" href="#g%d"/>`, i, i+1)
			}
			b.WriteString(`<linearGradient id="g70"/></defs>`)
			return b.String()
		}(),
	} {
		if icon, err := ReadIconStream(strings.NewReader(`<svg width="20" height="20">`+body+`</svg>`), StrictErrorMode); err == nil || icon != nil {
			t.Fatal("excessive nesting/reference depth accepted")
		}
	}
}

func TestRasterizePrimitives(t *testing.T) {
	for _, tt := range []struct {
		name, src string
		want      color.RGBA
	}{
		{"percentage intrinsic", `<svg width="100%" height="100%" viewBox="0 0 20 20"><rect width="20" height="20" fill="blue"/></svg>`, color.RGBA{B: 255, A: 255}},
		{"explicit zero radius", `<svg viewBox="0 0 20 20"><rect width="20" height="20" rx="0" ry="10" fill="blue"/></svg>`, color.RGBA{B: 255, A: 255}},
		{"two translucent layers", `<svg viewBox="0 0 20 20"><rect width="20" height="20" fill="blue" fill-opacity=".5"/><rect width="20" height="20" fill="red" fill-opacity=".5"/></svg>`, color.RGBA{R: 128, B: 64, A: 192}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			icon, err := ReadIconStream(strings.NewReader(tt.src), StrictErrorMode)
			if err != nil {
				t.Fatal(err)
			}
			// (1,1) 既是稳定内部像素，也能区分直角和错误的 10 单位圆角。
			got := icon.Rasterize(40, 40).RGBAAt(1, 1)
			for _, pair := range [][2]uint8{{got.R, tt.want.R}, {got.G, tt.want.G}, {got.B, tt.want.B}, {got.A, tt.want.A}} {
				if d := int(pair[0]) - int(pair[1]); d < -1 || d > 1 {
					t.Fatalf("pixel=%v want=%v", got, tt.want)
				}
			}
		})
	}
}

func TestSVGColorEdgeCases(t *testing.T) {
	for _, src := range []string{"hsl(0,,)", "hsl(0,%,)", "rgba(0,0,0,NaN)"} {
		if _, err := ParseSVGColor(src); err == nil {
			t.Fatalf("invalid color accepted: %s", src)
		}
	}
	for _, src := range []string{"#f008", "#ff000088", "rgba(255,0,0,53.3333333333%)"} {
		got, err := ParseSVGColor(src)
		if err != nil {
			t.Fatal(err)
		}
		if n := color.NRGBAModel.Convert(got).(color.NRGBA); n != (color.NRGBA{R: 255, A: 136}) {
			t.Fatalf("%s => %v", src, n)
		}
	}
}

func TestRasterizeIsolation(t *testing.T) {
	icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="0 0 20 20"><defs><linearGradient id="g"><stop offset=".7" stop-color="red"/><stop offset=".2" stop-color="blue"/></linearGradient></defs><rect x="2" y="2" width="16" height="16" fill="url(#g)" stroke="blue" stroke-dasharray="2 1"/></svg>`), StrictErrorMode)
	if err != nil {
		t.Fatal(err)
	}
	icon.SetTarget(10, 20, 100, 200)
	transform := icon.Transform
	expected := icon.Rasterize(60, 40)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 8; j++ {
				img := icon.Rasterize(20+i*3, 40+j)
				if img == nil {
					t.Error("missing image")
				}
				if got := icon.Rasterize(60, 40); !bytes.Equal(got.Pix, expected.Pix) {
					t.Error("shared parse result changed")
				}
			}
		}(i)
	}
	wg.Wait()
	if icon.Transform != transform || math.Abs(icon.Grads["g"].Stops[1].Offset-.7) > 1e-9 {
		t.Fatal("rendering mutated document state")
	}
	expected.Pix[0] = 255
	if icon.Rasterize(60, 40).Pix[0] != 0 {
		t.Fatal("rendering retained a caller-owned bitmap")
	}
	for _, size := range [][2]int{{0, 1}, {1, 0}, {-1, 3}, {int(^uint(0) >> 1), 2}} {
		if icon.Rasterize(size[0], size[1]) != nil {
			t.Fatalf("invalid size accepted: %v", size)
		}
	}
}

func FuzzReadIconStream(f *testing.F) {
	for _, src := range []string{`<svg viewBox="0 0 10 10"><path d="M1 1H9V9Z"/></svg>`, `<svg viewBox="0 0 1 1"><rect fill="hsl(0,,)"/></svg>`, `<!DOCTYPE svg><svg width="4" height="4" xml:space="preserve"/>`} {
		f.Add(src)
	}
	f.Fuzz(func(t *testing.T, src string) {
		if len(src) > 16384 {
			return
		}
		icon, err := ReadIconStream(strings.NewReader(src), StrictErrorMode)
		if err != nil && icon != nil {
			t.Fatal("partial icon returned on failure")
		}
	})
}

func TestSVGColorNormalization(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  color.NRGBA
	}{
		{"hsl(120,200%,25%)", color.NRGBA{G: 128, A: 255}},
		{"hsl(999,100%,25%)", color.NRGBA{R: 83, B: 128, A: 255}},
		{"hsl(-120,100%,50%)", color.NRGBA{B: 255, A: 255}},
		{"hsl(120.0,100%,50%)", color.NRGBA{G: 255, A: 255}},
		{"hsla(120,100%,50%,.5)", color.NRGBA{G: 255, A: 128}},
		{"rgb(255 0 0 / 50%)", color.NRGBA{R: 255, A: 128}},
		{"rgb(255.0,0,0)", color.NRGBA{R: 255, A: 255}},
	} {
		got, err := ParseSVGColor(tt.input)
		if err != nil {
			t.Fatal(tt.input, err)
		}
		if actual := color.NRGBAModel.Convert(got).(color.NRGBA); actual != tt.want {
			t.Fatalf("%s=%v want %v", tt.input, actual, tt.want)
		}
	}
	for _, src := range []string{"rgb(0,100%,255)", "rgba(0,50%,0,.5)", "hsl(0,50,50%)", "rgb(1,2,3", "hsl(0,,)", "hsla(NaN,0%,0%,1)"} {
		if _, err := ParseSVGColor(src); err == nil {
			t.Fatal("invalid color accepted", src)
		}
	}
}

// 采样点避开边缘，预期由 1in=96px 及起始 stop 的 pad 语义确定。
func TestRasterizeCorpusRegressions(t *testing.T) {
	for _, tt := range []struct {
		name, svg  string
		size, x, y int
		want       color.RGBA
	}{
		{"absolute units", `<svg viewBox="0 0 120 120"><rect x="1in" y="2.54mm" width=".25in" height="10pt" fill="blue"/></svg>`, 120, 100, 15, color.RGBA{B: 255, A: 255}},
		{"duplicate start pad", `<svg viewBox="0 0 20 20"><defs><linearGradient id="g" x1=".25" x2=".75"><stop offset="0" stop-color="black"/><stop offset="0" stop-color="gray"/><stop offset="1" stop-color="white"/></linearGradient></defs><rect x="2" y="2" width="16" height="16" fill="url(#g)"/></svg>`, 40, 6, 20, color.RGBA{A: 255}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			icon, err := ReadIconStream(strings.NewReader(tt.svg), StrictErrorMode)
			if err != nil {
				t.Fatal(err)
			}
			got := icon.Rasterize(tt.size, tt.size).RGBAAt(tt.x, tt.y)
			if got != tt.want {
				t.Fatalf("(%d,%d)=%v want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

// 固定来源的原始 SVG；采样预期来自 Chrome/套件 PNG，并避开抗锯齿边缘。
func TestOnlineCorpusFixtures(t *testing.T) {
	for _, tt := range []struct {
		file string
		x, y int
		want color.RGBA
	}{
		{"lucide/avocado.svg", 64, 64, color.RGBA{A: 255}},
		{"lucide/azerbaijani-manat.svg", 64, 32, color.RGBA{A: 255}},
		{"lucide/book-bookmark.svg", 48, 48, color.RGBA{A: 255}},
		{"lucide/braces.svg", 14, 63, color.RGBA{A: 255}},
		{"lucide/dumbbell.svg", 85, 29, color.RGBA{A: 255}},
		{"lucide/squirrel.svg", 36, 33, color.RGBA{A: 255}},
		{"lucide/mop.svg", 52, 108, color.RGBA{}},
		{"lucide/parentheses.svg", 86, 14, color.RGBA{A: 255}},
		{"bootstrap/heptagon.svg", 64, 127, color.RGBA{A: 255}},
		{"resvg/shapes/rect/mm-values.svg", 64, 64, color.RGBA{G: 128, A: 255}},
		{"resvg/paint-servers/stop/stops-with-equal-offset-5.svg", 20, 64, color.RGBA{A: 255}},
		{"resvg/paint-servers/stop/stops-with-equal-offset-6.svg", 20, 64, color.RGBA{A: 255}},
		{"resvg/painting/fill/hsl-120-200percent-25percent.svg", 64, 64, color.RGBA{G: 128, A: 255}},
		{"resvg/painting/fill/hsl-999-100percent-25percent.svg", 64, 64, color.RGBA{R: 83, B: 128, A: 255}},
		{"resvg/structure/style-attribute/non-presentational-attribute.svg", 64, 64, color.RGBA{R: 255, A: 255}},
		{"resvg/painting/stroke-dasharray/on-a-circle.svg", 57, 27, color.RGBA{G: 128, A: 255}},
		{"resvg/painting/stroke-dasharray/n-0.svg", 22, 22, color.RGBA{}},
	} {
		t.Run(tt.file, func(t *testing.T) {
			icon, err := ReadIcon("testdata/corpus/"+tt.file, StrictErrorMode)
			if err != nil {
				t.Fatal(err)
			}
			img := icon.Rasterize(128, 128)
			if got := img.RGBAAt(tt.x, tt.y); got != tt.want {
				t.Fatalf("(%d,%d)=%v want=%v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

// SVG 2 的几何属性使用现有局部级联；内部颜色与外部透明由明确坐标独立确定。
func TestCSSGeometry(t *testing.T) {
	for _, tt := range []struct {
		name, body               string
		x, y, outsideX, outsideY int
		want                     color.RGBA
	}{
		{"inline size", `<rect x="2" y="2" width="16" style="height:16px" fill="red"/>`, 10, 10, 1, 10, color.RGBA{R: 255, A: 255}},
		{"class position", `<style>.s {x:4px;y:4px;width:12px;height:12px;fill:blue}</style><rect class="s" width="20" height="20"/>`, 10, 10, 2, 10, color.RGBA{B: 255, A: 255}},
		{"inline precedence", `<style>.s {width:8px;height:8px;fill:red}</style><rect class="s" x="2" y="2" width="2" height="2" style="width:16px;height:16px;fill:blue"/>`, 14, 14, 1, 10, color.RGBA{B: 255, A: 255}},
		{"geometry does not inherit", `<g style="width:20px;height:20px"><rect width="4" height="4" fill="red"/></g>`, 2, 2, 10, 10, color.RGBA{R: 255, A: 255}},
		{"circle", `<circle cx="1" cy="1" r="1" style="cx:10px;cy:10px;r:6px" fill="blue"/>`, 14, 10, 18, 10, color.RGBA{B: 255, A: 255}},
		{"ellipse", `<ellipse style="cx:50%;cy:50%;rx:6px;ry:4px" fill="blue"/>`, 14, 10, 10, 16, color.RGBA{B: 255, A: 255}},
		{"rounded rectangle", `<rect x="2" y="2" width="16" height="16" style="rx:6px;ry:6px" fill="blue"/>`, 10, 10, 2, 2, color.RGBA{B: 255, A: 255}},
		{"use position", `<defs><rect id="r" width="4" height="4" fill="blue"/></defs><use href="#r" style="x:8px;y:8px"/>`, 10, 10, 2, 2, color.RGBA{B: 255, A: 255}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="0 0 20 20">`+tt.body+`</svg>`), StrictErrorMode)
			if err != nil {
				t.Fatal(err)
			}
			img := icon.Rasterize(20, 20)
			if got := img.RGBAAt(tt.x, tt.y); got != tt.want {
				t.Fatalf("内部 (%d,%d)=%v，期望 %v", tt.x, tt.y, got, tt.want)
			}
			if got := img.RGBAAt(tt.outsideX, tt.outsideY); got.A != 0 {
				t.Fatalf("外部 (%d,%d)=%v，期望透明", tt.outsideX, tt.outsideY, got)
			}
		})
	}
	icon, err := ReadIconStream(strings.NewReader(`<svg style="width:20px;height:20px"><rect width="20" height="20" fill="red"/></svg>`), StrictErrorMode)
	if err != nil {
		t.Fatal(err)
	}
	if icon.ViewBox.W != 20 || icon.ViewBox.H != 20 {
		t.Fatalf("根尺寸 = %v，期望 20×20", icon.ViewBox)
	}
	if got := icon.Rasterize(20, 20).RGBAAt(10, 10); got != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("根几何属性未生效: %v", got)
	}
}

// 140 次 0.1 的累加应到达 y=16；浮点尾差不能在 8x 放大后留下灰色底边。
func TestFractionalPathBoundaryCoverage(t *testing.T) {
	path := "M2 2" + strings.Repeat("v0.1", 140) + "H14V2Z"
	icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="0 0 16 16"><path d="`+path+`"/></svg>`), StrictErrorMode)
	if err != nil {
		t.Fatal(err)
	}
	img := icon.Rasterize(128, 128)
	if got := img.RGBAAt(64, 127); got != (color.RGBA{A: 255}) {
		t.Fatalf("底部内部像素=%v，期望不透明黑色", got)
	}
	if got := img.RGBAAt(0, 127); got.A != 0 {
		t.Fatalf("矩形外部像素=%v，期望透明", got)
	}
}

func TestSVGCompatibilityExtensions(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		want       color.RGBA
	}{
		{"percentage geometry", `<rect x="25%" y="25%" width="50%" height="50%" fill="blue"/>`, color.RGBA{B: 255, A: 255}},
		{"missing paint fallback", `<rect width="20" height="20" fill="url(#missing) blue"/>`, color.RGBA{B: 255, A: 255}},
		{"fallback currentColor", `<rect width="20" height="20" color="blue" fill="url(#missing) currentColor"/>`, color.RGBA{B: 255, A: 255}},
		{"fallback none", `<rect width="20" height="20" fill="url(#missing) none"/>`, color.RGBA{}},
		{"external paint fallback", `<rect width="20" height="20" fill="url(https://invalid.example/gradient) blue"/>`, color.RGBA{B: 255, A: 255}},
		{"class comments", `<style>/* } { */ .blue { /* fill:red; */ fill:blue }</style><rect class="blue" width="20" height="20"/>`, color.RGBA{B: 255, A: 255}},
		{"inline comments", `<rect width="20" height="20" fill="red" style="/* note */ fill:blue"/>`, color.RGBA{B: 255, A: 255}},
		{"static link", `<a href="https://invalid.example"><rect width="20" height="20" fill="blue"/></a>`, color.RGBA{B: 255, A: 255}},
		{"ellipse auto radius", `<ellipse cx="10" cy="10" ry="8" fill="blue"/>`, color.RGBA{B: 255, A: 255}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="0 0 20 20">`+tt.body+`</svg>`), StrictErrorMode)
			if err != nil {
				t.Fatal(err)
			}
			if got := icon.Rasterize(40, 40).RGBAAt(20, 20); got != tt.want {
				t.Fatalf("pixel=%v want=%v", got, tt.want)
			}
		})
	}
	// 非零 viewBox 原点不计入百分比坐标；独立参考 Chrome 的中心应为透明。
	icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="10 20 40 20"><rect x="0%" y="0%" width="100%" height="100%" fill="red"/></svg>`), StrictErrorMode)
	if err != nil {
		t.Fatal(err)
	}
	if got := icon.Rasterize(101, 51).RGBAAt(50, 25); got.A != 0 {
		t.Fatalf("percent origin shifted: %v", got)
	}
}

func TestUnsupportedCorpusFeatures(t *testing.T) {
	for _, body := range []string{
		`<style>rect { fill:red }</style><rect width="20" height="20"/>`,
		`<rect width="20" height="20" style="mix-blend-mode:multiply"/>`,
		`<rect width="20" height="20" paint-order="stroke fill"/>`,
		`<rect width="20" height="20" transform-origin="center"/>`,
		`<defs><radialGradient id="g" fr=".2"/></defs>`,
		`<rect width="20" height="20" systemLanguage="ru"/>`,
	} {
		if icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="0 0 20 20">`+body+`</svg>`), StrictErrorMode); err == nil || icon != nil {
			t.Fatalf("unsupported feature accepted: %s", body)
		}
	}
}

func TestRelativeIntrinsicSizeWithViewBox(t *testing.T) {
	for _, unit := range []string{"em", "rem", "ex"} {
		src := `<svg width="1` + unit + `" height="1` + unit + `" viewBox="0 0 20 20"><rect width="20" height="20" fill="blue"/></svg>`
		icon, err := ReadIconStream(strings.NewReader(src), StrictErrorMode)
		if err != nil {
			t.Fatal(err)
		}
		if got := icon.Rasterize(40, 40).RGBAAt(20, 20); got != (color.RGBA{B: 255, A: 255}) {
			t.Fatal(got)
		}
	}
	if icon, err := ReadIconStream(strings.NewReader(`<svg width="1em" height="1em"/>`), StrictErrorMode); err == nil || icon != nil {
		t.Fatal("unresolved font size accepted without viewBox")
	}
}

// 圆的等价路径从三点钟位置顺时针前进，10/100 的虚线不能上下镜像。
func TestCircleDashDirection(t *testing.T) {
	icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="0 0 80 80"><circle cx="40" cy="40" r="20" fill="none" stroke="blue" stroke-width="4" stroke-dasharray="10 100"/></svg>`), StrictErrorMode)
	if err != nil {
		t.Fatal(err)
	}
	img := icon.Rasterize(160, 160)
	if a := img.RGBAAt(119, 87).A; a != 255 {
		t.Fatalf("clockwise first dash alpha=%d", a)
	}
	if a := img.RGBAAt(119, 75).A; a != 0 {
		t.Fatalf("counterclockwise gap alpha=%d", a)
	}
	for _, d := range []string{"A5 5 0 0 1 50 50", "L10 10", "Z"} {
		if icon, err := ReadIconStream(strings.NewReader(`<svg viewBox="0 0 80 80"><path d="`+d+`"/></svg>`), StrictErrorMode); err == nil || icon != nil {
			t.Fatal("path without moveto accepted", d)
		}
	}
}
