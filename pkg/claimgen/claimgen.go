package claimgen

import (
	"errors"
	"fmt"
	rewardsCoordinator "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/IRewardsCoordinator"
	"github.com/Layr-Labs/eigenlayer-rewards-proofs/pkg/distribution"
	"github.com/Layr-Labs/eigenlayer-rewards-proofs/pkg/utils"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/wealdtech/go-merkletree/v2"
)

var ErrEarnerIndexNotFound = errors.New("earner index not found")
var ErrTokenIndexNotFound = errors.New("token not found")
var ErrAmountNotFound = errors.New("amount not found")

// Helper function for getting the proof for the specified earner and tokens
func GetProofForEarner(
	distribution *distribution.Distribution,
	rootIndex uint32,
	accountTree *merkletree.MerkleTree,
	tokenTrees map[gethcommon.Address]*merkletree.MerkleTree,
	earner gethcommon.Address,
	tokens []gethcommon.Address,
) (*rewardsCoordinator.IRewardsCoordinatorRewardsMerkleClaim, error) {
	earnerIndex, found := distribution.GetAccountIndex(earner)
	if !found {
		return nil, fmt.Errorf("%w for earner %s", ErrEarnerIndexNotFound, earner.Hex())
	}

	// get the token proofs
	tokenIndices := make([]uint32, 0)
	tokenProofsBytes := make([][]byte, 0)
	tokenLeaves := make([]rewardsCoordinator.IRewardsCoordinatorTokenTreeMerkleLeaf, 0)
	for _, token := range tokens {
		tokenIndex, found := distribution.GetTokenIndex(earner, token)
		if !found {
			return nil, fmt.Errorf("%w for token %s and earner %s", ErrTokenIndexNotFound, token.Hex(), earner.Hex())
		}
		tokenIndices = append(tokenIndices, uint32(tokenIndex))

		tokenProof, err := tokenTrees[earner].GenerateProofWithIndex(tokenIndex, 0)
		if err != nil {
			return nil, err
		}
		tokenProofsBytes = append(tokenProofsBytes, flattenHashes(tokenProof.Hashes))

		amount, found := distribution.Get(earner, token)
		if !found {
			// this should never happen due to the token index check above
			return nil, fmt.Errorf("%w for token %s and earner %s", ErrAmountNotFound, token.Hex(), earner.Hex())
		}
		tokenLeaves = append(tokenLeaves, rewardsCoordinator.IRewardsCoordinatorTokenTreeMerkleLeaf{
			Token:              token,
			CumulativeEarnings: amount,
		})
	}

	var earnerRoot [32]byte
	copy(earnerRoot[:], tokenTrees[earner].Root())

	// get the account proof
	earnerTreeProof, err := accountTree.GenerateProofWithIndex(earnerIndex, 0)
	if err != nil {
		return nil, err
	}

	earnerTreeProofBytes := flattenHashes(earnerTreeProof.Hashes)

	return &rewardsCoordinator.IRewardsCoordinatorRewardsMerkleClaim{
		RootIndex:       rootIndex,
		EarnerIndex:     uint32(earnerIndex),
		EarnerTreeProof: earnerTreeProofBytes,
		EarnerLeaf: rewardsCoordinator.IRewardsCoordinatorEarnerTreeMerkleLeaf{
			Earner:          earner,
			EarnerTokenRoot: earnerRoot,
		},
		TokenIndices:    tokenIndices,
		TokenTreeProofs: tokenProofsBytes,
		TokenLeaves:     tokenLeaves,
	}, nil
}

func flattenHashes(hashes [][]byte) []byte {
	result := make([]byte, 0)
	for i := 0; i < len(hashes); i++ {
		result = append(result, hashes[i]...)
	}
	return result
}

type IRewardsCoordinatorEarnerTreeMerkleLeafStrings struct {
	Earner          gethcommon.Address
	EarnerTokenRoot string
}

type IRewardsCoordinatorPaymentMerkleClaimStrings struct {
	Root               string
	RootIndex          uint32
	EarnerIndex        uint32
	EarnerTreeProof    string
	EarnerLeaf         IRewardsCoordinatorEarnerTreeMerkleLeafStrings
	LeafIndices        []uint32
	TokenTreeProofs    []string
	TokenLeaves        []rewardsCoordinator.IRewardsCoordinatorTokenTreeMerkleLeaf
	TokenTreeProofsNum uint32
	TokenLeavesNum     uint32
}

func FormatProofForSolidity(accountTreeRoot []byte, proof *rewardsCoordinator.IRewardsCoordinatorRewardsMerkleClaim) *IRewardsCoordinatorPaymentMerkleClaimStrings {
	return &IRewardsCoordinatorPaymentMerkleClaimStrings{
		Root:            utils.ConvertBytesToString(accountTreeRoot),
		RootIndex:       proof.RootIndex,
		EarnerIndex:     proof.EarnerIndex,
		EarnerTreeProof: utils.ConvertBytesToString(proof.EarnerTreeProof),
		EarnerLeaf: IRewardsCoordinatorEarnerTreeMerkleLeafStrings{
			Earner:          proof.EarnerLeaf.Earner,
			EarnerTokenRoot: utils.ConvertBytes32ToString(proof.EarnerLeaf.EarnerTokenRoot),
		},
		LeafIndices:        proof.TokenIndices,
		TokenTreeProofs:    utils.ConvertBytesToStrings(proof.TokenTreeProofs),
		TokenLeaves:        proof.TokenLeaves,
		TokenTreeProofsNum: uint32(len(proof.TokenTreeProofs)),
		TokenLeavesNum:     uint32(len(proof.TokenLeaves)),
	}
}

type Claimgen struct {
	distribution *distribution.Distribution
}

func NewClaimgen(distro *distribution.Distribution) *Claimgen {
	return &Claimgen{
		distribution: distro,
	}
}

func (c *Claimgen) GenerateClaimProofForEarner(
	earner gethcommon.Address,
	tokens []gethcommon.Address,
	rootIndex uint32,
) (
	*merkletree.MerkleTree,
	*rewardsCoordinator.IRewardsCoordinatorRewardsMerkleClaim,
	error,
) {
	accountTree, tokenTrees, err := c.distribution.Merklize()
	if err != nil {
		return nil, nil, err
	}

	merkleClaim, err := GetProofForEarner(
		c.distribution,
		rootIndex,
		accountTree,
		tokenTrees,
		earner,
		tokens,
	)

	if err != nil {
		return nil, nil, err
	}

	return accountTree, merkleClaim, err
}
