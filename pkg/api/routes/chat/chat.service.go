package chat

import (
	"easyflow-backend/pkg/api/errors"
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/enum"
	"easyflow-backend/pkg/jwt"
	"easyflow-backend/pkg/logger"

	"net/http"
	"slices"

	"gorm.io/gorm"
)

func createChat(db *gorm.DB, payload *CreateChatRequest, jwtPayload *jwt.JWTTokenPayload, logger *logger.Logger) (*ChatResponse, *errors.ApiError) {
	chat := database.Chat{
		Name:        payload.Name,
		Description: payload.Description,
	}

	// Start a transaction
	tx := db.Begin()

	if err := tx.Create(&chat).Error; err != nil {
		tx.Rollback()
		logger.PrintfError("Error creating chat: %s", err)
		return nil, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	var chatUserKeys []database.ChatsUsers

	for _, user := range payload.UserKeys {
		chatUserKey := database.ChatsUsers{
			ChatID: chat.ID,
			UserID: user.UserID,
			Key:    user.Key,
		}
		chatUserKeys = append(chatUserKeys, chatUserKey)
	}

	if err := tx.CreateInBatches(&chatUserKeys, 100).Error; err != nil {
		tx.Rollback()
		logger.PrintfError("Error creating chat user key: %s", err)
		return nil, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	if err := tx.Commit().Error; err != nil {
		logger.PrintfError("Error committing transaction: %s", err)
		return nil, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	logger.Printf("Successfully created chat with id: %s", chat.ID)

	keyIndex := slices.IndexFunc(chatUserKeys, func(chatUser database.ChatsUsers) bool {
		return chatUser.UserID == jwtPayload.UserID
	})
	if keyIndex == -1 {
		logger.PrintfError("Error getting chat user key for user")
		return nil, &errors.ApiError{
			Code:    http.StatusBadRequest,
			Error:   enum.MalformedRequest,
			Details: "Users and keys are mismatched or missing",
		}
	}

	key := chatUserKeys[keyIndex].Key

	return &ChatResponse{
		Id:          chat.ID,
		CreatedAt:   chat.CreatedAt.String(),
		UpdateAt:    chat.UpdatedAt.String(),
		Name:        chat.Name,
		Picture:     chat.Picture,
		Description: chat.Description,
		Key:         key,
	}, nil
}

func getChatPreviews(db *gorm.DB, jwtPayload *jwt.JWTTokenPayload, logger *logger.Logger) ([]ChatResponse, *errors.ApiError) {
	logger.PrintfInfo("Attempting to get chat previews for user: %s", jwtPayload.UserID)
	var chatUserKeys []database.ChatsUsers
	chatPreviews := []ChatResponse{}

	if err := db.Where("user_id = ?", jwtPayload.UserID).Find(&chatUserKeys).Error; err != nil {
		logger.PrintfError("Error getting chats for user: %s", err)
		return nil, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	for _, chatUserKey := range chatUserKeys {
		var chat database.Chat
		if err := db.Where("id = ?", chatUserKey.ChatID).First(&chat).Error; err != nil {
			logger.PrintfError("Error getting chat with id: %s. %s", chatUserKey.ChatID, err)
			return nil, &errors.ApiError{
				Code:  http.StatusInternalServerError,
				Error: enum.ApiError,
			}
		}

		var lastMessage database.Message
		if err := db.Where("chat_id = ?", chatUserKey.ChatID).Order("created_at desc").First(&lastMessage).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				// If there's another error, log it and return
				logger.PrintfError("Error getting last message for chat with id: %s. Error: %s", chatUserKey.ChatID, err.Error())
				return nil, &errors.ApiError{
					Code:  http.StatusInternalServerError,
					Error: enum.ApiError,
				}
			}

			var createChatResponse = ChatResponse{
				Id:          chat.ID,
				CreatedAt:   chat.CreatedAt.String(),
				UpdateAt:    chat.UpdatedAt.String(),
				Name:        chat.Name,
				Picture:     chat.Picture,
				Description: chat.Description,
				Key:         chatUserKey.Key,
			}

			chatPreviews = append(chatPreviews, createChatResponse)
		}
	}

	logger.Printf("Successfully got chat previews for user: %s", jwtPayload.UserID)

	return chatPreviews, nil
}

func getChatById(db *gorm.DB, chatId string, jwtPayload *jwt.JWTTokenPayload, logger *logger.Logger) (*GetChatByIdResponse, *errors.ApiError) {
	var chat database.Chat
	if err := db.Where("id = ?", chatId).First(&chat).Error; err != nil {
		logger.PrintfError("Error getting chat with id: %s. Error: %s", chatId, err)
		return nil, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	var chatUserKeys []database.ChatsUsers
	if err := db.Where("chat_id = ? AND user_id = ?", chatId, jwtPayload.UserID).Find(&chatUserKeys).Error; err != nil {
		logger.PrintfError("Error getting chat user key for chat with id: %s. Error: %s", chatId, err)
		return nil, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	var Messages []database.Message
	if err := db.Where("chat_id = ?", chatId).Order("created_at desc").Find(&Messages).Limit(50).Error; err != nil {
		logger.PrintfError("Error getting messages for chat with id: %s. Error: %s", chatId, err)
		return nil, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	// Mappings
	usersEntries := []UserEntry{}
	for _, chatUserKey := range chatUserKeys {
		var user database.User
		if err := db.Where("id = ?", chatUserKey.UserID).First(&user).Error; err != nil {
			logger.PrintfError("Error getting user with id: %s. Error: %s", chatUserKey.UserID, err)
			return nil, &errors.ApiError{
				Code:  http.StatusInternalServerError,
				Error: enum.ApiError,
			}
		}

		usersEntries = append(usersEntries,
			UserEntry{
				Id:   user.ID,
				Name: user.Name,
				Bio:  user.Bio,
			},
		)
	}

	messageEntries := []MessageEntry{}
	for _, message := range Messages {
		messageEntries = append(messageEntries,
			MessageEntry{
				Id:        message.ID,
				CreatedAt: message.CreatedAt.String(),
				UpdatedAt: message.UpdatedAt.String(),
				Content:   message.Content,
				Iv:        message.Iv,
				SenderId:  message.SenderID,
			},
		)
	}

	logger.Printf("Successfully got chat with id: %s", chatId)

	return &GetChatByIdResponse{
		ChatResponse: ChatResponse{
			Id:          chat.ID,
			CreatedAt:   chat.CreatedAt.String(),
			UpdateAt:    chat.UpdatedAt.String(),
			Name:        chat.Name,
			Picture:     chat.Picture,
			Description: chat.Description,
		},
		Users:    usersEntries,
		Messages: messageEntries,
	}, nil

}
