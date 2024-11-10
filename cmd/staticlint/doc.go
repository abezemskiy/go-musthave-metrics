/*
main - основной пакет мультианализатора staticlint.

Запуск анализатора из папки с исполняемым файлом

	./staticlint ./../../...

Эта команда запустит анализатор со всеми реализованными проверками в проекте go-musthave-metrics.

# Доступные анализаторы

staticlint объединяет в себе такие анализаторы как:
- mainexitcheckanalyzer
- анализаторы из пакета golang.org/x/tools/go/analysis/passes bn
- анализаторы из пакета honnef.co/go/tools/staticcheck
*/
package main
