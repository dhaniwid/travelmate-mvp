package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"travelmate/internal/domain"
)

type CollaboratorRepository struct {
	DB *sql.DB
}

func NewCollaboratorRepository(db *sql.DB) *CollaboratorRepository {
	return &CollaboratorRepository{DB: db}
}

// AddCollaborator adds a new collaborator to a trip
func (r *CollaboratorRepository) AddCollaborator(ctx context.Context, collab *domain.Collaborator) error {
	query := `
		INSERT INTO trip_collaborators (trip_id, user_id, role, status, invited_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (trip_id, user_id) DO UPDATE SET
			role = EXCLUDED.role,
			status = EXCLUDED.status,
			updated_at = NOW()
	`

	_, err := r.DB.ExecContext(ctx, query,
		collab.TripID,
		collab.UserID,
		collab.Role,
		collab.Status,
		collab.InvitedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to add collaborator: %w", err)
	}

	return nil
}

// GetCollaborators retrieves all collaborators for a trip
func (r *CollaboratorRepository) GetCollaborators(ctx context.Context, tripID string) ([]domain.Collaborator, error) {
	query := `
		SELECT id, trip_id, user_id, role, status, invited_by, created_at, updated_at
		FROM trip_collaborators
		WHERE trip_id = $1
		ORDER BY 
			CASE role 
				WHEN 'owner' THEN 1
				WHEN 'editor' THEN 2
				WHEN 'viewer' THEN 3
			END,
			created_at ASC
	`

	rows, err := r.DB.QueryContext(ctx, query, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to query collaborators: %w", err)
	}
	defer rows.Close()

	var collaborators []domain.Collaborator
	for rows.Next() {
		var c domain.Collaborator
		err := rows.Scan(
			&c.ID,
			&c.TripID,
			&c.UserID,
			&c.Role,
			&c.Status,
			&c.InvitedBy,
			&c.CreatedAt,
			&c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan collaborator: %w", err)
		}
		collaborators = append(collaborators, c)
	}

	return collaborators, nil
}

// RemoveCollaborator removes a collaborator from a trip
func (r *CollaboratorRepository) RemoveCollaborator(ctx context.Context, tripID, userID string) error {
	query := `DELETE FROM trip_collaborators WHERE trip_id = $1 AND user_id = $2`

	result, err := r.DB.ExecContext(ctx, query, tripID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove collaborator: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("collaborator not found")
	}

	return nil
}

// UpdateCollaboratorRole updates a collaborator's role
func (r *CollaboratorRepository) UpdateCollaboratorRole(ctx context.Context, tripID, userID, newRole string) error {
	query := `
		UPDATE trip_collaborators 
		SET role = $1, updated_at = NOW()
		WHERE trip_id = $2 AND user_id = $3
	`

	result, err := r.DB.ExecContext(ctx, query, newRole, tripID, userID)
	if err != nil {
		return fmt.Errorf("failed to update collaborator role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("collaborator not found")
	}

	return nil
}

// GetUserRole retrieves a user's role for a specific trip
func (r *CollaboratorRepository) GetUserRole(ctx context.Context, tripID, userID string) (string, error) {
	// 1. Check if user is the trip owner directly
	var isOwner bool
	err := r.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM trips WHERE id = $1 AND user_id = $2)", tripID, userID).Scan(&isOwner)
	if err != nil {
		return "", fmt.Errorf("failed to check ownership: %w", err)
	}
	if isOwner {
		return domain.RoleOwner, nil
	}

	// 2. Check collaborators table
	query := `
		SELECT role 
		FROM trip_collaborators 
		WHERE trip_id = $1 AND user_id = $2 AND status = 'accepted'
	`

	var role string
	err = r.DB.QueryRowContext(ctx, query, tripID, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", nil // User is not a collaborator
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

// HasAccess checks if a user has access to a trip (either as owner or collaborator)
func (r *CollaboratorRepository) HasAccess(ctx context.Context, tripID, userID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM trips WHERE id = $1 AND user_id = $2
			UNION
			SELECT 1 FROM trip_collaborators 
			WHERE trip_id = $1 AND user_id = $2 AND status = 'accepted'
		)
	`

	var hasAccess bool
	err := r.DB.QueryRowContext(ctx, query, tripID, userID).Scan(&hasAccess)
	if err != nil {
		return false, fmt.Errorf("failed to check access: %w", err)
	}

	return hasAccess, nil
}
