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
	Description *string        `json:"description" validate:"omitempty"`
	UserKeys    []UserKeyEntry `json:"userKeys" validate:"required,dive"`
}

type ChatResponse struct {
	Id          string  `json:"id"`
	CreatedAt   string  `json:"createdAt"`
	UpdateAt    string  `json:"updatedAt"`
	Name        string  `json:"name"`
	Picture     *string `json:"picture"`
	Description *string `json:"description"`
	Key         string  `json:"key"`
	Iv          string  `json:"iv"`
}

type GetChatByIdResponse struct {
	ChatResponse
	UserKeys []UserKeyEntry `json:"userKeys"`
	Messages []MessageEntry `json:"messages"`
	Users    []UserEntry    `json:"users"`
}
