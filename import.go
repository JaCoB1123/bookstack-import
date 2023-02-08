package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	cache "github.com/Code-Hex/go-generics-cache"
)

type bookstackImport struct {
	Client   *bookStackClient
	Books    *cache.Cache[string, *book]
	Chapters *cache.Cache[string, *chapter]
	Pages    *cache.Cache[string, *page]
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
	imp.Books = cache.New[string, *book]()
	for _, book := range books.Data {
		book2 := book
		imp.Books.Set(book.String(), &book2)
	}

	chapters, err := client.GetChapters()
	if err != nil {
		log.Fatal("Could not get list of chapters:", err)
		return nil
	}
	imp.Chapters = cache.New[string, *chapter]()
	for _, chapter := range chapters.Data {
		chapter2 := chapter
		imp.Chapters.Set(chapter.String(), &chapter2)
	}

	pages, err := client.GetPages()
	if err != nil {
		log.Fatal("Could not get list of pages:", err)
		return nil
	}
	imp.Pages = cache.New[string, *page]()
	for _, page := range pages.Data {
		page2 := page
		imp.Pages.Set(page.String(), &page2)
	}
	return imp
}

func (imp *bookstackImport) GetBook(name string) *book {
	book := &book{
		Name: name,
	}

	existingBook, ok := imp.Books.Get(book.String())
	if ok {
		return existingBook
	}

	log.Println("Creating new book", name)
	newBook, err := imp.Client.CreateBook(name)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	imp.Books.Set(newBook.String(), newBook)
	log.Println("New book:", newBook)
	return newBook
}

func (imp *bookstackImport) GetChapter(name string, bookID int) *chapter {
	chapter := &chapter{
		BookID: bookID,
		Name:   name,
	}

	existingChapter, ok := imp.Chapters.Get(chapter.String())
	if ok {
		return existingChapter
	}

	log.Println("Creating new chapter", name)
	newChapter, err := imp.Client.CreateChapter(bookID, name)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	imp.Chapters.Set(newChapter.String(), newChapter)
	log.Println("New chapter:", newChapter)
	return newChapter
}

func (imp *bookstackImport) GetPage(name string, chapterID int, content []byte) *page {
	page := &page{
		ChapterID: chapterID,
		Name:      name,
	}

	existingPage, ok := imp.Pages.Get(page.String())
	if !ok {
		log.Println("Creating new page", name)
		newPage, err := imp.Client.CreatePage(chapterID, name, content)
		if err != nil {
			return nil
		}

		imp.Pages.Set(newPage.String(), newPage)
		return newPage
	}

	page, err := imp.Client.UpdatePageContent(existingPage.ID, content)
	if err != nil {
		log.Printf("could not update page %d: %s\n", existingPage.ID, err)
		return nil
	}
	imp.Pages.Set(page.String(), page)
	return page
}

func (imp *bookstackImport) ImportFolder(importPath string) error {
	return filepath.WalkDir(importPath, func(fullPath string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if info.Name() == "media" || info.Name() == "docx" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(fullPath, ".md") {
			return nil
		}

		path := fullPath[len(importPath)+1:]
		segments := strings.FieldsFunc(path, IsDirSeparator)

		book := imp.GetBook(segments[0])
		chapter := imp.GetChapter(segments[1], book.ID)
		pageName := strings.Join(segments[2:], "/")

		content, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return err
		}

		page := imp.GetPage(pageName, chapter.ID, content)
		content, err = imp.ReplaceAllImages(page.ID, content, fullPath)
		if err != nil {
			return err
		}

		// TODO Replace all Embeds \<\<[.../...]\>\>
		// TODO Replace all Links [text](url)

		imp.GetPage(pageName, chapter.ID, content)
		return nil
	})
}

func (imp *bookstackImport) ReplaceAllImages(pageID int, content []byte, path string) ([]byte, error) {
	for i := 0; i < len(content); i++ {
		if content[i] != '!' {
			continue
		}

		bracketStart, bracketEnd := FindNext(content, i+1, '[', ']')
		if bracketEnd == -1 {
			continue
		}

		parenthesisStart, parenthesisEnd := FindNext(content, bracketEnd+1, '(', ')')
		if parenthesisEnd == -1 {
			continue
		}

		os.WriteFile(fmt.Sprintf("C:\\temp\\notes\\%d-%d.md", pageID, i), content, fs.ModePerm)
		name := content[bracketStart+1 : bracketEnd]
		src := content[parenthesisStart+1 : parenthesisEnd]
		path := filepath.Join(filepath.Dir(path), string(src))
		attachment, err := imp.Client.UploadAttachment(pageID, string(name), path)
		if err != nil {
			return nil, err
		}

		src = []byte(fmt.Sprintf("/attachments/%d", attachment.ID))
		contentTail := content[parenthesisEnd+1:]
		content = append(content[:i], []byte(fmt.Sprintf("![%s](%s)", name, src))...)
		i = len(content)
		content = append(content, contentTail...)
	}

	return content, nil
}

func FindNext(content []byte, start int, nested byte, char byte) (int, int) {
	if content[start] != nested {
		return -1, -1
	}

	end := start + 1
	nestedCount := 0
	for ; end < len(content); end++ {
		switch content[end] {
		case nested:
			nestedCount++
		case char:
			nestedCount--
			if nestedCount < 0 {
				return start, end
			}
		}
	}
	return -1, -1
}
