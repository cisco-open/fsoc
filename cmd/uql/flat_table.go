// Copyright 2023 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package uql

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	verticalBorder  string = "|"
	bottomBorder    string = "-"
	cornerBorder    string = "+"
	headerSeparator string = "="
)

// FlatTable is transformed complex UQL response as a single table.
type FlatTable struct {
	header  genericCell
	content genericCell
}

// genericCell is a general building block of a table.
// A cell represents a value from a column.
// The columns might contain sub-columns. The structure of columns is a tree.
// Therefore, a cell might contain child sub-cells
type genericCell interface {
	// childWidths returns maximal widths of columns in the leaves of the column tree.
	childWidths() []int
	// inflateWidths sets minimal widths of columns in the leaves of the column tree.
	// If a particular column is currently wider, it stays wider.
	inflateWidths(widths []int)
	// childColumns returns number of all sub-columns.
	childColumns() int
	// render provides string representation of the cell.
	render() string
	// renderBottomBorder renders divider for the bottom part of the cell.
	renderBottomBorder() string
}

// atomicCell is the simplest table cell containing a scalar value.
type atomicCell struct {
	value  string
	width  int
	height int
}

// simpleTable is a 2D table of atomicCells.
type simpleTable struct {
	data      [][]genericCell
	colNum    int
	maxWidths []int
}

// emptyTable is a table with no value. Only this cell could be used for missing or no values.
type emptyTable struct {
	colNum    int
	maxWidths []int
}

// combinedTable is a generic table where each cell might be an another sub-table.
type combinedTable struct {
	data        [][]genericCell
	topCols     int
	flattenCols int
	maxWidths   []int
}

// headerCell is a cell used for header part of the table.
// HeaderCell optionally contains name of a column and sub-column header cells.
type headerCell struct {
	colName     string
	subColumns  []*headerCell
	flattenCols int
	width       int
	maxWidths   []int
}

// MakeFlatTable builds the table representation from parsed UQL response.
func MakeFlatTable(response *Response) FlatTable {
	content := makeCell(response.Main(), response.Model())
	header := makeHeaderCell(response.Model())

	// align header columns widths and content columns widths
	mainWidths := content.childWidths()
	headerWidths := header.childWidths()
	if len(mainWidths) != len(headerWidths) {
		panic("Header and body of the table does not have the same number of columns. This is a bug!")
	}
	header.inflateWidths(mainWidths)
	headerWidths = header.childWidths()

	unifiedWidths := make([]int, len(mainWidths))
	for i := range unifiedWidths {
		unifiedWidths[i] = max(mainWidths[i], headerWidths[i])
	}
	content.inflateWidths(unifiedWidths)
	return FlatTable{
		header:  header,
		content: content,
	}
}

// Render prepares string representation of the FlatTable.
func (t *FlatTable) Render() string {
	return lipgloss.JoinVertical(lipgloss.Left, t.header.render(), t.header.renderBottomBorder(), t.content.render())
}

func makeCell(dataset Complex, model *Model) genericCell {
	allSimpleCells := true
	for _, field := range model.Fields {
		if field.Model != nil {
			allSimpleCells = false
			break
		}
	}
	colNo := len(model.Fields)
	if complexIsEmpty(dataset) {
		return makeEmptyTable(model)
	}
	rowNo := len(dataset.Values())
	rows := make([][]genericCell, rowNo)
	for r, row := range dataset.Values() {
		rowCells := make([]genericCell, colNo)
		for c, field := range model.Fields {
			// is simple scalar field
			if field.Model == nil {
				// use empty string instead of "<nil>" for missing values
				var toString string
				if row[c] != nil {
					toString = fmt.Sprint(row[c])
				}
				rowCells[c] = makeAtomicCell(toString)
			} else {
				rowCells[c] = makeCell(row[c].(Complex), field.Model)
			}
		}
		rows[r] = rowCells
	}
	if allSimpleCells {
		return makeSimpleTable(rows, colNo)
	} else {
		return makeCombinedTable(rows, colNo)
	}
}

func makeAtomicCell(value string) *atomicCell {
	spaced := lipgloss.NewStyle().Margin(0).Padding(0, 1).Render(replaceTabs(value))
	return &atomicCell{
		value:  spaced,
		width:  lipgloss.Width(spaced),
		height: lipgloss.Height(spaced),
	}
}

func (c *atomicCell) childWidths() []int {
	return []int{c.width}
}

func (c *atomicCell) inflateWidths(widths []int) {
	if len(widths) != 1 {
		panic(fmt.Sprintf(
			"Trying to set width for %d columns, but there is just one. This is a bug",
			len(widths),
		),
		)
	}
	c.width = max(c.width, widths[0])
}

