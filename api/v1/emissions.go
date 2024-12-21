package v1

import (
	"encoding/json"
	"io"
	"net/http"
	v2 "scope3proxy/internal/scope3/v2"
)

type MeasureRequestBodyRow struct {
	Country     string `json:"country,omitempty"`
	Channel     string `json:"channel,omitempty"`
	InventoryId string `json:"inventoryId" validate:"required"`
	Impressions string `json:"impressions" validate:"required"`
	UtcDatetime string `json:"utcDatetime" validate:"required"`
	Priority    int    `json:"priority"`
}

func (h *APIV1Handler) GetEmissionsBreakdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.notOk(w, r, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}
	// Read the request body
	defer r.Body.Close()
	requestBodyInBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.notOk(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	var requestBody v2.MeasureRequestBody
	err = json.Unmarshal(requestBodyInBytes, &requestBody)
	if err != nil {
		h.notOk(w, r, http.StatusBadRequest, "Invalid request body")
		h.logAppError("Unable to parse request body", r, &requestBodyInBytes, err)
		return
	}

	emissionsBreakdownByInventoryId, err := h.scope3APIClient.GetEmissionsBreakdown(requestBody)
	if err != nil {
		h.notOk(w, r, http.StatusInternalServerError, GenericCustomerError)
		h.logAppError("Unable to fetch emissions breakdown", r, &requestBodyInBytes, err)
		return
	}
	h.ok(w, r, emissionsBreakdownByInventoryId)
}
