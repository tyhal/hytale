package main

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/tyhal/hytale/pkg/auth"
)

func basicAuthFlow(cmd *cobra.Command, args []string) {
	deviceCode, err := auth.RequestDeviceCode()
	if err != nil {
		log.Errorf("RequestDeviceCode: %v\n", err)
		return
	}
	fmt.Println(deviceCode.VerificationUriComplete)
	token, err := auth.WaitForToken(deviceCode)
	if err != nil {
		log.Errorf("WaitingForToken: %v\n", err)
		return
	}
	profiles, err := auth.GetProfiles(token.AccessToken)
	if err != nil {
		log.Errorf("GetProfiles: %v\n", err)
		return
	}
	if len(profiles.Profiles) <= 0 {
		log.Errorf("No profiles found")
		return
	}
	session, err := auth.CreateGameSession(token.AccessToken, profiles.Profiles[0])
	if err != nil {
		log.Errorf("CreateGameSession: %v\n", err)
		return
	}
	fmt.Println(session.SessionToken)
	fmt.Println(session.IdentityToken)
	fmt.Println(profiles.Owner)
}
