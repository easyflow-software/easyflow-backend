package user

import (
	"easyflow-backend/pkg/api/errors"
	"easyflow-backend/pkg/api/turnstile"
	"easyflow-backend/pkg/config"
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/enum"
	"easyflow-backend/pkg/jwt"
	"easyflow-backend/pkg/logger"
	"easyflow-backend/pkg/minio"

	e "errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func createUser(db *gorm.DB, payload *CreateUserRequest, cfg *config.Config, logger *logger.Logger, ip string) (*database.User, *errors.ApiError) {
	ok, checkTurnstileErr := turnstile.CheckCloudflareTurnstile(logger, cfg, ip, payload.TurnstileToken)
	if !ok {
		return nil, checkTurnstileErr
	}

	var user database.User
	if err := db.Where("email = ?", payload.Email).First(&user).Error; err == nil {
		logger.PrintfError("User with email: %s already exists", payload.Email)
		return nil, &errors.ApiError{
			Code:  http.StatusConflict,
			Error: enum.AlreadyExists,
		}
	}

	password, err := bcrypt.GenerateFromPassword([]byte(payload.Password), cfg.SaltRounds)
	if err != nil {
		logger.PrintfError("Error hashing password: %s", err)
		return nil, &errors.ApiError{
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
		return nil, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	return &user, nil
}

func getUserById(db *gorm.DB, jwtPayload *jwt.JWTTokenPayload, logger *logger.Logger) (*database.User, *errors.ApiError) {
	var user database.User
	if err := db.Where("id = ?", jwtPayload.UserID).First(&user).Error; err != nil {
		logger.PrintfError("Error getting user: %s", err)
		return nil, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	logger.Printf("Successfully got user: %s", user.ID)

	return &user, nil
}

func getUserByEmail(db *gorm.DB, email string, logger *logger.Logger) (bool, *errors.ApiError) {
	var user database.User
	err := db.Where("email = ?", email).First(&user).Error

	if e.Is(err, gorm.ErrRecordNotFound) {
		logger.PrintfInfo("No user with email: %s found", err)
		return false, nil
	}

	if err != nil {
		logger.PrintfInfo("An error occured while trying to find user: %s ", err)
		return false, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	logger.PrintfInfo("User with email: %s found", email)

	return true, nil
}

func getProfilePictureURL(db *gorm.DB, jwtPayload *jwt.JWTTokenPayload, logger *logger.Logger) (*string, *errors.ApiError) {
	var user database.User
	if err := db.Where("id = ?", jwtPayload.UserID).First(&user).Error; err != nil {
		logger.PrintfError("Error getting user: %s", err)
		return nil, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.NotFound,
		}
	}

	logger.PrintfInfo("Successfully retrieved profile picture URL for user: %s", user.ID)

	return user.ProfilePicture, nil
}

func generateUploadProfilePictureURL(db *gorm.DB, jwtPayload *jwt.JWTTokenPayload, logger *logger.Logger, cfg *config.Config) (*string, *errors.ApiError) {
	var user database.User
	if err := db.Where("id = ?", jwtPayload.UserID).First(&user).Error; err != nil {
		logger.PrintfError("Error getting user: %s", err)
		return nil, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.NotFound,
		}
	}

	uploadURL, err := minio.GenerateUploadURL(logger, cfg, cfg.ProfilePictureBucketName, user.ID, 60*60)
	if err != nil {
		logger.PrintfError("Error uploading profile picture: %s", err.Error)
		return nil, &errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	logger.Printf("Successfully generated profile picture upload URL for user: %s", user.ID)

	return uploadURL, nil
}

func updateUser(db *gorm.DB, jwtPayload *jwt.JWTTokenPayload, payload *UpdateUserRequest, logger *logger.Logger) (*database.User, *errors.ApiError) {
	var user database.User
	if err := db.Where("id = ?", jwtPayload.UserID).First(&user).Error; err != nil {
		logger.PrintfError("Error getting user: %s", err)
		return nil, &errors.ApiError{
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

	if err := db.Update(user.ID, &user).Error; err != nil {
		logger.PrintfError("Error updating user: %s", err)
		return nil, &errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err.Error(),
		}
	}

	logger.Printf("Successfully updated user: %s", user.ID)

	return &user, nil
}

func deleteUser(db *gorm.DB, jwtPayload *jwt.JWTTokenPayload, logger *logger.Logger) *errors.ApiError {
	var user database.User
	if err := db.Where("id = ?", jwtPayload.UserID).First(&user).Error; err != nil {
		logger.PrintfError("Error getting user: %s", err)
		return &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.NotFound,
		}
	}

	if err := db.Delete(&user).Error; err != nil {
		logger.PrintfError("Error deleting user: %s", err)
		return &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	logger.Printf("Successfully deleted user: %s", user.ID)

	return nil
}
