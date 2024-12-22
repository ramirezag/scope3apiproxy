package v1

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"scope3apiproxy/internal"
	"scope3apiproxy/internal/cache"
	v2 "scope3apiproxy/internal/scope3/v2"
	"strings"
	"testing"
	"time"
)

// dummyEmissionInEachProperties represents the emission from scope3 measure API response.
//
//	{
//	 "rows": [
//	   {
//	     "emissionsBreakdown": {
//	       "breakdown": <whatever json fields that comes here>
//	     },
//	     "internal": {
//	       "propertyName": "nytimes.com"
//	     }
//	   }
//	 ]
//	}
const dummyEmissionInEachProperties = `{"someproperty1":"somevalue1"}`

func TestGetEmissions(t *testing.T) {
	t.Run("with uncached property", func(t *testing.T) {
		propertiesQueriedFromScope3APIServer := make(map[string]bool)
		scope3MockAPIServer := createMockHttpServerForEmissions(t, propertiesQueriedFromScope3APIServer)
		defer scope3MockAPIServer.Close()

		apiHandler, appCache := createTestApiHandler(scope3MockAPIServer.URL, 1)
		handler := http.HandlerFunc(apiHandler.getEmissions)

		propertyName := "nytimes.com"
		requestBody := emissionRequestBody{
			Rows: []EmissionRequestBodyRow{
				{
					InventoryId: propertyName,
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
				},
			},
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, createTestHttpRequest(t, requestBody))
		verifyPerPropertyEmissionAppResponse(t, rr, propertyName)
		verifyScope3APIServerCalls(t, propertiesQueriedFromScope3APIServer, propertyName)
		verifyCache(t, appCache, propertyName)
	})

	t.Run("with cached property", func(t *testing.T) {
		propertiesQueriedFromScope3APIServer := make(map[string]bool)
		scope3MockAPIServer := createMockHttpServerForEmissions(t, propertiesQueriedFromScope3APIServer)
		defer scope3MockAPIServer.Close()

		apiHandler, appCache := createTestApiHandler(scope3MockAPIServer.URL, 1)
		handler := http.HandlerFunc(apiHandler.getEmissions)

		propertyName := "nytimes.com"
		requestBody := emissionRequestBody{
			Rows: []EmissionRequestBodyRow{
				{
					InventoryId: propertyName,
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
				},
			},
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, createTestHttpRequest(t, requestBody))
		verifyPerPropertyEmissionAppResponse(t, rr, propertyName)
		verifyScope3APIServerCalls(t, propertiesQueriedFromScope3APIServer, propertyName)
		verifyCache(t, appCache, propertyName)

		clearPropertiesQueriedFromScope3APIServerMap(propertiesQueriedFromScope3APIServer)
		rr = httptest.NewRecorder()

		handler.ServeHTTP(rr, createTestHttpRequest(t, requestBody))
		verifyPerPropertyEmissionAppResponse(t, rr, propertyName)
		verifyNoScope3APIServerCalls(t, propertiesQueriedFromScope3APIServer)
		verifyCache(t, appCache, propertyName)
	})

	t.Run("with 1 cached property + 1 uncached property", func(t *testing.T) {
		propertiesQueriedFromScope3APIServer := make(map[string]bool)
		scope3MockAPIServer := createMockHttpServerForEmissions(t, propertiesQueriedFromScope3APIServer)
		defer scope3MockAPIServer.Close()

		apiHandler, appCache := createTestApiHandler(scope3MockAPIServer.URL, 2)
		handler := http.HandlerFunc(apiHandler.getEmissions)

		requestBody := emissionRequestBody{
			Rows: []EmissionRequestBodyRow{
				{
					InventoryId: "nytimes.com",
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
				},
			},
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, createTestHttpRequest(t, requestBody))
		verifyPerPropertyEmissionAppResponse(t, rr, "nytimes.com")
		verifyScope3APIServerCalls(t, propertiesQueriedFromScope3APIServer, "nytimes.com")
		verifyCache(t, appCache, "nytimes.com")

		clearPropertiesQueriedFromScope3APIServerMap(propertiesQueriedFromScope3APIServer)
		rr = httptest.NewRecorder()
		requestBody = emissionRequestBody{
			Rows: []EmissionRequestBodyRow{
				{
					InventoryId: "nytimes.com",
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
				},
				{
					InventoryId: "foxnews.com",
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
				},
			},
		}

		handler.ServeHTTP(rr, createTestHttpRequest(t, requestBody))
		verifyPerPropertyEmissionAppResponse(t, rr, "nytimes.com", "foxnews.com")
		verifyScope3APIServerCalls(t, propertiesQueriedFromScope3APIServer, "foxnews.com")
		verifyCache(t, appCache, "nytimes.com", "foxnews.com")
	})

	t.Run("with eviction based on least frequently used/queried (LFU)", func(t *testing.T) {
		propertiesQueriedFromScope3APIServer := make(map[string]bool)
		scope3MockAPIServer := createMockHttpServerForEmissions(t, propertiesQueriedFromScope3APIServer)
		defer scope3MockAPIServer.Close()

		apiHandler, appCache := createTestApiHandler(scope3MockAPIServer.URL, 2)
		handler := http.HandlerFunc(apiHandler.getEmissions)

		requestBody := emissionRequestBody{
			Rows: []EmissionRequestBodyRow{
				{
					InventoryId: "nytimes.com",
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
				},
			},
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, createTestHttpRequest(t, requestBody))
		verifyPerPropertyEmissionAppResponse(t, rr, "nytimes.com")
		verifyScope3APIServerCalls(t, propertiesQueriedFromScope3APIServer, "nytimes.com")
		verifyCache(t, appCache, "nytimes.com")

		clearPropertiesQueriedFromScope3APIServerMap(propertiesQueriedFromScope3APIServer)
		rr = httptest.NewRecorder()
		requestBody = emissionRequestBody{
			Rows: []EmissionRequestBodyRow{
				{
					InventoryId: "nytimes.com",
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
				},
				{
					InventoryId: "foxnews.com",
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
				},
			},
		}
		handler.ServeHTTP(rr, createTestHttpRequest(t, requestBody))
		verifyPerPropertyEmissionAppResponse(t, rr, "nytimes.com", "foxnews.com")
		verifyScope3APIServerCalls(t, propertiesQueriedFromScope3APIServer, "foxnews.com")
		verifyCache(t, appCache, "nytimes.com", "foxnews.com")

		clearPropertiesQueriedFromScope3APIServerMap(propertiesQueriedFromScope3APIServer)
		rr = httptest.NewRecorder()
		requestBody = emissionRequestBody{
			Rows: []EmissionRequestBodyRow{
				{
					InventoryId: "usatoday.com",
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
				},
			},
		}
		handler.ServeHTTP(rr, createTestHttpRequest(t, requestBody))
		verifyPerPropertyEmissionAppResponse(t, rr, "usatoday.com")
		verifyScope3APIServerCalls(t, propertiesQueriedFromScope3APIServer, "usatoday.com")
		verifyCache(t, appCache, "nytimes.com", "usatoday.com")
		_, exists := appCache.Get("foxnews.com" + internal.EmissionCacheKeySuffix)
		assert.False(t, exists, "foxnews.com should not be in cache")
	})

	t.Run("with priority based eviction", func(t *testing.T) {
		propertiesQueriedFromScope3APIServer := make(map[string]bool)
		scope3MockAPIServer := createMockHttpServerForEmissions(t, propertiesQueriedFromScope3APIServer)
		defer scope3MockAPIServer.Close()

		apiHandler, appCache := createTestApiHandler(scope3MockAPIServer.URL, 2)
		handler := http.HandlerFunc(apiHandler.getEmissions)

		requestBody := emissionRequestBody{
			Rows: []EmissionRequestBodyRow{
				{
					InventoryId: "nytimes.com",
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
					Priority:    1,
				},
				{
					InventoryId: "foxnews.com",
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
				},
			},
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, createTestHttpRequest(t, requestBody))
		verifyPerPropertyEmissionAppResponse(t, rr, "nytimes.com", "foxnews.com")
		verifyScope3APIServerCalls(t, propertiesQueriedFromScope3APIServer, "nytimes.com", "foxnews.com")
		verifyCache(t, appCache, "nytimes.com", "foxnews.com")

		clearPropertiesQueriedFromScope3APIServerMap(propertiesQueriedFromScope3APIServer)
		rr = httptest.NewRecorder()
		requestBody = emissionRequestBody{
			Rows: []EmissionRequestBodyRow{
				{
					InventoryId: "usatoday.com",
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
					Priority:    2,
				},
			},
		}
		handler.ServeHTTP(rr, createTestHttpRequest(t, requestBody))
		verifyPerPropertyEmissionAppResponse(t, rr, "usatoday.com")
		verifyScope3APIServerCalls(t, propertiesQueriedFromScope3APIServer, "usatoday.com")
		verifyCache(t, appCache, "nytimes.com", "usatoday.com")
		_, exists := appCache.Get("foxnews.com" + internal.EmissionCacheKeySuffix)
		assert.False(t, exists, "foxnews.com should not be in cache")

		clearPropertiesQueriedFromScope3APIServerMap(propertiesQueriedFromScope3APIServer)
		rr = httptest.NewRecorder()
		requestBody = emissionRequestBody{
			Rows: []EmissionRequestBodyRow{
				{
					InventoryId: "washingtonpost.com",
					Impressions: 1000,
					UtcDatetime: "2024-10-31",
					Priority:    3,
				},
			},
		}
		handler.ServeHTTP(rr, createTestHttpRequest(t, requestBody))
		verifyPerPropertyEmissionAppResponse(t, rr, "washingtonpost.com")
		verifyScope3APIServerCalls(t, propertiesQueriedFromScope3APIServer, "washingtonpost.com")
		verifyCache(t, appCache, "usatoday.com", "washingtonpost.com")
		_, exists = appCache.Get("nytimes.com" + internal.EmissionCacheKeySuffix)
		assert.False(t, exists, "nytimes.com should not be in cache")
		_, exists = appCache.Get("foxnews.com" + internal.EmissionCacheKeySuffix)
		assert.False(t, exists, "foxnews.com should not be in cache")
	})
}

