package telegram

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (c *APIClient) GetFileURL(fileID string) (string, error) {
	file, err := c.getFile(fileID)
	if err != nil {
		return "", err
	}

	return c.buildTelegramFileURL("/" + file.FilePath), nil
}

type getFileResponse struct {
	OK     bool `json:"ok"`
	Result File `json:"result"`
}

func (c *APIClient) getFile(fileID string) (*File, error) {
	getFileURL := c.buildTelegramURL(telegramPathGetFile)
	getFileURL = getFileURL + "?file_id=" + fileID

	req, err := http.NewRequest(http.MethodGet, getFileURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected status 200 but got %s", resp.Status)
	}

	var response getFileResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return &response.Result, nil
}

func (c *APIClient) buildTelegramFileURL(path string) string { //nolint:unparam
	b := strings.Builder{}
	b.Grow(len(telegramFileBasePath) + len(c.token) + len(path))
	b.WriteString(telegramFileBasePath)
	b.WriteString(c.token)
	b.WriteString(path)
	return b.String()
}
