package auth

import (
	"easyflow-backend/pkg/api"
	"easyflow-backend/pkg/api/utils"
	"easyflow-backend/pkg/common"
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/enum"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func generateJwt[T interface{ jwt.Claims }](cfg *common.Config, payload T) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	signedToken, err := token.SignedString([]byte(cfg.JwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

func ValidateToken(cfg *common.Config, token string) (*JWTAccessTokenPayload, error) {
	var claims JWTAccessTokenPayload
	_, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(token *jwt.Token) (interface{}, error) {
			// Verify that the signing method is what we expect
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.JwtSecret), nil
		},
	)

	if err != nil {
		return nil, err
	}

	return &claims, nil
}

func LoginService(db *gorm.DB, cfg *common.Config, payload *LoginRequest, logger *common.Logger) (JWTPair, *api.ApiError) {
	var user database.User
	if err := db.Where("email = ?", payload.Email).First(&user).Error; err != nil {
		logger.PrintfWarning("User with email: %s not found", payload.Email)
		return JWTPair{}, &api.ApiError{
			Code:    http.StatusUnauthorized,
			Error:   enum.WrongCredentials,
			Details: err,
		}
	}

	//check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.Password)); err != nil {
		logger.PrintfWarning("Wrong password for user with email: %s", payload.Email)
		return JWTPair{}, &api.ApiError{
			Code:    http.StatusUnauthorized,
			Error:   enum.WrongCredentials,
			Details: err,
		}
	}

	random := uuid.New()
	expires := time.Now().Add(time.Duration(cfg.JwtExpirationTime) * time.Second)
	refreshExpires := time.Now().Add(time.Duration(cfg.RefreshExpirationTime) * time.Second)

	accessTokenPayload := JWTAccessTokenPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expires),
			Issuer:    "easyflow",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserId:      user.Id,
		RefreshRand: &random,
	}

	refreshTokenPayload := JWTAccessTokenPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpires),
			Issuer:    "easyflow",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserId:      user.Id,
		RefreshRand: &random,
	}

	accessToken, err := generateJwt[JWTAccessTokenPayload](cfg, accessTokenPayload)

	if err != nil {
		logger.PrintfError("Error generating jwt: %s", err)
		return JWTPair{}, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	refreshToken, err := generateJwt[JWTAccessTokenPayload](cfg, refreshTokenPayload)

	if err != nil {
		logger.PrintfError("Error generating jwt: %s", err)
		return JWTPair{}, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	//write refresh token to db
	entry := database.UserKeys{
		Random:    random.String(),
		ExpiredAt: refreshExpires,
		UserId:    user.Id,
	}

	if err := db.Save(&entry).Error; err != nil {
		logger.PrintfError("Error updating user key: %s", err)
		return JWTPair{}, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	if user.ProfilePicture == nil {
		utils.GenerateNewProfilePictureUrl(logger, cfg, db, &user)
	} else {
		expired := false

		pictureUrl, err := url.Parse(*user.ProfilePicture)
		if err != nil {
			expired = true
		}

		query := pictureUrl.Query()
		issuedAt, err := time.Parse(time.RFC3339, query.Get("X-Amz-Date"))
		if err != nil {
			expired = true
		}
		expiryTime, err := strconv.ParseInt(query.Get("X-Amz-Expires"), 10, 64)
		if err != nil {
			expired = true
		}

		if issuedAt.Add(time.Duration(expiryTime) * time.Second).After(time.Now()) {
			expired = true
		}

		if expired {
			utils.GenerateNewProfilePictureUrl(logger, cfg, db, &user)
		}

	}

	logger.Printf("Logged in user: %s", user.Id)

	return JWTPair{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, nil
}

func RefreshService(db *gorm.DB, cfg *common.Config, payload *JWTAccessTokenPayload, logger *common.Logger) (JWTPair, *api.ApiError) {
	//get user from db
	var user database.User
	if err := db.First(&user, "id = ?", payload.UserId).Error; err != nil {
		logger.PrintfWarning("Could not get user with id: %s", payload.UserId)
		return JWTPair{}, &api.ApiError{
			Code:    http.StatusUnauthorized,
			Error:   enum.Unauthorized,
			Details: err,
		}
	}

	random := uuid.New()
	expires := time.Now().Add(time.Duration(cfg.JwtExpirationTime) * time.Second)
	refreshExpires := time.Now().Add(time.Duration(cfg.RefreshExpirationTime) * time.Second)

	accessTokenPayload := JWTAccessTokenPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expires),
			Issuer:    "easyflow",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserId:      user.Id,
		RefreshRand: &random,
	}

	refreshTokenPayload := JWTAccessTokenPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpires),
			Issuer:    "easyflow",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserId:      user.Id,
		RefreshRand: &random,
	}

	accessToken, err := generateJwt(cfg, &accessTokenPayload)
	if err != nil {
		logger.PrintfError("Error generating jwt: %s", err)
		return JWTPair{}, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	refreshToken, err := generateJwt(cfg, &refreshTokenPayload)
	if err != nil {
		logger.PrintfError("Error generating jwt: %s", err)
		return JWTPair{}, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	//write refresh token random to db
	err = db.Model(database.UserKeys{}).Where(
		&database.UserKeys{
			UserId: payload.UserId,
			Random: payload.RefreshRand.String(),
		},
	).Updates(
		database.UserKeys{
			Random:    random.String(),
			ExpiredAt: refreshExpires,
		}).Error

	if err != nil {
		logger.PrintfError("Error updating user key with user id: %s and random: %s", payload.UserId, payload.RefreshRand)

	}

	logger.Printf("Refreshed token for user with id: %s", payload.UserId)

	return JWTPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func LogoutService(db *gorm.DB, payload *JWTAccessTokenPayload, logger *common.Logger) *api.ApiError {
	if err := db.Delete(&database.UserKeys{}, payload.RefreshRand).Error; err != nil {
		logger.PrintfError("Could not delete Refresh Token with random: %s and user id: %s", payload.RefreshRand, payload.UserId)
		return &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	logger.Printf("Successfully ended session for user with id: %s and random: %s", payload.UserId, payload.RefreshRand)

	return nil
}
