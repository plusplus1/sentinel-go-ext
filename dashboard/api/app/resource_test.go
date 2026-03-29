package app

import "testing"

func TestRulesChanged(t *testing.T) {
	tests := []struct {
		name          string
		currentFlow   []map[string]interface{}
		publishedFlow []map[string]interface{}
		currentCB     []map[string]interface{}
		publishedCB   []map[string]interface{}
		want          bool
	}{
		{
			name:          "both empty",
			currentFlow:   []map[string]interface{}{},
			publishedFlow: []map[string]interface{}{},
			currentCB:     []map[string]interface{}{},
			publishedCB:   []map[string]interface{}{},
			want:          false,
		},
		{
			name:          "identical flow rules",
			currentFlow:   []map[string]interface{}{{"resource": "test", "threshold": 10}},
			publishedFlow: []map[string]interface{}{{"resource": "test", "threshold": 10}},
			currentCB:     []map[string]interface{}{},
			publishedCB:   []map[string]interface{}{},
			want:          false,
		},
		{
			name:          "flow rule added",
			currentFlow:   []map[string]interface{}{{"resource": "test", "threshold": 10}},
			publishedFlow: []map[string]interface{}{},
			currentCB:     []map[string]interface{}{},
			publishedCB:   []map[string]interface{}{},
			want:          true,
		},
		{
			name:          "flow rule removed",
			currentFlow:   []map[string]interface{}{},
			publishedFlow: []map[string]interface{}{{"resource": "test", "threshold": 10}},
			currentCB:     []map[string]interface{}{},
			publishedCB:   []map[string]interface{}{},
			want:          true,
		},
		{
			name:          "flow rule value changed",
			currentFlow:   []map[string]interface{}{{"resource": "test", "threshold": 20}},
			publishedFlow: []map[string]interface{}{{"resource": "test", "threshold": 10}},
			currentCB:     []map[string]interface{}{},
			publishedCB:   []map[string]interface{}{},
			want:          true,
		},
		{
			name:          "cb rule added",
			currentFlow:   []map[string]interface{}{},
			publishedFlow: []map[string]interface{}{},
			currentCB:     []map[string]interface{}{{"resource": "test", "threshold": 5}},
			publishedCB:   []map[string]interface{}{},
			want:          true,
		},
		{
			name:          "cb rule value changed",
			currentFlow:   []map[string]interface{}{},
			publishedFlow: []map[string]interface{}{},
			currentCB:     []map[string]interface{}{{"resource": "test", "threshold": 5}},
			publishedCB:   []map[string]interface{}{{"resource": "test", "threshold": 10}},
			want:          true,
		},
		{
			name:          "same count different content",
			currentFlow:   []map[string]interface{}{{"resource": "test", "threshold": 10}},
			publishedFlow: []map[string]interface{}{{"resource": "test", "threshold": 20}},
			currentCB:     []map[string]interface{}{},
			publishedCB:   []map[string]interface{}{},
			want:          true,
		},
		{
			name:          "int vs float64 equality",
			currentFlow:   []map[string]interface{}{{"resource": "test", "threshold": 10}},
			publishedFlow: []map[string]interface{}{{"resource": "test", "threshold": 10.0}},
			currentCB:     []map[string]interface{}{},
			publishedCB:   []map[string]interface{}{},
			want:          false,
		},
		{
			name:          "int64 vs float64 equality",
			currentFlow:   []map[string]interface{}{{"resource": "test", "threshold": int64(10)}},
			publishedFlow: []map[string]interface{}{{"resource": "test", "threshold": 10.0}},
			currentCB:     []map[string]interface{}{},
			publishedCB:   []map[string]interface{}{},
			want:          false,
		},
		{
			name:          "identical cb rules",
			currentFlow:   []map[string]interface{}{},
			publishedFlow: []map[string]interface{}{},
			currentCB:     []map[string]interface{}{{"resource": "test", "strategy": 0, "threshold": 0.5}},
			publishedCB:   []map[string]interface{}{{"resource": "test", "strategy": 0, "threshold": 0.5}},
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rulesChanged(tt.currentFlow, tt.publishedFlow, tt.currentCB, tt.publishedCB)
			if got != tt.want {
				t.Errorf("rulesChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapEqual(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]interface{}
		b    map[string]interface{}
		want bool
	}{
		{
			name: "identical maps",
			a:    map[string]interface{}{"key": "value", "num": 10},
			b:    map[string]interface{}{"key": "value", "num": 10},
			want: true,
		},
		{
			name: "different values",
			a:    map[string]interface{}{"key": "value1"},
			b:    map[string]interface{}{"key": "value2"},
			want: false,
		},
		{
			name: "different keys",
			a:    map[string]interface{}{"key1": "value"},
			b:    map[string]interface{}{"key2": "value"},
			want: false,
		},
		{
			name: "different lengths",
			a:    map[string]interface{}{"key": "value", "extra": 1},
			b:    map[string]interface{}{"key": "value"},
			want: false,
		},
		{
			name: "int vs float64",
			a:    map[string]interface{}{"num": 10},
			b:    map[string]interface{}{"num": 10.0},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("mapEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueEqual(t *testing.T) {
	tests := []struct {
		name string
		a    interface{}
		b    interface{}
		want bool
	}{
		{"int int", 10, 10, true},
		{"int float64", 10, 10.0, true},
		{"int64 float64", int64(10), 10.0, true},
		{"float32 float64", float32(10.5), 10.5, true},
		{"string string", "test", "test", true},
		{"different int", 10, 20, false},
		{"different string", "test1", "test2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := valueEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("valueEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
