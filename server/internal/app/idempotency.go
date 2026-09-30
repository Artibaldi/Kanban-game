package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// command runs a state-changing request exactly once per Idempotency-Key (pattern from Devops-pilot):
//   - the key is serialized with an advisory lock, so two identical requests in flight cannot both apply;
//   - a repeat with the same body gets the stored response (header Idempotent-Replay: true) and changes nothing;
//   - the same key with a different body is refused;
//   - a refusal (4xx) is not stored, so the client may correct and retry with the same key.
//
// fn works inside the transaction and returns the HTTP status and the response value.
func (s *Server) command(w http.ResponseWriter, r *http.Request, a actor, fn func(ctx context.Context, tx pgx.Tx, raw []byte) (int, any, error)) {
	key := r.Header.Get("Idempotency-Key")
	if len(key) < 8 || len(key) > 128 {
		fail(w, 428, "idempotency_key_required", "Нужен заголовок Idempotency-Key.")
		return
	}
	raw, err := readBody(r)
	if err != nil {
		failErr(w, err)
		return
	}
	sum := sha256.Sum256(raw)
	route := r.Method + " " + r.URL.Path
	ctx := r.Context()
	var status int
	var out []byte
	replay := false
	err = s.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, a.ID+"/"+route+"/"+key); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM idempotency_records WHERE expires_at<=now() AND ctid IN (SELECT ctid FROM idempotency_records WHERE expires_at<=now() LIMIT 50)`); err != nil {
			return err
		}
		var old []byte
		err := tx.QueryRow(ctx, `SELECT body_digest,status,response::text FROM idempotency_records WHERE actor_id=$1 AND route=$2 AND key=$3 AND expires_at>now()`, a.ID, route, key).Scan(&old, &status, &out)
		if err == nil {
			if !bytes.Equal(old, sum[:]) {
				return refuse(422, "idempotency_conflict", "Этот запрос уже был отправлен с другими данными. Обновите страницу.")
			}
			replay = true
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		st, value, err := fn(ctx, tx, raw)
		if err != nil {
			return err
		}
		status = st
		if out, err = json.Marshal(value); err != nil {
			return err
		}
		// Answer with the stored form, so the first response and every replay are byte-identical.
		return tx.QueryRow(ctx, `INSERT INTO idempotency_records(actor_id,route,key,body_digest,status,response,expires_at) VALUES($1,$2,$3,$4,$5,$6,now()+interval '7 days') RETURNING response::text`, a.ID, route, key, sum[:], status, out).Scan(&out)
	})
	if err != nil {
		failErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if replay {
		w.Header().Set("Idempotent-Replay", "true")
	}
	w.WriteHeader(status)
	_, _ = w.Write(out)
}