func (c *atomicCell) childColumns() int {
	return 1
}

func (c *atomicCell) render() string {
	render := lipgloss.
		NewStyle().
		Width(c.width).
		MaxWidth(c.width).
		Height(c.height).
		MaxHeight(c.height).
		Render(c.value)
	return render
}

func (c *atomicCell) renderBottomBorder() string {
	return strings.Repeat(bottomBorder, c.width)
}

func makeSimpleTable(cells [][]genericCell, colNum int) *simpleTable {
	if len(cells) == 0 {
		panic("No row data provided for creation of a table. This is a bug.")
	}
	return &simpleTable{
		data:      cells,
		colNum:    colNum,
		maxWidths: calculateMaxWidths(cells),
	}
}

func (t *simpleTable) childWidths() []int {
	widths := make([]int, len(t.maxWidths))
	copy(widths, t.maxWidths)
	return widths
}

func (t *simpleTable) inflateWidths(widths []int) {
	if len(widths) != t.childColumns() {
		panic(fmt.Sprintf(
			"Trying to set width for %d columns, but there are %d of them. This is a bug.",
			len(widths),
			t.childColumns()),
		)
	}
	for _, row := range t.data {
		for c := range row {
			row[c].inflateWidths(widths[c : c+1])
		}
	}
	for i := range widths {
		t.maxWidths[i] = max(t.maxWidths[i], widths[i])
	}
}

func (t *simpleTable) childColumns() int {
	return t.colNum
}

func (t *simpleTable) render() string {
	renderedRows := make([]string, len(t.data))
	for ri, row := range t.data {
		renderedRows[ri] = renderTableRow(&row, t.colNum)
	}
	tableContent := lipgloss.JoinVertical(lipgloss.Left, renderedRows...)
	return lipgloss.JoinVertical(lipgloss.Left, tableContent, t.renderBottomBorder())
}

func (t *simpleTable) renderBottomBorder() string {
	cellBottoms := make([]string, t.colNum)
	for i, c := range t.data[0] {
		cellBottoms[i] = c.renderBottomBorder()
	}
	return strings.Join(cellBottoms, cornerBorder)
}

func makeEmptyTable(model *Model) *emptyTable {
	var recCount func(fields []ModelField) int
	recCount = func(fields []ModelField) int {
		s := 0
		for _, f := range fields {
			if f.Model != nil {
				s += recCount(f.Model.Fields)
			} else {
				s += 1
			}
		}
		return s
	}
	colNum := recCount(model.Fields)
	return &emptyTable{
		colNum:    colNum,
		maxWidths: make([]int, colNum),
	}
}

func (t *emptyTable) childWidths() []int {
	widths := make([]int, len(t.maxWidths))
	copy(widths, t.maxWidths)
	return widths
}

func (t *emptyTable) inflateWidths(widths []int) {
	if len(widths) != t.childColumns() {
		panic(fmt.Sprintf(
			"Trying to set width for %d columns, but there are %d of them. This is a bug",
			len(widths),
			t.childColumns()),
		)
	}
	copy(t.maxWidths, widths)
}

func (t *emptyTable) childColumns() int {
	return len(t.maxWidths)
}

func (t *emptyTable) render() string {
	width := sum(t.maxWidths...) + ((t.colNum - 1) * lipgloss.Width(verticalBorder))
	return lipgloss.
		NewStyle().
		Width(width).
		MaxWidth(width).
		Height(1).
		MaxHeight(1).
		Render("")
}

func (t *emptyTable) renderBottomBorder() string {
	cellBottoms := make([]string, t.colNum)
	for i, width := range t.maxWidths {
		cellBottoms[i] = strings.Repeat(bottomBorder, width)
	}
	return strings.Join(cellBottoms, cornerBorder)
}

func makeCombinedTable(cells [][]genericCell, colNum int) *combinedTable {
	if len(cells) == 0 || len(cells[0]) == 0 {
		panic("Trying to render a sub-table with either zero rows or zero columns. This is a bug")
	}
	maxWidths := calculateMaxWidths(cells)
	return &combinedTable{
		data:        cells,
		topCols:     colNum,
		flattenCols: len(maxWidths),
		maxWidths:   maxWidths,
	}
}

func (t *combinedTable) childWidths() []int {
	widths := make([]int, len(t.maxWidths))
	copy(widths, t.maxWidths)
	return widths
}

