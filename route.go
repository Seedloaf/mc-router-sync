package mcroutersync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type Route struct {
	ServerAddress string `json:"serverAddress"`
	Backend       string `json:"backend"`
}

type Routes []Route

func (r *Routes) Parse(reader io.Reader) error {
	return json.NewDecoder(reader).Decode(r)
}

func (r Route) Json() (*bytes.Reader, error) {
	body, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal route: %w", err)
	}

	return bytes.NewReader(body), nil
}
