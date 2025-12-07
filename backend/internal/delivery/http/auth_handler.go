package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"control_page/internal/adaptor"
	"control_page/internal/usecase"
)

type AuthHandler struct {
	authUseCase adaptor.AuthUseCase
}

func NewAuthHandler(authUseCase adaptor.AuthUseCase) *AuthHandler {
	return &AuthHandler{authUseCase: authUseCase}
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ActivateAccountRequest struct {
	UserID int64  `json:"user_id"`
	Code   string `json:"code"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type VerifyTOTPRequest struct {
	UserID int64  `json:"user_id"`
	Code   string `json:"code"`
}

type SetupTOTPRebindRequest struct {
	Password string `json:"password"`
}

type ConfirmTOTPRebindRequest struct {
	Code string `json:"code"`
}

type LoginResponse struct {
	RequiresTOTP      bool   `json:"requires_totp"`
	RequiresTOTPSetup bool   `json:"requires_totp_setup"`
	Token             string `json:"token,omitempty"`
	User              any    `json:"user,omitempty"`
	TempUserID        int64  `json:"temp_user_id,omitempty"`
	TOTPSetup         any    `json:"totp_setup,omitempty"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "username, email and password are required"})
		return
	}

	if len(req.Password) < 6 {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "password must be at least 6 characters"})
		return
	}

	result, err := h.authUseCase.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrUserAlreadyExists) {
			WriteJSON(w, http.StatusConflict, ErrorResponse{Error: "user already exists"})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to register user"})
		return
	}

	WriteJSON(w, http.StatusCreated, SuccessResponse{
		Message: "user registered, please setup 2FA to activate your account",
		Data:    result,
	})
}

func (h *AuthHandler) ActivateAccount(w http.ResponseWriter, r *http.Request) {
	var req ActivateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.UserID == 0 || req.Code == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "user_id and code are required"})
		return
	}

	err := h.authUseCase.ActivateAccount(r.Context(), req.UserID, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
		case errors.Is(err, usecase.ErrInvalidTOTPCode):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid verification code"})
		case errors.Is(err, usecase.ErrTOTPNotSetup):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "2FA is not set up"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to activate account"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Message: "account activated successfully"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "username and password are required"})
		return
	}

	result, err := h.authUseCase.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound), errors.Is(err, usecase.ErrInvalidCredentials):
			WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid username or password"})
		case errors.Is(err, usecase.ErrUserInactive):
			WriteJSON(w, http.StatusForbidden, ErrorResponse{Error: "user account is inactive"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to login"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, LoginResponse{
		RequiresTOTP:      result.RequiresTOTP,
		RequiresTOTPSetup: result.RequiresTOTPSetup,
		Token:             result.Token,
		User:              result.User,
		TempUserID:        result.TempUserID,
		TOTPSetup:         result.TOTPSetup,
	})
}

func (h *AuthHandler) VerifyTOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifyTOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.UserID == 0 || req.Code == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "user_id and code are required"})
		return
	}

	token, user, err := h.authUseCase.VerifyTOTP(r.Context(), req.UserID, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid user"})
		case errors.Is(err, usecase.ErrInvalidTOTPCode):
			WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid verification code"})
		case errors.Is(err, usecase.ErrTOTPNotSetup):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "2FA is not enabled"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to verify code"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, LoginResponse{
		RequiresTOTP: false,
		Token:        token,
		User:         user,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	WriteJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "current_password and new_password are required"})
		return
	}

	if len(req.NewPassword) < 6 {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "new password must be at least 6 characters"})
		return
	}

	err := h.authUseCase.ChangePassword(r.Context(), user.ID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrIncorrectPassword):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "current password is incorrect"})
		case errors.Is(err, usecase.ErrPasswordSameAsOld):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "new password cannot be the same as current password"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to change password"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Message: "password changed successfully"})
}

func (h *AuthHandler) SetupTOTPRebind(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	var req SetupTOTPRebindRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Password == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "password is required"})
		return
	}

	setup, err := h.authUseCase.SetupTOTPRebind(r.Context(), user.ID, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrIncorrectPassword):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "incorrect password"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to setup 2FA rebind"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{
		Message: "2FA rebind initiated, scan the QR code and verify",
		Data:    setup,
	})
}

func (h *AuthHandler) ConfirmTOTPRebind(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	var req ConfirmTOTPRebindRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Code == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "verification code is required"})
		return
	}

	err := h.authUseCase.ConfirmTOTPRebind(r.Context(), user.ID, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrTOTPNotSetup):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "please initiate 2FA rebind first"})
		case errors.Is(err, usecase.ErrInvalidTOTPCode):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid verification code"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to confirm 2FA rebind"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Message: "2FA rebind successful"})
}

func (h *AuthHandler) CancelTOTPRebind(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	err := h.authUseCase.CancelTOTPRebind(r.Context(), user.ID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to cancel 2FA rebind"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Message: "2FA rebind cancelled"})
}
