package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The city-planning migration moves every existing city two cells toward the centre. Buildings next to each
// other must not collide on the way (the (user, x, y) uniqueness is checked row by row).
func TestCityShiftWithExistingBuildings(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	name := "kanban_mig_" + hex.EncodeToString(b)
	if _, err = admin.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(dsn)
	u.Path = "/" + name
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), `DROP DATABASE `+name+` WITH (FORCE)`)
		admin.Close()
	})
	apply := func(file string) {
		body, err := migrations.ReadFile("migrations/" + file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, string(body), pgx.QueryExecModeSimpleProtocol); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
	for _, f := range []string{"0001_initial.sql", "0002_coop.sql", "0003_challenges.sql"} {
		apply(f)
	}
	var user string
	if err = pool.QueryRow(ctx, `INSERT INTO users(login,display_name,role) VALUES('a','A','admin') RETURNING id::text`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	// A row of walls two cells apart: shifting (0,0) by +2 lands on (2,2) before that one has moved.
	for _, c := range [][2]int{{0, 0}, {2, 2}, {4, 4}, {11, 11}} {
		if _, err = pool.Exec(ctx, `INSERT INTO buildings(user_id,type,level,x,y) VALUES($1,'wall',1,$2,$3)`, user, c[0], c[1]); err != nil {
			t.Fatal(err)
		}
	}
	apply("0004_city_planning.sql")
	rows, err := pool.Query(ctx, `SELECT x, y FROM buildings ORDER BY x`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) ([2]int, error) {
		var c [2]int
		err := r.Scan(&c[0], &c[1])
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]int{{2, 2}, {4, 4}, {6, 6}, {13, 13}}
	if len(got) != len(want) {
		t.Fatalf("buildings: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("shifted: %v, want %v", got, want)
		}
	}
}
