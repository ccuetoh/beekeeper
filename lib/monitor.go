/*
 * Copyright © 2020 Camilo Hernández <me@camiloh.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package beekeeper

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"log"
	"os"
	"time"

	"github.com/rivo/tview"
)

const monitorMaxWorkersPerPage = 5

// Monitor represents a Beekeeper Monitor.
type Monitor struct {
	App         *tview.Application
	Pages       *tview.Pages
	CurrentPage int
}

// NewMonitor creates and returns a *Monitor struct.
func NewMonitor() *Monitor {
	return &Monitor{
		App:         tview.NewApplication(),
		Pages:       tview.NewPages(),
		CurrentPage: 1,
	}
}

// Run starts the Monitor, renders it and updates it regularly.
func (m *Monitor) Run(configs ...Config) {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	} else {
		config = NewDefaultConfig()
	}

	config.DisableConnectionWatchdog = true

	go func() {
		err := StartPrimary(config)
		if err != nil {
			log.Panic("Unable to start server:", err.Error())
		}
	}()

	m.App.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyCtrlC:
			m.Stop()
		case tcell.KeyEsc:
			m.Stop()
		case tcell.KeyRight:
			m.NextPage()
		case tcell.KeyLeft:
			m.PreviousPage()
		}

		return e
	})

	sleepTime := time.Second
	justBegan := true

	go func() {
		for {
			onlineWorkersLock.Lock()
			onlineWorkers = Workers{}
			onlineWorkersLock.Unlock()

			err := broadcastMessage(Message{
				Operation:     OperationStatus,
				Token:         config.Token,
				RespondOnPort: config.InboundPort}, true)

			if err != nil {
				log.Println("Unable to broadcast status request:", err.Error())

				time.Sleep(sleepTime)
				continue
			}

			if !justBegan {
				time.Sleep(sleepTime)
			} else {
				justBegan = false
			}

			m.App.QueueUpdateDraw(func() {
				onlineWorkersLock.RLock()
				m.Render(onlineWorkers)
				onlineWorkersLock.RUnlock()
			})

		}
	}()

	err := m.App.Run()
	if err != nil {
		log.Panic("Unable to start monitor interface:", err.Error())
	}
}

// Render prints the Monitor to the console.
func (m *Monitor) Render(ws Workers) {
	// Order the workers so their position keeps regular between updates
	ws = ws.sort()

	// Generate details
	var detailBoxes []*tview.Flex
	for _, w := range ws {
		detailBoxes = append(detailBoxes, newWorkerDetailBox(w))
	}

	// Generate pages
	chunks := chunkDetails(detailBoxes, monitorMaxWorkersPerPage)
	for pageNum, chunk := range chunks {
		pageNum += 1

		pageName := fmt.Sprintf("%d", pageNum)
		content := pageContentFromChunk(chunk, pageNum, len(chunks))

		m.Pages.AddPage(pageName, content, true, false)
	}

	m.Pages.SwitchToPage(fmt.Sprintf("%d", m.CurrentPage))
	m.App.SetRoot(m.Pages, true)
}

// PreviousPage changes the page to the n+1 page.
func (m *Monitor) NextPage() {
	next := m.CurrentPage + 1
	if m.Pages.GetPageCount() < next {
		return
	}

	m.CurrentPage = next
	m.Pages.SwitchToPage(fmt.Sprintf("%d", next))
}

// PreviousPage changes the page to the n-1 page.
func (m *Monitor) PreviousPage() {
	previous := m.CurrentPage - 1
	if previous < 1 {
		return
	}

	m.CurrentPage = previous
	m.Pages.SwitchToPage(fmt.Sprintf("%d", previous))
}

// Stop stops the App and exits with code 0.
func (m *Monitor) Stop() {
	m.App.Stop()
	os.Exit(0)
}

// pageContentFromChunk creates a new detailed view box of a Worker to be rendered on the Monitor.
func pageContentFromChunk(chunk []*tview.Flex, pageNum int, totalPages int) *tview.Flex {
	content := tview.NewFlex().SetDirection(tview.FlexRow)

	content.SetBorder(true)
	content.SetTitle(" Beekeeper Monitor ") // Spaces for formatting
	content.SetTitleAlign(tview.AlignCenter)

	for _, row := range chunk {
		content.AddItem(row, 5, 5, false)
	}

	// Check if the page has missing workers
	emptySlots := (monitorMaxWorkersPerPage - len(chunk)) + 1 // Always keep an empty slot as to keep the footer down
	if emptySlots != 0 {
		for x := 0; x < emptySlots; x++ {
			content.AddItem(nil, 0, 5, false)
		}
	}

	// Prepare the text and arrows when needed
	footerText := fmt.Sprintf("Page %d/%d", pageNum, totalPages)
	if pageNum+1 <= totalPages {
		footerText += " >"
	} else {
		footerText += "  " // So it looks centered
	}

	if pageNum-1 >= 1 {
		footerText = "< " + footerText
	} else {
		footerText = "  " + footerText // So it looks centered
	}

	content.AddItem(newPrimitive(footerText), 1, 1, false)

	return content
}

// newWorkerDetailBox creates a new detailed view box of a Worker to be rendered on the Monitor.
func newWorkerDetailBox(w Worker) *tview.Flex {
	ip := tview.NewFlex()
	ip.SetTitle("IP").
		SetBorder(true).
		SetTitleAlign(tview.AlignCenter)
	ip.AddItem(newPrimitive(w.Addr.IP.String()), 0, 1, false)

	status := tview.NewFlex()
	status.SetTitle("Status").
		SetBorder(true).
		SetTitleAlign(tview.AlignCenter)
	status.AddItem(newPrimitive(w.Status.String()), 0, 1, false)

	cpuTemp := tview.NewFlex()
	cpuTemp.SetTitle("CPU Temp.").
		SetBorder(true).
		SetTitleAlign(tview.AlignCenter)
	cpuTemp.AddItem(newPrimitive(fmt.Sprintf("%d°C", int(w.Info.CPUTemp))), 0, 1, false)

	usage := tview.NewFlex()
	usage.SetTitle("Usage").
		SetBorder(true).
		SetTitleAlign(tview.AlignCenter)
	usage.AddItem(newPrimitive(fmt.Sprintf("%d%%", int(w.Info.Usage))), 0, 1, false)

	flex := tview.NewFlex()
	flex.Box.SetTitle(w.Name).SetBorder(true).SetTitleAlign(tview.AlignLeft)

	flex.AddItem(ip, 0, 1, false)
	flex.AddItem(status, 0, 1, false)
	flex.AddItem(cpuTemp, 0, 1, false)
	flex.AddItem(usage, 0, 1, false)

	return flex
}

// chunkDetails utility function to chunk a slice of details into pages.
func chunkDetails(details []*tview.Flex, perPage int) (chunks [][]*tview.Flex) {
	for perPage < len(details) {
		details, chunks = details[perPage:], append(chunks, details[0:perPage:perPage])
	}

	return append(chunks, details)
}

// newPrimitive utility function to create a centered text primitive.
func newPrimitive(text string) tview.Primitive {
	return tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(text)
}