func clearPropertiesQueriedFromScope3APIServerMap(propertiesQueriedFromScope3APIServer map[string]bool) {
	for key := range propertiesQueriedFromScope3APIServer {
		delete(propertiesQueriedFromScope3APIServer, key)
	}
}

func verifyCache(t *testing.T, appCache *cache.Cache, propertyNames ...string) {
	// Give a few moment for the cache to do its thing since caching is done in goroutine
	time.Sleep(5 * time.Millisecond)
	for _, propertyName := range propertyNames {
		_, exists := appCache.Get(propertyName + internal.EmissionCacheKeySuffix)
		assert.True(t, exists, propertyName+" should be cached")
	}
}

func verifyScope3APIServerCalls(t *testing.T, propertiesQueriedFromScope3APIServer map[string]bool, propertyNames ...string) {
	t.Helper()
	assert.Equal(t, len(propertyNames), len(propertiesQueriedFromScope3APIServer))
	for _, propertyName := range propertyNames {
		assert.True(t, propertiesQueriedFromScope3APIServer[propertyName])
	}
}

func verifyNoScope3APIServerCalls(t *testing.T, propertiesQueriedFromScope3APIServer map[string]bool) {
	t.Helper()
	assert.Equal(t, 0, len(propertiesQueriedFromScope3APIServer))
}

