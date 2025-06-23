package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

// Serializes the results, either as json (default), or raw (close to FujiNet json parsing result)
// raw=1 -  or as key[char 0]value[char 0] pairs
// - fc=U/L - (may use with raw) force data case all upper or lower

func serializeResults(c *gin.Context, obj any) {

	if debugMode {
		// Log the request details
		log.Printf("DEBUG Request - Method: %s, Path: %s, Query: %s", c.Request.Method, c.Request.URL.Path, c.Request.URL.RawQuery)

		// Log the request body if present
		// Read the body into a byte slice
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("DEBUG Error reading request body: %v", err)
		} else if len(bodyBytes) > 0 {
			log.Printf("DEBUG Request Body: %s", string(bodyBytes))
		}
		// Restore the body for subsequent handlers
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Log the response
		jsonBytes, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			log.Printf("DEBUG Error marshaling response for logging: %v", err)
		} else {
			log.Printf("DEBUG Response: %s", string(jsonBytes))
		}
	}

	if c.Query("raw") == "1" {
		lineDelimiter := "\u0000"
		if c.Query("lf") == "1" {
			lineDelimiter = "\n"
		}
		jsonBytes, _ := json.Marshal(obj)
		jsonResult := string(jsonBytes)

		// Strip out [,],{,}
		jsonResult = strings.ReplaceAll(jsonResult, "{", "")
		jsonResult = strings.ReplaceAll(jsonResult, "}", "")
		jsonResult = strings.ReplaceAll(jsonResult, "[", "")
		jsonResult = strings.ReplaceAll(jsonResult, "]", "")

		// Convert : to new line
		jsonResult = strings.ReplaceAll(jsonResult, ":", lineDelimiter)

		// Convert commas to new line
		jsonResult = strings.ReplaceAll(jsonResult, "\",", lineDelimiter)
		jsonResult = strings.ReplaceAll(jsonResult, ",\"", lineDelimiter)
		jsonResult = strings.ReplaceAll(jsonResult, "\"", "")

		if c.Query("uc") == "1" {
			jsonResult = strings.ToUpper(jsonResult)
		}

		if c.Query("lc") == "1" {
			jsonResult = strings.ToLower(jsonResult)
		}

		c.String(http.StatusOK, jsonResult)

	} else {
		c.JSON(http.StatusOK, obj)
	}
}
