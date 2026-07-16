package gateway

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func uuidToPg(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func pgToUUID(id pgtype.UUID) (uuid.UUID, bool) {
	if !id.Valid {
		return uuid.Nil, false
	}
	return uuid.UUID(id.Bytes), true
}
