package game

import (
	"fmt"
	"math"
	"time"
)

const (
	// MaxGrid is the whole island; the land a player can build on grows from its centre with the town hall.
	MaxGrid     = 16
	MaxBuilders = 2
	// Base caps provided by the town hall itself, before any storage.
	BaseGoldCap   = 600
	BaseEnergyCap = 200
)

// LevelSpec describes one level of a building: the price to reach it, the time it takes and what it gives.
type LevelSpec struct {
	Level      int   `json:"level"`
	Cost       int64 `json:"cost"`        // gold
	Seconds    int   `json:"seconds"`     // demo timer
	RequiresTH int   `json:"requires_th"` // minimum town hall level
	// RequiresLevel is the player level (from work experience) a town hall level needs.
	RequiresLevel int   `json:"requires_level,omitempty"`
	HP            int   `json:"hp"`
	DPS           int   `json:"dps,omitempty"`        // tower damage per round
	Capacity      int   `json:"capacity,omitempty"`   // barracks housing
	GoldCap       int64 `json:"gold_cap,omitempty"`   // storage
	EnergyCap     int64 `json:"energy_cap,omitempty"` // storage, crystal well
	FeePct        int   `json:"fee_pct,omitempty"`    // market: the share kept on every exchange
	Builders      int   `json:"builders,omitempty"`   // builder hut
}

type BuildingDef struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// MaxCount[i] is how many of this building a town hall of level i+1 allows.
	MaxCount []int       `json:"max_count"`
	Levels   []LevelSpec `json:"levels"`
	Defense  bool        `json:"defense"`
	// Decor: looks only — instant, no builder, no hit points, never part of a defence.
	Decor bool `json:"decor,omitempty"`
}

func (d BuildingDef) MaxLevel() int { return len(d.Levels) }

// Spec returns the specification of level n (1-based) or false.
func (d BuildingDef) Spec(level int) (LevelSpec, bool) {
	if level < 1 || level > len(d.Levels) {
		return LevelSpec{}, false
	}
	return d.Levels[level-1], true
}

// Allowed is how many buildings of this type a town hall of the given level allows.
func (d BuildingDef) Allowed(thLevel int) int {
	if thLevel < 1 {
		return 0
	}
	if thLevel > len(d.MaxCount) {
		return d.MaxCount[len(d.MaxCount)-1]
	}
	return d.MaxCount[thLevel-1]
}

// BuildingOrder is the order buildings are offered in the city.
var BuildingOrder = []string{"townhall", "barracks", "storage", "wall", "tower", "tavern", "hut", "market", "well", "road", "tree", "flowers", "lantern", "pond"}

// Land is the square a player may build on: (start, size) on the MaxGrid island, centred.
// 12×12 for town halls 1–2 (the original field), 14×14 for 3, the whole island from 4.
func Land(townHall int) (start, size int) {
	switch {
	case townHall >= 4:
		size = 16
	case townHall == 3:
		size = 14
	default:
		size = 12
	}
	return (MaxGrid - size) / 2, size
}

// InLand: the cell is on the player's land.
func InLand(townHall, x, y int) bool {
	start, size := Land(townHall)
	return x >= start && y >= start && x < start+size && y < start+size
}

