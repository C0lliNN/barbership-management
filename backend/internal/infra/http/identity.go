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
func RegisterIdentityRoutes(rg *gin.RouterGroup, svc identity.Signer) {
	rg.POST("/signup", handleSignUp(svc))
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
		Shop: shopDTO{
			ID:        resp.Shop.ID,
			Name:      resp.Shop.Name,
			Slug:      resp.Shop.Slug,
			Phone:     resp.Shop.Phone,
			Address:   resp.Shop.Address,
			City:      resp.Shop.City,
			State:     resp.Shop.State,
			CreatedAt: time.Unix(resp.Shop.CreatedAt, 0).UTC().Format(time.RFC3339),
		},
		Owner: ownerDTO{
			ID:        resp.Owner.ID,
			Email:     resp.Owner.Email,
			FullName:  resp.Owner.FullName,
			Phone:     resp.Owner.Phone,
			CreatedAt: time.Unix(resp.Owner.CreatedAt, 0).UTC().Format(time.RFC3339),
		},
	}
}
