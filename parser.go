package oksvg

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"math"
	"strings"

	"github.com/srwiley/rasterx"
	"golang.org/x/net/html/charset"
)

const svgNamespace = "http://www.w3.org/2000/svg"
const xlinkNamespace = "http://www.w3.org/1999/xlink"
const maxElementDepth = 64

// Elements are parsing-time data only; drawing retains compiled paths.
type svgElement struct {
	name     string
	attrs    []xml.Attr
	text     string
	parent   *svgElement
	children []*svgElement
}

func (e *svgElement) attr(name string) string {
	for _, a := range e.attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func readDocument(stream io.Reader) (*svgElement, error) {
	if stream == nil {
		return nil, fmt.Errorf("nil SVG reader")
	}
	d := xml.NewDecoder(stream)
	d.CharsetReader = charset.NewReaderLabel
	var root *svgElement
	var stack []*svgElement
	for {
		token, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			if len(stack) == 0 {
				if root != nil || t.Name.Local != "svg" || (t.Name.Space != "" && t.Name.Space != svgNamespace) {
					return nil, fmt.Errorf("exactly one SVG root required")
				}
			} else if (t.Name.Space != "" && t.Name.Space != svgNamespace) || t.Name.Local == "metadata" {
				if err := d.Skip(); err != nil {
					return nil, err
				}
				continue
			}
			if len(stack) >= maxElementDepth {
				return nil, fmt.Errorf("SVG nesting exceeds %d elements", maxElementDepth)
			}
			e := &svgElement{name: t.Name.Local}
			seen := make(map[xml.Name]bool)
			// Namespace URI is significant. Unknown foreign attributes cannot become
			// ordinary SVG attributes merely because their local names match.
			unqualified := make(map[string]bool)
			for _, a := range t.Attr {
				if seen[a.Name] {
					return nil, fmt.Errorf("duplicate XML attribute %q", a.Name.Local)
				}
				seen[a.Name] = true
				if a.Name.Space == "" {
					unqualified[a.Name.Local] = true
				}
			}
			for _, a := range t.Attr {
				if a.Name.Space == "" {
					if a.Name.Local != "xmlns" {
						e.attrs = append(e.attrs, a)
					}
				} else if a.Name.Space == xlinkNamespace && a.Name.Local == "href" && !unqualified["href"] {
					e.attrs = append(e.attrs, xml.Attr{Name: xml.Name{Local: "href"}, Value: a.Value})
				}
			}
			if len(stack) == 0 {
				root = e
			} else {
				e.parent = stack[len(stack)-1]
				e.parent.children = append(e.parent.children, e)
			}
			stack = append(stack, e)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("unbalanced SVG elements")
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 {
				if strings.TrimSpace(string(t)) != "" {
					return nil, fmt.Errorf("text outside SVG root")
				}
			} else {
				stack[len(stack)-1].text += string(t)
			}
		// encoding/xml does not resolve DTDs or custom entities. Do not fetch them.
		case xml.Directive, xml.ProcInst, xml.Comment:
		}
	}
	if root == nil || len(stack) != 0 {
		return nil, fmt.Errorf("complete SVG root required")
	}
	return root, nil
}

type gradientDefinition struct {
	element    *svgElement
	state      uint8 // 0 = unseen, 1 = visiting, 2 = compiled
	attributes map[string]string
	depth      int
}

type svgCompiler struct {
	cursor    *IconCursor
	ids       map[string]*svgElement
	gradients map[string]*gradientDefinition
	active    map[*svgElement]bool
}

func parseDocument(stream io.Reader, mode ErrorMode) (*SvgIcon, error) {
	root, err := readDocument(stream)
	if err != nil {
		return nil, err
	}
	icon := &SvgIcon{Defs: make(map[string][]definition), Grads: make(map[string]*rasterx.Gradient), Transform: rasterx.Identity}
	c := &svgCompiler{
		cursor: &IconCursor{icon: icon}, ids: make(map[string]*svgElement),
		gradients: make(map[string]*gradientDefinition), active: make(map[*svgElement]bool),
	}
	c.cursor.ErrorMode = mode
	if err := c.collect(root); err != nil {
		return nil, err
	}
	if err := drawSvg(c.cursor, root.attrs); err != nil {
		return nil, err
	}
	v := icon.ViewBox
	if !isFinite(v.X) || !isFinite(v.Y) || !isFinite(v.W) || !isFinite(v.H) || v.W <= 0 || v.H <= 0 {
		return nil, fmt.Errorf("finite positive viewBox or width/height required")
	}
	icon.aspect, err = parseAspectRatio(root.attr("preserveAspectRatio"))
	if err != nil {
		return nil, err
	}
	for id := range c.gradients {
		if err := c.gradient(id, 0); err != nil {
			return nil, err
		}
	}
	c.cursor.StyleStack = []PathStyle{DefaultStyle}
	if err := c.compile(root, 0); err != nil {
		return nil, err
	}
	return icon, nil
}

