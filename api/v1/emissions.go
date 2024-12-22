package v1

import (
	"encoding/json"
	"io"
	"net/http"
	"scope3proxy/internal"
)

type MeasureRequestBody struct {
	Rows []MeasureRequestBodyRow `json:"rows"`
}

type MeasureRequestBodyRow struct {
	Country     string `json:"country,omitempty"`
	Channel     string `json:"channel,omitempty"`
	InventoryId string `json:"inventoryId" validate:"required"`
	Impressions int    `json:"impressions" validate:"required"`
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

	var requestBody MeasureRequestBody
	err = json.Unmarshal(requestBodyInBytes, &requestBody)
	if err != nil {
		h.notOk(w, r, http.StatusBadRequest, "Invalid request body")
		h.logAppError("Unable to parse request body", r, &requestBodyInBytes, err)
		return
	}

	var filters []internal.EmissionFilter
	for _, row := range requestBody.Rows {
		filters = append(filters, internal.EmissionFilter{
			Country:     row.Country,
			Channel:     row.Channel,
			InventoryId: row.InventoryId,
			Impressions: row.Impressions,
			UtcDatetime: row.UtcDatetime,
			Priority:    row.Priority,
		})
	}

	result, err := h.emissionService.GetEmissions(filters)
	if err != nil {
		h.notOk(w, r, http.StatusInternalServerError, GenericClientError)
		h.logAppError("Unable to fetch emissions breakdown", r, &requestBodyInBytes, err)
		return
	}
	h.ok(w, r, result)
}
