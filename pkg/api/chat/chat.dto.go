package chat

type UserKeyEntry struct {
	UserID string `json:"userId" validate:"required"`
	Key    string `json:"key" validate:"required"`
}

type UserEntry struct {
	Id   string  `json:"id"`
	Name string  `json:"name"`
	Bio  *string `json:"bio"`
}

type MessageEntry struct {
	Id        string `json:"id"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Content   string `json:"content"`
	Iv        string `json:"iv"`
	SenderId  string `json:"sender_id"`
}

type CreateChatRequest struct {
	Name        string         `json:"name" validate:"required"`
	Picture     *string        `json:"picture" validate:"omitempty,url"`
	Description *string        `json:"description" validate:"omitempty"`
	UserKeys    []UserKeyEntry `json:"userKeys" validate:"required,dive"`
}

type CreateChatResponse struct {
	Id          string  `json:"id"`
	CreatedAt   string  `json:"createdAt"`
	UpdateAt    string  `json:"updatedAt"`
	Name        string  `json:"name"`
	Picture     *string `json:"picture"`
	Description *string `json:"description"`
}

type GetChatPreviewResponse struct {
	CreateChatResponse
	LastMessage *string `json:"last_message"`
}

type GetChatByIdResponse struct {
	CreateChatResponse
	UserKeys []UserKeyEntry `json:"userKeys"`
	Messages []MessageEntry `json:"messages"`
	Users    []UserEntry    `json:"users"`
}
