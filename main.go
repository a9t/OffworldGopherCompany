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
	worldX = 160
	worldY = 50
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

	world := GenerateWorld(worldY, worldX)
	g.SetManagerFunc(generateLayout(world, worldViewX, worldViewY))

	if err := initKeybindings(g, world); err != nil {
		log.Fatalln(err)
	}

	ch := make(chan coord)
	posChan = &ch
	go tileUpdater(g, posChan)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalln(err)
	}
}

func generateLayout(world [][]int, worldX int, worldY int) func (g *gocui.Gui) error{
	canDisplay := false

	maxWorldWindowX := worldX + 2
	maxWorldWindowY := worldY + 2

	var lastY int 

	return func(g *gocui.Gui) error {
		maxX, maxY := g.Size()

		if maxX < mainViewMinX || maxY < mainViewMinY {
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
			fmt.Fprintln(v, "Type    : 0-0")
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

			printWorld(v, world, 0, 0)
			lastY = 0
		} else {
			// on fast resizing, if the cursur happens to be on the last line,
			// this triggers a panic; even this fix does not completely remove
			// the issue, but it is better
			if lastY > maxY {
				_, yc := v.Cursor()
				if yc == maxY - 2 {
					v.SetCursor(0, 0)
				}
			}
			lastY = maxY
		}

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

func initKeybindings(g *gocui.Gui, world [][]int) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			return gocui.ErrQuit
		}); err != nil {
		return err
	}

	xOffset := 0
	yOffset := 0
	if err := g.SetKeybinding("Map", gocui.KeyArrowDown, gocui.ModNone,
		moveCursor(world, 0, 1, &xOffset, &yOffset)); err != nil {
		return err
	}
	if err := g.SetKeybinding("Map", gocui.KeyArrowUp, gocui.ModNone,
		moveCursor(world, 0, -1, &xOffset, &yOffset)); err != nil {
		return err
	}
	if err := g.SetKeybinding("Map", gocui.KeyArrowLeft, gocui.ModNone,
		moveCursor(world, -1, 0, &xOffset, &yOffset)); err != nil {
		return err
	}
	if err := g.SetKeybinding("Map", gocui.KeyArrowRight, gocui.ModNone,
		moveCursor(world, 1, 0, &xOffset, &yOffset)); err != nil {
		return err
	}
	if err := g.SetKeybinding("Map", gocui.MouseLeft, gocui.ModNone,
		moveCursor(world, 0, 0, &xOffset, &yOffset)); err != nil {
		return err
	}
	if err := g.SetKeybinding("Map", gocui.MouseRelease, gocui.ModNone,
		moveCursor(world, 0, 0, &xOffset, &yOffset)); err != nil {
		return err
	}

	return nil
}

func moveCursor(world [][]int, dx, dy int, xOffset *int, yOffset *int) func(g *gocui.Gui, v *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		xc, yc := v.Cursor()
		maxX, maxY := v.Size()

		worldY := len(world)
		worldX := len(world[0])

		newX := xc + dx
		newY := yc + dy

		if newX < 0 {
			if newX + *xOffset >= 0 {
				*xOffset = newX + *xOffset
				newX = 0
			} else {
				*xOffset = 0
				newX = 0
			}
		} else if newX >= maxX {
			if newX + *xOffset < worldX {
				*xOffset = *xOffset + newX - maxX + 1
				newX = maxX - 1
			} else {
				*xOffset = worldX - maxX
				newX = maxX - 1
			}
		}

		if newY < 0 {
			if newY + *yOffset >= 0 {
				*yOffset = newY + *yOffset
				newY = 0
			} else {
				*yOffset = 0
				newY = 0
			}
		} else if newY >= maxY {
			if newY + *yOffset < worldY {
				*yOffset = *yOffset + newY - maxY + 1
				newY = maxY - 1
			} else {
				*yOffset = worldY - maxY
				newY = maxY - 1
			}
		}

		v.Clear()
		v.SetCursor(newX, newY)
		*posChan <- coord{newX + *xOffset, newY + *yOffset}

		printWorld(v, world, *yOffset, *xOffset)
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

func printWorld(v *gocui.View, world [][]int, offsetY, offsetX int) {
	width, height := v.Size()
	width -= 2
	height -= 2

	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			if world[i + offsetY][j + offsetX] == 0 {
				fmt.Fprint(v, " ")
			} else {
				fmt.Fprintf(v, "%d", world[i + offsetY][j + offsetX])
			}
		}
		fmt.Fprintln(v, "")
	}
}
