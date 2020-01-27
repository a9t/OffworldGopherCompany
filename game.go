package main

import (
	"math/rand"
	"time"
)

// TileType identifies the type of the tile on the map
type TileType = int

const (
	// TileEmpty identifies an empty tile
	TileEmpty TileType = iota
	// TileMetal identifies a tile with metal
	TileMetal
	// TileWater identifies a tile with water
	TileWater
	// TileCarbon identifies a tile with carbon
	TileCarbon

	// TileHQ identifies a tile the HQ of a player
	TileHQ
	// TileWind identifies a tile with a wind farm
	TileWind
	// TileElectro identifies a tile with an electrolysis center
	TileElectro
	// TileChem identifies a tile with a chemical plant
	TileChem
)

// TileInfo data about the map tile
type TileInfo struct {
	TileType TileType
	Quantity int
	Level int
	player *Player
}

func (p *Player) updateResources(tile *TileInfo) {
	var percentage = 0.0
	switch tile.Level {
	case 1: percentage = 0.4
	case 2: percentage = 0.65
	case 3: percentage = 1.
	default: return
	}

	switch tile.TileType {
	case TileMetal: p.metal += float64(tile.Quantity) * percentage
	case TileWater: p.water += float64(tile.Quantity) * percentage
	case TileCarbon: p.carbon += float64(tile.Quantity) * percentage
	}
}

// Player data about the player
type Player struct {
	name *string
	tiles []*TileInfo
	game *Game
	notification chan struct{}

	funds float64
	debt float64
	carbon float64
	water float64
	metal float64
	energy float64
	chemicals float64
	fuel float64
	oxygen float64
}

// Game struct
type Game struct {
	WorldMap [][]*TileInfo
	players []*Player
}

func (game *Game) run(done *chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-*done:
			return
		case <-ticker.C:
			for _, p := range game.players {
				for _, t := range p.tiles {
					p.updateResources(t)
				}
				p.notification <- struct{}{}
			}
		}
	}
}

func isWithinLimits(worldMap [][]*TileInfo, x, y int) bool {
	return x >= 0 && y >= 0 && y < len(worldMap) && x < len(worldMap[0])
}

func (p *Player) claim(x, y int) int {
	if !isWithinLimits(p.game.WorldMap, x, y) {
		return -1
	}

	tile := p.game.WorldMap[y][x]
	if tile == nil {
		tile = new(TileInfo)
		tile.TileType = TileEmpty
		tile.Quantity = 0
		tile.player = p

		p.game.WorldMap[y][x] = tile
	} else if tile.player == nil {
		tile.player = p
	} else {
		return -1
	}

	p.tiles = append(p.tiles, tile)

	return 0
}

func (p *Player) buildExtractor(x, y int) int {
	if !isWithinLimits(p.game.WorldMap, x, y) {
		return -1
	}

	tile := p.game.WorldMap[y][x]
	if tile == nil {
		return -1
	} else if tile.player != p {
		return -1
	} else if tile.Level != 0 {
		return -1
	}

	if tile.TileType == TileWater || tile.TileType == TileMetal || 	tile.TileType == TileCarbon {
		tile.Level++
		return 0
	}

	return -1
}

func (p *Player) buildWindTurbine(x, y int) int {
	if !isWithinLimits(p.game.WorldMap, x, y) {
		return -1
	}

	tile := p.game.WorldMap[y][x]
	if tile == nil {
		return -1
	} else if tile.player != p {
		return -1
	} else if tile.TileType != TileEmpty {
		return -1
	}

	tile.TileType = TileWind
	tile.Level++

	return 0
}

func (p *Player) buildElectrolysisCenter(x, y int) int {
	if !isWithinLimits(p.game.WorldMap, x, y) {
		return -1
	}

	tile := p.game.WorldMap[y][x]
	if tile == nil {
		return -1
	} else if tile.player != p {
		return -1
	} else if tile.TileType != TileEmpty {
		return -1
	}

	tile.TileType = TileElectro
	tile.Level++

	return 0
}

