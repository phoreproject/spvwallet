package spvwallet

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"time"
)

type Checkpoint struct {
	Height uint32
	Header wire.BlockHeader
}

var mainnetCheckpoints []Checkpoint

func init() {
	// Mainnet
	mainnetPrev, _ := chainhash.NewHashFromStr("66aa54b9a85e32c3d466fe38cbd40553120a32e15f99b107e31c20a6f56882e7")
	mainnetMerk, _ := chainhash.NewHashFromStr("5f19fc2b20e25c9ebdc373fd7b4fe49d164b0d139f1121e603f956a1ae8d0b28")
	mainnetCheckpoints = append(mainnetCheckpoints, Checkpoint{
		Height: 400000,
		Header: wire.BlockHeader{
			Version:    4,
			PrevBlock:  *mainnetPrev,
			MerkleRoot: *mainnetMerk,
			Timestamp:  time.Unix(1529796056, 0),
			Bits:       453089485,
			Nonce:      0,
		},
	})
	if mainnetCheckpoints[0].Header.BlockHash().String() != "9491894ab30da4bae4e8a2ca9547f2f6a01ac29fc4006342cc690fd61dbe55b3" {
		panic("Invalid checkpoint")
	}
}


func GetCheckpoint(walletCreationDate time.Time, params *chaincfg.Params) Checkpoint {
	for i := len(mainnetCheckpoints) - 1; i >= 0; i-- {
		if walletCreationDate.After(mainnetCheckpoints[i].Header.Timestamp) {
			return mainnetCheckpoints[i]
		}
	}
	return mainnetCheckpoints[0]
}
