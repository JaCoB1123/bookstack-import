package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	url := os.Getenv("BOOKSTACK_URL")
	tokenID := os.Getenv("BOOKSTACK_TOKEN_ID")
	tokenSecret := os.Getenv("BOOKSTACK_TOKEN_SECRET")
	importPath := os.Getenv("BOOKSTACK_IMPORT_PATH")
	importMode := os.Getenv("BOOKSTACK_IMPORT_MODE") // "folder" (default) or "notion"
	bookName := os.Getenv("BOOKSTACK_BOOK_NAME")     // Required for notion mode

	if url == "" || tokenID == "" || tokenSecret == "" || importPath == "" {
		fmt.Println("Required environment variables:")
		fmt.Println("  BOOKSTACK_URL          - BookStack URL (e.g., https://bookstack.example.com)")
		fmt.Println("  BOOKSTACK_TOKEN_ID     - API Token ID")
		fmt.Println("  BOOKSTACK_TOKEN_SECRET - API Token Secret")
		fmt.Println("  BOOKSTACK_IMPORT_PATH  - Path to import folder")
		fmt.Println("")
		fmt.Println("Optional environment variables:")
		fmt.Println("  BOOKSTACK_IMPORT_MODE  - 'folder' (default) or 'notion'")
		fmt.Println("  BOOKSTACK_BOOK_NAME    - Book name (required for notion mode, defaults to folder name)")
		fmt.Println("")
		fmt.Println("Import modes:")
		fmt.Println("  folder - Structure: Book/Chapter/Page.md")
		fmt.Println("  notion - Structure: Notion export (folder = book, subfolders = chapters)")
		os.Exit(1)
	}

	client, err := NewBookStackClient(url, tokenID, tokenSecret)
	if err != nil {
		panic(err)
	}

	imp := NewImport(client)

	switch importMode {
	case "notion":
		// Use folder name as book name if not specified
		if bookName == "" {
			bookName = filepath.Base(importPath)
		}
		fmt.Printf("Importing Notion export from '%s' as book '%s'\n", importPath, bookName)
		err = imp.ImportNotionExport(importPath, bookName)
	default:
		fmt.Printf("Importing folder structure from '%s'\n", importPath)
		err = imp.ImportFolder(importPath)
	}

	if err != nil {
		panic(err)
	}

	fmt.Println("Import completed successfully!")
}

func IsDirSeparator(r rune) bool {
	return r == '\\' || r == '/'
}