func (t *combinedTable) inflateWidths(widths []int) {
	if len(widths) != t.flattenCols {
		panic(fmt.Sprintf(
			"Trying to set width for %d columns, but there are %d of them. This is a bug",
			len(widths),
			t.flattenCols),
		)
	}
	offset := 0
	divided := make([][]int, t.topCols)
	for col, cell := range t.data[0] {
		divided[col] = widths[offset : offset+cell.childColumns()]
		offset += cell.childColumns()
	}
	for _, row := range t.data {
		for col := range row {
			row[col].inflateWidths(divided[col])
		}
	}
	for i := range widths {
		t.maxWidths[i] = max(t.maxWidths[i], widths[i])
	}
}

func (t *combinedTable) childColumns() int {
	return t.flattenCols
}

func (t *combinedTable) render() string {
	renderedRows := make([]string, len(t.data))
	hDivider := t.renderBottomBorder()
	for r, row := range t.data {
		renderedRow := renderTableRow(&row, t.topCols)
		if lastLineIsBorder(row) {
			// Cut cell bottom borders that would touch row horizontal divider.
			// A simpleTable or combinedTable is always higher than any other cell and has the bottom border.
			renderedRow = renderedRow[:strings.LastIndex(renderedRow, "\n")]
		}
		renderedRows[r] = lipgloss.JoinVertical(lipgloss.Left, renderedRow, hDivider)
	}
	return lipgloss.JoinVertical(lipgloss.Left, renderedRows...)
}

func (t *combinedTable) renderBottomBorder() string {
	cellBottoms := make([]string, t.topCols)
	for i, c := range t.data[0] {
		cellBottoms[i] = c.renderBottomBorder()
	}
	return strings.Join(cellBottoms, cornerBorder)
}

func makeHeaderCell(model *Model) *headerCell {
	columns := make([]*headerCell, len(model.Fields))
	totalNestedCols := 0
	for fi, field := range model.Fields {
		colName := lipgloss.NewStyle().Margin(0).Padding(0, 1).Render(field.Alias)
		colWidth := lipgloss.Width(colName)
		var childColumns *headerCell
		if field.Model != nil {
			childColumns = makeHeaderCell(field.Model)
			childWidths := childColumns.childWidths()
			columns[fi] = &headerCell{
				colName:     colName,
				width:       colWidth,
				maxWidths:   childWidths,
				flattenCols: len(childWidths),
				subColumns:  []*headerCell{childColumns},
			}
			totalNestedCols += columns[fi].childColumns()
		} else {
			columns[fi] = &headerCell{
				colName:     colName,
				width:       colWidth,
				maxWidths:   []int{colWidth},
				flattenCols: 1,
				subColumns:  nil,
			}
			totalNestedCols += 1
		}
	}
	maxWidths := make([]int, totalNestedCols)
	offset := 0
	for c := range columns {
		widths := columns[c].childWidths()
		copy(maxWidths[offset:offset+len(widths)], widths)
		offset += len(widths)
	}
	// wrapping cell without name
	return &headerCell{
		colName:     "",
		width:       0,
		maxWidths:   maxWidths,
		flattenCols: totalNestedCols,
		subColumns:  columns,
	}
}

func (c *headerCell) childWidths() []int {
	widths := make([]int, len(c.maxWidths))
	copy(widths, c.maxWidths)
	return widths
}

func (c *headerCell) inflateWidths(widths []int) {
	if len(widths) != c.flattenCols {
		panic(fmt.Sprintf(
			"Trying to set width for %d columns, but there are %d of them. This is a bug",
			len(widths),
			c.flattenCols),
		)
	}
	var newMaxWidths = make([]int, len(widths))
	copy(newMaxWidths, widths)
	totalSubWidth := sum(newMaxWidths...) + ((len(newMaxWidths) - 1) * lipgloss.Width(verticalBorder))
	// resize most-right child columns if current header column is wider than all child columns together
	if c.width > totalSubWidth {
		newMaxWidths[len(newMaxWidths)-1] += c.width - totalSubWidth
	} else {
		c.width = totalSubWidth
	}
	// propagate the resize
	offset := 0
	for _, cell := range c.subColumns {
		cell.inflateWidths(newMaxWidths[offset : offset+cell.childColumns()])
		copy(newMaxWidths[offset:offset+cell.childColumns()], cell.maxWidths)
		offset += cell.childColumns()
	}
	c.maxWidths = newMaxWidths
}

func (c *headerCell) childColumns() int {
	return c.flattenCols
}

