package checker

import (
	"math"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

// equalFloat - функция для проверки на равенство двух чисел типа float64.
func EqualFloat(n1 float64, n2 float64) bool {
	return (math.Abs(n1 - n2)) < 0.0001
}

// Equal - функция для проверки, что две метрики одинаковые.
func Equal(m1 repositories.Metric, m2 repositories.Metric) bool {
	if m1.ID != m2.ID {
		return false
	}
	if m1.MType != m2.MType {
		return false
	}
	switch m1.MType {
	case "gauge":
		if !EqualFloat(*m1.Value, *m2.Value) {
			return false
		}
	case "counter":
		if *m1.Delta != *m2.Delta {
			return false
		}
	}
	return true
}
