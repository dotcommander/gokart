package actions

import (
	"testing"
)

func TestGreet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   GreetInput
		want    string
		wantErr error
	}{
		{
			name:    "simple greeting",
			input:   GreetInput{Name: "World", Loud: false},
			want:    "Hello, World",
			wantErr: nil,
		},
		{
			name:    "loud greeting",
			input:   GreetInput{Name: "World", Loud: true},
			want:    "HELLO, WORLD!",
			wantErr: nil,
		},
		{
			name:    "empty name",
			input:   GreetInput{Name: "", Loud: false},
			want:    "",
			wantErr: ErrNameRequired,
		},
		{
			name:    "whitespace name",
			input:   GreetInput{Name: "   ", Loud: false},
			want:    "",
			wantErr: ErrNameRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Greet(tt.input)
			if err != tt.wantErr {
				t.Errorf("Greet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Greet() = %q, want %q", got, tt.want)
			}
		})
	}
}