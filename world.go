package main

import (
	"fmt"
	"math/rand"
	"time"
)

func showWorld(world [][]int) {

		for _, row := range world {
				for _, cell := range row {
						fmt.Printf("%d ", cell)
				}
				fmt.Println("")
		}

}

func addResouce(world [][]int, minNodes int, maxResources int, maxNodes int, maxPerNode int) {
		lines, cols := len(world), len(world[0])
		size := lines * cols
		assigned := make(map[int]int)

		rand.Seed(time.Now().UnixNano())

		resources := 0
		nodes := 0
		for nodes < minNodes {
				pos := rand.Intn(size)
				if _, ok := assigned[pos]; !ok {
						assigned[pos] = 2
						resources += 2
						nodes++
				} else {
						if assigned[pos] < maxPerNode {
								assigned[pos]++
								resources++
						}
				}
		}

		extraNodes := rand.Intn(maxNodes - minNodes)
		for i := 0; i < extraNodes; i++ {
				pos := rand.Intn(size)
				if _, ok := assigned[pos]; !ok {
						assigned[pos] = 1
						resources++
						nodes++
						resources++

						if resources > maxResources {
								break
						}
				}
		}

		for pos, value := range assigned {
				line := pos / cols
				col := pos % cols
				world[line][col] = value
		}
}

// GenerateWorld creates a lines x columns matrix with randomly
// distributed resources
func GenerateWorld(lines, columns int) [][]int {
	world := make([][]int, lines)
	for i := range world {
		world[i] = make([]int, columns)
	}

	addResouce(world, 5, 15, 10, 4)
	return world
}
