package tablelite

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/lunixbochs/vtclean"
	"github.com/mattn/go-runewidth"
)

type Surrounder func(string) string

type FillFunc func(string, int) string

type Column struct {
	Value          string
	Surrounder     Surrounder
	Filler         FillFunc
	NoColumnSpacer bool

	ls []string
}

func (c *Column) lines() []string {
	if c.ls != nil {
		return c.ls
	}
	c.ls = strings.Split(c.Value, "\n")
	return c.ls
}

func (c *Column) filler(defaultFiller FillFunc) FillFunc {
	if c.Filler != nil {
		return c.Filler
	}
	return defaultFiller
}

func (c *Column) string(w *Writer, colidx, lineidx int) (string, bool) {
	filler := c.filler(w.DefaultFiller)
	width := w.columnWidths()[colidx]
	surrounder := c.Surrounder
	if surrounder == nil {
		surrounder = func(s string) string {
			return s
		}
	}

	if lineidx >= len(c.lines()) {
		return filler("", width), false
	}
	return surrounder(filler(c.lines()[lineidx], width)), true
}

type Writer struct {
	RuneCondition *runewidth.Condition
	ColSpacer     string
	DefaultFiller FillFunc

	rows           [][]Column
	colWidths      []int
	colSpacerWidth int
}

func (w *Writer) Append(row []string) {
	cols := []Column{}
	for _, col := range row {
		cols = append(cols, Column{Value: col})
	}
	w.rows = append(w.rows, cols)
}

func (w *Writer) AppendColumns(row []Column) {
	w.rows = append(w.rows, row)
}

func (w *Writer) columnWidths() (widths []int) {
	if w.colWidths != nil {
		return w.colWidths
	}

	widths = []int{}

	for _, row := range w.rows {
		for index, col := range row {
			if len(widths) <= index {
				widths = append(widths, 0)
			}
			for _, line := range strings.Split(vtclean.Clean(col.Value, false), "\n") {
				width := w.RuneCondition.StringWidth(line)
				if widths[index] < width {
					widths[index] = width
				}
			}
		}
	}

	w.colWidths = widths
	return w.colWidths
}

func (w *Writer) renderRow(writer io.Writer, row []Column) {
	lineidx := 0
	for {
		found := false
		line := make([]byte, 0, 1024)
		useNextSpacer := false
		for colidx, col := range row {
			s, b := col.string(w, colidx, lineidx)
			if useNextSpacer && !col.NoColumnSpacer {
				line = append(line, []byte(w.ColSpacer)...)
			}
			useNextSpacer = w.columnWidths()[colidx] > 0
			line = append(line, []byte(s)...)
			if b {
				found = true
			}
		}
		if !found {
			break
		}
		fmt.Fprintln(writer, string(line))
		lineidx++
	}
}

func (w *Writer) RenderTo(writer io.Writer) {
	w.colSpacerWidth = w.RuneCondition.StringWidth(w.ColSpacer)
	for _, row := range w.rows {
		w.renderRow(writer, row)
	}
}

func (w *Writer) Render() string {
	buf := new(bytes.Buffer)
	w.RenderTo(buf)
	return buf.String()
}

func New() *Writer {
	condition := runewidth.DefaultCondition

	return &Writer{
		RuneCondition: condition,
		ColSpacer:     " ",
		DefaultFiller: condition.FillRight,
		rows:          [][]Column{},
	}
}
