package optimize

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/apex/log"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const TIME_FORMAT = "2006-01-02 15:04:05"

const TEXT_COLOR = tcell.ColorLightSkyBlue
const TEXT_COLOR_WARNING = tcell.ColorRed
const BACKGROUND_COLOR = tcell.ColorBlack
const PAGE_EVENTS_TITLE = "Events"
const PAGE_EVENT_DETAILS_TITLE = "Event Details"

type EventTableData struct {
	events          []EventsRow
	showOptimizerID bool
	tview.TableContent
}

func (d *EventTableData) GetCell(row, column int) *tview.TableCell {
	// Return the cell for the given row and column.
	event := d.events[row]

	if row == 0 {
		var headers []string
		if d.showOptimizerID {
			headers = []string{"Timestamp↓", "Optimizer ID", "Event Type", "OPT/STG/EXP", "Summary"}
		} else {
			headers = []string{"Timestamp↓", "Event Type", "OPT/STG/EXP", "Summary"}
		}

		return &tview.TableCell{
			Text:          strings.ToUpper(headers[column]),
			Align:         tview.AlignCenter,
			Color:         tview.Styles.PrimaryTextColor,
			NotSelectable: true,
		}

	} else {
		if !d.showOptimizerID && column > 0 {
			column++
		}

		switch column {
		case 0:
			t := event.Timestamp.Format(TIME_FORMAT)
			return &tview.TableCell{Text: t, Align: tview.AlignLeft, Color: TEXT_COLOR}
		case 1:
			optID := event.EventAttributes["optimize.optimization.optimizer_id"].(string)
			return &tview.TableCell{Text: optID, Align: tview.AlignLeft, Color: TEXT_COLOR}
		case 2:
			ev_type := event.EventAttributes["appd.event.type"].(string)
			return &tview.TableCell{Text: ev_type, Align: tview.AlignLeft, Color: TEXT_COLOR}
		case 3:
			return &tview.TableCell{Text: event.EntityInfo, Align: tview.AlignLeft, Color: TEXT_COLOR}
		case 4:
			return &tview.TableCell{Text: event.Summary, Align: tview.AlignLeft, Color: TEXT_COLOR}
		}
	}

	panic(fmt.Sprintf("Invalid cell requested, row %d, column %d", row, column))
}

func (d *EventTableData) GetRowCount() int {
	return len(d.events)
}

func (d *EventTableData) GetColumnCount() int {
	if d.showOptimizerID {
		return 5
	}
	return 4
}

func (d *EventTableData) InsertRow(row int) {
	log.Debugf("Inserting row %d", row)
}

func showInteractive(
	eventsChan chan []EventsRow,
	errorChan chan error,
	loadChan chan bool,
	showOptimizerID bool,
	following bool,
) {

	// Close STDERR to avoid progress coming from the UQL client
	os.Stderr, _ = os.Open(os.DevNull)

	app := tview.NewApplication()
	pages := tview.NewPages()

	pages.SetBackgroundColor(BACKGROUND_COLOR)
	f := tview.NewFrame(pages).SetBorders(0, 0, 0, 0, 0, 0)
	eventsData := &EventTableData{showOptimizerID: showOptimizerID}
	eventsListTable := addEventsPage(pages, eventsData)

	go func() {
		startEventsListender(eventsChan, errorChan, loadChan, eventsData, app, eventsListTable, f, following)
	}()

	if err := app.SetRoot(f, true).SetFocus(f).Run(); err != nil {
		panic(err)
	}

}

func startEventsListender(
	eventsChan chan []EventsRow,
	errorChan chan error,
	loadChan chan bool,
	eventsData *EventTableData,
	app *tview.Application,
	table *tview.Table,
	f *tview.Frame,
	following bool,
) {
	for {
		select {
		case loading := <-loadChan:
			var text string
			if loading {
				text = "Loading events..."
			} else if following {
				text = "Waiting for updates"
			} else {
				text = "Loaded events"
			}

			app.QueueUpdateDraw(func() {
				f.Clear().AddText(text, false, tview.AlignCenter, TEXT_COLOR)
			})
		case err := <-errorChan:
			if err == nil {
				continue
			}
			log.Warnf("startEventsListender err: %v", err)

			app.QueueUpdateDraw(func() {
				f.Clear().AddText("Failed to load events", false, tview.AlignCenter, TEXT_COLOR_WARNING)
			})

		case eventsRows := <-eventsChan:
			for _, event := range eventsRows {
				// Insert at right place in the list, using the event timestamp
				index := sort.Search(len(eventsData.events), func(i int) bool {
					return event.Timestamp.After(eventsData.events[i].Timestamp)
				})
				eventsData.events = append(eventsData.events[:index], append([]EventsRow{event}, eventsData.events[index:]...)...)

				app.QueueUpdateDraw(func() {
					table.SetTitle(fmt.Sprintf(" %s [%d] ", PAGE_EVENTS_TITLE, len(eventsData.events)))
					table.InsertRow(0)
				})
			}
		}
	}
}

func makeTableWithHeaders(title string, headers []string) *tview.Table {
	table := tview.NewTable().SetSelectable(true, false).SetFixed(1, 0).SetEvaluateAllRows(true)
	table.SetBackgroundColor(BACKGROUND_COLOR)
	table.SetSelectedStyle(tcell.StyleDefault.Background(TEXT_COLOR).Foreground(BACKGROUND_COLOR))

	paddedTitle := fmt.Sprintf(" %s ", title)
	table.SetTitle(paddedTitle).SetTitleColor(TEXT_COLOR).SetTitleAlign(tview.AlignCenter)
	table.SetBorder(true).SetBorderPadding(0, 0, 1, 1).SetBorderColor(TEXT_COLOR).SetBorderAttributes(tcell.AttrNone)
	for i, header := range headers {
		table.SetCell(0, i, &tview.TableCell{
			Text:          strings.ToUpper(header),
			Align:         tview.AlignCenter,
			Color:         tview.Styles.PrimaryTextColor,
			NotSelectable: true,
		})
	}

	return table
}

func addEventsPage(pages *tview.Pages, eventsData *EventTableData) *tview.Table {

	title := fmt.Sprintf("%s [%d]", PAGE_EVENTS_TITLE, len(eventsData.events))
	table := makeTableWithHeaders(title, nil)
	table.SetContent(eventsData)

	pages.AddAndSwitchToPage(PAGE_EVENTS_TITLE, table, true)

	table.SetSelectedFunc(func(row int, column int) {
		addEventDetailsPage(pages, eventsData.events[row])
	})

	return table

}
func addEventDetailsPage(pages *tview.Pages, eventsRow EventsRow) {

	table := makeTableWithHeaders(PAGE_EVENT_DETAILS_TITLE, []string{"Attribute", "Value"})

	table.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			pages.RemovePage(PAGE_EVENT_DETAILS_TITLE)
		}
	})

	row := 1
	// sort event attributes
	attributes := make([]string, 0, len(eventsRow.EventAttributes))
	for k := range eventsRow.EventAttributes {
		attributes = append(attributes, k)
	}
	sort.Strings(attributes)

	for _, k := range attributes {
		v := fmt.Sprintf("%s", eventsRow.EventAttributes[k])
		table.SetCell(row, 0, &tview.TableCell{Text: k, Align: tview.AlignLeft, Color: TEXT_COLOR})
		table.SetCell(row, 1, &tview.TableCell{Text: v, Align: tview.AlignLeft, Color: TEXT_COLOR, Expansion: 1})
		row++
	}

	pages.AddAndSwitchToPage(PAGE_EVENT_DETAILS_TITLE, table, true)
}
