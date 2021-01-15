package bech32_test

import (
	"strings"
	"testing"

	"github.com/tendermint/tendermint/libs/bech32"
)

func TestBech32(t *testing.T) {
	tests := []struct {
		str   string
		valid bool
	}{
		{"A12UEL5L", true},
		{"an83characterlonghumanreadablepartthatcontainsthenumber1andtheexcludedcharactersbio1tt5tgs", true},
		{"abcdef1qpzry9x8gf2tvdw0s3jn54khce6mua7lmqqqxw", true},
		{"11qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqc8247j", true},
		{"split1checkupstagehandshakeupstreamerranterredcaperred2y9e3w", true},
		{"split1checkupstagehandshakeupstreamerranterredcaperred2y9e2w", false},                         // invalid checksum
		{"s lit1checkupstagehandshakeupstreamerranterredcaperredp8hs2p", false},                         // invalid character (space) in hrp
		{"spl" + string(rune(127)) + "t1checkupstagehandshakeupstreamerranterredcaperred2y9e3w", false}, // invalid character (DEL) in hrp
		{"split1cheo2y9e2w", false}, // invalid character (o) in data part
		{"split1a2y9w", false},      // too short data part
		{"1checkupstagehandshakeupstreamerranterredcaperred2y9e3w", false}, // empty hrp
		// 165 characters plus using LIMIT
		{"linkvalconspub1668lhsfs4zxq4gakkq5yj8ujn22g43s7qz7498el7k57cw88t30y56sxa6eszusd470ncx3j79slqg3aphy2c93ymejzqxzpu36pjrv5u0wphpf5eluc7am08m0ekkfns4jcnrj7u2hc5f6240tlze", true},
		// 201 characters plus using LIMIT
		{"linkvalconspub1668lhsfs4zxq4gakkq5yj8ujn22g43s7qz7498el7k57cw88t30y56sxa6eszusd470ncx3j79slqg3aphy2c93ymejzqxzpu36pjrv5u0wphpf5eluc7am08m0ekkfns4jcnrj7u2hc5f6240tlze668lhsfs4zxq4gakkq5yj8ujn22g43s7qz74", false},
	}

	for _, test := range tests {
		str := test.str
		hrp, decoded, err := bech32.Decode(str)
		if !test.valid {
			// Invalid string decoding should result in error.
			if err == nil {
				t.Error("expected decoding to fail for "+
					"invalid string %v", test.str)
			}
			continue
		}

		// Valid string decoding should result in no error.
		if err != nil {
			t.Errorf("expected string to be valid bech32: %v", err)
		}

		// Check that it encodes to the same string
		encoded, err := bech32.Encode(hrp, decoded)
		if err != nil {
			t.Errorf("encoding failed: %v", err)
		}

		if encoded != strings.ToLower(str) {
			t.Errorf("expected data to encode to %v, but got %v",
				str, encoded)
		}

		// Flip a bit in the string an make sure it is caught.
		pos := strings.LastIndexAny(str, "1")
		flipped := str[:pos+1] + string((str[pos+1] ^ 1)) + str[pos+2:]
		_, _, err = bech32.Decode(flipped)
		if err == nil {
			t.Error("expected decoding to fail")
		}
	}
}