func verifyPerPropertyEmissionAppResponse(t *testing.T, rr *httptest.ResponseRecorder, propertyNames ...string) {
	t.Helper()
	assert.Equal(t, http.StatusOK, rr.Code)
	var apiResult APIResult
	_ = json.Unmarshal(rr.Body.Bytes(), &apiResult)
	assert.Equal(t, "", apiResult.Error)
	emissionPerProperty := apiResult.Data.(map[string]interface{})
	for _, propertyName := range propertyNames {
		actualEmission, _ := json.Marshal(emissionPerProperty[propertyName])
		assert.Equal(t, dummyEmissionInEachProperties, string(actualEmission))
	}
}

func createTestHttpRequest(t *testing.T, requestBody interface{}) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(requestBody)
	if err != nil {
		t.Fatal("Unable to encode request body")
	}

	req, err := http.NewRequest(http.MethodPost, "/", &buf)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}
	return req
}

func createMockHttpServerForEmissions(t *testing.T, propertiesQueriedFromScope3APIServer map[string]bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the GetEmissionsBreakdown in internal.scope3.v2.measure.go calls the
		// correct full API from the scope3 API server
		assert.Equal(t, r.Method, "POST")
		assert.Equal(t, r.URL.String(), "/v2/measure?includeRows=true&latest=true&fields=emissionsBreakdown")

		// Capture the property names that are queried from the scope3 API server for each test scenarios to verify
		defer r.Body.Close()
		requestBodyInBytes, _ := io.ReadAll(r.Body)
		var requestBody map[string]interface{}
		_ = json.Unmarshal(requestBodyInBytes, &requestBody)
		requestBodyRows := requestBody["rows"].([]interface{})
		var responseBodyRows []string
		for _, row := range requestBodyRows {
			// Should be the same fields with MeasureFilterRow
			rowMap := row.(map[string]interface{})
			propertyName := rowMap["inventoryId"].(string)
			responseBodyRows = append(responseBodyRows, `{"emissionsBreakdown":{"breakdown":`+dummyEmissionInEachProperties+`},"internal":{"propertyName":"`+propertyName+`"}}`)
			propertiesQueriedFromScope3APIServer[propertyName] = true
		}
		// Return the response as json
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"rows":[` + strings.Join(responseBodyRows, ",") + `]}`))
	}))
}

func createTestApiHandler(mockServerHost string, cacheCapacity int) (*APIV1Handler, *cache.Cache) {
	logger := zap.NewNop()
	scope3APIClient := v2.NewScope3APIClient(v2.Scope3APIClientConfig{
		Host: mockServerHost,
	})
	appCache := cache.NewCache(cacheCapacity)
	emissionService := internal.NewEmissionService(logger, scope3APIClient, appCache, 1*time.Hour)
	return &APIV1Handler{zap.NewNop(), emissionService, http.NewServeMux()}, appCache
}
