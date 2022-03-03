package nut

import (
	"fmt"
	"time"
)

// This example connects to NUT, authenticates and returns the first UPS listed.
func ExampleGetUPSList() {
	client, connectErr := Connect("127.0.0.1", 10*time.Second, 30*time.Second)
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