func (c *svgCompiler) collect(e *svgElement) error {
	switch e.name {
	case "svg":
		if e.parent != nil {
			return c.cursor.unsupported("nested svg")
		}
	case "g", "defs", "path", "rect", "circle", "ellipse", "line", "polyline", "polygon", "use", "linearGradient", "radialGradient", "stop", "style", "title", "desc":
	default:
		// Unsupported containers are opaque: their IDs and styles must not leak.
		return c.cursor.unsupported("element " + e.name)
	}
	id := e.attr("id")
	if id != "" {
		if c.ids[id] != nil {
			return fmt.Errorf("duplicate SVG id %q", id)
		}
		c.ids[id] = e
		if e.name == "linearGradient" || e.name == "radialGradient" {
			c.gradients[id] = &gradientDefinition{element: e}
		} else {
			c.cursor.icon.Defs[id] = definitionList(e)
		}
	}
	if e.name == "style" {
		for _, block := range strings.Split(e.text, "}") {
			if strings.TrimSpace(block) == "" {
				continue
			}
			classes, err := parseClasses(block + "}")
			if err != nil {
				return err
			}
			for name, attrs := range classes {

				c.cursor.icon.classRules = append(c.cursor.icon.classRules, classRule{name, attrs})
			}
		}
	}
	for _, child := range e.children {
		if err := c.collect(child); err != nil {
			return err
		}
	}
	return nil
}

func definitionList(e *svgElement) []definition {
	out := []definition{{ID: e.attr("id"), Tag: e.name, Attrs: e.attrs}}
	for _, child := range e.children {
		out = append(out, definitionList(child)...)
	}
	if e.name == "g" {
		out = append(out, definition{Tag: "endg"})
	}
	return out
}

// Resolve paint-server attributes before coordinates so inherited gradientUnits
// and percentage defaults are independent of XML attribute order.
func (c *svgCompiler) gradient(id string, depth int) error {
	if depth >= maxElementDepth {
		return fmt.Errorf("gradient reference depth exceeds %d", maxElementDepth)
	}
	definition := c.gradients[id]
	if definition == nil {
		return fmt.Errorf("missing gradient %q", id)
	}
	switch definition.state {
	case 1:
		return fmt.Errorf("cyclic gradient reference %q", id)
	case 2:
		if depth+definition.depth > maxElementDepth {
			return fmt.Errorf("gradient reference depth exceeds %d", maxElementDepth)
		}
		return nil
	}
	e := definition.element
	definition.state = 1
	attrs := make(map[string]string)
	referenceDepth := 1
	var stops []rasterx.GradStop
	if href := e.attr("href"); href != "" {
		if !strings.HasPrefix(href, "#") {
			return fmt.Errorf("external gradient reference is unsupported")
		}
		if err := c.gradient(href[1:], depth+1); err != nil {
			return err
		}
		referenceDepth += c.gradients[href[1:]].depth
		for k, v := range c.gradients[href[1:]].attributes {
			if c.gradients[href[1:]].element.name == e.name || k == "gradientUnits" || k == "gradientTransform" || k == "spreadMethod" {
				attrs[k] = v
			}
		}
		stops = c.cursor.icon.Grads[href[1:]].Stops
	}
	for _, a := range e.attrs {
		attrs[a.Name.Local] = a.Value
	}
	g := &rasterx.Gradient{IsRadial: e.name == "radialGradient", Bounds: c.cursor.icon.ViewBox, Matrix: rasterx.Identity, Stops: stops}
	c.cursor.grad = g
	for _, k := range []string{"gradientUnits", "gradientTransform", "spreadMethod"} {
		if v, ok := attrs[k]; ok {
			if err := c.cursor.ReadGradAttr(xml.Attr{Name: xml.Name{Local: k}, Value: v}); err != nil {
				return err
			}
		}
	}
	names := []string{"x1", "y1", "x2", "y2"}
	defaults := []string{"0%", "0%", "100%", "0%"}
	if g.IsRadial {
		names = []string{"cx", "cy", "fx", "fy", "r"}
		defaults = []string{"50%", "50%", "", "", "50%"}
	}
	for i, k := range names {
		v := attrs[k]
		if v == "" {
			v = defaults[i]
		}
		if v == "" {
			g.Points[i] = g.Points[i-2]
			continue
		}
		n, err := gradientLength(v, i, g.Units, c.cursor.icon.ViewBox)
		if err != nil {
			return fmt.Errorf("gradient %s: %w", k, err)
		}
		g.Points[i] = n
	}
	if g.IsRadial && g.Points[4] < 0 {
		return fmt.Errorf("negative gradient radius")
	}
	// Stop properties inherit from the definition's real ancestors, not its href.
	var chain []*svgElement
	for n := e; n != nil; n = n.parent {
		chain = append(chain, n)
	}
	c.cursor.StyleStack = []PathStyle{DefaultStyle}
	for i := len(chain) - 1; i >= 0; i-- {
		if err := c.cursor.PushStyle(chain[i].attrs); err != nil {
			return err
		}
	}
	ownStops := false
	for _, child := range e.children {
		if child.name != "stop" {
			if child.name != "title" && child.name != "desc" {
				if err := c.cursor.unsupported("gradient child " + child.name); err != nil {
					return err
				}
			}
			continue
		}
		if !ownStops {
			g.Stops = nil
			ownStops = true
		}
		if err := c.cursor.PushStyle(child.attrs); err != nil {
			return err
		}
		if err := drawStop(c.cursor, child.attrs); err != nil {
			return err
		}
		c.cursor.StyleStack = c.cursor.StyleStack[:len(c.cursor.StyleStack)-1]
	}
	if ownStops {
		for i := range g.Stops {
			g.Stops[i].Offset = clampUnit(g.Stops[i].Offset)
			if i > 0 && g.Stops[i].Offset < g.Stops[i-1].Offset {
				g.Stops[i].Offset = g.Stops[i-1].Offset
			}
		}
	}
	c.cursor.icon.Grads[id] = g
	definition.attributes = attrs
	definition.depth = referenceDepth
	definition.state = 2
	return nil
}

