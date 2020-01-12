package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/jroimartin/gocui"
)

const (
	mainViewMinX = 60
	mainViewMinY = 20
	worldViewX = 120
	worldViewY = 40
)

type coord struct {
	x int
	y int
}

var posChan *chan coord

func main() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Fatalln(err)
	}
	defer g.Close()

	g.Mouse = true
	g.Cursor = true

	g.SetManagerFunc(generateLayout(worldViewX, worldViewY))

	viewPos := coord{0, 0}
	if err := initKeybindings(g, &viewPos); err != nil {
		log.Fatalln(err)
	}

	ch := make(chan coord)
	posChan = &ch
	go tileUpdater(g, posChan)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalln(err)
	}
}

func generateLayout(worldX int, worldY int) func (g *gocui.Gui) error{
	canDisplay := false
	//topLeftX := 0
	//topLeftY := 0

	maxWorldWindowX := worldX + 2
	maxWorldWindowY := worldY + 2

	return func(g *gocui.Gui) error {
		maxX, maxY := g.Size()

		if maxX < mainViewMinX || maxY < mainViewMinY {
			if canDisplay {
				v, err := g.View("Map"); if err == nil {
					v.SetCursor(0, 0)
				}
			}
			canDisplay = false
			return errLayout(g)
		}

		if !canDisplay {
			g.DeleteView("Error")
			canDisplay = true
		}

		sidePanelX := maxX - 23
		if sidePanelX > maxWorldWindowX {
			sidePanelX = maxWorldWindowX
		}
		windowY := maxWorldWindowY
		if windowY > maxY {
			windowY = maxY
		}

		if v, err := g.SetView("Market", sidePanelX, 0, sidePanelX+22, 4); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Title = "Market"
			fmt.Fprintln(v, "Metal :   A    1 V")
			fmt.Fprintln(v, "Water :   A    4 V")
			fmt.Fprintln(v, "Carbon:   A  105 V")
		}

		if v, err := g.SetView("TileInfo", sidePanelX, 5, sidePanelX+22, windowY-1); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Title = "Tile Info"
			fmt.Fprintln(v, "Type    : plain")
			fmt.Fprintln(v, "Quantity: -")
			fmt.Fprintln(v, "Owned   : -")
		}

		v, err := g.SetView("Map", 0, 0, sidePanelX-1, windowY-1); if err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			if _, err := g.SetCurrentView("Map"); err != nil {
				return err
			}
			v.Title = "Map"
			v.SetCursor(0, 0)
		}

		xc, yc := v.Cursor()
		*posChan <- coord{xc, yc}

		return nil
	}
}

func errLayout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	
	v, err := g.SetView("Error", -1, -1, maxX, maxY); if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Error"
	} 

	v.Clear()
	fmt.Fprint(v, strings.Repeat("\n", (maxY - 1) / 2))

	indent := (maxX - 1) / 2 - 16; if indent > 0 {
		fmt.Fprint(v, strings.Repeat(" ", indent))
	}
	fmt.Fprint(v, "Window too small, please resize!")

	return nil
}

func initKeybindings(g *gocui.Gui, viewPos *coord) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			return gocui.ErrQuit
		}); err != nil {
		return err
	}
	if err := g.SetKeybinding("Map", gocui.KeyArrowDown, gocui.ModNone,
		moveCursor(0, 1)); err != nil {
		return err
	}
	if err := g.SetKeybinding("Map", gocui.KeyArrowUp, gocui.ModNone,
		moveCursor(0, -1)); err != nil {
		return err
	}
	if err := g.SetKeybinding("Map", gocui.KeyArrowLeft, gocui.ModNone,
		moveCursor(-1, 0)); err != nil {
		return err
	}
	if err := g.SetKeybinding("Map", gocui.KeyArrowRight, gocui.ModNone,
		moveCursor(1, 0)); err != nil {
		return err
	}
	if err := g.SetKeybinding("Map", gocui.MouseLeft, gocui.ModNone,
		moveCursor(0, 0)); err != nil {
		return err
	}
	if err := g.SetKeybinding("Map", gocui.MouseRelease, gocui.ModNone,
		moveCursor(0, 0)); err != nil {
		return err
	}

	return nil
}

func moveCursor(dx, dy int) func(g *gocui.Gui, v *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		xc, yc := v.Cursor()
		maxX, maxY := v.Size()

		newX := xc + dx
		newY := yc + dy

		if newX >= 0 && newX < maxX && newY >= 0 && newY < maxY {
			v.SetCursor(newX, newY)
			*posChan <- coord{newX, newY}
		}

		return nil
	}
}

func tileUpdater(g *gocui.Gui, c *chan coord) {
	for {
		cursor := <- *c
		if v, err := g.View("TileInfo"); err == nil {
			v.Clear()
			fmt.Fprintf(v, "Type    : %d-%d\n", cursor.x, cursor.y)
			fmt.Fprintln(v, "Quantity: -")
			fmt.Fprintln(v, "Owned   : -")
		}
	}
}