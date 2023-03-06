package blockchain

import (
	"encoding/hex"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/line/ostracon/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"testing"

	bcproto "github.com/tendermint/tendermint/proto/tendermint/blockchain"

	ocbcproto "github.com/line/ostracon/proto/ostracon/blockchain"
	sm "github.com/line/ostracon/state"
	"github.com/line/ostracon/types"
)

func TestBcBlockRequestMessageValidateBasic(t *testing.T) {
	testCases := []struct {
		testName      string
		requestHeight int64
		expectErr     bool
	}{
		{"Valid Request Message", 0, false},
		{"Valid Request Message", 1, false},
		{"Invalid Request Message", -1, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			request := bcproto.BlockRequest{Height: tc.requestHeight}
			assert.Equal(t, tc.expectErr, ValidateMsg(&request) != nil, "Validate Basic had an unexpected result")
		})
	}
}

func TestBcNoBlockResponseMessageValidateBasic(t *testing.T) {
	testCases := []struct {
		testName          string
		nonResponseHeight int64
		expectErr         bool
	}{
		{"Valid Non-Response Message", 0, false},
		{"Valid Non-Response Message", 1, false},
		{"Invalid Non-Response Message", -1, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			nonResponse := bcproto.NoBlockResponse{Height: tc.nonResponseHeight}
			assert.Equal(t, tc.expectErr, ValidateMsg(&nonResponse) != nil, "Validate Basic had an unexpected result")
		})
	}
}

func TestBcStatusRequestMessageValidateBasic(t *testing.T) {
	request := bcproto.StatusRequest{}
	assert.NoError(t, ValidateMsg(&request))
}

func TestBcStatusResponseMessageValidateBasic(t *testing.T) {
	testCases := []struct {
		testName       string
		responseHeight int64
		expectErr      bool
	}{
		{"Valid Response Message", 0, false},
		{"Valid Response Message", 1, false},
		{"Invalid Response Message", -1, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			response := bcproto.StatusResponse{Height: tc.responseHeight}
			assert.Equal(t, tc.expectErr, ValidateMsg(&response) != nil, "Validate Basic had an unexpected result")
		})
	}
}

// nolint:lll // ignore line length in tests
func TestBlockchainMessageVectors(t *testing.T) {
	block := types.MakeBlock(int64(3), []types.Tx{types.Tx("Hello World")}, nil, nil, sm.InitStateVersion.Consensus)
	block.Version.Block = 11 // overwrite updated protocol version
	block.Version.App = 11   // overwrite updated protocol version

	bpb, err := block.ToProto()
	require.NoError(t, err)

	testCases := []struct {
		testName string
		bmsg     proto.Message
		expBytes string
	}{
		{"BlockRequestMessage", &ocbcproto.Message{Sum: &ocbcproto.Message_BlockRequest{
			BlockRequest: &bcproto.BlockRequest{Height: 1}}}, "0a020801"},
		{"BlockRequestMessage", &ocbcproto.Message{Sum: &ocbcproto.Message_BlockRequest{
			BlockRequest: &bcproto.BlockRequest{Height: math.MaxInt64}}},
			"0a0a08ffffffffffffffff7f"},
		{"BlockResponseMessage", &ocbcproto.Message{Sum: &ocbcproto.Message_BlockResponse{
			BlockResponse: &ocbcproto.BlockResponse{Block: bpb}}}, "1a750a730a5d0a04080b100b1803220b088092b8c398feffffff012a0212003a20c4da88e876062aa1543400d50d0eaa0dac88096057949cfb7bca7f3a48c04bf96a20e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855120d0a0b48656c6c6f20576f726c641a00c23e00"},
		{"NoBlockResponseMessage", &ocbcproto.Message{Sum: &ocbcproto.Message_NoBlockResponse{
			NoBlockResponse: &bcproto.NoBlockResponse{Height: 1}}}, "12020801"},
		{"NoBlockResponseMessage", &ocbcproto.Message{Sum: &ocbcproto.Message_NoBlockResponse{
			NoBlockResponse: &bcproto.NoBlockResponse{Height: math.MaxInt64}}},
			"120a08ffffffffffffffff7f"},
		{"StatusRequestMessage", &ocbcproto.Message{Sum: &ocbcproto.Message_StatusRequest{
			StatusRequest: &bcproto.StatusRequest{}}},
			"2200"},
		{"StatusResponseMessage", &ocbcproto.Message{Sum: &ocbcproto.Message_StatusResponse{
			StatusResponse: &bcproto.StatusResponse{Height: 1, Base: 2}}},
			"2a0408011002"},
		{"StatusResponseMessage", &ocbcproto.Message{Sum: &ocbcproto.Message_StatusResponse{
			StatusResponse: &bcproto.StatusResponse{Height: math.MaxInt64, Base: math.MaxInt64}}},
			"2a1408ffffffffffffffff7f10ffffffffffffffff7f"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			bz, _ := proto.Marshal(tc.bmsg)

			require.Equal(t, tc.expBytes, hex.EncodeToString(bz))
		})
	}
}

