package spvwallet

import (
	"github.com/phoreproject/btcd/chaincfg"
	"github.com/phoreproject/btcd/chaincfg/chainhash"
	"github.com/phoreproject/btcd/wire"
	"time"
)

type Checkpoint struct {
	Height uint32
	Header wire.BlockHeader
}

var mainnetCheckpoints []Checkpoint

func init() {
	// Mainnet
	mainnetPrev, _ := chainhash.NewHashFromStr("000000000000000001389446206ebcd378c32cd00b4920a8a1ba7b540ca7d699")
	mainnetMerk, _ := chainhash.NewHashFromStr("ddc4fede55aeebe6e3bfd3292145b011a4f16ead187ed90d7df0fd4c020b6ab6")
	mainnetCheckpoints = append(mainnetCheckpoints, Checkpoint{
		Height: 473760,
		Header: wire.BlockHeader{
			Version:    536870914,
			PrevBlock:  *mainnetPrev,
			MerkleRoot: *mainnetMerk,
			Timestamp:  time.Unix(1498956437, 0),
			Bits:       402754864,
			Nonce:      134883004,
		},
	})
	if mainnetCheckpoints[0].Header.BlockHash().String() != "000000000000000000802ba879f1b7a638dcea6ff0ceb614d91afc8683ac0502" {
		panic("Invalid checkpoint")
	}

	mainnetPrev1, _ := chainhash.NewHashFromStr("000000000000000000d0ad638ad61e7c4c3113618b8b26b2044347c00c042278")
	mainnetMerk1, _ := chainhash.NewHashFromStr("87b940030e48d97625b923c3ebc0626c2cb1123b78135380306eb6dcfd50703c")
	mainnetCheckpoints = append(mainnetCheckpoints, Checkpoint{
		Height: 483840,
		Header: wire.BlockHeader{
			Version:    536870912,
			PrevBlock:  *mainnetPrev1,
			MerkleRoot: *mainnetMerk1,
			Timestamp:  time.Unix(1504704195, 0),
			Bits:       402731275,
			Nonce:      1775134070,
		},
	})
	if mainnetCheckpoints[1].Header.BlockHash().String() != "0000000000000000008e5d72027ef42ca050a0776b7184c96d0d4b300fa5da9e" {
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
