<p align="center">
  <a href="https://pkg.go.dev/github.com/golang-gui/oksvg?tab=doc" title="Go API Reference" rel="nofollow"><img src="https://img.shields.io/badge/go-documentation-blue.svg?style=flat" alt="Go API Reference"></a>
</p>

# oksvg
Oksvg is a rasterizer for a partial implementation of the SVG2.0 specification in Go.

This is GOUI's fork of [fyne-io/oksvg](https://github.com/fyne-io/oksvg),
originally by [srwiley](https://github.com/srwiley/oksvg). Fixes are proposed
upstream separately from fork-specific module changes. The repeated-arc fix is
tracked in [fyne-io/oksvg#6](https://github.com/fyne-io/oksvg/pull/6).

Although many SVG elements will not be read by oksvg, it is good enough to faithfully produce thousands, but certainly not all, SVG icons available both for free and commercially. A list of valid and invalid elements is in the doc folder.

The package can be installed by running:

```
go get github.com/golang-gui/oksvg@latest
```

### Rendering an icon

```go
icon, err := oksvg.ReadIconStream(strings.NewReader(svgSource), oksvg.StrictErrorMode)
if err != nil {
    return err
}
bitmap := icon.Rasterize(64, 64)
```

`Rasterize` returns a new, caller-owned premultiplied `*image.RGBA`, with
transparent pixels outside the artwork. It applies `viewBox` and
`preserveAspectRatio` (default: `xMidYMid meet`). Invalid target sizes return
`nil`. It has no image cache and does not change `SetTarget`/`Transform` state;
concurrent calls are safe while the parsed icon is not modified by callers.
It uses the pure Go [scanFT](https://github.com/golang-gui/scanFT) scanner for both
nonzero and evenodd filling. No cgo or native SVG library is required.

The existing `SetTarget`/`Draw` interface remains available. `SetTarget`
explicitly stretches the viewBox to its target rectangle. A caller supplying a
scanner to `Draw` must choose one that implements the requested winding rule;
rasterx's `ScannerGV` does not implement evenodd filling.

### Parsing and supported SVG features

DOCTYPE declarations, XML metadata attributes such as `xml:space`, and foreign
namespace metadata are accepted. The parser never fetches DTDs, expands custom
entities, or loads external files or URLs. Foreign attributes cannot override
ordinary SVG attributes with the same local name.

The icon subset supports shapes and paths, affine transforms, local `use`
references, simple class rules, inline styles, `currentColor`, fill rules,
fill/stroke opacity, stroke caps/joins/dashes, and linear/radial gradients.
Definitions may follow their uses. Inline styles override presentation
attributes, and inherited fill/stroke opacity is overridden rather than
multiplied by a child's declaration.

Compact arc flags, percentage geometry and stroke lengths, 96-dpi absolute
length units, and local paint fallbacks such as `url(#paint) currentColor` are
supported. Function colors include RGB/HSL alpha and space/slash forms.
Relative intrinsic `em`/`ex`/`rem` sizes can accompany an explicit viewBox;
font-relative shape lengths still require an unavailable font context.

SVG 2 geometry properties `x`, `y`, `width`, `height`, `cx`, `cy`, `r`, `rx`
and `ry` also accept supported lengths/percentages in class rules and inline
styles, using the same precedence as paint properties. Geometry is local to
each element and does not implicitly inherit. This does not add full CSS
layout, `calc()`, or CSS path syntax.

`StrictErrorMode` rejects recognized unsupported rendering features, including
object/group opacity other than 1, clipping, masks, filters, text and animation.
Unknown attributes are ignored for forward compatibility. `IgnoreErrorMode`
(the legacy default) and `WarnErrorMode` remain available; unsupported elements
are skipped with their entire subtrees. Malformed XML, numbers, paths and
missing/cyclic local references fail in every mode and return no partial icon.

See [the feature list](doc/SVG_Element_List.txt) and
[the parser/rasterization design](doc/DesignSVG.md) for scope and limitations.
Known renderer limitations and online corpus results are recorded in
[the corpus validation notes](doc/CorpusValidation.md). Reversing round joins,
collinear curve cusps, and implicit subpaths following closepath have regression
coverage; remaining reference differences are recorded separately.


Oksvg uses the maintained [rasterx](https://github.com/golang-gui/rasterx) fork for
path stroking, including the extended 'arc' join mode.

The current development branches use the `golang-gui` module paths throughout.
Dependencies are pinned to exact fork commits. Those commits must be available
on their remotes before standalone module resolution; local integration before
pushing uses a temporary Go workspace. See the validation notes for the staging
and release order. Legacy `Draw` users must use the matching fork's rasterx types.

![arcs and caps](doc/TestShapes.png)

### Extra non-standard features

In addition to 'arc' as a valid join mode value, oksvg also allows 'arc-clip' which is the arc analog of miter-clip and some extra capping and gap values. It can also specify different capping functions for line starts and ends.

#### Rasterizations of SVG to PNG from creative commons 3.0 sources

Example renderings of unedited open source SVG files by oksvg and rasterx are shown below.

Thanks to [Freepik](http://www.freepik.com) from [Flaticon](https://www.flaticon.com/)
Licensed by [Creative Commons 3.0](http://creativecommons.org/licenses/by/3.0/) for the example icons shown below, and also used as test icons in the testdata folder.

![Jupiter](doc/jupiter.png)

![lander](doc/lander.png)

![mountains](doc/mountains.png)

![bus](doc/school-bus.png)
