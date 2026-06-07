package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gcollin65/barbershop/internal/identity"
	"github.com/gcollin65/barbershop/internal/logger"
)

// RegisterShopRoutes wires shop-scoped endpoints onto the provided router
// group. Every route requires authentication and verified membership in the
// :shopID shop; PATCH additionally requires the Owner role.
func RegisterShopRoutes(rg *gin.RouterGroup, svc identity.ShopManager, memberships identity.MembershipRepository, jwtSecret string) {
	scoped := rg.Group("/shops/:shopID", AuthRequired(jwtSecret), RequireShopMembership(memberships))
	scoped.GET("", handleGetShop(svc))
	scoped.PATCH("", RequireRole(identity.RoleOwner), handleUpdateShop(svc))
}

func handleGetShop(svc identity.ShopManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		shopID, _ := identity.TenantFromCtx(ctx)

		shop, err := svc.GetShop(ctx, shopID)
		if err != nil {
			if errors.Is(err, identity.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			logger.FromContext(ctx).Error("get shop failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, toShopDTO(shop))
	}
}

func handleUpdateShop(svc identity.ShopManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		shopID, _ := identity.TenantFromCtx(ctx)

		var input identity.ShopUpdateInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusUnprocessableEntity, formatValidationError(err))
			return
		}

		shop, err := svc.UpdateShop(ctx, shopID, input)
		if err != nil {
			if errors.Is(err, identity.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			logger.FromContext(ctx).Error("update shop failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, toShopDTO(shop))
	}
}
