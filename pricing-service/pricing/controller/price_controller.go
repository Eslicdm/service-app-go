package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"service-app-go/pricing-service/core/entity"
	"service-app-go/pricing-service/core/exception"
	"service-app-go/pricing-service/pricing/dto"
	"service-app-go/pricing-service/pricing/service"
)

// PriceController handles HTTP requests for pricing, mirroring the Spring
// pricing-service PriceController: GET /api/v1/prices (public) and
// PUT /api/v1/prices/{priceType} (manager/admin).
type PriceController struct {
	priceService service.PriceService
}

// NewPriceController creates a new PriceController.
func NewPriceController(s service.PriceService) *PriceController {
	return &PriceController{priceService: s}
}

// GetAllPrices retrieves all prices.
// @Summary Get all prices
// @Description Retrieve the 3 pricing tiers (free, half-price, full-price)
// @Tags prices
// @Produce json
// @Success 200 {array} []entity.Price
// @Failure 500 {object} exception.ErrorResponse
// @Router /prices [get]
func (c *PriceController) GetAllPrices(ctx *gin.Context) {
	prices, err := c.priceService.GetAllPrices(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, exception.ErrorResponse{Detail: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, prices)
}

// UpdatePrice updates a price by its PriceType (manager/admin only).
// @Summary Update a price by type
// @Description Upsert value/description for a pricing tier by PriceType
// @Tags prices
// @Accept json
// @Produce json
// @Param priceType path string true "Price Type (free, half-price, full-price)"
// @Param price body dto.UpdatePriceDTO true "Updated price"
// @Success 200 {object} entity.Price
// @Failure 400 {object} exception.ErrorResponse
// @Failure 403 {object} exception.ErrorResponse
// @Failure 500 {object} exception.ErrorResponse
// @Router /prices/{priceType} [put]
func (c *PriceController) UpdatePrice(ctx *gin.Context) {
	priceTypeStr := ctx.Param("priceType")
	priceType, err := entity.PriceTypeFromString(priceTypeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, exception.ErrorResponse{Detail: err.Error()})
		return
	}

	var updateDTO dto.UpdatePriceDTO
	if err := ctx.ShouldBindJSON(&updateDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, exception.ErrorResponse{Detail: err.Error()})
		return
	}

	result, err := c.priceService.UpdatePriceByType(ctx.Request.Context(), priceType, updateDTO)
	if err != nil {
		switch e := err.(type) {
		case *exception.InvalidInputError:
			ctx.JSON(http.StatusBadRequest, exception.ErrorResponse{Detail: e.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, exception.ErrorResponse{Detail: err.Error()})
		}
		return
	}
	ctx.JSON(http.StatusOK, result)
}
