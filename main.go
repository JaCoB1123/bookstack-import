package main

import (
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/slices"
)

type bookstackImport struct {
	Client   *bookStackClient
	Books    []book
	Chapters []chapter
	Pages    []page
}

func NewImport(client *bookStackClient) *bookstackImport {
	imp := &bookstackImport{
		Client: client,
	}
	books, err := client.GetBooks()
	if err != nil {
		log.Fatal("Could not get list of books:", err)
		return nil
	}
	imp.Books = books.Data

	chapters, err := client.GetChapters()
	if err != nil {
		log.Fatal("Could not get list of chapters:", err)
		return nil
	}
	imp.Chapters = chapters.Data

	pages, err := client.GetPages()
	if err != nil {
		log.Fatal("Could not get list of pages:", err)
		return nil
	}
	imp.Pages = pages.Data
	return imp
}

func (imp bookstackImport) GetBook(name string) *book {
	matchingBookIndex := slices.IndexFunc(imp.Books, func(b book) bool { return b.Name == name })
	if matchingBookIndex == -1 {
		log.Println("Creating new book", name)
		newBook, err := imp.Client.CreateBook(name)
		if err != nil {
			log.Fatal(err)
			return nil
		}

		matchingBookIndex = len(imp.Books)
		imp.Books = append(imp.Books, *newBook)
		log.Println("New book:", newBook)
	}
	return &imp.Books[matchingBookIndex]
}

func (imp bookstackImport) GetChapter(name string, bookID int) *chapter {
	matchingChapterIndex := slices.IndexFunc(imp.Chapters, func(b chapter) bool { return b.Name == name && b.BookID == bookID })
	if matchingChapterIndex == -1 {
		log.Println("Creating new chapter", name)
		newChapter, err := imp.Client.CreateChapter(bookID, name)
		if err != nil {
			log.Fatal(err)
			return nil
		}

		matchingChapterIndex = len(imp.Chapters)
		imp.Chapters = append(imp.Chapters, *newChapter)
		log.Println("New chapter:", newChapter)
	}

	return &imp.Chapters[matchingChapterIndex]
}

func (imp bookstackImport) GetPage(name string, bookID int, chapterID int, content []byte) *page {
	matchingPageIndex := slices.IndexFunc(imp.Pages, func(b page) bool {
		return b.Name == name && b.BookID == bookID && b.ChapterID == chapterID
	})
	if matchingPageIndex == -1 {
		log.Println("Creating new page", name)
		newPage, err := imp.Client.CreatePage(bookID, chapterID, name, content)
		if err != nil {
			return nil
		}

		imp.Pages = append(imp.Pages, *newPage)
		return newPage
	}

	existingPage := &imp.Pages[matchingPageIndex]
	page, err := imp.Client.UpdatePageContent(existingPage.ID, content)
	if err != nil {
		log.Printf("could not update page %d: %s\n", page.ID, err)
		return nil
	}
	imp.Pages[matchingPageIndex] = *page
	return page
}

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

	err = filepath.WalkDir(importPath, func(fullPath string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		path := fullPath[len(importPath)+1:]

		segments := strings.Split(path, "\\")

		log.Println(path)

		book := imp.GetBook(segments[0])
		chapter := imp.GetChapter(segments[1], book.ID)
		pageName := strings.Join(segments[2:], "\\")

		content, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return err
		}

		imp.GetPage(pageName, book.ID, chapter.ID, content)
		return nil
	})

	if err != nil {
		panic(err)
	}
}
