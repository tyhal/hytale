package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tyhal/hytale/pkg/auth"
)

func main() {
	reauth := flag.Bool("reauth", false, "Force re-authentication via interactive device code flow")
	flag.BoolVar(reauth, "force", false, "Alias for --reauth")
	flag.Parse()

	var (
		authData *auth.AuthData
		err      error
	)
	if *reauth {
		authData, err = auth.Authenticate()
	} else {
		authData, err = auth.EnsureAuth()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Authentication error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Owner UUID:      %s\n", authData.Owner)
	fmt.Printf("Profile UUID:    %s (%s)\n", authData.Profile.Uuid, authData.Profile.Username)
	fmt.Printf("Session Token:   %s\n", authData.Session.SessionToken)
	fmt.Printf("Identity Token:  %s\n", authData.Session.IdentityToken)
	if !authData.Session.ExpiresAt.IsZero() {
		fmt.Printf("Session Expires: %s\n", authData.Session.ExpiresAt.Format("2006-01-02 15:04:05 MST"))
	}
}
