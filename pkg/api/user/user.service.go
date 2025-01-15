package user

import (
	"errors"
	"net/http"

	"easyflow-backend/pkg/api"
	"easyflow-backend/pkg/api/auth"
	"easyflow-backend/pkg/api/s3"
	"easyflow-backend/pkg/common"
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/enum"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func CreateUser(db *gorm.DB, payload *CreateUserRequest, cfg *common.Config, logger *common.Logger) (*database.User, *api.ApiError) {
	var user database.User
	if err := db.Where("email = ?", payload.Email).First(&user).Error; err == nil {
		logger.PrintfError("User with email: %s already exists", payload.Email)
		return nil, &api.ApiError{
			Code:  http.StatusConflict,
			Error: enum.AlreadyExists,
		}
	}

	password, err := bcrypt.GenerateFromPassword([]byte(payload.Password), cfg.SaltRounds)
	if err != nil {
		logger.PrintfError("Error hashing password: %s", err)
		return nil, &api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	//create a new user
	user = database.User{
		Email:      payload.Email,
		Name:       payload.Name,
		Password:   string(password),
		PublicKey:  payload.PublicKey,
		PrivateKey: payload.PrivateKey,
		Iv:         payload.Iv,
	}

	if err := db.Create(&user).Error; err != nil {
		logger.PrintfError("Error creating user: %s", err)
		return nil, &api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	return &user, nil
}

func GetUserById(db *gorm.DB, jwtPayload *auth.JWTAccessTokenPayload, logger *common.Logger) (*database.User, *api.ApiError) {
	var user database.User
	if err := db.Where("id = ?", jwtPayload.UserId).First(&user).Error; err != nil {
		logger.PrintfError("Error getting user: %s", err)
		return nil, &api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	logger.Printf("Successfully got user: %s", user.Id)

	return &user, nil
}

func GetUserByEmail(db *gorm.DB, email string, logger *common.Logger) (bool, *api.ApiError) {
	var user database.User
	err := db.Where("email = ?", email).First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		logger.PrintfInfo("No user with email: %s found", err)
		return false, nil
	}

	if err != nil {
		logger.PrintfInfo("An error occured while trying to find user: %s ", err)
		return false, &api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	logger.PrintfInfo("User with email: %s found", email)

	return true, nil
}

func GenerateGetProfilePictureURL(db *gorm.DB, jwtPayload *auth.JWTAccessTokenPayload, logger *common.Logger, cfg *common.Config) (*string, *api.ApiError) {
	var user database.User
	if err := db.Where("id = ?", jwtPayload.UserId).First(&user).Error; err != nil {
		logger.PrintfError("Error getting user: %s", err)
		return nil, &api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.NotFound,
		}
	}

	imageURL, err := s3.GenerateDownloadURL(logger, cfg, cfg.ProfilePictureBucketName, user.Id, 60*60*24*7) // 1 week expiration time
	if err != nil {
		return nil, err
	}

	user.ProfilePicture = imageURL
	if err := db.Update(user.Id, &user).Error; err != nil {
		logger.PrintfError("Error saving user: %s", err)
		return nil, &api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	logger.Printf("Successfully generated profile picture URL for user: %s", user.Id)

	return imageURL, nil
}

func GenerateUploadProfilePictureURL(db *gorm.DB, jwtPayload *auth.JWTAccessTokenPayload, logger *common.Logger, cfg *common.Config) (*string, *api.ApiError) {
	var user database.User
	if err := db.Where("id = ?", jwtPayload.UserId).First(&user).Error; err != nil {
		logger.PrintfError("Error getting user: %s", err)
		return nil, &api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.NotFound,
		}
	}

	uploadURL, err := s3.GenerateUploadURL(logger, cfg, cfg.ProfilePictureBucketName, user.Id, 60*60)
	if err != nil {
		logger.PrintfError("Error uploading profile picture: %s", err.Error)
		return nil, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	logger.Printf("Successfully generated profile picture upload URL for user: %s", user.Id)

	return uploadURL, nil
}

func UpdateUser(db *gorm.DB, jwtPayload *auth.JWTAccessTokenPayload, payload *UpdateUserRequest, logger *common.Logger) (*database.User, *api.ApiError) {
	var user database.User
	if err := db.Where("id = ?", jwtPayload.UserId).First(&user).Error; err != nil {
		logger.PrintfError("Error getting user: %s", err)
		return nil, &api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.NotFound,
		}
	}

	if payload.Name != nil {
		user.Name = *payload.Name
	}
	if payload.Bio != nil {
		user.Bio = payload.Bio
	}

	if err := db.Update(user.Id, &user).Error; err != nil {
		logger.PrintfError("Error updating user: %s", err)
		return nil, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err.Error(),
		}
	}

	logger.Printf("Successfully updated user: %s", user.Id)

	return &user, nil
}

func DeleteUser(db *gorm.DB, jwtPayload *auth.JWTAccessTokenPayload, logger *common.Logger) *api.ApiError {
	var user database.User
	if err := db.Where("id = ?", jwtPayload.UserId).First(&user).Error; err != nil {
		logger.PrintfError("Error getting user: %s", err)
		return &api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.NotFound,
		}
	}

	if err := db.Delete(&user).Error; err != nil {
		logger.PrintfError("Error deleting user: %s", err)
		return &api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	logger.Printf("Successfully deleted user: %s", user.Id)

	return nil
}
