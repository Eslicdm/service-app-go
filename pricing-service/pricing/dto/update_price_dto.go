package dto

// UpdatePriceDTO mirrors the Spring PriceUpdateDTO record:
// value (@NotNull @DecimalMin("0.0")) and description (@NotBlank).
// value=0 is allowed (the Free "Garden Pass" tier is 0.00).
type UpdatePriceDTO struct {
	Value       float64 `json:"value" binding:"gte=0"`
	Description string  `json:"description" binding:"required"`
}