var Buildings = map[string]BuildingDef{
	"townhall": {
		Type: "townhall", Name: "Ратуша", Description: "Сердце города. Её уровень открывает новые здания и уровни.",
		MaxCount: []int{1},
		Levels: []LevelSpec{
			{Level: 1, Cost: 0, Seconds: 0, RequiresTH: 0, HP: 900},
			{Level: 2, Cost: 400, Seconds: 30, RequiresTH: 1, RequiresLevel: 2, HP: 1300},
			{Level: 3, Cost: 1000, Seconds: 45, RequiresTH: 2, RequiresLevel: 3, HP: 1800},
			{Level: 4, Cost: 2000, Seconds: 60, RequiresTH: 3, RequiresLevel: 5, HP: 2400},
			{Level: 5, Cost: 3500, Seconds: 90, RequiresTH: 4, RequiresLevel: 7, HP: 3100},
		},
	},
	"barracks": {
		Type: "barracks", Name: "Казарма", Description: "Готовит войска. Уровень увеличивает численность армии.",
		MaxCount: []int{1, 1, 2, 2, 3},
		Levels: []LevelSpec{
			{Level: 1, Cost: 150, Seconds: 10, RequiresTH: 1, HP: 400, Capacity: 10},
			{Level: 2, Cost: 400, Seconds: 20, RequiresTH: 2, HP: 550, Capacity: 18},
			{Level: 3, Cost: 900, Seconds: 35, RequiresTH: 3, HP: 700, Capacity: 28},
			{Level: 4, Cost: 1700, Seconds: 50, RequiresTH: 4, HP: 900, Capacity: 40},
		},
	},
	"storage": {
		Type: "storage", Name: "Хранилище", Description: "Увеличивает запас золота и энергии.",
		MaxCount: []int{1, 1, 2, 2, 3},
		Levels: []LevelSpec{
			{Level: 1, Cost: 120, Seconds: 10, RequiresTH: 1, HP: 350, GoldCap: 800, EnergyCap: 200},
			{Level: 2, Cost: 350, Seconds: 20, RequiresTH: 2, HP: 500, GoldCap: 1600, EnergyCap: 400},
			{Level: 3, Cost: 800, Seconds: 35, RequiresTH: 3, HP: 700, GoldCap: 3000, EnergyCap: 700},
			{Level: 4, Cost: 1500, Seconds: 50, RequiresTH: 4, HP: 950, GoldCap: 5000, EnergyCap: 1100},
		},
	},
	"wall": {
		Type: "wall", Name: "Стена", Description: "Сегмент укреплений: войска сначала пробивают стены.", Defense: true,
		MaxCount: []int{4, 8, 12, 16, 20},
		Levels: []LevelSpec{
			{Level: 1, Cost: 25, Seconds: 3, RequiresTH: 1, HP: 120},
			{Level: 2, Cost: 70, Seconds: 5, RequiresTH: 2, HP: 220},
			{Level: 3, Cost: 160, Seconds: 8, RequiresTH: 3, HP: 360},
			{Level: 4, Cost: 320, Seconds: 12, RequiresTH: 4, HP: 540},
			{Level: 5, Cost: 550, Seconds: 15, RequiresTH: 5, HP: 780},
		},
	},
	"tavern": {
		Type: "tavern", Name: "Таверна", Description: "Здесь бросают и принимают вызовы на дружеский бой. Уровень добавляет дружеские бои в день.",
		MaxCount: []int{1},
		Levels: []LevelSpec{
			{Level: 1, Cost: 100, Seconds: 8, RequiresTH: 1, HP: 300},
			{Level: 2, Cost: 300, Seconds: 15, RequiresTH: 2, HP: 420},
			{Level: 3, Cost: 700, Seconds: 25, RequiresTH: 3, HP: 560},
		},
	},
	// Economy: none of these earns anything by itself — gold and energy still come only from work and wave chests.
	"hut": {
		Type: "hut", Name: "Хижина строителя", Description: "Здесь живёт ещё один строитель: можно вести на одну стройку больше.",
		MaxCount: []int{0, 0, 1, 1, 2},
		Levels:   []LevelSpec{{Level: 1, Cost: 600, Seconds: 25, RequiresTH: 3, HP: 300, Builders: 1}},
	},
	"market": {
		Type: "market", Name: "Рынок", Description: "Обмен золота на энергию и обратно. Рынок берёт комиссию, уровень её снижает.",
		MaxCount: []int{0, 1},
		Levels: []LevelSpec{
			{Level: 1, Cost: 300, Seconds: 15, RequiresTH: 2, HP: 400, FeePct: 30},
			{Level: 2, Cost: 700, Seconds: 25, RequiresTH: 3, HP: 550, FeePct: 20},
			{Level: 3, Cost: 1400, Seconds: 40, RequiresTH: 4, HP: 700, FeePct: 10},
		},
	},
	"well": {
		Type: "well", Name: "Кристальный колодец", Description: "Отдельный запас энергии: хранит её, но не добывает.",
		MaxCount: []int{0, 1, 1, 2, 2},
		Levels: []LevelSpec{
			{Level: 1, Cost: 200, Seconds: 12, RequiresTH: 2, HP: 300, EnergyCap: 250},
			{Level: 2, Cost: 500, Seconds: 25, RequiresTH: 3, HP: 420, EnergyCap: 500},
			{Level: 3, Cost: 1100, Seconds: 40, RequiresTH: 4, HP: 560, EnergyCap: 900},
		},
	},
	// Decor: looks only. Built at once, no builder, no hit points, never part of a defence.
	"road":    decor("road", "Дорога", "Мостовая между зданиями. Укладывается цепочкой.", 5, []int{40, 60, 80, 100, 120}),
	"tree":    decor("tree", "Дерево", "Тень и зелень во дворе.", 15, []int{6, 10, 14, 18, 22}),
	"flowers": decor("flowers", "Клумба", "Цветы у дорожек.", 10, []int{6, 10, 14, 18, 22}),
	"lantern": decor("lantern", "Фонарь", "Свет на улицах города.", 20, []int{4, 6, 8, 10, 12}),
	"pond":    decor("pond", "Пруд", "Вода с кувшинками.", 40, []int{1, 1, 2, 2, 3}),
	"tower": {
		Type: "tower", Name: "Башня", Description: "Стреляет по атакующим каждый раунд боя.", Defense: true,
		MaxCount: []int{1, 2, 3, 4, 5},
		Levels: []LevelSpec{
			{Level: 1, Cost: 200, Seconds: 10, RequiresTH: 1, HP: 300, DPS: 14},
			{Level: 2, Cost: 500, Seconds: 20, RequiresTH: 2, HP: 430, DPS: 22},
			{Level: 3, Cost: 1000, Seconds: 30, RequiresTH: 3, HP: 600, DPS: 32},
			{Level: 4, Cost: 1800, Seconds: 45, RequiresTH: 4, HP: 800, DPS: 44},
			{Level: 5, Cost: 3000, Seconds: 60, RequiresTH: 5, HP: 1050, DPS: 58},
		},
	},
}

