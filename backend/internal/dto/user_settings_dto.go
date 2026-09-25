package dto

type UserSettingsResponse struct {
	DefaultLanding string `json:"default_landing"`
	ShowAllCards   bool   `json:"show_all_cards"`
	Timezone       string `json:"timezone"`
}

type UpdateUserSettingsRequest struct {
	DefaultLanding *string `json:"default_landing" validate:"omitempty,oneof=today my_work all_boards"`
	ShowAllCards   *bool   `json:"show_all_cards"`
	Timezone       *string `json:"timezone"        validate:"omitempty,min=1,max=50"`
}
