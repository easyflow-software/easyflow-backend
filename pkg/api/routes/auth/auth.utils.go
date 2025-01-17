package auth

import (
	"easyflow-backend/pkg/config"
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/logger"
	"easyflow-backend/pkg/minio"

	"gorm.io/gorm"
)

func generateNewProfilePictureUrl(logger *logger.Logger, cfg *config.Config, db *gorm.DB, user *database.User) {
	pictureUrl, err := minio.GenerateDownloadURL(logger, cfg, cfg.ProfilePictureBucketName, user.ID, 7*24*60*60)
	if err == nil {
		user.ProfilePicture = pictureUrl

		if err := db.Save(user).Error; err != nil {
			logger.PrintfWarning("Could not save the new ProfilePicture url for user: %s. Error: %s", user.ID, err)
		}
	}
}
