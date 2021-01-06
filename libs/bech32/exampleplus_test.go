package bech32_test

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/tendermint/tendermint/libs/bech32"
)

// This example demonstrates how to decode a bech32 encoded string.
func TestExampleDecode(t *testing.T) {
	encoded := "bc1pw508d6qejxtdg4y5r3zarvary0c5xw7kw508d6qejxtdg4y5r3zarvary0c5xw7k7grplx"
	hrp, decoded, err := bech32.Decode(encoded)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Show the decoded data.
	fmt.Println("Decoded human-readable part:", hrp)
	fmt.Println("Decoded Data:", hex.EncodeToString(decoded))

	// Output:
	// Decoded human-readable part: bc
	// Decoded Data: 010e140f070d1a001912060b0d081504140311021d030c1d03040f1814060e1e160e140f070d1a001912060b0d081504140311021d030c1d03040f1814060e1e16
}

// This example demonstrates how to encode data into a bech32 string.
func TestExampleEncode(t *testing.T) {
	data := []byte("Test data")
	// Convert test data to base32:
	conv, err := bech32.ConvertBits(data, 8, 5, true)
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Println(conv)
	encoded, err := bech32.Encode("customHrp!11111q", conv)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Show the encoded data.
	fmt.Println("Encoded Data:", encoded)

	// Output:
	// Encoded Data: customHrp!11111q123jhxapqv3shgcgumastr
}

// This example demonstrates how to encode data into a bech32 string.
func TestExampleEncodeDecode(t *testing.T) {
	// ------ encode --------
	data := []byte("Test data")
	// Convert test data to base32:
	conv, err := bech32.ConvertBits(data, 8, 5, true)
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Println(conv)
	encoded, err := bech32.Encode("linkvalconspub", conv)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Show the encoded data.
	fmt.Println("Encoded Data:", encoded)

	// -------- decode ---------------
	// kokeshi Error: string not all lowercase or all uppercase
	encoded = strings.ToLower(encoded)

	// decode start
	hrp, decoded, err := bech32.Decode(encoded)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Show the decoded data.
	fmt.Println("Decoded human-readable part:", hrp)
	fmt.Println("Decoded Data:", hex.EncodeToString(decoded))

	// Output:
	// Encoded Data: customHrp!11111q123jhxapqv3shgcgumastr
}
