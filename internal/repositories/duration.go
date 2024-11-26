package repositories

import (
	"encoding/json"
	"time"
)

// Duration оборачивает time.Duration для кастомной десериализации
type Duration struct {
	time.Duration
}

// UnmarshalJSON реализует кастомный Unmarshal для Duration
func (d *Duration) UnmarshalJSON(b []byte) error {
	// Убираем кавычки и парсим как строку
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	// Парсим в time.Duration
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = duration
	return nil
}
