package pdf2png

import (
	"image/png"
	"log"
	"os"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
)

const (
	// defaultDPI for rendering:
	defaultDPI = 200
)

var pool *pdfium.Pool
var instance pdfium.Pdfium

func init() {
	// Init the PDFium library and return the instance to open documents.
	// You can tweak these configs to your need. Be aware that workers can use quite some memory.
	pool, err := webassembly.Init(webassembly.Config{
		MinIdle:  1, // Makes sure that at least x workers are always available
		MaxIdle:  1, // Makes sure that at most x workers are ever available
		MaxTotal: 1, // Maxium amount of workers in total, allows the amount of workers to grow when needed, items between total max and idle max are automatically cleaned up, while idle workers are kept alive so they can be used directly.
	})
	if err != nil {
		log.Fatal(err)
	}

	instance, err = pool.GetInstance(time.Second * 30)
	if err != nil {
		log.Fatal(err)
	}
}

// RenderPage tuns a specific page of the PDF into a PNG file:
func RenderPage(filePath string, output string, page int) error {
	pdfBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	doc, err := instance.OpenDocument(&requests.OpenDocument{
		File: &pdfBytes,
	})
	if err != nil {
		return err
	}
	defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{
		Document: doc.Document,
	})

	pageRender, err := instance.RenderPageInDPI(&requests.RenderPageInDPI{
		DPI: defaultDPI,
		Page: requests.Page{
			ByIndex: &requests.PageByIndex{
				Document: doc.Document,
				Index:    page,
			},
		},
	})
	if err != nil {
		return err
	}
	defer pageRender.Cleanup()

	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()

	err = png.Encode(f, pageRender.Result.Image)
	if err != nil {
		return err
	}

	return nil
}

// GetPageCount returns the amount of pages in a PDF file:
func GetPageCount(filePath string) (int, error) {
	pdfBytes, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}

	doc, err := instance.OpenDocument(&requests.OpenDocument{
		File: &pdfBytes,
	})
	if err != nil {
		return 0, err
	}
	defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{
		Document: doc.Document,
	})

	pageCount, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{
		Document: doc.Document,
	})
	if err != nil {
		return 0, err
	}

	return pageCount.PageCount, nil
}
