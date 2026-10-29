package oksvg

import (
	"fmt"
	"strings"
)

type aspectRatio struct {
	none, slice bool
	x, y        float64
	set         bool
}

func parseAspectRatio(value string) (aspectRatio, error) {
	a := aspectRatio{x: .5, y: .5, set: true}
	f := strings.Fields(value)
	if len(f) == 0 {
		return a, nil
	}
	if f[0] == "defer" {
		f = f[1:]
		if len(f) == 0 {
			return a, fmt.Errorf("invalid preserveAspectRatio")
		}
	}
	if f[0] == "none" {
		a.none = true
	} else {
		align := f[0]
		if len(align) != 8 || align[0] != 'x' || align[4] != 'Y' {
			return a, fmt.Errorf("invalid preserveAspectRatio %q", value)
		}
		values := map[string]float64{"Min": 0, "Mid": .5, "Max": 1}
		var ok bool
		a.x, ok = values[align[1:4]]
		if !ok {
			return a, fmt.Errorf("invalid preserveAspectRatio %q", value)
		}
		a.y, ok = values[align[5:]]
		if !ok {
			return a, fmt.Errorf("invalid preserveAspectRatio %q", value)
		}
	}
	if len(f) > 2 {
		return a, fmt.Errorf("invalid preserveAspectRatio %q", value)
	}
	if len(f) == 2 {
		switch f[1] {
		case "meet":
		case "slice":
			a.slice = true
		default:
			return a, fmt.Errorf("invalid preserveAspectRatio %q", value)
		}
	}
	return a, nil
}
