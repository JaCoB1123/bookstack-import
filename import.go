package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
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

func (imp *bookstackImport) GetPageID(name string, chapterID int) (int, error) {
	page := &page{
		ChapterID: chapterID,
		Name:      name,
	}

	existingPage, ok := imp.Pages.Get(page.String())
	if !ok {
		log.Println("Creating new page", name)
		newPage, err := imp.Client.CreatePage(chapterID, name, []byte("empty"))
		if err != nil {
			return -1, fmt.Errorf("create page: %w", err)
		}

		imp.Pages.Set(newPage.String(), newPage)
		return newPage.ID, nil
	}

	return existingPage.ID, nil
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

	log.Println("Updating existing page", existingPage.ID, name)
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
			return fmt.Errorf("read file: %w", err)
		}

		// Header entfernen
		headerSeparator := FindNextMultiChar(content, 0, '-', '-', '-')
		content = content[headerSeparator+5:]

		// Pfeile sind in OneNote mit WingDings formatiert, durch ASCII-Pfeile ersetzen
		content = []byte(bytes.ReplaceAll(content, []byte("à"), []byte("->")))

		pageID, err := imp.GetPageID(pageName, chapter.ID)
		if err != nil {
			return fmt.Errorf("get page ID: %w", err)
		}

		content, err = imp.ReplaceAllImages(pageID, content, fullPath)
		if err != nil {
			return fmt.Errorf("replace images: %w", err)
		}

		content, err = imp.ReplaceAllEmbeds(pageID, content, fullPath)
		if err != nil {
			return fmt.Errorf("replace embeds: %w", err)
		}

		content, err = imp.ReplaceAllInternalLinks(pageID, content, fullPath)
		if err != nil {
			return fmt.Errorf("replace internal links: %w", err)
		}

		imp.GetPage(pageName, chapter.ID, content)
		return nil
	})
}

func (imp *bookstackImport) ReplaceAllInternalLinks(pageID int, content []byte, path string) ([]byte, error) {
	// TODO Implement
	for i := 0; i < len(content); i++ {
		if content[i] != '[' {
			continue
		}

		_, bracketEnd := FindNext(content, i, '[', ']')
		if bracketEnd == -1 {
			continue
		}

		parenthesisStart, parenthesisEnd := FindNext(content, bracketEnd+1, '(', ')')
		if parenthesisEnd == -1 {
			continue
		}

		//name := content[bracketStart+1 : bracketEnd]
		src, err := strconv.Unquote("\"" + string(content[parenthesisStart+1:parenthesisEnd]) + "\"")
		if err != nil {
			return nil, fmt.Errorf("unquote: %w", err)
		}

		if strings.HasPrefix(string(src), "onenote:") {
			log.Println("Found onenote link", src)
		}
	}

	return content, nil
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

		name := content[bracketStart+1 : bracketEnd]
		src, err := strconv.Unquote("\"" + string(content[parenthesisStart+1:parenthesisEnd]) + "\"")
		if err != nil {
			return nil, fmt.Errorf("unquote: %w", err)
		}
		path := filepath.Join(filepath.Dir(path), src)
		attachment, err := imp.Client.UploadAttachment(pageID, string(name), path)
		if err != nil {
			return nil, fmt.Errorf("upload attachment: %w", err)
		}

		src = fmt.Sprintf("/attachments/%d", attachment.ID)
		contentTail := content[parenthesisEnd+1:]
		newImage := []byte(fmt.Sprintf("![%s](%s)", filepath.Base(src), src))
		content = append(content[:i], newImage...)
		i = len(newImage) + i - 1
		content = append(content, contentTail...)
	}

	return content, nil
}

func (imp *bookstackImport) ReplaceAllEmbeds(pageID int, content []byte, path string) ([]byte, error) {
	for i := 0; i < len(content); i++ {
		if content[i] != '\\' || content[i+1] != '<' {
			continue
		}

		if content[i+2] != '\\' || content[i+3] != '<' {
			continue
		}

		firstClosing := FindNextMultiChar(content, i+3, '\\', '>', '\\', '>')
		if firstClosing == -1 {
			continue
		}

		bracketStart, bracketEnd := FindNext(content, i+4, '[', ']')
		if bracketEnd == -1 {
			continue
		}

		parenthesisStart, parenthesisEnd := FindNext(content, bracketEnd+1, '(', ')')
		if parenthesisEnd == -1 {
			continue
		}

		name := content[bracketStart+1 : bracketEnd]
		src, err := strconv.Unquote("\"" + string(content[parenthesisStart+1:parenthesisEnd]) + "\"")
		if err != nil {
			return nil, fmt.Errorf("unquote: %w", err)
		}
		path := filepath.Join(filepath.Dir(path), string(src))
		attachment, err := imp.Client.UploadAttachment(pageID, string(name), path)
		if err != nil {
			return nil, fmt.Errorf("upload attachment: %w", err)
		}

		src = fmt.Sprintf("/attachments/%d", attachment.ID)
		contentTail := content[parenthesisEnd+5:]
		newImage := []byte(fmt.Sprintf("[%s](%s)", filepath.Base(src), src))
		content = append(content[:i], newImage...)
		i = len(newImage) + i - 1
		content = append(content, contentTail...)
	}

	return content, nil
}

func FindNextAnywhere(content []byte, start int, nested byte, char byte) (int, int) {
	for ; start < len(content)-1; start++ {
		if content[start] != nested {
			continue
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
	}
	return -1, -1
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

func FindNextMultiChar(content []byte, start int, chars ...byte) int {
	end := start + 1
	for ; end < len(content); end++ {
		switch content[end] {
		case chars[0]:
			for i := 1; i < len(chars); i++ {
				if content[end+i] == chars[i] {
					return end + i
				}
			}
		}
	}
	return -1
}
