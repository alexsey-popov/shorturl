package compact

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/alexsey-popov/shorturl/pkg/content_type"
)

// CompactWriter - обвёртка ResponseWriter, которая записывает данные в кастомный io.Writer
type CompactWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

// Переопределяем метод записи
func (cw CompactWriter) Write(data []byte) (int, error) {
	return cw.Writer.Write(data)
}

// HTTPMiddleware - обработчик для сжатия/распаковки данных
func HTTPMiddleware(handler http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Распаковываем данные только если они подходят по формату
		requestContentType := r.Header.Get("Content-Type")
		if requestContentType == content_type.Json || requestContentType == content_type.Plain {
			// Проверяем наличие сжатия данных на входе
			contentEncoding := r.Header.Get("Content-Encoding")
			if strings.Contains(contentEncoding, "gzip") {
				gzipReader, err := gzip.NewReader(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				defer gzipReader.Close()
				r.Body = gzipReader
			}
		}

		// Запаковываем данные только если они подходят по формату
		ResponseContentType := w.Header().Get("Content-Type")
		if ResponseContentType == content_type.Json || ResponseContentType == content_type.Plain {
			// Проверяем возможность сжатия данных на выходе
			acceptEncoding := r.Header.Get("Accept-Encoding")
			if strings.Contains(acceptEncoding, "gzip") {
				gz := gzip.NewWriter(w)
				defer gz.Close()

				w.Header().Set("Content-Encoding", "gzip")
				w = CompactWriter{ResponseWriter: w, Writer: gz}
			}
		}

		handler.ServeHTTP(w, r)
	})

}
