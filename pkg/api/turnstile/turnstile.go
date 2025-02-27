package turnstile

import (
	"easyflow-backend/pkg/api/errors"
	"easyflow-backend/pkg/config"
	"easyflow-backend/pkg/enum"
	"easyflow-backend/pkg/logger"

	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

func CheckCloudflareTurnstile(logger *logger.Logger, cfg *config.Config, ip string, token string) (bool, *errors.ApiError) {
	formData := url.Values{}
	formData.Add("secret", cfg.TurnstileSecret)
	formData.Add("response", token)
	formData.Add("remoteip", ip)

	res, err := http.PostForm(cfg.TurnstileUrl, formData)
	if err != nil {
		logger.PrintfError("Error verifying turnstile token: %s", err)
		return false, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.PrintfError("Error reading turnstile response: %s", err)
		return false, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}
	var jsonBody CloudflareTurnstileResponse
	err = json.Unmarshal(body, &jsonBody)
	if err != nil {
		logger.PrintfError("Error unmarshalling turnstile response: %s", err)
		return false, &errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		}
	}

	logger.PrintfDebug("Action: %s", jsonBody.Action)

	if !jsonBody.Success {
		logger.PrintfWarning("Turnstile token verification failed: %s", jsonBody.ErrorCodes)
		return false, &errors.ApiError{
			Code:    http.StatusUnauthorized,
			Error:   enum.InvalidTurnstile,
			Details: "Failed to validate the provided Turnstile token",
		}
	}

	return true, nil
}
