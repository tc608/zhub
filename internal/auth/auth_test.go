package auth

import (
	"fmt"
	"log"
	"regexp"
	"testing"
)

func TestAuth(t *testing.T) {
	// Create an instance of PermissionManager
	p := &PermissionManager{}
	// Initialize the permission manager
	err := p.Init()
	if err != nil {
		log.Fatal(err)
	}
	// Example usage: Get permissions by token
	token := "token-12345"
	userId, err := p.GetUserIdByToken(token)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Permissions:")

	check := p.AuthCheck(userId, "zcore:abx", "r")
	check2 := p.AuthCheck(userId, "zcore:abx", "r")

	fmt.Println("check:", check)
	fmt.Println("check2:", check2)
}

func TestName(t *testing.T) {
	exp := "zcore:*"
	channel := "zcore:axxx"
	compile, _ := regexp.Compile(exp)

	//matched, _ := regexp.MatchString(exp, channel)
	matched := compile.MatchString(channel)
	fmt.Println("matched:", matched)
}
