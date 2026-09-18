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


Oksvg uses the [rasterx](https://github.com/srwiley/rasterx) rasterizer package which implements full SVG2.0 path functions, including the newer 'arc' join-mode.

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
