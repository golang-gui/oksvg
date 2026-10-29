// Copyright 2018 The oksvg Authors. All rights reserved.
// created: 2018 by S.R.Wiley
package oksvg_test

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"

	. "github.com/golang-gui/oksvg"
	. "github.com/srwiley/rasterx"
	// . "github.com/srwiley/scanFT"
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
