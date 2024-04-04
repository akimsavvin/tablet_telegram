package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/akimsavvin/tablet_telegram/internal/dto"
)

type TabletService struct {
	backendURL string
}

func NewTabletService(backendURL string) *TabletService {
	if backendURL == "" {
		log.Fatalf("Empty backend URL")
	}

	return &TabletService{backendURL}
}

func (s *TabletService) Create(createDTO *dto.CreateTabletDTO) error {
	url := fmt.Sprintf("%s/api/tablets", s.backendURL)

	jsonDTO, err := json.Marshal(createDTO)
	if err != nil {
		log.Printf("Could not parse create tablet dto due to error: %s", err.Error())
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonDTO))
	if err != nil {
		log.Printf("Could not create tablet due to error: %s", err.Error())
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		log.Printf("Could not create tablet due to error: %s", string(body))
		return errors.New("could not create tablet due to unknown error")
	}

	return nil
}
