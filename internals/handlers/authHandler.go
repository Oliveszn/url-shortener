package handlers

import (
	"net/http"
	"url-shortener/internals/dtos"
	"url-shortener/internals/services"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type AuthHandler struct {
	BaseHandler
	service *services.AuthService
}

func NewAuthHandler(db *mongo.Database, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		BaseHandler: BaseHandler{
			Logger: logger,
		},
		service: services.NewAuthService(db, logger),
	}
}

// @Summary Register a new user
// @Description Register a new user with the provided details
// @Tags auth
// @Accept json
// @Produce json
// @Param user body dtos.RegisterUserDto true "User registration data"
// @Success 201 {object} dtos.StructuredResponse "User registered successfully"
// @Failure 500 {object} dtos.StructuredResponse "Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	h.Logger.Info("Register request received")

	var req dtos.RegisterUserDto

	if !h.DecodeJSONBody(w, r, &req) {
		return
	}

	h.Logger.Debug("Registering user", zap.String("email", req.Email))

	response, err := h.service.RegisterUser(r.Context(), req)

	if err != nil {
		h.Logger.Warn("Failed to register user", zap.Error(err))
		h.ReturnJSONResponse(w, dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
			Payload: nil,
		})
		return
	}

	h.ReturnJSONResponse(w, response)
}

// @Summary Login a user
// @Description Login a user with the provided credentials
// @Tags auth
// @Accept json
// @Produce json
// @Param user body dtos.LoginUserDto true "User login data"
// @Success 200 {object} dtos.StructuredResponse "User logged in successfully"
// @Failure 401 {object} dtos.StructuredResponse "Invalid credentials"
// @Failure 500 {object} dtos.StructuredResponse "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	h.Logger.Info("Login request received")

	var req dtos.LoginUserDto

	if !h.DecodeJSONBody(w, r, &req) {
		return
	}

	h.Logger.Debug("Logging in user", zap.String("email", req.Email))

	response, err := h.service.LoginUser(r.Context(), req)

	if err != nil {
		h.Logger.Warn("Failed to login user", zap.Error(err))
		h.ReturnJSONResponse(w, dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
			Payload: nil,
		})
		return
	}

	h.ReturnJSONResponse(w, response)
}

// @Summary Logout user
// @Description Logs out the current user
// @Tags auth
// @Produce json
// @Success 200 {object} dtos.StructuredResponse "Logout successful"
// @Failure 500 {object} dtos.StructuredResponse "Internal server error"
// @Router /auth/logout [post]
func (h *AuthHandler) LogoutUser(w http.ResponseWriter, r *http.Request) {
	h.Logger.Info("Logout request received")

	response, err := h.service.LogoutUser(r.Context())

	if err != nil {
		h.Logger.Error("Logout failed", zap.Error(err))
	}

	h.ReturnJSONResponse(w, response)
}
