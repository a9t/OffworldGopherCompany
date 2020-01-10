package main

import (
	"fmt"
	"log"

	"github.com/jroimartin/gocui"
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

	g.SetManagerFunc(layout)

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

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if maxX < 40 || maxY < 20 {
		return nil
	}

	if v, err := g.SetView("Market", maxX-23, 0, maxX-1, 4); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Market"
		fmt.Fprintln(v, "Metal :   A    1 V")
		fmt.Fprintln(v, "Water :   A    4 V")
		fmt.Fprintln(v, "Carbon:   A  105 V")
	}

	if v, err := g.SetView("TileInfo", maxX-23, 5, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Tile Info"
		fmt.Fprintln(v, "Type    : plain")
		fmt.Fprintln(v, "Quantity: -")
		fmt.Fprintln(v, "Owned   : -")
	}

	if v, err := g.SetView("Map", 0, 0, maxX-24, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		if _, err := g.SetCurrentView("Map"); err != nil {
			return err
		}
		v.Title = "Map"
		v.SetCursor(1, 1)
		*posChan <- coord{1, 1}
		v.Size()
	}

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