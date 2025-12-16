package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/url"
	"path/filepath"
	"regexp"
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

// ImportFolder imports markdown files with structure: Book/Chapter/Page.md
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

		if len(segments) < 3 {
			log.Printf("Skipping %s: need at least Book/Chapter/Page.md structure", path)
			return nil
		}

		book := imp.GetBook(segments[0])
		chapter := imp.GetChapter(segments[1], book.ID)
		pageName := strings.Join(segments[2:], "/")

		return imp.processMarkdownFile(fullPath, pageName, chapter.ID)
	})
}

// ImportNotionExport imports Notion export with structure:
// - Root folder = Book name
// - .md files in root = Pages in a default chapter
// - Subfolders = Chapters
// - .md files in subfolders = Pages in that chapter
func (imp *bookstackImport) ImportNotionExport(importPath string, bookName string) error {
	book := imp.GetBook(bookName)
	
	// Default chapter for pages directly in root
	defaultChapter := imp.GetChapter("General", book.ID)
	
	return filepath.WalkDir(importPath, func(fullPath string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip media/image folders
			if info.Name() == "media" || info.Name() == "docx" || info.Name() == "images" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(fullPath, ".md") {
			return nil
		}

		path := fullPath[len(importPath)+1:]
		segments := strings.FieldsFunc(path, IsDirSeparator)
		
		var chapterID int
		var pageName string
		
		if len(segments) == 1 {
			// File directly in root -> use default chapter
			chapterID = defaultChapter.ID
			pageName = CleanNotionPageName(segments[0])
		} else {
			// File in subfolder -> subfolder is chapter name
			chapterName := CleanNotionPageName(segments[0])
			chapter := imp.GetChapter(chapterName, book.ID)
			chapterID = chapter.ID
			// Combine remaining segments as page name
			pageName = CleanNotionPageName(strings.Join(segments[1:], "/"))
		}

		return imp.processMarkdownFile(fullPath, pageName, chapterID)
	})
}

// CleanNotionPageName removes Notion's UUID suffix from page names
// Example: "DM Hub 96d5b0bea9d7476c88788225804e3b27.md" -> "DM Hub"
func CleanNotionPageName(name string) string {
	// Remove .md extension
	name = strings.TrimSuffix(name, ".md")
	
	// Notion adds a 32-character hex ID at the end, separated by space
	// Pattern: "Page Name abc123def456..."
	if len(name) > 33 {
		// Check if last 32 chars are hex
		potentialID := name[len(name)-32:]
		isHex := true
		for _, c := range potentialID {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				isHex = false
				break
			}
		}
		if isHex && len(name) > 33 && name[len(name)-33] == ' ' {
			name = name[:len(name)-33]
		}
	}
	
	return strings.TrimSpace(name)
}

// processMarkdownFile handles the common logic for processing a markdown file
func (imp *bookstackImport) processMarkdownFile(fullPath string, pageName string, chapterID int) error {
	log.Printf("Processing: %s -> Chapter %d / %s", fullPath, chapterID, pageName)
	
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Remove YAML frontmatter if exists (must start with --- on first line)
	content = RemoveYAMLFrontmatter(content)

	// Clean up markdown for better BookStack compatibility
	content = CleanMarkdownForBookStack(content)

	// Pfeile sind in OneNote mit WingDings formatiert, durch ASCII-Pfeile ersetzen
	content = []byte(bytes.ReplaceAll(content, []byte("à"), []byte("->")))

	pageID, err := imp.GetPageID(pageName, chapterID)
	if err != nil {
		return fmt.Errorf("get page ID: %w", err)
	}

	// Use regex-based image replacement for better handling of special characters
	content, err = imp.ReplaceAllImagesRegex(pageID, content, fullPath)
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

	imp.GetPage(pageName, chapterID, content)
	return nil
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
		src := SafeUnquote("\"" + string(content[parenthesisStart+1:parenthesisEnd]) + "\"")

		if strings.HasPrefix(string(src), "onenote:") {
			log.Println("Found onenote link", src)
		}
	}

	return content, nil
}

func SafeUnquote(text string) string {
	unquoted, err := strconv.Unquote(text)
	if err != nil {
		log.Printf("unquote failed for: '%s': %s", text, err)
		return text
	}

	return unquoted
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
		rawSrc := string(content[parenthesisStart+1 : parenthesisEnd])
		
		// Handle URL-encoded paths and paths with spaces
		src := rawSrc
		// Try to URL-decode the path first
		if decoded, err := url.PathUnescape(rawSrc); err == nil {
			src = decoded
		}
		// Also try SafeUnquote for escaped characters
		src = SafeUnquote("\"" + src + "\"")
		
		imgPath := filepath.Join(filepath.Dir(path), src)
		
		// Log the image path for debugging
		log.Printf("Processing image: name=%s, rawSrc=%s, src=%s, imgPath=%s", string(name), rawSrc, src, imgPath)
		
		image, err := imp.Client.UploadImage(pageID, string(name), imgPath)
		if err != nil {
			log.Printf("Warning: failed to upload image %s: %v", imgPath, err)
			// Continue processing other images instead of failing completely
			continue
		}

		src = image.Path
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
		src := SafeUnquote("\"" + string(content[parenthesisStart+1:parenthesisEnd]) + "\"")
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

// RemoveYAMLFrontmatter removes YAML frontmatter from markdown content
// YAML frontmatter must start with --- on the very first line and end with ---
func RemoveYAMLFrontmatter(content []byte) []byte {
	text := string(content)
	lines := strings.Split(text, "\n")
	
	// Check if file starts with --- (YAML frontmatter)
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		// No YAML frontmatter, return as is
		return content
	}
	
	// Find the closing ---
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			// Found closing ---, remove frontmatter
			// Skip to content after frontmatter
			remainingLines := lines[i+1:]
			// Trim leading empty lines
			for len(remainingLines) > 0 && strings.TrimSpace(remainingLines[0]) == "" {
				remainingLines = remainingLines[1:]
			}
			return []byte(strings.Join(remainingLines, "\n"))
		}
	}
	
	// No closing --- found, return original content
	return content
}

