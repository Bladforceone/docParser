package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Функция для скачивания файла с тайм-аутом
func downloadFile(fileURL, filePath string) error {
	client := &http.Client{
		Timeout: 30 * time.Second, // Тайм-аут 30 секунд
	}

	// Выполняем запрос для скачивания файла
	resp, err := client.Get(fileURL)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем успешность загрузки
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Создаем файл для сохранения
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Копируем содержимое ответа в файл
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

// Общая функция для выполнения HTTP-запросов
func fetchHTML(url string) (*html.Node, error) {
	client := &http.Client{
		Timeout: 30 * time.Second, // Тайм-аут 30 секунд
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем успешность загрузки
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	// Парсим HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return doc, nil
}

// Функция для поиска всех .doc файлов на странице
func findAllDocFiles(pageURL string) ([]string, error) {
	doc, err := fetchHTML(pageURL)
	if err != nil {
		return nil, err
	}

	var docFiles []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" && strings.HasSuffix(attr.Val, ".doc") {
					docFiles = append(docFiles, attr.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return docFiles, nil
}

// Функция для конструирования полного URL
func resolveURL(baseURL string, relativeURL string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}
	ref, err := url.Parse(relativeURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse relative URL: %w", err)
	}

	// Конструируем полный URL
	return base.ResolveReference(ref).String(), nil
}

func main() {
	// URL сайта для парсинга
	baseURL := "https://tstu.ru/r.php?r=tgtu.general.docum.standart"

	// Парсим сайт и ищем ссылки на .doc файлы
	docFiles, err := findAllDocFiles(baseURL)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	if len(docFiles) == 0 {
		fmt.Println("no .doc files found")
		return
	}

	// Создаем директорию для загрузок
	if err := os.MkdirAll("downloads", os.ModePerm); err != nil {
		fmt.Println("error creating directory:", err)
		return
	}

	// Параллельное скачивание файлов
	for _, relativeFileURL := range docFiles {
		go func(relativeFileURL string) {
			// Преобразуем относительный путь в полный URL
			fullURL, err := resolveURL(baseURL, relativeFileURL)
			if err != nil {
				fmt.Printf("error resolving URL %s: %v\n", relativeFileURL, err)
				return
			}

			// Скачиваем файл
			fileName := path.Base(fullURL)
			savePath := path.Join("downloads", fileName)

			fmt.Println("downloading file:", fullURL)
			if err := downloadFile(fullURL, savePath); err != nil {
				fmt.Printf("error downloading %s: %v\n", fullURL, err)
			} else {
				fmt.Printf("file %s successfully downloaded\n", fileName)
			}
		}(relativeFileURL)
	}

	// Ожидаем завершения скачивания (в простом случае можно использовать time.Sleep)
	time.Sleep(10 * time.Second)
}
