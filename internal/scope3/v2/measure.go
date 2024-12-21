package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
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

type measureResponse struct {
	Rows []measureRow `json:"rows"`
}

type measureRow struct {
	EmissionsBreakdown map[string]interface{} `json:"emissionsBreakdown"`
	Internal           map[string]interface{} `json:"internal"`
}

func (s *Scope3APIClient) GetEmissionsBreakdown(requestBody MeasureRequestBody) (map[string]interface{}, error) {
	requestBodyInBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall request body: %w", err)
	}

	url := s.baseUrl + "/measure?includeRows=true&latest=true&fields=emissionsBreakdown"

	resp, err := s.doPost(url, requestBodyInBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to call scope3 measure api: %w", err)
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
		propertyName := row.Internal["propertyName"].(string)
		result[propertyName] = row.EmissionsBreakdown["breakdown"]
	}
	return result, nil
}
