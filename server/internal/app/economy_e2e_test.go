package app

import (
	"context"
	"testing"
)

func TestMarketAndBuilders(t *testing.T) {
	ts, pool, _ := testServer(t)
	alexey := login(t, ts, "alexey")
	ctx := context.Background()

	// Without a market nothing is exchanged.
	alexey.refused("no_market", "POST", "/city/market", map[string]any{"give": "gold", "amount": 100})

	// A finished market (level 1, 30% fee) and a builder hut from the seed town hall 3.
	if _, err := pool.Exec(ctx, `INSERT INTO buildings(user_id,type,level,x,y) SELECT id,'market',1,1,1 FROM users WHERE login='alexey'
	 UNION ALL SELECT id,'hut',1,2,1 FROM users WHERE login='alexey'`); err != nil {
		t.Fatal(err)
	}
	p := alexey.must(200, "GET", "/player", nil, "").Body
	if num(p["builders"]) != 3 {
		t.Fatalf("a hut adds the third builder: %v", p["builders"])
	}
	gold, energy := num(p["gold"]), num(p["energy"])
	out := alexey.must(200, "POST", "/city/market", map[string]any{"give": "gold", "amount": 100}, "market-trade-1").Body
	if num(out["gold"]) != gold-100 || num(out["energy"]) != energy+17 {
		t.Fatalf("100 gold → 17 energy: %v %v → %v %v", gold, energy, out["gold"], out["energy"])
	}
	// A repeated request with the same key is the same trade.
	again := alexey.must(200, "POST", "/city/market", map[string]any{"give": "gold", "amount": 100}, "market-trade-1").Body
	if num(again["gold"]) != gold-100 {
		t.Fatal("idempotent trade")
	}
	var kind string
	if err := pool.QueryRow(ctx, `SELECT kind FROM ledger WHERE user_id=(SELECT id FROM users WHERE login='alexey') ORDER BY id DESC LIMIT 1`).Scan(&kind); err != nil || kind != "market" {
		t.Fatalf("the trade is in the ledger: %q %v", kind, err)
	}
	alexey.refused("insufficient_energy", "POST", "/city/market", map[string]any{"give": "energy", "amount": 1_000_000})
	alexey.refused("too_little", "POST", "/city/market", map[string]any{"give": "gold", "amount": 3})
	// Gold for energy cannot overflow the energy store.
	if _, err := pool.Exec(ctx, `UPDATE players SET gold=3000, energy=990 WHERE user_id=(SELECT id FROM users WHERE login='alexey')`); err != nil {
		t.Fatal(err)
	}
	alexey.refused("storage_full", "POST", "/city/market", map[string]any{"give": "gold", "amount": 3000})
}
