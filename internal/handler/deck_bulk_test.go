package handler

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestParseDeckLine(t *testing.T) {
	tests := []struct {
		line    string
		want    deckLine
		wantErr bool
	}{
		{"4 Lightning Bolt", deckLine{Quantity: 4, Name: "Lightning Bolt"}, false},
		{"1x Sol Ring", deckLine{Quantity: 1, Name: "Sol Ring"}, false},
		{"City of Brass", deckLine{Quantity: 1, Name: "City of Brass"}, false},
		{"  2 City of Brass  ", deckLine{Quantity: 2, Name: "City of Brass"}, false},
		{"1 Atraxa, Praetors' Voice *F*", deckLine{Quantity: 1, Name: "Atraxa, Praetors' Voice"}, false},
		{"4 Lightning Bolt (M20) 141", deckLine{Quantity: 4, Name: "Lightning Bolt", SetCode: "m20", CollectorNumber: "141"}, false},
		{"1 Teferi, Hero of Dominaria (DOM) 207 *F*", deckLine{Quantity: 1, Name: "Teferi, Hero of Dominaria", SetCode: "dom", CollectorNumber: "207"}, false},
		{"1 Delver of Secrets // Insectile Aberration", deckLine{Quantity: 1, Name: "Delver of Secrets // Insectile Aberration"}, false},
		// A bare number is a quantity with no name.
		{"0 Sol Ring", deckLine{}, true},
		{"", deckLine{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got, err := parseDeckLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		in, want string
		wantErr  bool
	}{
		{"", "en", false},
		{"  ", "en", false},
		{"ja", "ja", false},
		{"DE", "de", false},
		{"zhs", "zhs", false},
		{"english", "", true},
		{"e", "", true},
		{"en-US", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := normalizeLanguage(tt.in)
			if tt.wantErr {
				if status.Code(err) != codes.InvalidArgument {
					t.Fatalf("want InvalidArgument, got %v", err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("got (%q, %v), want %q", got, err, tt.want)
			}
		})
	}
}
