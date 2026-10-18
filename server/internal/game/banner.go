package game

// RewardOrder lists the wave-reward decor in the order the team usually earns it.
var RewardOrder = []string{"flags", "arch", "fountain", "statue"}

// Trophies: which reward decor the team has unlocked, from its settled waves' crowns, oldest first.
// Team achievements only — nobody's personal share is counted.
func Trophies(crowns []int) map[string]bool {
	out := map[string]bool{}
	run, threes := 0, 0
	for _, c := range crowns {
		if c >= 1 {
			out["flags"] = true
		}
		if c >= 3 {
			out["arch"] = true
			threes++
		}
		if c >= 2 {
			run++
		} else {
			run = 0
		}
		if run >= 3 {
			out["fountain"] = true
		}
	}
	if threes >= 3 {
		out["statue"] = true
	}
	return out
}

// RewardLocked: reward decor the team has not earned yet cannot be placed; other buildings are never locked here.
func RewardLocked(kind string, crowns []int) *Refusal {
	d, ok := Buildings[kind]
	if !ok || d.Reward == "" || Trophies(crowns)[kind] {
		return nil
	}
	return refuse("reward_locked", "Откроется, когда команда этого добьётся: "+d.Reward)
}

// GarrisonCapacity: places for colleagues' troops at the finished team banner.
func (c City) GarrisonCapacity() int {
	n := 0
	for _, b := range c.Buildings {
		if spec, ok := Buildings[b.Type].Spec(b.Level); ok && b.Type == "banner" {
			n += spec.Garrison
		}
	}
	return n
}

// CheckDonate validates sending count troops of kind to a banner that already holds `held` places;
// `ready` is how many of that kind the donor has trained and at home.
func CheckDonate(owner City, held int, kind string, count, ready int) *Refusal {
	t, ok := Troops[kind]
	if !ok || count < 1 {
		return refuse("invalid_input", "Выберите отряд и количество.")
	}
	room := owner.GarrisonCapacity()
	if room == 0 {
		return refuse("no_banner", "У коллеги ещё нет знамени команды.")
	}
	if count > ready {
		return refuse("not_enough_troops", "Столько готовых войск этого типа у вас нет.")
	}
	if held+t.Housing*count > room {
		return refuse("banner_full", "На знамени не хватит места.")
	}
	return nil
}
