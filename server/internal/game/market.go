package game

import "math"

// maxPayout is far above any storage cap and far below the int64 limit.
const maxPayout = 1e15

// Exchange is one trade at the market: give gold for energy or energy for gold. It returns the balance changes.
// The rate follows the work rewards (gold and energy per point), and the market keeps FeePct of what it pays,
// so a round trip always loses: the market moves value between resources and never creates it.
func Exchange(s Settings, c City, give string, amount int64) (dGold, dEnergy int64, r *Refusal) {
	if amount <= 0 || (give != "gold" && give != "energy") {
		return 0, 0, refuse("invalid_input", "Укажите, что и сколько обменять.")
	}
	fee := -1
	for _, b := range c.Buildings {
		// A market being upgraded keeps trading at the level it has.
		if spec, ok := Buildings[b.Type].Spec(b.Level); ok && b.Type == "market" && (fee < 0 || spec.FeePct < fee) {
			fee = spec.FeePct
		}
	}
	if fee < 0 {
		return 0, 0, refuse("no_market", "Сначала постройте рынок.")
	}
	rate := s.GoldPerPoint / s.EnergyPerPoint // gold for one energy
	keep := float64(100-fee) / 100
	pay := float64(amount) * rate * keep
	if give == "gold" {
		pay = float64(amount) / rate * keep
	}
	// No store holds this much; refusing here also keeps the conversion to int64 from wrapping around.
	if pay > maxPayout {
		return 0, 0, refuse("storage_full", "Столько не поместится в запас. Обменяйте меньше или постройте хранилище.")
	}
	if give == "gold" {
		got := int64(math.Floor(pay + 1e-9))
		if got < 1 {
			return 0, 0, refuse("too_little", "Слишком мало для обмена: получится меньше одной единицы.")
		}
		return -amount, got, nil
	}
	got := int64(math.Floor(pay + 1e-9))
	if got < 1 {
		return 0, 0, refuse("too_little", "Слишком мало для обмена: получится меньше одной единицы.")
	}
	return got, -amount, nil
}

// TradeFits: the player has what they give, and what they get fits the storage — nothing is lost to a full store.
func TradeFits(c City, gold, energy, dGold, dEnergy int64) *Refusal {
	if gold+dGold < 0 {
		return refuse("insufficient_gold", "Недостаточно золота.")
	}
	if energy+dEnergy < 0 {
		return refuse("insufficient_energy", "Недостаточно энергии.")
	}
	capG, capE := c.Caps()
	// Compared as room left, so a huge change cannot wrap around past the cap.
	if (dGold > 0 && dGold > capG-gold) || (dEnergy > 0 && dEnergy > capE-energy) {
		return refuse("storage_full", "Столько не поместится в запас. Обменяйте меньше или постройте хранилище.")
	}
	return nil
}
