package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type PinataResponse struct {
	IpfsHash string `json:"IpfsHash"`
}

func deployPinata(filePath, apiKey, secretApiKey string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create a new buffer to store the multipart/form-data request body
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Create a part for the file
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}

	// Copy the file content into the part
	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}

	writer.Close()

	// Create the HTTP POST request
	req, err := http.NewRequest("POST", "https://api.pinata.cloud/pinning/pinFileToIPFS", &requestBody)
	if err != nil {
		return "", err
	}

	// Set content type header
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Set API key headers
	req.Header.Add("pinata_api_key", apiKey)
	req.Header.Add("pinata_secret_api_key", secretApiKey)

	// Perform the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status code %d, %s", resp.StatusCode, body)
	}

	var pinataResponse PinataResponse
	if err := json.NewDecoder(resp.Body).Decode(&pinataResponse); err != nil {
		return "", err
	}

	imageUrl := "https://ipfs.io/ipfs/" + pinataResponse.IpfsHash

	return imageUrl, nil
}
