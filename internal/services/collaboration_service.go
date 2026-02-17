package services

import (
	"context"
	"fmt"
	"log"
	"travelmate/internal/domain"
	"travelmate/internal/repositories"
)

type CollaborationService struct {
	CollabRepo *repositories.CollaboratorRepository
	UserRepo   *repositories.UserRepository
	TripRepo   *repositories.TripRepository
}

func NewCollaborationService(collabRepo *repositories.CollaboratorRepository, userRepo *repositories.UserRepository, tripRepo *repositories.TripRepository) *CollaborationService {
	return &CollaborationService{
		CollabRepo: collabRepo,
		UserRepo:   userRepo,
		TripRepo:   tripRepo,
	}
}

// InviteUser checks permissions, finds user by email, and adds them as a collaborator
func (s *CollaborationService) InviteUser(ctx context.Context, tripID, inviterID, email, role string) (*domain.Collaborator, error) {
	// 1. Check if Inviter has permission (Owner or Editor)
	canInvite, err := s.CheckTripAccess(ctx, tripID, inviterID, domain.RoleEditor)
	if err != nil {
		return nil, fmt.Errorf("permission check failed: %w", err)
	}
	if !canInvite {
		return nil, fmt.Errorf("forbidden: only owners and editors can invite")
	}

	// 2. Find User by Email
	targetUser, err := s.UserRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	if targetUser == nil {
		// MVP: Fail if user doesn't exist. Future: Create pending invite record.
		return nil, fmt.Errorf("user not found with email: %s", email)
	}

	// 3. Prevent inviting self
	if targetUser.UserID == inviterID {
		return nil, fmt.Errorf("cannot invite yourself")
	}

	// 4. Check if user is already a collaborator or owner
	existingRole, err := s.CollabRepo.GetUserRole(ctx, tripID, targetUser.UserID)
	if err != nil {
		return nil, err
	}
	if existingRole != "" {
		return nil, fmt.Errorf("user is already a collaborator with role: %s", existingRole)
	}

	// 5. Add Collaborator
	collab := &domain.Collaborator{
		TripID:    tripID,
		UserID:    targetUser.UserID,
		Role:      role,
		Status:    domain.StatusAccepted, // Auto-accept for MVP if user exists
		InvitedBy: inviterID,
	}

	if err := s.CollabRepo.AddCollaborator(ctx, collab); err != nil {
		return nil, err
	}

	// 6. Return with user details
	collab.User = targetUser
	return collab, nil

}

// RemoveCollaborator removes a user from the trip
func (s *CollaborationService) RemoveCollaborator(ctx context.Context, tripID, requestorID, targetUserID string) error {
	// 1. Self-removal (Leave Trip) is always allowed
	if requestorID == targetUserID {
		// Double check not Owner leaving (Owner must transfer ownership first - simplified: Owner cannot leave)
		role, err := s.CollabRepo.GetUserRole(ctx, tripID, requestorID)
		if err != nil {
			return err
		}
		if role == domain.RoleOwner {
			return fmt.Errorf("owner cannot leave trip. delete trip or transfer ownership instead")
		}
		return s.CollabRepo.RemoveCollaborator(ctx, tripID, requestorID)
	}

	// 2. Check if Requestor is Owner
	role, err := s.CollabRepo.GetUserRole(ctx, tripID, requestorID)
	if err != nil {
		return err
	}
	if role != domain.RoleOwner {
		return fmt.Errorf("forbidden: only owner can remove other collaborators")
	}

	// 3. Remove target
	return s.CollabRepo.RemoveCollaborator(ctx, tripID, targetUserID)
}

// GetCollaborators fetches all collaborators with user details
func (s *CollaborationService) GetCollaborators(ctx context.Context, tripID, requestorID string) ([]domain.Collaborator, error) {
	// 1. Check Access
	hasAccess, err := s.CollabRepo.HasAccess(ctx, tripID, requestorID)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, fmt.Errorf("forbidden: no access to trip")
	}

	// 2. Fetch Collaborators
	collabs, err := s.CollabRepo.GetCollaborators(ctx, tripID)
	if err != nil {
		return nil, err
	}

	// 3. Enrich with User Details (N+1 query pattern, acceptable for small lists)
	for i, c := range collabs {
		user, err := s.UserRepo.GetUserByClerkID(ctx, c.UserID)
		if err != nil {
			log.Printf("Warning: failed to fetch user details for %s: %v", c.UserID, err)
			continue
		}
		// If user not found (e.g. deleted), we still show the collaborator record but without details
		if user != nil {
			// Create a copy to avoid mutating cache/db structs if any
			u := *user
			collabs[i].User = &u
		}
	}

	return collabs, nil
}

// UpdateRole updates a collaborator's role
func (s *CollaborationService) UpdateRole(ctx context.Context, tripID, requestorID, targetUserID, newRole string) error {
	// 1. Only Owner can update roles
	role, err := s.CollabRepo.GetUserRole(ctx, tripID, requestorID)
	if err != nil {
		return err
	}
	if role != domain.RoleOwner {
		return fmt.Errorf("forbidden: only owner can update roles")
	}

	// 2. Prevent changing own role (Owner must always be Owner)
	if requestorID == targetUserID {
		return fmt.Errorf("cannot change your own role")
	}

	// 3. Update
	return s.CollabRepo.UpdateCollaboratorRole(ctx, tripID, targetUserID, newRole)
}

// CheckTripAccess verifies if a user has at least the required role
// Hierarchy: Owner > Editor > Viewer
func (s *CollaborationService) CheckTripAccess(ctx context.Context, tripID, userID, requiredRole string) (bool, error) {
	role, err := s.CollabRepo.GetUserRole(ctx, tripID, userID)
	if err != nil {
		return false, err
	}
	if role == "" {
		return false, nil // No access
	}

	// Normalize roles for comparison
	switch requiredRole {
	case domain.RoleViewer:
		// Viewer, Editor, Owner all have read access
		return true, nil
	case domain.RoleEditor:
		// Only Editor and Owner have write access
		return role == domain.RoleEditor || role == domain.RoleOwner, nil
	case domain.RoleOwner:
		// Only Owner
		return role == domain.RoleOwner, nil
	default:
		return false, fmt.Errorf("unknown role requirement: %s", requiredRole)
	}
}
