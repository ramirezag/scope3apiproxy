package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type MeasureFilterRow struct {
	Country     string `json:"country,omitempty"`
	Channel     string `json:"channel,omitempty"`
	InventoryId string `json:"inventoryId" validate:"required"`
	Impressions int    `json:"impressions" validate:"required"`
	UtcDatetime string `json:"utcDatetime" validate:"required"`
}

type measureResponse struct {
	Rows []measureRow `json:"rows"`
}

type measureRow struct {
	// scope3 returns HTTP 200 but set the error message for field validation issues (eg, missing or < 1 impressions)
	Error              scope3Error            `json:"error,omitempty"`
	EmissionsBreakdown map[string]interface{} `json:"emissionsBreakdown,omitempty"`
	Internal           map[string]interface{} `json:"internal,omitempty"`
}

type scope3Error struct {
	Message string `json:"message"`
}

func (s *Scope3APIClient) GetEmissionsBreakdown(rows *[]MeasureFilterRow) (map[string]interface{}, error) {
	requestBodyInBytes, err := json.Marshal(map[string]interface{}{
		"rows": *rows,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall request body: %w", err)
	}

	url := s.baseUrl + "/measure?includeRows=true&latest=true&fields=emissionsBreakdown"

	resp, err := s.doPost(url, requestBodyInBytes)
	if err != nil {
		// Maybe server is unreachable
		return nil, Scope3ServerError{
			Message: "Failed to call scope3 measure api",
			Err:     err,
		}
	}
	responseBodyInBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read the response body: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("scope3 server returns http status " + strconv.Itoa(resp.StatusCode) +
			" with response body: " + string(responseBodyInBytes))
	}
	var responseBody measureResponse
	err = json.Unmarshal(responseBodyInBytes, &responseBody)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall scope3 measure api response: %w", err)
	}

	result := make(map[string]interface{}, len(responseBody.Rows))
	for _, row := range responseBody.Rows {
		if row.Error.Message == "" {
			propertyName := row.Internal["propertyName"].(string)
			result[propertyName] = row.EmissionsBreakdown["breakdown"]
		} else {
			return nil, fmt.Errorf("scope3 server request error: %s", row.Error.Message)
		}
	}
	return result, nil
}
