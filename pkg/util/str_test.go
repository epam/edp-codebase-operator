package util

import "testing"

func TestSearchVersion(t *testing.T) {
	type args struct {
		a []string
		b string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Should return false", args{a: []string{"0-0-0"}, b: "0-0-1"}, false},
		{"Should return false for len is zero", args{a: []string{}, b: "0-0-1"}, false},
		{"Should return false for len is zero", args{a: []string{"fake", "found"}, b: "found"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SearchVersion(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("SearchVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
