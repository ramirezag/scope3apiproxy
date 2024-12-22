package v1

import (
	"encoding/json"
	"go.uber.org/zap"
	"net/http"
	"scope3proxy/internal"
)

const GenericClientError = "Something went wrong. Please try again later or contact scope3 team."
const GenericLogUnsentResponseError = "Unable to send the response payload to customers"
const LoggerKeyRequestMethod = "requestMethod"
const LoggerKeyRequestUrl = "requestUrl"

type APIV1Handler struct {
	logger          *zap.Logger
	emissionService *internal.EmissionService
	*http.ServeMux
}

func NewHandler(logger *zap.Logger, emissionService *internal.EmissionService) http.Handler {
	handler := &APIV1Handler{logger, emissionService, http.NewServeMux()}
	handler.HandleFunc("/api/v1/emissions", handler.GetEmissionsBreakdown)
	return handler
}

func (h *APIV1Handler) ok(
	w http.ResponseWriter,
	r *http.Request,
	result interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(APIResult{Data: result}); err != nil {
		// Less likely to happen, but it is best to handle errors even in extreme cases.
		// In this case, we log the error with details for observability
		h.logger.Error(GenericLogUnsentResponseError,
			zap.Error(err),
			zap.String(LoggerKeyRequestMethod, r.Method),
			zap.String(LoggerKeyRequestUrl, r.URL.String()),
		)
	}
}

func (h *APIV1Handler) notOk(
	w http.ResponseWriter,
	r *http.Request,
	code int,
	errorMessage string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(APIResult{Error: errorMessage}); err != nil {
		// Less likely to happen, but it is best to handle errors even in extreme cases.
		// In this case, we log the error with details for observability
		h.logger.Error(GenericLogUnsentResponseError,
			zap.Error(err),
			zap.String(LoggerKeyRequestMethod, r.Method),
			zap.String(LoggerKeyRequestUrl, r.URL.String()),
		)
	}
}

func (h *APIV1Handler) logAppError(
	errorMessage string,
	r *http.Request,
	requestBody *[]byte,
	err error,
) {
	var fields = []zap.Field{
		zap.Error(err),
		zap.String(LoggerKeyRequestMethod, r.Method),
		zap.String(LoggerKeyRequestUrl, r.URL.String()),
	}
	if requestBody != nil {
		fields = append(fields, zap.ByteString("requestBody", *requestBody))
	}
	h.logger.Error(errorMessage, append(fields, zap.Error(err))...)
}

type APIResult struct {
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}
