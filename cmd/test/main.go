package main

import (
	"context"
	"fmt"
	"log"

	"github.com/focusd-so/focusd/internal/usage"
)

func main() {
	u := "https://news.ycombinator.com/item?id=46903556"

	mainContent, err := usage.FetchMainContent(context.Background(), u)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(mainContent)

}
