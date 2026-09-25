package dto

// is_done is derived from the target column's category.
type MoveCardRequest struct {
	ColumnID string  `json:"column_id" validate:"required,uuid4"`
	Position float64 `json:"position"  validate:"required,gt=0"`
}

// Pointer so an omitted field is a 400, not a silent false.
type ToggleCardDoneRequest struct {
	IsDone *bool `json:"is_done" validate:"required"`
}

// Color is a palette key (ColumnOptionsModal.COLUMN_COLOR_PALETTE), not a hex value.

type CreateColumnRequest struct {
	Title    string  `json:"title"    validate:"required,min=1,max=100"`
	Category string  `json:"category" validate:"required,oneof=TODO DONE"`
	Color    *string `json:"color"    validate:"omitempty,oneof=slate blue purple green amber rose pink cyan"`
}

// Both required: the SQL sets them outright, not via COALESCE.
type UpdateColumnRequest struct {
	Title    string  `json:"title"    validate:"required,min=1,max=100"`
	Category string  `json:"category" validate:"required,oneof=TODO DONE"`
	Color    *string `json:"color"    validate:"omitempty,oneof=slate blue purple green amber rose pink cyan"`
}
