package dto

type UpdatePriceDTO struct {
	Value       float64 `json:"value" binding:"gte=0"` // Removed 'required' tag
	Description string  `json:"description" binding:"required"`
}
