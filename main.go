package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"vend/cmd"
	"vend/internal/sudo"
)

func main() {
	if len(os.Args) == 3 && os.Args[1] == "link" {
		if err := link(os.Args[2]); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	} else {
		cmd.Execute()
	}
}

func link(encoded string) error {
	var linkData []sudo.LinkData
	b64Dec := base64.NewDecoder(base64.URLEncoding, strings.NewReader(encoded))
	json.NewDecoder(b64Dec).Decode(&linkData)

	for _, data := range linkData {
		if data.Old == data.New {
			continue
		}
		if data.New == "" || data.Old == "" {
			return fmt.Errorf("invalid link data: %v", data)
		}
		if err := os.Symlink(data.Old, data.New); err != nil {
			return fmt.Errorf("failed to create link: %w", err)
		}
	}
	return nil
}
