package main

import (
	"log"
	"os"
	"path/filepath"

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

	bookDirectories, err := os.ReadDir(importPath)
	if err != nil {
		log.Fatal("Could not get files from "+importPath, err)
		return
	}

	books, err := client.GetBooks()
	if err != nil {
		log.Fatal("Could not get list of books:", err)
		return
	}

	chapters, err := client.GetChapters()
	if err != nil {
		log.Fatal("Could not get list of chapters:", err)
		return
	}

	pages, err := client.GetPages()
	if err != nil {
		log.Fatal("Could not get list of pages:", err)
		return
	}

	for _, bookDirectory := range bookDirectories {
		matchingBookIndex := slices.IndexFunc(books.Data, func(b book) bool { return b.Name == bookDirectory.Name() })
		if matchingBookIndex == -1 {
			log.Println("Creating new book", bookDirectory.Name())
			newBook, err := client.CreateBook(bookDirectory.Name())
			if err != nil {
				log.Fatal(err)
				return
			}

			matchingBookIndex = len(books.Data)
			books.Data = append(books.Data, *newBook)
			log.Println("New book:", newBook)
		}

		matchingBook := books.Data[matchingBookIndex]

		importPath := filepath.Join(importPath, bookDirectory.Name())
		chapterDirectories, err := os.ReadDir(importPath)
		if err != nil {
			log.Fatal("Could not get files from "+importPath, err)
			return
		}

		for _, chapterDirectory := range chapterDirectories {
			matchingChapterIndex := slices.IndexFunc(chapters.Data, func(b chapter) bool { return b.Name == chapterDirectory.Name() && b.BookID == matchingBook.ID })
			if matchingChapterIndex == -1 {
				log.Println("Creating new chapter", chapterDirectory.Name())
				newChapter, err := client.CreateChapter(matchingBook.ID, chapterDirectory.Name())
				if err != nil {
					log.Fatal(err)
					return
				}

				matchingChapterIndex = len(chapters.Data)
				chapters.Data = append(chapters.Data, *newChapter)
				log.Println("New chapter:", newChapter)
			}

			matchingChapter := chapters.Data[matchingChapterIndex]

			importPath := filepath.Join(importPath, chapterDirectory.Name())

			// from here, handle leafs only, as the nesting is limited
			err := filepath.Walk(importPath,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if info.IsDir() {
						return nil
					}

					matchingPageIndex := slices.IndexFunc(pages.Data, func(b page) bool {
						return b.Name == chapterDirectory.Name() && b.BookID == matchingBook.ID && b.ChapterID == matchingChapter.ID
					})
					if matchingPageIndex == -1 {
						log.Println("Creating new page", chapterDirectory.Name())
						newPage, err := client.CreatePage(matchingBook.ID, matchingChapter.ID, chapterDirectory.Name())
						if err != nil {
							return err
						}

						matchingPageIndex = len(pages.Data)
						pages.Data = append(pages.Data, *newPage)
						log.Println("New page:", newPage)
					}
					return nil
				})
			if err != nil {
				panic(err)
			}
			return
		}

		return
	}
}
