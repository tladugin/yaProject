package repository

import (
	"reflect"
	"testing"
)

func TestMemStorage_AddCounter(t *testing.T) {

	tests := []struct {
		name      string
		initial   []counter // начальное состояние
		inputName string    // имя счетчика
		inputVal  int64     // значение для добавления
		want      []counter // ожидаемое состояние
		wantLen   int       // ожидаемая длина слайса
	}{
		{
			name:      "Add new counter to empty storage",
			initial:   []counter{},
			inputName: "cnt1",
			inputVal:  10,
			want:      []counter{{Name: "cnt1", Value: 10}},
			wantLen:   1,
		},
		{
			name: "Add new counter to existing storage",
			initial: []counter{
				{Name: "cnt1", Value: 10},
			},
			inputName: "cnt2",
			inputVal:  5,
			want: []counter{
				{Name: "cnt1", Value: 10},
				{Name: "cnt2", Value: 5},
			},
			wantLen: 2,
		},
		{
			name: "Update existing counter",
			initial: []counter{
				{Name: "cnt1", Value: 10},
				{Name: "cnt2", Value: 5},
			},
			inputName: "cnt1",
			inputVal:  3,
			want: []counter{
				{Name: "cnt1", Value: 13}, // 10 + 3
				{Name: "cnt2", Value: 5},
			},
			wantLen: 2,
		},
		{
			name: "Update with negative value",
			initial: []counter{
				{Name: "cnt1", Value: 10},
			},
			inputName: "cnt1",
			inputVal:  -2,
			want: []counter{
				{Name: "cnt1", Value: 8}, // 10 + (-2)
			},
			wantLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MemStorage{
				counterSlice: tt.initial,
			}
			s.AddCounter(tt.inputName, tt.inputVal)
			if len(s.counterSlice) != tt.wantLen {
				t.Errorf("got length %d, want %d", len(s.counterSlice), tt.wantLen)
			}

			// Проверка содержимого слайса
			for i, c := range s.counterSlice {
				if c.Name != tt.want[i].Name || c.Value != tt.want[i].Value {
					t.Errorf("at index %d: got {Name: %s, Value: %d}, want {Name: %s, Value: %d}",
						i, c.Name, c.Value, tt.want[i].Name, tt.want[i].Value)
				}
			}
		})
	}
}

func TestMemStorage_AddGauge(t *testing.T) {
	/*type fields struct {
		counterSlice []counter
		gaugeSlice   []gauge
	}
	type args struct {
		name  string
		value float64
	}

	*/
	tests := []struct {
		name      string
		initial   []gauge
		inputName string
		inputVal  float64
		want      []gauge
		wantLen   int
	}{

		{
			name:      "Add new gauge to empty storage",
			initial:   []gauge{},
			inputName: "test1",
			inputVal:  10.5,
			want:      []gauge{{Name: "test1", Value: 10.5}},
			wantLen:   1,
		},
		{
			name: "Add new gauge to non-empty storage",
			initial: []gauge{
				{Name: "test1", Value: 10.5},
			},
			inputName: "test2",
			inputVal:  20.3,
			want: []gauge{
				{Name: "test1", Value: 10.5},
				{Name: "test2", Value: 20.3},
			},
			wantLen: 2,
		},
		{
			name: "Update existing gauge",
			initial: []gauge{
				{Name: "test1", Value: 10.5},
				{Name: "test2", Value: 20.3},
			},
			inputName: "test1",
			inputVal:  15.7,
			want: []gauge{
				{Name: "test1", Value: 15.7},
				{Name: "test2", Value: 20.3},
			},
			wantLen: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MemStorage{
				gaugeSlice: tt.initial,
			}
			s.AddGauge(tt.inputName, tt.inputVal)
			if len(s.gaugeSlice) != tt.wantLen {
				t.Errorf("got slice length %d, want %d", len(s.gaugeSlice), tt.wantLen)
			}
			for i, g := range s.gaugeSlice {
				if g.Name != tt.want[i].Name || g.Value != tt.want[i].Value {
					t.Errorf("at index %d: got {Name: %s, Value: %f}, want {Name: %s, Value: %f}",
						i, g.Name, g.Value, tt.want[i].Name, tt.want[i].Value)
				}
			}
		})
	}
}

func TestMemStorage_CounterSlice(t *testing.T) {
	type fields struct {
		counterSlice []counter
		gaugeSlice   []gauge
	}
	tests := []struct {
		name   string
		fields fields
		want   []counter
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MemStorage{
				counterSlice: tt.fields.counterSlice,
				gaugeSlice:   tt.fields.gaugeSlice,
			}
			if got := s.CounterSlice(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CounterSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemStorage_GaugeSlice(t *testing.T) {
	type fields struct {
		counterSlice []counter
		gaugeSlice   []gauge
	}
	tests := []struct {
		name   string
		fields fields
		want   []gauge
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MemStorage{
				counterSlice: tt.fields.counterSlice,
				gaugeSlice:   tt.fields.gaugeSlice,
			}
			if got := s.GaugeSlice(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GaugeSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewMemStorage(t *testing.T) {
	tests := []struct {
		name string
		want *MemStorage
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMemStorage(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMemStorage() = %v, want %v", got, tt.want)
			}
		})
	}
}
