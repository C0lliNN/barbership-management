package http

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/gcollin65/barbershop/internal/identity"
	"github.com/gcollin65/barbershop/internal/logger"
)

// RegisterIdentityRoutes wires identity endpoints onto the provided router group.
func RegisterIdentityRoutes(rg *gin.RouterGroup, svc identity.Signer, jwtSecret string) {
	rg.POST("/signup", handleSignUp(svc))
	rg.POST("/login", handleLogin(svc))
	rg.POST("/logout", handleLogout())

	protected := rg.Group("", AuthRequired(jwtSecret))
	protected.GET("/me", handleMe())
}

func handleSignUp(svc identity.Signer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req identity.SignUpRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusUnprocessableEntity, formatValidationError(err))
			return
		}
		resp, err := svc.SignUp(c.Request.Context(), req)
		if err != nil {
			switch {
			case errors.Is(err, identity.ErrEmailTaken):
				c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			case errors.Is(err, identity.ErrSlugTaken):
				c.JSON(http.StatusConflict, gin.H{"error": "shop name too similar to an existing shop; try a different name"})
			default:
				logger.FromContext(c.Request.Context()).Error("signup failed", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			}
			return
		}

		c.JSON(http.StatusCreated, toSignUpDTO(resp))
	}
}

func handleLogin(svc identity.Signer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req identity.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusUnprocessableEntity, formatValidationError(err))
			return
		}
		resp, err := svc.Login(c.Request.Context(), req)
		if err != nil {
			if errors.Is(err, identity.ErrInvalidCredentials) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
				return
			}
			logger.FromContext(c.Request.Context()).Error("login failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		c.JSON(http.StatusOK, toLoginDTO(resp))
	}
}

func handleLogout() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	}
}

func handleMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := claimsFromContext(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id": claims.Subject,
			"email":   claims.Email,
			"shop_id": claims.ShopID,
			"role":    claims.Role,
		})
	}
}

func formatValidationError(err error) gin.H {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		details := make([]string, len(ve))
		for i, fe := range ve {
			details[i] = fmt.Sprintf("%s: %s", fe.Field(), fe.Tag())
		}
		return gin.H{"error": "validation failed", "details": details}
	}
	return gin.H{"error": "invalid request body"}
}

// --- DTOs ---

type shopDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Phone     string `json:"phone,omitempty"`
	Address   string `json:"address,omitempty"`
	City      string `json:"city,omitempty"`
	State     string `json:"state,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toShopDTO(shop identity.Shop) shopDTO {
	return shopDTO{
		ID:        shop.ID,
		Name:      shop.Name,
		Slug:      shop.Slug,
		Phone:     shop.Phone,
		Address:   shop.Address,
		City:      shop.City,
		State:     shop.State,
		CreatedAt: time.Unix(shop.CreatedAt, 0).UTC().Format(time.RFC3339),
		UpdatedAt: time.Unix(shop.UpdatedAt, 0).UTC().Format(time.RFC3339),
	}
}

type ownerDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Phone     string `json:"phone,omitempty"`
	CreatedAt string `json:"created_at"`
}

type signUpDTO struct {
	Shop  shopDTO  `json:"shop"`
	Owner ownerDTO `json:"owner"`
}

func toSignUpDTO(resp identity.SignUpResponse) signUpDTO {
	return signUpDTO{
		Shop: toShopDTO(resp.Shop),
		Owner: ownerDTO{
			ID:        resp.Owner.ID,
			Email:     resp.Owner.Email,
			FullName:  resp.Owner.FullName,
			Phone:     resp.Owner.Phone,
			CreatedAt: time.Unix(resp.Owner.CreatedAt, 0).UTC().Format(time.RFC3339),
		},
	}
}

type loginDTO struct {
	Token  string   `json:"token"`
	User   ownerDTO `json:"user"`
	ShopID string   `json:"shop_id"`
	Role   string   `json:"role"`
}

func toLoginDTO(resp identity.LoginResponse) loginDTO {
	return loginDTO{
		Token: resp.Token,
		User: ownerDTO{
			ID:        resp.User.ID,
			Email:     resp.User.Email,
			FullName:  resp.User.FullName,
			Phone:     resp.User.Phone,
			CreatedAt: time.Unix(resp.User.CreatedAt, 0).UTC().Format(time.RFC3339),
		},
		ShopID: resp.ShopID,
		Role:   string(resp.Role),
	}
}