func gradientLength(value string, axis int, units rasterx.GradientUnits, viewport struct{ X, Y, W, H float64 }) (float64, error) {
	value = strings.TrimSpace(value)
	if !strings.HasSuffix(value, "%") {
		return parseFloat(value, 64)
	}
	n, err := readFraction(value)
	if err != nil || units == rasterx.ObjectBoundingBox {
		return n, err
	}
	switch axis {
	case 0, 2:
		return viewport.X + n*viewport.W, nil
	case 1, 3:
		return viewport.Y + n*viewport.H, nil
	default:
		return n * math.Hypot(viewport.W, viewport.H) / math.Sqrt2, nil
	}
}

func (c *svgCompiler) compile(e *svgElement, depth int) error {
	if depth >= maxElementDepth || c.active[e] {
		return fmt.Errorf("cyclic or excessively deep SVG reference")
	}
	switch e.name {
	case "defs", "linearGradient", "radialGradient", "stop", "style":
		return nil
	}
	if e.name == "svg" && depth > 0 {
		return c.cursor.unsupported("nested svg")
	}
	if err := c.cursor.PushStyle(e.attrs); err != nil {
		return fmt.Errorf("%s: %w", e.name, err)
	}
	defer func() { c.cursor.StyleStack = c.cursor.StyleStack[:len(c.cursor.StyleStack)-1] }()
	s := &c.cursor.StyleStack[len(c.cursor.StyleStack)-1]
	if !s.display {
		return nil
	}
	c.active[e] = true
	defer delete(c.active, e)
	switch e.name {
	case "svg", "g":
		for _, child := range e.children {
			if err := c.compile(child, depth+1); err != nil {
				return err
			}
		}
	case "title":
		c.cursor.icon.Titles = append(c.cursor.icon.Titles, e.text)
	case "desc":
		c.cursor.icon.Descriptions = append(c.cursor.icon.Descriptions, e.text)
	case "use":
		href := e.attr("href")
		if !strings.HasPrefix(href, "#") {
			return c.cursor.unsupported("external or missing use reference")
		}
		target := c.ids[href[1:]]
		if target == nil {
			return fmt.Errorf("missing use reference %q", href)
		}
		x, y := 0.0, 0.0
		var err error
		if v := e.attr("x"); v != "" {
			x, err = parseFloat(v, 64)
			if err != nil {
				return err
			}
		}
		if v := e.attr("y"); v != "" {
			y, err = parseFloat(v, 64)
			if err != nil {
				return err
			}
		}
		s.mAdder.M = s.mAdder.M.Translate(x, y)
		return c.compile(target, depth+1)
	case "path", "rect", "circle", "ellipse", "line", "polyline", "polygon":
		if !s.visible {
			return nil
		}
		c.cursor.Path = nil
		if err := draw(e.name, c.cursor, e.attrs); err != nil {
			return fmt.Errorf("%s: %w", e.name, err)
		}
		if len(c.cursor.Path) == 0 {
			return nil
		}
		style := *s
		var err error
		style.fillerColor, err = c.resolvePaint(style.fillerColor, style.color)
		if err != nil {
			return err
		}
		style.linerColor, err = c.resolvePaint(style.linerColor, style.color)
		if err != nil {
			return err
		}
		path := append(rasterx.Path(nil), c.cursor.Path...)
		c.cursor.icon.SVGPaths = append(c.cursor.icon.SVGPaths, SvgPath{PathStyle: style, Path: path})
	default:
		return c.cursor.unsupported("element " + e.name)
	}
	return nil
}

func (c *svgCompiler) resolvePaint(value any, currentColor any) (any, error) {
	switch v := value.(type) {
	case currentColorPaint:
		return currentColor, nil
	case paintReference:
		g := c.cursor.icon.Grads[string(v)]
		if g == nil {
			return nil, fmt.Errorf("missing paint server %q", v)
		}
		return *g, nil
	default:
		return value, nil
	}
}

func (c *IconCursor) unsupported(feature string) error {
	err := fmt.Errorf("unsupported SVG %s", feature)
	if c.ErrorMode == StrictErrorMode {
		return err
	}
	if c.ErrorMode == WarnErrorMode {
		log.Print(err)
	}
	return nil
}

func isFinite(v float64) bool     { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func clampUnit(v float64) float64 { return math.Max(0, math.Min(1, v)) }
