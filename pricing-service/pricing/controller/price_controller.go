package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"service-app-go/pricing-service/core/exception"
	"service-app-go/pricing-service/pricing/dto"
	"service-app-go/pricing-service/pricing/service"
)

// PriceController handles HTTP requests related to prices.
type PriceController struct {
	priceService service.PriceService
}

// NewPriceController creates a new PriceController.
func NewPriceController(s service.PriceService) *PriceController {
	return &PriceController{
		priceService: s,
	}
}

// CreatePrice handles the creation of a new price.
// @Summary Create a new price
// @Description Create a new price with the provided details
// @Tags prices
// @Accept json
// @Produce json
// @Param price body dto.CreatePriceDTO true "Price object to be created"
// @Success 201 {object} entity.Price
// @Failure 400 {object} exception.ErrorResponse "Invalid input"
// @Failure 500 {object} exception.ErrorResponse "Internal server error"
// @Router /prices [post]
func (c *PriceController) CreatePrice(ctx *gin.Context) {
	var createDTO dto.CreatePriceDTO
	if err := ctx.ShouldBindJSON(&createDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, exception.ErrorResponse{Detail: err.Error()})
		return
	}

	createdPrice, err := c.priceService.CreatePrice(ctx.Request.Context(), createDTO)
	if err != nil {
		switch e := err.(type) {
		case *exception.InvalidInputError:
			ctx.JSON(http.StatusBadRequest, exception.ErrorResponse{Detail: e.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, exception.ErrorResponse{Detail: err.Error()})
		}
		return
	}
	ctx.JSON(http.StatusCreated, createdPrice)
}

// GetPriceByID retrieves a price by its ID.
// @Summary Get a price by ID
// @Description Get a single price by its unique identifier
// @Tags prices
// @Produce json
// @Param id path string true "Price ID"
// @Success 200 {object} entity.Price
// @Failure 400 {object} exception.ErrorResponse "Invalid ID format"
// @Failure 404 {object} exception.ErrorResponse "Price not found"
// @Failure 500 {object} exception.ErrorResponse "Internal server error"
// @Router /prices/{id} [get]
func (c *PriceController) GetPriceByID(ctx *gin.Context) {
	id := ctx.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		ctx.JSON(http.StatusBadRequest, exception.ErrorResponse{Detail: "Invalid ID format"})
		return
	}

	price, err := c.priceService.GetPriceByID(ctx.Request.Context(), id)
	if err != nil {
		switch e := err.(type) {
		case *exception.PriceNotFoundError:
			ctx.JSON(http.StatusNotFound, exception.ErrorResponse{Detail: e.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, exception.ErrorResponse{Detail: err.Error()})
		}
		return
	}
	ctx.JSON(http.StatusOK, price)
}

// GetAllPrices retrieves all prices.
// @Summary Get all prices
// @Description Retrieve a list of all prices
// @Tags prices
// @Produce json
// @Success 200 {array} []entity.Price
// @Failure 500 {object} exception.ErrorResponse "Internal server error"
// @Router /prices [get]
func (c *PriceController) GetAllPrices(ctx *gin.Context) {
	prices, err := c.priceService.GetAllPrices(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, exception.ErrorResponse{Detail: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, prices)
}

// UpdatePrice updates an existing price.
// @Summary Update an existing price
// @Description Update the details of an existing price by PriceType
// @Tags prices
// @Accept json
// @Produce json
// @Param priceType path string true "Price Type (e.g., 'free', 'half-price', 'full-price')"
// @Param price body dto.UpdatePriceDTO true "Updated price object"
// @Success 200 {object} entity.Price
// @Failure 400 {object} exception.ErrorResponse "Invalid input"
// @Failure 404 {object} exception.ErrorResponse "Price not found"
// @Failure 500 {object} exception.ErrorResponse "Internal server error"
// @Router /prices/{priceType} [put]
func (c *PriceController) UpdatePrice(ctx *gin.Context) {
	priceType := ctx.Param("id") // The parameter is still named "id" in the route definition

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
		case *exception.PriceNotFoundError:
			ctx.JSON(http.StatusNotFound, exception.ErrorResponse{Detail: e.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, exception.ErrorResponse{Detail: err.Error()})
		}
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// DeletePrice deletes a price by its ID.
// @Summary Delete a price
// @Description Delete a price by its unique identifier
// @Tags prices
// @Produce json
// @Param id path string true "Price ID"
// @Success 204 "No Content"
// @Failure 400 {object} exception.ErrorResponse "Invalid ID format"
// @Failure 404 {object} exception.ErrorResponse "Price not found"
// @Failure 500 {object} exception.ErrorResponse "Internal server error"
// @Router /prices/{id} [delete]
func (c *PriceController) DeletePrice(ctx *gin.Context) {
	id := ctx.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		ctx.JSON(http.StatusBadRequest, exception.ErrorResponse{Detail: "Invalid ID format"})
		return
	}

	err := c.priceService.DeletePrice(ctx.Request.Context(), id)
	if err != nil {
		switch e := err.(type) {
		case *exception.PriceNotFoundError:
			ctx.JSON(http.StatusNotFound, exception.ErrorResponse{Detail: e.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, exception.ErrorResponse{Detail: err.Error()})
		}
		return
	}
	ctx.Status(http.StatusNoContent)
}