func TestEncodeDecodeValidateMsg(t *testing.T) {
	height := int64(3)
	block := types.MakeBlock(
		height,
		[]types.Tx{types.Tx("Hello World")},
		&types.Commit{
			Signatures: []types.CommitSig{
				{
					BlockIDFlag:      types.BlockIDFlagCommit,
					ValidatorAddress: make([]byte, crypto.AddressSize),
					Signature:        make([]byte, crypto.AddressSize),
				},
			},
		},
		nil,
		sm.InitStateVersion.Consensus)
	block.ProposerAddress = make([]byte, crypto.AddressSize)
	bpb, err := block.ToProto()
	require.NoError(t, err)

	type args struct {
		pb proto.Message
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "bcproto.BlockRequest",
			args:    args{pb: &bcproto.BlockRequest{}},
			want:    []byte{0xa, 0x0},
			wantErr: assert.NoError,
		},
		{
			name:    "ocbcproto.BlockResponse", // Ostracon
			args:    args{pb: &ocbcproto.BlockResponse{Block: bpb}},
			want:    []byte{0x1a, 0xf0, 0x1, 0xa, 0xed, 0x1, 0xa, 0x93, 0x1, 0xa, 0x2, 0x8, 0xb, 0x18, 0x3, 0x22, 0xb, 0x8, 0x80, 0x92, 0xb8, 0xc3, 0x98, 0xfe, 0xff, 0xff, 0xff, 0x1, 0x2a, 0x2, 0x12, 0x0, 0x32, 0x20, 0x1e, 0xba, 0x40, 0x13, 0xa, 0xf2, 0x5e, 0xd1, 0x9, 0x5f, 0x67, 0x86, 0xe5, 0x8d, 0xb9, 0x4d, 0xeb, 0xf4, 0x6a, 0x0, 0x7f, 0xc6, 0x8c, 0x20, 0x32, 0x39, 0x2f, 0xde, 0xdd, 0x32, 0x26, 0x7e, 0x3a, 0x20, 0xc4, 0xda, 0x88, 0xe8, 0x76, 0x6, 0x2a, 0xa1, 0x54, 0x34, 0x0, 0xd5, 0xd, 0xe, 0xaa, 0xd, 0xac, 0x88, 0x9, 0x60, 0x57, 0x94, 0x9c, 0xfb, 0x7b, 0xca, 0x7f, 0x3a, 0x48, 0xc0, 0x4b, 0xf9, 0x6a, 0x20, 0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14, 0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9, 0x24, 0x27, 0xae, 0x41, 0xe4, 0x64, 0x9b, 0x93, 0x4c, 0xa4, 0x95, 0x99, 0x1b, 0x78, 0x52, 0xb8, 0x55, 0x72, 0x14, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x12, 0xd, 0xa, 0xb, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x57, 0x6f, 0x72, 0x6c, 0x64, 0x1a, 0x0, 0x22, 0x41, 0x1a, 0x2, 0x12, 0x0, 0x22, 0x3b, 0x8, 0x2, 0x12, 0x14, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1a, 0xb, 0x8, 0x80, 0x92, 0xb8, 0xc3, 0x98, 0xfe, 0xff, 0xff, 0xff, 0x1, 0x22, 0x14, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xc2, 0x3e, 0x0},
			wantErr: assert.NoError,
		},
		{
			name:    "bcproto.NoBlockResponse",
			args:    args{pb: &bcproto.NoBlockResponse{}},
			want:    []byte{0x12, 0x0},
			wantErr: assert.NoError,
		},
		{
			name:    "bcproto.StatusRequest",
			args:    args{pb: &bcproto.StatusRequest{}},
			want:    []byte{0x22, 0x0},
			wantErr: assert.NoError,
		},
		{
			name:    "bcproto.StatusResponse",
			args:    args{pb: &bcproto.StatusResponse{}},
			want:    []byte{0x2a, 0x0},
			wantErr: assert.NoError,
		},
		{
			name:    "default: unknown message type",
			args:    args{pb: nil},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			{
				// Encode
				got, err := EncodeMsg(tt.args.pb)
				if !tt.wantErr(t, err, fmt.Sprintf("EncodeMsg(%v)", tt.args.pb)) {
					return
				}
				assert.Equalf(t, tt.want, got, "EncodeMsg(%v)", tt.args.pb)
			}
			{
				// Decode
				got, err := DecodeMsg(tt.want)
				if !tt.wantErr(t, err, fmt.Sprintf("DecodeMsg(%v)", tt.want)) {
					return
				}
				if got == nil {
					assert.Equalf(t, tt.args.pb, got, "DecodeMsg(%v)", tt.want)
				} else {
					// NOTE: "tt.args.pb != got" since got.evidence is nil by DecodeMsg, but these can compare each "String"
					assert.Equalf(t, tt.args.pb.String(), got.String(), "DecodeMsg(%v)", tt.want)
				}
			}
			{
				// Validate
				tt.wantErr(t, ValidateMsg(tt.args.pb), fmt.Sprintf("ValidateMsg(%v)", tt.args.pb))
			}
		})
	}
}
