// This file is deprecated. Use cmd/server/main.go instead.
// This file is kept for backward compatibility only.

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("This main.go file is deprecated.")
	fmt.Println("Please use: go run cmd/server/main.go")
	fmt.Println("Or build with: go build -o reelchoice-backend ./cmd/server")
	os.Exit(1)
}
