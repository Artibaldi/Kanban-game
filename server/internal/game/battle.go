package game

import (
	"math"
	"math/rand/v2"
	"sort"
	"time"
)

type TroopDef struct {
	Type      string  `json:"type"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	HP        int     `json:"hp"`
	Attack    int     `json:"attack"`
	SiegeMult float64 `json:"siege_mult"` // damage multiplier against walls and towers
	Housing   int     `json:"housing"`
	Cost      int64   `json:"cost"`    // energy per unit
	Seconds   int     `json:"seconds"` // training time per unit
}

// Three original troop types.
var Troops = map[string]TroopDef{
	"guard":  {Type: "guard", Name: "Страж", Role: "Сбалансированный боец", HP: 100, Attack: 20, SiegeMult: 0.8, Housing: 1, Cost: 8, Seconds: 2},
	"raider": {Type: "raider", Name: "Налётчик", Role: "Сильная атака, слабая защита", HP: 45, Attack: 42, SiegeMult: 0.35, Housing: 1, Cost: 10, Seconds: 3},
	"ram":    {Type: "ram", Name: "Таран", Role: "Разрушает укрепления", HP: 150, Attack: 10, SiegeMult: 6, Housing: 2, Cost: 18, Seconds: 6},
}

// TroopOrder is the display and battle order.
var TroopOrder = []string{"guard", "raider", "ram"}

func TrainDuration(s Settings, t TroopDef, count int) time.Duration {
	return time.Duration(math.Round(float64(t.Seconds*count)*s.TimeScale*1000)) * time.Millisecond
}

// Housing of an army.
func Housing(army map[string]int) int {
	n := 0
	for kind, count := range army {
		n += Troops[kind].Housing * count
	}
	return n
}

// Defense is the defender's city as a battle sees it.
type Defense struct {
	WallHP     int `json:"wall_hp"`
	TowerHP    int `json:"tower_hp"`
	TowerDPS   int `json:"tower_dps"`
	Towers     int `json:"towers"`
	Walls      int `json:"walls"`
	TownHallHP int `json:"townhall_hp"`
	OtherHP    int `json:"other_hp"` // barracks and storages
	TownHall   int `json:"townhall_level"`
}

// DefenseOf builds the defense from completed levels; a building under construction defends at its current level.
func DefenseOf(c City) Defense {
	d := Defense{TownHall: c.TownHall()}
	for _, b := range c.Buildings {
		spec, ok := Buildings[b.Type].Spec(b.Level)
		if !ok || Buildings[b.Type].Decor {
			continue
		}
		switch b.Type {
		case "wall":
			d.WallHP += spec.HP
			d.Walls++
		case "tower":
			d.TowerHP += spec.HP
			d.TowerDPS += spec.DPS
			d.Towers++
		case "townhall":
			d.TownHallHP += spec.HP
		default:
			d.OtherHP += spec.HP
		}
	}
	return d
}

type Round struct {
	N           int            `json:"n"`
	TowerDamage int            `json:"tower_damage"`
	Killed      map[string]int `json:"killed"`
	Alive       map[string]int `json:"alive"`
	WallDamage  int            `json:"wall_damage"`
	TowerHit    int            `json:"tower_hit"`
	BaseDamage  int            `json:"base_damage"`
	WallHP      int            `json:"wall_hp"`
	TowerHP     int            `json:"tower_hp"`
	TownHallHP  int            `json:"townhall_hp"`
	OtherHP     int            `json:"other_hp"`
	Events      []string       `json:"events"`
}

type Outcome struct {
	Rounds       []Round        `json:"rounds"`
	Losses       map[string]int `json:"losses"`
	Survivors    map[string]int `json:"survivors"`
	Destruction  int            `json:"destruction"` // percent of towers + buildings (walls are not counted)
	Stars        int            `json:"stars"`
	TownHallDown bool           `json:"townhall_down"`
}

const MaxRounds = 6

// Fight resolves an asynchronous battle. The same seed and inputs always give the same result, so a stored
// battle can be re-checked. Randomness is limited to ±spread on each damage roll.
//
// Each round: towers fire first (damage spread over troops by their share of hit points), then the survivors
// strike. Troops must break through walls before they reach towers and buildings; rams deal siege_mult× damage
// to walls and towers, raiders are weak against fortifications. Destruction counts towers and buildings.
func Fight(army map[string]int, d Defense, seed int64, spread float64) Outcome {
	rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9e3779b97f4a7c15))
	roll := func() float64 { return 1 + (rng.Float64()*2-1)*spread }
	alive := map[string]int{}
	for _, k := range TroopOrder {
		alive[k] = army[k]
	}
	// Damage carried against a partly wounded unit of each type.
	wound := map[string]float64{}
	wall, tower, th, other := float64(d.WallHP), float64(d.TowerHP), float64(d.TownHallHP), float64(d.OtherHP)
	totalTarget := float64(d.TowerHP + d.TownHallHP + d.OtherHP)
	out := Outcome{Losses: map[string]int{}, Survivors: map[string]int{}}
	towerDPSNow := func() float64 {
		if d.TowerHP == 0 {
			return 0
		}
		return float64(d.TowerDPS) * tower / float64(d.TowerHP) // wounded towers shoot less
	}
	count := func() int {
		n := 0
		for _, k := range TroopOrder {
			n += alive[k]
		}
		return n
	}
	for r := 1; r <= MaxRounds && count() > 0 && tower+th+other > 0; r++ {
		round := Round{N: r, Killed: map[string]int{}, Alive: map[string]int{}}
		// 1. Towers fire. Three volleys per round.
		incoming := towerDPSNow() * 3 * roll()
		round.TowerDamage = int(math.Round(incoming))
		if incoming > 0 {
			var pool float64
			for _, k := range TroopOrder {
				pool += float64(alive[k] * Troops[k].HP)
			}
			for _, k := range TroopOrder {
				if alive[k] == 0 || pool == 0 {
					continue
				}
				share := incoming * float64(alive[k]*Troops[k].HP) / pool
				wound[k] += share
				dead := int(wound[k] / float64(Troops[k].HP))
				if dead > alive[k] {
					dead = alive[k]
				}
				wound[k] -= float64(dead * Troops[k].HP)
				alive[k] -= dead
				if alive[k] == 0 {
					wound[k] = 0
				}
				round.Killed[k] = dead
				out.Losses[k] += dead
			}
		}
		// 2. Survivors strike: walls first, then towers, then the town hall and other buildings.
		for _, k := range TroopOrder {
			if alive[k] == 0 {
				continue
			}
			t := Troops[k]
			hit := float64(alive[k]*t.Attack) * roll()
			// Walls.
			if wall > 0 {
				dmg := hit * t.SiegeMult
				used := math.Min(dmg, wall)
				wall -= used
				round.WallDamage += int(math.Round(used))
				hit = (dmg - used) / t.SiegeMult
				if wall <= 0 {
					round.Events = append(round.Events, "wall_breached")
				}
			}
			if hit <= 0 {
				continue
			}
			// Towers.
			if tower > 0 {
				dmg := hit * t.SiegeMult
				used := math.Min(dmg, tower)
				tower -= used
				round.TowerHit += int(math.Round(used))
				hit = (dmg - used) / t.SiegeMult
				if tower <= 0 {
					round.Events = append(round.Events, "towers_down")
				}
			}
			if hit <= 0 {
				continue
			}
			// Town hall, then other buildings; plain damage.
			if th > 0 {
				used := math.Min(hit, th)
				th -= used
				round.BaseDamage += int(math.Round(used))
				hit -= used
				if th <= 0 {
					round.Events = append(round.Events, "townhall_down")
				}
			}
			if hit > 0 && other > 0 {
				used := math.Min(hit, other)
				other -= used
				round.BaseDamage += int(math.Round(used))
			}
		}
		for _, k := range TroopOrder {
			round.Alive[k] = alive[k]
		}
		round.WallHP, round.TowerHP, round.TownHallHP, round.OtherHP = int(math.Round(wall)), int(math.Round(tower)), int(math.Round(th)), int(math.Round(other))
		if round.Events == nil {
			round.Events = []string{}
		}
		out.Rounds = append(out.Rounds, round)
	}
	for _, k := range TroopOrder {
		out.Survivors[k] = alive[k]
		if _, ok := out.Losses[k]; !ok {
			out.Losses[k] = 0
		}
	}
	if totalTarget > 0 {
		out.Destruction = int(math.Floor((totalTarget - (tower + th + other)) / totalTarget * 100))
	}
	out.TownHallDown = d.TownHallHP > 0 && th <= 0
	if out.Destruction >= 50 {
		out.Stars++
	}
	if out.TownHallDown {
		out.Stars++
	}
	if out.Destruction >= 100 {
		out.Stars++
	}
	return out
}

// SortedTypes returns troop types present in an army in battle order.
func SortedTypes(army map[string]int) []string {
	out := []string{}
	for k := range army {
		out = append(out, k)
	}
	order := map[string]int{"guard": 0, "raider": 1, "ram": 2}
	sort.Slice(out, func(i, j int) bool { return order[out[i]] < order[out[j]] })
	return out
}
