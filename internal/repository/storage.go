package repository

import "sync"

type gauge struct {
	Name  string
	Value float64
}
type counter struct {
	Name  string
	Value int64
}
type MemStorage struct {
	counterSlice []counter
	gaugeSlice   []gauge
}

func (s *MemStorage) GaugeSlice() []gauge {
	return s.gaugeSlice
}

func (s *MemStorage) CounterSlice() []counter {
	return s.counterSlice
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		counterSlice: make([]counter, 0),
		gaugeSlice:   make([]gauge, 0),
	}
}

var mutex sync.Mutex

func (s *MemStorage) AddGauge(name string, value float64) {
	//fmt.Println(name, value)
	mutex.Lock()
	defer mutex.Unlock()
	for i, m := range s.gaugeSlice {
		if m.Name == name {
			s.gaugeSlice[i].Value = value
			return
		}
	}

	s.gaugeSlice = append(s.gaugeSlice, gauge{Name: name, Value: value})
}
func (s *MemStorage) AddCounter(name string, value int64) {
	mutex.Lock()
	defer mutex.Unlock()
	for i, m := range s.counterSlice {
		if m.Name == name {
			s.counterSlice[i].Value += value
			return
		}
	}

	s.counterSlice = append(s.counterSlice, counter{Name: name, Value: value})
}
