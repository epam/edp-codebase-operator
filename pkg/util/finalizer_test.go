package util

import (
	"reflect"
	"testing"
)

func TestContainsString(t *testing.T) {
	type args struct {
		slice []string
		s     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Contains_string", args{[]string{"foo", "bar", "buz"}, "bar"}, true},
		{"No_string", args{[]string{"foo", "bar", "buz"}, "asd"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsString(tt.args.slice, tt.args.s); got != tt.want {
				t.Errorf("ContainsString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveString(t *testing.T) {
	type args struct {
		slice []string
		s     string
	}
	tests := []struct {
		name       string
		args       args
		wantResult []string
	}{
		{"Remove_existing_string", args{[]string{"foo", "bar", "buz"}, "bar"}, []string{"foo", "buz"}},
		{"Nothing_to_remove", args{[]string{"foo", "bar", "buz"}, "asd"}, []string{"foo", "bar", "buz"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotResult := RemoveString(tt.args.slice, tt.args.s); !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("RemoveString() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}
