package game

import "time"

// ChallengeTTL: a challenge to a friendly duel waits this long for an answer or an attack.
const ChallengeTTL = 24 * time.Hour

// FriendlyLimit: friendly duels a player may start per day; a tavern adds one per level, at most two.
func FriendlyLimit(s Settings, tavernLevel int) int {
	return s.FriendlyPerDay + min(2, max(0, tavernLevel))
}