func (c *headerCell) render() string {
	if c.subColumns != nil {
		renderedChildren := make([]string, len(c.subColumns))
		maxHeight := 0
		for i, column := range c.subColumns {
			subColumn := column.render()
			maxHeight = max(maxHeight, lipgloss.Height(subColumn))
			renderedChildren[i] = subColumn
		}
		verticalSpace := lipgloss.NewStyle().Margin(0).Padding(0).Height(maxHeight).MaxHeight(maxHeight)
		// all same height
		for i := range c.subColumns {
			renderedChildren[i] = verticalSpace.Render(renderedChildren[i])
		}
		vDivider := verticalDivider(verticalBorder, maxHeight)
		interleaved := interleave(renderedChildren, vDivider)
		joinedChildren := lipgloss.JoinHorizontal(lipgloss.Top, interleaved...)
		if c.colName != "" {
			var colName = lipgloss.NewStyle().Width(c.width).MaxWidth(c.width).Render(c.colName)
			return lipgloss.JoinVertical(lipgloss.Left, colName, joinedChildren)
		} else {
			return joinedChildren
		}
	} else {
		return lipgloss.NewStyle().Width(c.width).MaxWidth(c.width).Render(c.colName)
	}
}

func (c *headerCell) renderBottomBorder() string {
	if c.subColumns != nil {
		cellBottoms := make([]string, len(c.subColumns))
		for i, c := range c.subColumns {
			cellBottoms[i] = c.renderBottomBorder()
		}
		return strings.Join(cellBottoms, headerSeparator)
	}
	return strings.Repeat(headerSeparator, c.width)
}

// calculateMaxWidths traverses the whole table of values and collects maximum widths of all leaf cells
// from the column tree structure.
func calculateMaxWidths(cells [][]genericCell) []int {
	var nestedCols int
	for _, c := range cells[0] {
		nestedCols += c.childColumns()
	}
	maxWidths := make([]int, nestedCols)
	for _, row := range cells {
		var offset = 0
		var colWidths = make([]int, nestedCols)
		for _, c := range row {
			celWidths := c.childWidths()
			copy(colWidths[offset:offset+len(celWidths)], celWidths)
			offset += len(celWidths)
		}
		for i, current := range maxWidths {
			maxWidths[i] = max(current, colWidths[i])
		}
	}
	return maxWidths
}

// renderTableRow renders all generic cells for a single row of a table.
func renderTableRow(row *[]genericCell, cols int) string {
	renderedRowCells := make([]string, cols)
	maxCellHeight := 0
	for c, cell := range *row {
		renderedCell := cell.render()
		maxCellHeight = max(maxCellHeight, lipgloss.Height(renderedCell))
		renderedRowCells[c] = renderedCell
	}
	forcedHeight := lipgloss.NewStyle().Margin(0).Padding(0).Height(maxCellHeight).MaxHeight(maxCellHeight)
	for c := range *row {
		renderedRowCells[c] = forcedHeight.Render(renderedRowCells[c])
	}
	vDivider := verticalDivider(verticalBorder, maxCellHeight)
	interleaved := interleave(renderedRowCells, vDivider)
	return lipgloss.JoinHorizontal(lipgloss.Top, interleaved...)
}

// verticalDivider makes vertical divider of a given height.
func verticalDivider(segment string, height int) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		strings.Split(strings.Repeat(segment, height), "")...,
	)
}

// interleave makes a new array of all given items where each odd indexed item would the given separator item.
func interleave(items []string, separator string) []string {
	interleaved := make([]string, (len(items)*2)-1)
	for i := 0; i < (len(items)*2)-1; i++ {
		if i%2 == 0 {
			interleaved[i] = items[i/2]
		} else {
			interleaved[i] = separator
		}
	}
	return interleaved
}

// lastLineIsBorder checks if rendering of the row will produce a bottom border for the highest cell of the row.
func lastLineIsBorder(row []genericCell) bool {
	for _, cell := range row {
		switch cell.(type) {
		case *simpleTable, *combinedTable:
			return true
		}
	}
	return false
}

// replaceTabs will replace tabs with space inside given string.
func replaceTabs(text string) string {
	// go-runewidth reports tab control characters as zero length. Table cell widths are therefore wrongly calculated.
	tabWidth := 4
	return strings.ReplaceAll(text, "\t", strings.Repeat(" ", tabWidth))
}

// complexIsEmpty checks presence of any data in complex data structures.
func complexIsEmpty(data Complex) bool {
	switch typed := data.(type) {
	case *DataSet:
		// without pointer cast we cannot compare interface with nil.
		return typed == nil || len(typed.Values()) == 0
	case ComplexData:
		return len(typed.Values()) == 0
	}
	panic("Unexpected type implementing Complex type. This is a bug")
}

func max(a int, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func sum(a ...int) int {
	s := 0
	for _, v := range a {
		s += v
	}
	return s
}
