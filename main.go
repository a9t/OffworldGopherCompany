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

	rightViewWidth = 30
	notificationViewHeight = 5
	marketViewHeight = 5
	infoViewHeight = 6
)

type coord struct {
	x int
	y int
}

var posChan *chan coord

func main() {
	g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		log.Fatalln(err)
	}
	defer g.Close()

	g.Mouse = true
	g.Cursor = true

	game := GenerateGame(worldY, worldX, 1)
	player := game.registerPlayer("Player1")

	g.SetManagerFunc(generateLayout(game.WorldMap, worldViewX, worldViewY))

	if err := initKeybindings(g, player, game.WorldMap); err != nil {
		log.Fatalln(err)
	}

	ch := make(chan coord)
	posChan = &ch
	go tileUpdater(g, game, posChan)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalln(err)
	}
}

func generateLayout(world [][]*TileInfo, worldX int, worldY int) func (g *gocui.Gui) error{
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

		worldWindowX := maxX - rightViewWidth
		if worldWindowX > maxWorldWindowX {
			worldWindowX = maxWorldWindowX
		}
		worldWindowY := maxY - notificationViewHeight
		if worldWindowY > maxWorldWindowY {
			worldWindowY = maxWorldWindowY
		}

		if v, err := g.SetView("Market", worldWindowX, 0, worldWindowX+rightViewWidth-1, marketViewHeight-1); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Title = "Market"
			fmt.Fprintln(v, "Metal :   A    1 V")
			fmt.Fprintln(v, "Water :   A    4 V")
			fmt.Fprintln(v, "Carbon:   A  105 V")
		}

		if v, err := g.SetView("TileInfo", worldWindowX, marketViewHeight, worldWindowX+rightViewWidth-1, marketViewHeight+infoViewHeight-1); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Title = "Tile Info"
			fmt.Fprintln(v, "Coord   : 0-0")
			fmt.Fprintln(v, "Type    : -")
			fmt.Fprintln(v, "Quantity: -")
			fmt.Fprintln(v, "Owner   : -")
		}

		if v, err := g.SetView("Actions", worldWindowX, marketViewHeight+infoViewHeight, worldWindowX+rightViewWidth-1, worldWindowY+notificationViewHeight-1); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Title = "Actions"
			fmt.Fprintln(v, "c - claim")
			fmt.Fprintln(v, "b - build")
		}

		v, err := g.SetView("Map", 0, 0, worldWindowX-1, worldWindowY-1); if err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			if _, err := g.SetCurrentView("Map"); err != nil {
				return err
			}
			v.Title = "World"
			v.SetCursor(0, 0)

			printWorld(v, world, 0, 0)
			lastY = 0
		} else {
			// on fast resizing, if the cursor happens to be on the last line,
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

		if v, err := g.SetView("Notifications", 0, worldWindowY, worldWindowX-1, worldWindowY+notificationViewHeight-1); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Title = "Notifications"
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

func initKeybindings(g *gocui.Gui, player *Player, world [][]*TileInfo) error {
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

	if err := g.SetKeybinding("Map", 'c', gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			xc, yc := v.Cursor()
			player.claim(xc + xOffset, yc + yOffset)
			moveCursor(world, 0, 0, &xOffset, &yOffset)(g, v)
			return nil
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("Map", 'e', gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			xc, yc := v.Cursor()
			player.buildExtractor(xc + xOffset, yc + yOffset)
			moveCursor(world, 0, 0, &xOffset, &yOffset)(g, v)
			return nil
		}); err != nil {
		return err
	}

	return nil
}

func moveCursor(world [][]*TileInfo, dx, dy int, xOffset *int, yOffset *int) func(g *gocui.Gui, v *gocui.View) error {
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

func tileUpdater(g *gocui.Gui, game *Game, c *chan coord) {
	for {
		cursor := <- *c
		if v, err := g.View("TileInfo"); err == nil {
			v.Clear()

			tileInfo := game.WorldMap[cursor.y][cursor.x]

			var typeString string
			if tileInfo == nil {
				typeString = "empty"
			} else {
				switch tileInfo.TileType {
				case TileEmpty: typeString = "empty"
				case TileMetal: typeString = "metal"
				case TileWater: typeString = "water"
				case TileCarbon: typeString = "carbon"
				default: typeString =  "unknown"
				}
			}

			var quantity int
			if tileInfo == nil {
				quantity = 0
			} else {
				quantity = tileInfo.Quantity
			}

			var name string
			if tileInfo == nil || tileInfo.player == nil {
				name = "-"
			} else {
				name = *(tileInfo.player.name)
			}

			fmt.Fprintf(v, "Coord   : %d-%d\n", cursor.x, cursor.y)
			fmt.Fprintf(v, "Type    : %s\n", typeString)
			fmt.Fprintf(v, "Quantity: %d\n", quantity)
			fmt.Fprintf(v, "Owner   : %s\n", name)
		}
	}
}

func getTileString(tileInfo *TileInfo) string {
	if tileInfo == nil {
		return " "
	}

	var ownerColor int
	if tileInfo.player == nil {
		ownerColor = 0
	} else {
		ownerColor = 4
	}

	var pattern = ""
	switch tileInfo.TileType {
	case TileEmpty: pattern = "\033[38;5;%dm\033[48;5;7m%d\033[0m"
	case TileMetal: pattern = "\033[38;5;%dm\033[48;5;1m%d\033[0m"
	case TileWater: pattern = "\033[38;5;%dm\033[48;5;2m%d\033[0m"
	case TileCarbon: pattern = "\033[38;5;%dm\033[48;5;3m%d\033[0m"
	default: return "\033[38;5;0m\033[48;5;3m?\033[0m"
	}

	return fmt.Sprintf(pattern, ownerColor, tileInfo.Quantity)
}

func printWorld(v *gocui.View, world [][]*TileInfo, offsetY, offsetX int) {
	width, height := v.Size()

	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			fmt.Fprint(v, getTileString(world[i + offsetY][j + offsetX]))
		}
		fmt.Fprintln(v, "")
	}
}
