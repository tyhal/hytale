package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const accountUrl = "https://account-data.hytale.com"

type UUID = string

type Profile struct {
	Uuid     UUID   `json:"uuid"`
	Username string `json:"username"`
}

type OwnerUUID = UUID

type OwnedProfiles struct {
	Owner    OwnerUUID `json:"owner"`
	Profiles []Profile `json:"profiles"`
}

func GetProfiles(at AccessToken) (OwnedProfiles, error) {
	req, err := http.NewRequest("GET", accountUrl+"/my-account/get-profiles", nil)
	if err != nil {
		return OwnedProfiles{}, err
	}
	req.Header.Set("Authorization", "Bearer "+at)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return OwnedProfiles{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return OwnedProfiles{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var profiles OwnedProfiles
	err = json.NewDecoder(resp.Body).Decode(&profiles)
	if err != nil {
		return OwnedProfiles{}, err
	}
	return profiles, nil
}
