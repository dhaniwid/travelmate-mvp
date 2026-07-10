package repositories

import (
	"context"
	"database/sql"
	"travelmate/internal/domain"
)

type PassportRepository struct {
	DB *sql.DB
}

func NewPassportRepository(db *sql.DB) *PassportRepository {
	return &PassportRepository{DB: db}
}

func (r *PassportRepository) GetUserStamps(ctx context.Context, userID string) ([]domain.PassportStamp, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, user_id, city, city_slug, date, serial, mood, image_url, rotation,
		       COALESCE(trip_id, ''), created_at
		FROM passport_stamps
		WHERE user_id = $1
		ORDER BY date DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stamps []domain.PassportStamp
	for rows.Next() {
		var s domain.PassportStamp
		if err := rows.Scan(&s.ID, &s.UserID, &s.City, &s.CitySlug, &s.Date,
			&s.Serial, &s.Mood, &s.ImageURL, &s.Rotation, &s.TripID, &s.CreatedAt); err != nil {
			return nil, err
		}
		stamps = append(stamps, s)
	}
	return stamps, rows.Err()
}

func (r *PassportRepository) GetStampsByCity(ctx context.Context, userID, citySlug string) ([]domain.PassportStamp, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, user_id, city, city_slug, date, serial, mood, image_url, rotation,
		       COALESCE(trip_id, ''), created_at
		FROM passport_stamps
		WHERE user_id = $1 AND city_slug = $2
		ORDER BY date DESC
	`, userID, citySlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stamps []domain.PassportStamp
	for rows.Next() {
		var s domain.PassportStamp
		if err := rows.Scan(&s.ID, &s.UserID, &s.City, &s.CitySlug, &s.Date,
			&s.Serial, &s.Mood, &s.ImageURL, &s.Rotation, &s.TripID, &s.CreatedAt); err != nil {
			return nil, err
		}
		stamps = append(stamps, s)
	}
	return stamps, rows.Err()
}

func (r *PassportRepository) UpsertStamp(ctx context.Context, s domain.PassportStamp) (*domain.PassportStamp, error) {
	tripID := sql.NullString{String: s.TripID, Valid: s.TripID != ""}
	var result domain.PassportStamp
	var tripIDOut sql.NullString

	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO passport_stamps (user_id, city, city_slug, date, serial, mood, image_url, rotation, trip_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, city_slug, mood)
		DO UPDATE SET
			date      = EXCLUDED.date,
			serial    = EXCLUDED.serial,
			image_url = EXCLUDED.image_url,
			rotation  = EXCLUDED.rotation,
			trip_id   = EXCLUDED.trip_id
		RETURNING id, user_id, city, city_slug, date, serial, mood, image_url, rotation,
		          trip_id, created_at
	`, s.UserID, s.City, s.CitySlug, s.Date, s.Serial, s.Mood, s.ImageURL, s.Rotation, tripID,
	).Scan(&result.ID, &result.UserID, &result.City, &result.CitySlug, &result.Date,
		&result.Serial, &result.Mood, &result.ImageURL, &result.Rotation,
		&tripIDOut, &result.CreatedAt)
	if err != nil {
		return nil, err
	}
	if tripIDOut.Valid {
		result.TripID = tripIDOut.String
	}
	return &result, nil
}
