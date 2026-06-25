package ui

import "math"

type scrollbarGeometry struct {
	visible bool
	start   int
	length  int
}

type scrollbarRenderer struct {
	geometry scrollbarGeometry
	row      int
}

func newScrollbarRenderer(geometry scrollbarGeometry) scrollbarRenderer {
	return scrollbarRenderer{geometry: geometry}
}

func (r *scrollbarRenderer) Next() string {
	scrollbar := renderScrollbar(r.row, r.geometry)
	r.row++
	return scrollbar
}

func (m SelectorModel) scrollbar(row, visible int) string {
	return renderScrollbar(row, scrollbarGeometryFor(len(m.items), visible, m.offset, m.scrollbarConfig))
}

func (m SelectorModel) inspectScrollbar(row, visible, total int) string {
	return renderScrollbar(row, scrollbarGeometryFor(total, visible, m.inspectOffset, m.scrollbarConfig))
}

func renderScrollbar(row int, geometry scrollbarGeometry) string {
	if !geometry.visible {
		return " "
	}
	if row >= geometry.start && row < geometry.start+geometry.length {
		return OKStyle.Render("|")
	}
	return DimStyle.Render(":")
}

func scrollbarGeometryFor(total, visible, offset int, config ScrollbarConfig) scrollbarGeometry {
	if visible <= 0 {
		return scrollbarGeometry{}
	}
	config = config.normalized()
	if total <= visible {
		if !config.ShowWhenContentFits {
			return scrollbarGeometry{}
		}
		return scrollbarGeometry{visible: true, length: visible}
	}

	track := visible
	minLength := ratioLength(track, config.MinThumbRatio, true)
	maxLengthCap := ratioLength(track, config.MaxThumbRatio, false)
	if maxLengthCap < minLength {
		maxLengthCap = minLength
	}
	naturalLength := int(math.Ceil(float64(track) * float64(visible) / float64(total)))
	if naturalLength < minLength {
		naturalLength = minLength
	}
	if naturalLength > maxLengthCap {
		naturalLength = maxLengthCap
	}

	maxOffset := total - visible
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	progress := float64(offset) / float64(maxOffset)
	length := boundaryAwareThumbLength(track, naturalLength, minLength, progress)
	center := progress * float64(track-1)
	start := int(math.Round(center - float64(length-1)/2))
	if start < 0 {
		start = 0
	}
	maxStart := track - length
	if start > maxStart {
		start = maxStart
	}
	return scrollbarGeometry{visible: true, start: start, length: length}
}

func boundaryAwareThumbLength(track, naturalLength, minLength int, progress float64) int {
	if naturalLength <= minLength || track <= 1 {
		return minLength
	}
	center := progress * float64(track-1)
	halfMax := float64(naturalLength-1) / 2
	start := center - halfMax
	end := center + halfMax
	overflow := 0.0
	if start < 0 {
		overflow = -start
	}
	if end > float64(track-1) {
		overflow = end - float64(track-1)
	}
	if overflow == 0 {
		return naturalLength
	}
	maxOverflow := halfMax
	if maxOverflow <= 0 {
		return minLength
	}
	shrink := int(math.Round(overflow / maxOverflow * float64(naturalLength-minLength)))
	length := naturalLength - shrink
	if length < minLength {
		return minLength
	}
	if length > naturalLength {
		return naturalLength
	}
	return length
}

func ratioLength(track int, ratio float64, roundUp bool) int {
	if track <= 0 {
		return 0
	}
	value := float64(track) * ratio
	length := int(math.Floor(value))
	if roundUp {
		length = int(math.Ceil(value))
	}
	if length < 1 {
		return 1
	}
	if length > track {
		return track
	}
	return length
}
