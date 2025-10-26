package repository

import (
	"sync"
)

// Типы метрик

// gauge представляет метрику типа "gauge" (значение произвольное)
type gauge struct {
	Name  string
	Value float64
}

// counter представляет метрику типа "counter" (значение накапливается)
type counter struct {
	Name  string
	Value int64
}

// MemStorage - хранилище метрик в памяти
type MemStorage struct {
	counterSlice []counter // Слайс для хранения метрик типа counter
	gaugeSlice   []gauge   // Слайс для хранения метрик типа gauge
}

// GaugeSlice возвращает слайс всех метрик типа gauge
func (s *MemStorage) GaugeSlice() []gauge {
	return s.gaugeSlice
}

// CounterSlice возвращает слайс всех метрик типа counter
func (s *MemStorage) CounterSlice() []counter {
	return s.counterSlice
}

// NewMemStorage создает новый экземпляр MemStorage с инициализированными слайсами
func NewMemStorage() *MemStorage {
	return &MemStorage{
		counterSlice: make([]counter, 0), // Инициализация пустого слайса для counter
		gaugeSlice:   make([]gauge, 0),   // Инициализация пустого слайса для gauge
	}
}

// mutex для обеспечения потокобезопасности операций с хранилищем
var mutex sync.Mutex

// AddGauge добавляет или обновляет метрику типа gauge
// Если метрика с таким именем уже существует - обновляет ее значение
// Если не существует - добавляет новую метрику
func (s *MemStorage) AddGauge(name string, value float64) {
	//fmt.Println(name, value)
	mutex.Lock()         // Блокируем мьютекс для обеспечения потокобезопасности
	defer mutex.Unlock() // Гарантируем разблокировку при выходе из функции

	// Поиск существующей метрики по имени
	for i, m := range s.gaugeSlice {
		if m.Name == name {
			// Если метрика найдена - обновляем ее значение
			s.gaugeSlice[i].Value = value
			return
		}
	}

	// Если метрика не найдена - добавляем новую
	s.gaugeSlice = append(s.gaugeSlice, gauge{Name: name, Value: value})
}

// AddCounter добавляет или обновляет метрику типа counter
// Если метрика с таким именем уже существует - увеличивает ее значение
// Если не существует - добавляет новую метрику с переданным значением
func (s *MemStorage) AddCounter(name string, value int64) {
	mutex.Lock()         // Блокируем мьютекс для обеспечения потокобезопасности
	defer mutex.Unlock() // Гарантируем разблокировку при выходе из функции

	// Поиск существующей метрики по имени
	for i, m := range s.counterSlice {
		if m.Name == name {
			// Если метрика найдена - увеличиваем ее значение (накапливаем)
			s.counterSlice[i].Value += value
			return
		}
	}

	// Если метрика не найдена - добавляем новую
	s.counterSlice = append(s.counterSlice, counter{Name: name, Value: value})
}