// BuildDuration scales a level's demo timer.
func BuildDuration(s Settings, spec LevelSpec) time.Duration {
	return time.Duration(math.Round(float64(spec.Seconds)*s.TimeScale*1000)) * time.Millisecond
}

// Building is the state the rules need; the store maps rows to it.
type Building struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Level       int        `json:"level"`
	TargetLevel *int       `json:"target_level"`
	BusyUntil   *time.Time `json:"busy_until"`
	X           int        `json:"x"`
	Y           int        `json:"y"`
}

func (b Building) Busy() bool { return b.BusyUntil != nil }

// City summarises a set of buildings.
type City struct{ Buildings []Building }

func (c City) TownHall() int {
	for _, b := range c.Buildings {
		if b.Type == "townhall" {
			return b.Level
		}
	}
	return 0
}

func (c City) Count(kind string) int {
	n := 0
	for _, b := range c.Buildings {
		if b.Type == kind {
			n++
		}
	}
	return n
}

// Builders: the two the town hall provides plus one for every finished builder hut.
func (c City) Builders() int {
	n := MaxBuilders
	for _, b := range c.Buildings {
		if spec, ok := Buildings[b.Type].Spec(b.Level); ok {
			n += spec.Builders
		}
	}
	return n
}

func (c City) BuildersBusy() int {
	n := 0
	for _, b := range c.Buildings {
		if b.Busy() {
			n++
		}
	}
	return n
}

func (c City) Occupied(x, y int) bool {
	for _, b := range c.Buildings {
		if b.X == x && b.Y == y {
			return true
		}
	}
	return false
}

// Caps are the storage limits: town hall base plus every completed storage and crystal well level.
func (c City) Caps() (gold, energy int64) {
	gold, energy = BaseGoldCap, BaseEnergyCap
	for _, b := range c.Buildings {
		if b.Type != "storage" && b.Type != "well" {
			continue
		}
		if spec, ok := Buildings[b.Type].Spec(b.Level); ok {
			gold += spec.GoldCap
			energy += spec.EnergyCap
		}
	}
	return
}

// ArmyCapacity is the housing of every completed barracks level.
func (c City) ArmyCapacity() int {
	n := 0
	for _, b := range c.Buildings {
		if b.Type != "barracks" {
			continue
		}
		if spec, ok := Buildings["barracks"].Spec(b.Level); ok {
			n += spec.Capacity
		}
	}
	return n
}

