package main

import (
	"log"
	"os"

	"golang.org/x/exp/slices"
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

	entries, err := os.ReadDir(importPath)
	if err != nil {
		log.Fatal("Could not get files from "+importPath, err)
		return
	}

	books, err := client.GetBooks()
	if err != nil {
		log.Fatal("Could not get list of books:", err)
		return
	}

	for _, entry := range entries {
		matchingBookIndex := slices.IndexFunc(books.Data, func(b book) bool { return b.Name == entry.Name() })
		if matchingBookIndex == -1 {
			log.Println("Creating new Book", entry.Name())
			newBook, err := client.CreateBook(entry.Name())
			if err != nil {
				log.Fatal(err)
				return
			}

			matchingBookIndex = len(books.Data)
			books.Data = append(books.Data, *newBook)
			log.Println("New book:", newBook)
		}

		matchingBook = books.Data[matchingBookIndex]

		return
	}
}
