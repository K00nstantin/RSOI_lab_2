package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

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
	libraryServiceURL = getEnv("LIBRARY_SERVICE_URL", "http://localhost:8060")
	reservationServiceURL = getEnv("RESERVATION_SERVICE_URL", "http://localhost:8070")

	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}

	r := gin.Default()

	r.GET("/api/v1/libraries", getLibrariesHandler)
	r.GET("/api/v1/libraries/:libraryUid/books", getLibraryBooksHandler)
	r.GET("/api/v1/reservations", getReservationsHandler)
	r.POST("/api/v1/reservations", createReservationHandler)
	r.POST("/api/v1/reservations/:reservationUid/return", returnBookHandler)
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
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-Name header is required"})
	}
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

func createReservationHandler(c *gin.Context) {
	username := c.GetHeader("X-User-Name")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-Name header must be contained"})
		return
	}
	var request struct {
		BookUid    string `json: "bookUid", binding: "required"`
		LibraryUid string `json: "bookUid", binding: "required"`
		TillDate   string `json: "tillDate", binding: "required"`
	}
	err := c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "validation error",
			"errors": map[string]string{
				"field": "request",
				"error": err.Error(),
			},
		})
		return
	}
	bookinfo := getBookInfo(request.LibraryUid, request.BookUid)
	if bookinfo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to find the book in the library"})
		return
	}
	avaliablecount, ok := bookinfo["avaliableCount"].(float64)
	if !ok || avaliablecount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "book not avaliable"})
		return
	}
	body, err := json.Marshal(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request body"})
		return
	}
	url := reservationServiceURL + "/api/v1/reservations"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create the request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Name", username)
	resp, err := httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to perform the request"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		rbody, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", rbody)
		return
	}
	var reservation map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&reservation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode the response"})
		return
	}
	libraryinfo := getLibraryInfo(request.LibraryUid)
	rating := getUserRating(username)
	response := map[string]interface{}{
		"reservationUid": reservation["reservationUid"],
		"status":         reservation["status"],
		"startDate":      reservation["startDate"],
		"tillDate":       reservation["tillDate"],
		"book": map[string]interface{}{
			"bookUid": bookinfo["bookUid"],
			"name":    bookinfo["name"],
			"author":  bookinfo["author"],
			"genre":   bookinfo["genre"],
		},
		"library": libraryinfo,
		"rating":  rating,
	}
	c.JSON(http.StatusOK, response)
}

func returnBookHandler(c *gin.Context) {
	username := c.GetHeader("X-User-Name")
	if username == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "header should be contained"})
		return
	}
	reservationUid := c.Param("reservationUid")
	var request struct {
		Condition string `json: "condition", binding: "required"`
		Date      string `json: "date", binding: "required"`
	}
	err := c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "validation error",
			"errors": map[string]string{
				"field": "request",
				"error": err.Error(),
			},
		})
		return
	}
	reqbody, err := json.Marshal(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request body"})
		return
	}
	url := fmt.Sprintf("%s/api/v1/reservations/%s/return", reservationServiceURL, reservationUid)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqbody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create the request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Name", username)
	resp, err := httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ewecute the response"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", body)
		return
	}
	c.Status(http.StatusNoContent)
}

func getRatingHandler(c *gin.Context) {
	username := c.GetHeader("X-User-Name")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-Name header is required"})
		return
	}

	url := ratingServiceURL + "api/v1/rating"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create the request"})
		return
	}
	req.Header.Set("X-User-Name", username)
	resp, err := httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ewecute the response"})
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response body"})
		return
	}
	c.Data(resp.StatusCode, "application/json", body)
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
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return map[string]interface{}{"stars": 0}
	}
	req.Header.Set("X-User-Name", username)

	resp, err := httpClient.Do(req)
	if err != nil {
		return map[string]interface{}{"stars": 0}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return map[string]interface{}{"stars": 0}
	}

	var rating map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rating); err != nil {
		return map[string]interface{}{"stars": 0}
	}

	return rating
}
