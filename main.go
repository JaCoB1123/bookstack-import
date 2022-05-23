package main

import (
	"os"
)

func main() {
	url := os.Getenv("BOOKSTACK_URL")
	tokenID := os.Getenv("BOOKSTACK_TOKEN_ID")
	tokenSecret := os.Getenv("BOOKSTACK_TOKEN_SECRET")
	importPath := os.Getenv("BOOKSTACK_IMPORT_PATH")

	client, err := NewBookStackClient(url, tokenID, tokenSecret)
	if err != nil {
		panic(err)
	}

	imp := NewImport(client)
	err = imp.ImportFolder(importPath)

	if err != nil {
		panic(err)
	}
}

func IsDirSeparator(r rune) bool {
	return r == '\\' || r == '/'
}
