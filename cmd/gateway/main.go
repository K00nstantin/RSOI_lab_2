package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var (
	ratingServiceURL      string
	libraryServiceURL     string
	reservationServiceURL string
	httpClient            *http.Client
)

func main() {
	ratingServiceURL = getEnv("RATING_SERVICE_URL", "http://localhost:8050")

	r := gin.Default()

	r.GET("/api/v1/rating", getRatingHandler)
	r.GET("/manage/health", healthCheck)

	log.Println("Gateway service starting on port 8080")
	r.Run(":8080")
}

func getLibrariesHandler(c *gin.Context) {
	params := c.Request.URL.Query().Encode()
	url := libraryServiceURL + "/api/v1/libraries"
	if params != "" {
		url += "?" + params
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to make a request"})
		return
	}
	response, err := httpClient.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to perform request"})
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to perform request"})
		return
	}
	c.Data(response.StatusCode, "application/json", body)
}
func getLibraryBooksHandler(c *gin.Context) {
	libraryUid := c.Param("libraryUid")
	queryparams := c.Request.URL.Query().Encode()
	url := fmt.Sprintf("%s/api/v1/libraries/%s/books", libraryServiceURL, libraryUid)
	if queryparams != "" {
		url += queryparams
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create a request"})
		return
	}
	response, err := httpClient.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to perform a request"})
		return
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read the response"})
		return
	}
	c.Data(response.StatusCode, "application/json", data)
}

func getReservationsHandler(c *gin.Context) {
	username := c.GetHeader("X-User-Name")
	url := reservationServiceURL + "/api/v1/reservations"
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	request.Header.Set("X-User-Name", username)
	response, err := httpClient.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to perform the request"})
		return
	}
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		c.Data(response.StatusCode, "application/json", body)
		return
	}
	var reservations []map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&reservations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode the response"})
		return
	}
	enrichedReservations := make([]map[string]interface{}, len(reservations))
	for i, res := range reservations {
		bookUid, _ := res["bookUid"].(string)
		libraryUid, _ := res["libraryUid"].(string)
		bookInfo := getBookInfo(libraryUid, bookUid)
		libraryInfo := getLibraryInfo(libraryUid)
		enrichedReservations[i] = map[string]interface{}{
			"reservationUid": res["reservationUid"],
			"status":         res["status"],
			"startDate":      res["startDate"],
			"tillDate":       res["tillDate"],
			"book":           bookInfo,
			"library":        libraryInfo,
		}
	}
	c.JSON(http.StatusOK, enrichedReservations)
}

func getRatingHandler(c *gin.Context) {
	username := c.GetHeader("X-User-Name")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-Name header is required"})
		return
	}

	resp, err := http.Get(ratingServiceURL + "/api/v1/rating?username=" + username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rating service unavailable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "failed to get rating"})
		return
	}

	var ratingResponse struct {
		Stars int `json:"stars"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ratingResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stars": ratingResponse.Stars})
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "UP",
		"details": "Host localhost:8080 is active",
	})
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// helper functions

func getBookInfo(libraryUid, bookUid string) map[string]interface{} {
	url := fmt.Sprintf("%s/api/v1/libraries/%s/books/%s", libraryServiceURL, libraryUid, bookUid)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil
	}
	var book map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&book)
	if err != nil {
		return nil
	}
	return book

}

func getLibraryInfo(libraryUid string) map[string]interface{} {
	url := fmt.Sprintf("%s/api/v1/libraries/%s", libraryServiceURL, libraryUid)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil
	}
	var library map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&library)
	if err != nil {
		return nil
	}
	return library
}

func getUserRating(username string) map[string]interface{} {
	url := ratingServiceURL + "/api/v1/rating"
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return map[string]interface{}{"stars": 0}
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return map[string]interface{}{"stars": 0}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return map[string]interface{}{"stars": 0}
	}
	var rating map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&rating)
	if err != nil {
		return map[string]interface{}{"stars": 0}
	}
	return rating
}
