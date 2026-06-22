package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

func LoadUrlsFromTxtFile(txtFileName string) ([]string, error) {
	dataBytes, err := os.ReadFile(txtFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", txtFileName, err)
	}

	lines := strings.Split(string(dataBytes), "\n")
	re := regexp.MustCompile(`https?://[^\s]+`)
	uniqueUrls := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		if url := re.FindString(line); url != "" {
			uniqueUrls[url] = true
		}
	}

	var urls []string
	for url := range uniqueUrls {
		urls = append(urls, url)
	}

	return urls, nil
}

func MakeAuthenticatedGETRequest[T any](url, token string) (*T, error) {
	startTime := time.Now()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("ERROR -> Execution time: [%.6f seconds] Failed to create request URL: [%s]\n", time.Since(startTime).Seconds(), url)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 10 * time.Second}

	res, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERROR -> Execution time: [%.6f seconds] Failed to send request URL: [%s]\n", time.Since(startTime).Seconds(), url)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer res.Body.Close()

	fmt.Printf("Status: [%d] Execution time: [%.6f seconds] Fetching URL: [%s]\n", res.StatusCode, time.Since(startTime).Seconds(), url)

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("ERROR -> Execution time: [%.6f seconds] Failed to read response body URL: [%s]\n", time.Since(startTime).Seconds(), url)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("ERROR -> Execution time: [%.6f seconds] Error un-marshalling response body URL: [%s]\n", time.Since(startTime).Seconds(), url)
		return nil, fmt.Errorf("error un-marshalling response body: %v", err)
	}

	return &result, nil
}

func SaveToJsonFile[T any](data T, fileName string) error {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling data to JSON: %w", err)
	}

	err = os.WriteFile(fileName, jsonBytes, 0644)
	if err != nil {
		return fmt.Errorf("error writing JSON to file: %w", err)
	}

	fmt.Printf("Json data has been saved into => %s\n", fileName)

	return nil
}

func SaveToMarkdown(templatePath string, data interface{}, outputPath string) error {
	mdTemplate, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	parsedTemplate, err := template.New("markdown").Parse(string(mdTemplate))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var output bytes.Buffer
	if err := parsedTemplate.Execute(&output, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if err := os.WriteFile(outputPath, output.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

func LoadJSONFromFile[T any](fileName string) (*T, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", fileName, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", fileName, err)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON from %s: %w", fileName, err)
	}

	return &result, nil
}
