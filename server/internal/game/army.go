package game

// ResearchSpec is one workshop step for a troop type: the gold and the time to reach Level.
type ResearchSpec struct {
	Level   int   `json:"level"`
	Cost    int64 `json:"cost"`
	Seconds int   `json:"seconds"`
}

// MaxTroopTop is the highest troop level a workshop can reach.
const MaxTroopTop = 4

// Research is the price of each troop level above 1 (index 0 is level 2).
var Research = []ResearchSpec{{Level: 2, Cost: 400, Seconds: 20}, {Level: 3, Cost: 900, Seconds: 35}, {Level: 4, Cost: 1600, Seconds: 50}}

// finishedLevel is the highest completed level of a building type in the city.
func (c City) finishedLevel(kind string) int {
	n := 0
	for _, b := range c.Buildings {
		if b.Type == kind && b.Level > n {
			n = b.Level
		}
	}
	return n
}

// MaxTroopLevel: troops can be raised to one above the finished workshop level, up to MaxTroopTop.
func MaxTroopLevel(c City) int {
	return min(MaxTroopTop, 1+c.finishedLevel("workshop"))
}

// CanTrain: archers need a finished archery range; the rest need only the barracks.
func CanTrain(c City, kind string) *Refusal {
	if kind == "archer" && c.finishedLevel("archery") == 0 {
		return refuse("no_archery", "Лучников учат на стрельбище. Постройте его в городе.")
	}
	return nil
}

// CheckResearch validates raising troop type kind from level current; busy means the workshop is already working.
func CheckResearch(c City, kind string, current int, busy bool, gold int64) (ResearchSpec, *Refusal) {
	if _, ok := Troops[kind]; !ok {
		return ResearchSpec{}, refuse("invalid_troop", "Такого отряда нет.")
	}
	if c.finishedLevel("workshop") == 0 {
		return ResearchSpec{}, refuse("no_workshop", "Сначала постройте мастерскую.")
	}
	if current < 1 {
		current = 1
	}
	if current >= MaxTroopTop {
		return ResearchSpec{}, refuse("max_level", "Отряд уже максимального уровня.")
	}
	if current+1 > MaxTroopLevel(c) {
		return ResearchSpec{}, refuse("workshop_level", "Улучшите мастерскую, чтобы открыть следующий уровень отряда.")
	}
	if busy {
		return ResearchSpec{}, refuse("workshop_busy", "Мастерская уже работает. Дождитесь окончания улучшения.")
	}
	spec := Research[current-1]
	if gold < spec.Cost {
		return ResearchSpec{}, refuse("insufficient_gold", "Недостаточно золота.")
	}
	return spec, nil
}
