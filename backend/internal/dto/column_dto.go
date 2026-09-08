package dto

// MoveCardRequest is the body of PATCH /api/cards/{cardID}/move. is_done is not
// accepted: the server derives it from the target column's category.
type MoveCardRequest struct {
	ColumnID string  `json:"column_id" validate:"required,uuid4"`
	Position float64 `json:"position"  validate:"required,gt=0"`
}

// ToggleCardDoneRequest is the body of PATCH /api/cards/{cardID}/done. IsDone is a
// pointer so an omitted field is a 400 rather than a silent "false".
type ToggleCardDoneRequest struct {
	IsDone *bool `json:"is_done" validate:"required"`
}

// CreateColumnRequest is the body of POST /api/boards/{boardID}/columns.
type CreateColumnRequest struct {
	Title    string  `json:"title"    validate:"required,min=1,max=100"`
	Category string  `json:"category" validate:"required,oneof=TODO DONE"`
	Color    *string `json:"color"    validate:"omitempty,hexcolor"`
}

// UpdateColumnRequest is the body of PATCH /api/columns/{columnID}. Both title and
// category are required: the underlying SQL sets them outright rather than COALESCE.
type UpdateColumnRequest struct {
	Title    string  `json:"title"    validate:"required,min=1,max=100"`
	Category string  `json:"category" validate:"required,oneof=TODO DONE"`
	Color    *string `json:"color"    validate:"omitempty,hexcolor"`
}
