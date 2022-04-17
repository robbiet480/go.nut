package nut

import (
	"context"
	"fmt"

	nut "github.com/andreyAKor/go.nut"
)

// This example connects to NUT, authenticates and returns the first UPS listed.
func ExampleGetUPSList() {
	client, connectErr := nut.NewClient(context.Background(), "127.0.0.1", 3493)
	if connectErr != nil {
		fmt.Print(connectErr)
	}

	_, authenticationError := client.Authenticate("username", "password")
	if authenticationError != nil {
		fmt.Print(authenticationError)
	}

	upsList, listErr := client.GetUPSList()
	if listErr != nil {
		fmt.Print(listErr)
	}

	fmt.Print("First UPS", upsList[0])
}
