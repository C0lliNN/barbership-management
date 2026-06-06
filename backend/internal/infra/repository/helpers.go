package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func nullText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