func (p *Player) buildChemicalPlant(x, y int) int {
	if !isWithinLimits(p.game.WorldMap, x, y) {
		return -1
	}

	tile := p.game.WorldMap[y][x]
	if tile == nil {
		return -1
	} else if tile.player != p {
		return -1
	} else if tile.TileType != TileEmpty {
		return -1
	}

	tile.TileType = TileChem
	tile.Level++

	return 0
}

func (p *Player) upgrade(x, y int) int {
	if !isWithinLimits(p.game.WorldMap, x, y) {
		return -1
	}

	tile := p.game.WorldMap[y][x]
	if tile == nil {
		return -1
	} else if tile.player != p {
		return -1
	} else if tile.Level < 1 || tile.Level == 3 {
		return -1
	}

	tile.Level++
	return 0
}

func (p *Player) destroy(x, y int) int {
	if !isWithinLimits(p.game.WorldMap, x, y) {
		return -1
	}

	tile := p.game.WorldMap[y][x]
	if tile == nil {
		return -1
	} else if tile.player != p {
		return -1
	} else if tile.Level == 0 {
		return -1
	}

	if tile.TileType != TileWater && tile.TileType != TileMetal && tile.TileType != TileCarbon {
		tile.TileType = TileEmpty
	}

	tile.Level = 0
	return 0
}

func (game *Game) registerPlayer(name string) *Player {
	for _, player := range game.players {
		if player.name == nil {
			player.name = &name
			return player
		}
	}
	return nil
}

func addResource(worldMap [][]*TileInfo, tileType TileType, maxResources int, maxPerTile int) {
	lines, cols := len(worldMap), len(worldMap[0])
	size := lines * cols

	resource := 0
	clusterProb := 1.
	clusterProbDecay := 0.6

	var cluster []int
	for resource < maxResources {
		var line, col, pos int
		newCluster := rand.Float64() < clusterProb

		if newCluster {
			cluster = make([]int, 1)
			pos = rand.Int() % size

			line = pos / cols
			col = pos % cols

			if worldMap[line][col] != nil {
				continue
			}
			clusterProb *= clusterProbDecay
		} else {
			pos = cluster[rand.Int() % len(cluster)]
			line = pos / cols
			col = pos % cols

			switch rand.Int() % 4 {
			case 0: line++; pos += cols
			case 1: line--; pos -= cols
			case 2: col++; pos++
			case 3: col--; pos--
			}

			if line >= lines || line < 0 || col >= cols || col < 0 {
				continue
			}

			if worldMap[line][col] != nil {
				continue
			}
		}

		cluster = append(cluster, pos)

		tileResource := rand.Int() % maxPerTile + 1
		worldMap[line][col] = new(TileInfo)
		worldMap[line][col].TileType = tileType
		worldMap[line][col].Quantity = tileResource

		resource += tileResource
	}
}

func addResources(worldMap [][]*TileInfo) {
	for _, resource := range []TileType{TileMetal, TileWater, TileCarbon} {
		addResource(worldMap, resource, 15, 4)
	}
}


// GenerateGame creates a new game instance
func GenerateGame(lines int, cols int, playerCount int) *Game {
	rand.Seed(time.Now().UTC().UnixNano())

	if lines <= 0 || cols <= 0  {
		return nil
	}

	if playerCount < 1 {
		return nil
	}

	worldMap := make([][]*TileInfo, lines)
	for i := 0; i < lines; i++ {
		worldMap[i] = make([]*TileInfo, cols)
	}
	addResources(worldMap)

	players := make([]*Player, playerCount)
	game := Game{worldMap, players}
	for i := 0; i < playerCount; i++ {
		players[i] = new(Player)
		players[i].tiles = make([]*TileInfo, 0)
		players[i].game = &game
		players[i].notification = make(chan struct{})

		players[i].funds = 0.
		players[i].debt = 0.
		players[i].water = 0.
		players[i].carbon = 0.
		players[i].metal = 0.
		players[i].energy = 0.
		players[i].fuel = 0.
		players[i].oxygen = 0.
		players[i].chemicals = 0.
	}

	return &game
}
