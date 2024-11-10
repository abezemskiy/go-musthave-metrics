package repositories

import (
	"compress/gzip"
	"io"
	"net/http"
)

// CompressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки.
type CompressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

// NewCompressWriter - фабричная функция для создания структуры CompressWriter.
func NewCompressWriter(w http.ResponseWriter) *CompressWriter {
	return &CompressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

// Header - установка заголовка.
func (c *CompressWriter) Header() http.Header {
	return c.w.Header()
}

// Write - запись ответа.
func (c *CompressWriter) Write(p []byte) (int, error) {
	// Устанавливаю заголовок о том, что данные сжаты, в основном на случай, когда в теле ответа будет содержаться ошибка
	// и агенту нужно будет корректно распаковать полученное от сервера тело с ошибкой
	c.w.Header().Set("Content-Encoding", "gzip")

	return c.zw.Write(p)
}

// WriteHeader - установка заголовка.
func (c *CompressWriter) WriteHeader(statusCode int) {
	// Устанавливаю заголовок о том, что данные сжаты, в основном на случай, когда в теле ответа будет содержаться ошибка
	// и агенту нужно будет корректно распаковать полученное от сервера тело с ошибкой
	c.w.Header().Set("Content-Encoding", "gzip")

	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *CompressWriter) Close() error {
	return c.zw.Close()
}

// CompressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные.
type CompressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// NewCompressReader - фабричная функци для создания структуры CompressReader.
func NewCompressReader(r io.ReadCloser) (*CompressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &CompressReader{
		r:  r,
		zr: zr,
	}, nil
}

// Read - метод чтения.
func (c CompressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Close - обертка над gzip.Reader_Close.
func (c *CompressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}
