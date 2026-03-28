package service

import (
	"database/sql"
	"net/http"
	"os"

	"go-file-server/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserInfo struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	Role           string `json:"role"`
	MfaEnabled     bool   `json:"mfa_enabled"`
	MfaMandatory   bool   `json:"mfa_mandatory"`
	StorageQuota   int64  `json:"storage_quota"`
	FailedAttempts int    `json:"failed_attempts"`
	LockedUntil    string `json:"locked_until,omitempty"`
	IsSuperAdmin   bool   `json:"is_super_admin"`
}

func (s *UserService) GetUsers(c *gin.Context) {
	superAdminUser := os.Getenv("ADMIN_USER")

	usersData, err := s.UserRepo.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	users := []UserInfo{}
	for _, u := range usersData {
		var locked string
		if u.LockedUntil != nil {
			locked = u.LockedUntil.Format("2006-01-02 15:04:05")
		}
		var quota int64
		if u.StorageQuota != nil {
			quota = *u.StorageQuota
		}

		userInfo := UserInfo{
			ID:             u.ID,
			Username:       u.Username,
			Role:           u.Role,
			MfaEnabled:     u.MFAEnabled,
			MfaMandatory:   u.MFAMandatory,
			StorageQuota:   quota,
			FailedAttempts: u.FailedAttempts,
			LockedUntil:    locked,
			IsSuperAdmin:   u.Username == superAdminUser,
		}
		users = append(users, userInfo)
	}

	c.JSON(http.StatusOK, users)
}

type CreateUserRequest struct {
	Username     string `json:"username" binding:"required"`
	Password     string `json:"password" binding:"required"`
	Role         string `json:"role"`
	MfaMandatory bool   `json:"mfa_mandatory"`
	StorageQuota int64  `json:"storage_quota"`
}

func (s *UserService) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	currentUsername := c.GetString("username")
	superAdminUser := os.Getenv("ADMIN_USER")

	if req.Role == "superadmin" && currentUsername != superAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the main superadmin can create other superadmins"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	id := uuid.New().String()

	user := &model.User{
		ID:           id,
		Username:     req.Username,
		PasswordHash: string(hashed),
		Role:         req.Role,
		MFAMandatory: req.MfaMandatory,
		StorageQuota: &req.StorageQuota,
	}

	if err := s.UserRepo.Create(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "username may already exist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "id": id})
}

func (s *UserService) RevokeSessions(c *gin.Context) {
	id := c.Param("id")
	currentUsername := c.GetString("username")
	superAdminUser := os.Getenv("ADMIN_USER")

	targetUser, err := s.UserRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if targetUser.Username == superAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot revoke super admin"})
		return
	}

	if targetUser.Role == "superadmin" && currentUsername != superAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the main superadmin can revoke other superadmins"})
		return
	}

	if targetUser.Role == "admin" && currentUsername != superAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "only super admin can revoke an admin"})
		return
	}

	// Increment token_version to invalidate all access tokens
	if err := s.UserRepo.IncrementTokenVersionByID(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke access tokens"})
		return
	}

	// Revoke refresh tokens
	s.TokenRepo.RevokeByUsername(targetUser.Username)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *UserService) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	currentUsername := c.GetString("username")
	superAdminUser := os.Getenv("ADMIN_USER")

	targetUser, err := s.UserRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if targetUser.Username == superAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete super admin"})
		return
	}

	if targetUser.Role == "superadmin" && currentUsername != superAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the main superadmin can delete other superadmins"})
		return
	}

	if targetUser.Role == "admin" && currentUsername != superAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "only super admin can delete an admin"})
		return
	}

	if err := s.UserRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *UserService) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.GetString("username")
		if username == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		user, err := s.UserRepo.GetByUsername(username)
		if err != nil || (user.Role != "admin" && user.Role != "superadmin") {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}