// CleanMarkdownForBookStack cleans up markdown content to be more compatible with BookStack
func CleanMarkdownForBookStack(content []byte) []byte {
	text := string(content)

	// 1. Fix regular links (not images) with spaces in URLs
	// Use negative lookbehind simulation - only match if not preceded by !
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		// Skip table separator lines (| --- | --- |)
		if regexp.MustCompile(`^\|[\s-:|]+\|$`).MatchString(strings.TrimSpace(line)) {
			continue
		}
		
		// Process links but preserve image syntax
		// Find all [text](url) patterns that are NOT preceded by !
		linkRegex := regexp.MustCompile(`([^!]|^)\[([^\]]+)\]\(([^)]+)\)`)
		lines[i] = linkRegex.ReplaceAllStringFunc(line, func(match string) string {
			submatches := linkRegex.FindStringSubmatch(match)
			if len(submatches) == 4 {
				prefix := submatches[1]
				linkText := submatches[2]
				linkURL := submatches[3]
				// URL encode spaces in the URL
				linkURL = strings.ReplaceAll(linkURL, " ", "%20")
				return fmt.Sprintf("%s[%s](%s)", prefix, linkText, linkURL)
			}
			return match
		})
	}
	text = strings.Join(lines, "\n")

	// 2. Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")

	// 3. Ensure proper spacing around headers (but not at the start of file)
	headerRegex := regexp.MustCompile(`\n(#{1,6})\s+`)
	text = headerRegex.ReplaceAllString(text, "\n\n$1 ")

	// 4. Clean up excessive blank lines (more than 2 consecutive)
	multipleNewlines := regexp.MustCompile(`\n{4,}`)
	text = multipleNewlines.ReplaceAllString(text, "\n\n\n")

	// 5. Fix code blocks with language specifiers that might not be recognized
	// Replace jsx with javascript for better compatibility
	text = strings.ReplaceAll(text, "```jsx", "```javascript")

	// 6. Handle image paths with spaces - URL encode them
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\(\.\/([^)]+)\)`)
	text = imageRegex.ReplaceAllStringFunc(text, func(match string) string {
		submatches := imageRegex.FindStringSubmatch(match)
		if len(submatches) == 3 {
			altText := submatches[1]
			imgPath := submatches[2]
			// Don't encode if already encoded
			if !strings.Contains(imgPath, "%20") {
				imgPath = strings.ReplaceAll(imgPath, " ", "%20")
			}
			return fmt.Sprintf("![%s](./%s)", altText, imgPath)
		}
		return match
	})

	return []byte(text)
}

// ReplaceAllImagesRegex uses regex for more reliable image replacement
func (imp *bookstackImport) ReplaceAllImagesRegex(pageID int, content []byte, mdPath string) ([]byte, error) {
	text := string(content)
	
	// Match markdown images: ![alt](path)
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	
	// Find all matches and their positions
	matches := imageRegex.FindAllStringSubmatchIndex(text, -1)
	
	// Process matches in reverse order to maintain correct positions
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		fullMatchStart := match[0]
		fullMatchEnd := match[1]
		altStart := match[2]
		altEnd := match[3]
		pathStart := match[4]
		pathEnd := match[5]
		
		altText := text[altStart:altEnd]
		rawPath := text[pathStart:pathEnd]
		
		// Skip external URLs
		if strings.HasPrefix(rawPath, "http://") || strings.HasPrefix(rawPath, "https://") {
			continue
		}
		
		// Decode URL-encoded paths
		imgSrc := rawPath
		if decoded, err := url.PathUnescape(rawPath); err == nil {
			imgSrc = decoded
		}
		
		// Build full path to image
		imgPath := filepath.Join(filepath.Dir(mdPath), imgSrc)
		
		log.Printf("Processing image: alt=%s, rawPath=%s, imgPath=%s", altText, rawPath, imgPath)
		
		// Upload image
		image, err := imp.Client.UploadImage(pageID, altText, imgPath)
		if err != nil {
			log.Printf("Warning: failed to upload image %s: %v", imgPath, err)
			// Keep original markdown if upload fails
			continue
		}
		
		// Replace with new image path
		newImageMd := fmt.Sprintf("![%s](%s)", altText, image.Path)
		text = text[:fullMatchStart] + newImageMd + text[fullMatchEnd:]
	}
	
	return []byte(text), nil
}