// Refusal is a rule violation with a stable code for the client and a Russian message for people.
type Refusal struct {
	Code    string
	Message string
}

func (r *Refusal) Error() string { return r.Message }

func refuse(code, message string) *Refusal { return &Refusal{Code: code, Message: message} }

// CheckPlace validates placing a new level-1 building. gold is the current balance.
func CheckPlace(c City, kind string, x, y int, gold int64) (LevelSpec, *Refusal) {
	def, ok := Buildings[kind]
	if !ok || kind == "townhall" {
		return LevelSpec{}, refuse("invalid_building", "Такое здание построить нельзя.")
	}
	if !InLand(c.TownHall(), x, y) {
		return LevelSpec{}, refuse("invalid_cell", "Клетка за пределами вашей земли. Земля растёт с уровнем ратуши.")
	}
	if c.Occupied(x, y) {
		return LevelSpec{}, refuse("cell_occupied", "Клетка уже занята.")
	}
	spec, _ := def.Spec(1)
	th := c.TownHall()
	if th < spec.RequiresTH {
		return LevelSpec{}, refuse("townhall_level", "Нужен более высокий уровень ратуши.")
	}
	if c.Count(kind) >= def.Allowed(th) {
		return LevelSpec{}, refuse("building_limit", "Достигнут лимит таких зданий для текущего уровня ратуши.")
	}
	if !def.Decor && c.BuildersBusy() >= c.Builders() {
		return LevelSpec{}, refuse("builders_busy", "Все строители заняты. Дождитесь окончания стройки.")
	}
	if gold < spec.Cost {
		return LevelSpec{}, refuse("insufficient_gold", "Недостаточно золота.")
	}
	return spec, nil
}

// CheckUpgrade validates upgrading building b to the next level. playerLevel comes from work experience.
func CheckUpgrade(c City, b Building, gold int64, playerLevel int) (LevelSpec, *Refusal) {
	def := Buildings[b.Type]
	if b.Busy() {
		return LevelSpec{}, refuse("building_busy", "Здание уже строится.")
	}
	spec, ok := def.Spec(b.Level + 1)
	if !ok {
		return LevelSpec{}, refuse("max_level", "Здание уже максимального уровня.")
	}
	th := c.TownHall()
	if b.Type != "townhall" && spec.RequiresTH > th {
		return LevelSpec{}, refuse("townhall_level", "Сначала улучшите ратушу.")
	}
	if playerLevel < spec.RequiresLevel {
		return LevelSpec{}, refuse("player_level", fmt.Sprintf("Нужен уровень игрока %d. Опыт дают подтверждённые задачи.", spec.RequiresLevel))
	}
	if c.BuildersBusy() >= c.Builders() {
		return LevelSpec{}, refuse("builders_busy", "Все строители заняты. Дождитесь окончания стройки.")
	}
	if gold < spec.Cost {
		return LevelSpec{}, refuse("insufficient_gold", "Недостаточно золота.")
	}
	return spec, nil
}

// CheckMove validates moving building b to (x, y): free and instant, anywhere on the player's land.
func CheckMove(c City, b Building, x, y int) *Refusal {
	if b.X == x && b.Y == y {
		return refuse("same_cell", "Здание уже стоит здесь.")
	}
	if !InLand(c.TownHall(), x, y) {
		return refuse("invalid_cell", "Клетка за пределами вашей земли. Земля растёт с уровнем ратуши.")
	}
	if c.Occupied(x, y) {
		return refuse("cell_occupied", "Клетка уже занята.")
	}
	return nil
}

func decor(kind, name, description string, cost int64, maxCount []int) BuildingDef {
	return BuildingDef{Type: kind, Name: name, Description: description, Decor: true, MaxCount: maxCount,
		Levels: []LevelSpec{{Level: 1, Cost: cost, Seconds: 0, RequiresTH: 1, HP: 0}}}
}

// Find returns the building with the given id.
func (c City) Find(id string) (Building, bool) {
	for _, b := range c.Buildings {
		if b.ID == id {
			return b, true
		}
	}
	return Building{}, false
}
