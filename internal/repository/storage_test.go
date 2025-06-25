package repository

import (
	"reflect"
	"testing"
)

func TestMemStorage_AddCounter(t *testing.T) {
	type fields struct {
		counterSlice []counter
		gaugeSlice   []gauge
	}
	type args struct {
		name  string
		value int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MemStorage{
				counterSlice: tt.fields.counterSlice,
				gaugeSlice:   tt.fields.gaugeSlice,
			}
			s.AddCounter(tt.args.name, tt.args.value)
		})
	}
}

func TestMemStorage_AddGauge(t *testing.T) {
	type fields struct {
		counterSlice []counter
		gaugeSlice   []gauge
	}
	type args struct {
		name  string
		value float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MemStorage{
				counterSlice: tt.fields.counterSlice,
				gaugeSlice:   tt.fields.gaugeSlice,
			}
			s.AddGauge(tt.args.name, tt.args.value)
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
