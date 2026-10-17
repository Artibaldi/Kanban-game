package app

import (
	"context"
	"testing"
)

func TestCoopSchema(t *testing.T) {
	_, pool, _ := testServer(t)
	ctx := context.Background()
	var hp, weeks int
	if err := pool.QueryRow(ctx, `SELECT f.hp, t.wave_weeks FROM fortresses f JOIN teams t ON t.id=f.team_id WHERE t.name='Платформа'`).Scan(&hp, &weeks); err != nil {
		t.Fatal(err)
	}
	if weeks != 1 {
		t.Fatalf("default wave length: %d", weeks)
	}
	var trophies int
	_ = pool.QueryRow(ctx, `SELECT COALESCE(MAX(trophies),0) FROM players`).Scan(&trophies)
	if trophies != 0 {
		t.Fatalf("raid trophies must be reset: %d", trophies)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ledger(user_id,kind,note) SELECT id,'wave_reward','t' FROM users WHERE login='alexey'`); err != nil {
		t.Fatalf("wave_reward kind must be allowed: %v", err)
	}
}
