package claimgen

import (
	"github.com/Layr-Labs/eigenlayer-rewards-proofs/internal/tests"
	"github.com/Layr-Labs/eigenlayer-rewards-proofs/pkg/distribution"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewClaimgen(t *testing.T) {
	distro := distribution.NewDistribution()

	cg := NewClaimgen(distro)

	assert.NotNil(t, cg)
	assert.NotNil(t, cg.distribution)
}

func TestGetProofForEarner(t *testing.T) {
	distro, err := distribution.NewDistributionWithData(tests.TestJsonDistribution)
	assert.Nil(t, err)

	accounts, tokens, err := distro.Merklize()

	earner := common.HexToAddress("0x0D6bA28b9919CfCDb6b233469Cc5Ce30b979e08E")
	token := common.HexToAddress("0x1006dd1B8C3D0eF53489beD27577C75299F71473")
	claim, err := GetProofForEarner(
		distro,
		0,
		accounts,
		tokens,
		earner,
		[]common.Address{token},
	)

	assert.Nil(t, err)

	assert.Equal(t, earner, claim.EarnerLeaf.Earner)
	assert.Equal(t, 1, len(claim.TokenLeaves))
	assert.Equal(t, token, claim.TokenLeaves[0].Token)
}

func TestFormatProofForSolidity(t *testing.T) {
	distro, err := distribution.NewDistributionWithData(tests.TestJsonDistribution)
	assert.Nil(t, err)

	accounts, tokens, err := distro.Merklize()
	assert.Nil(t, err)

	earner := common.HexToAddress("0x0D6bA28b9919CfCDb6b233469Cc5Ce30b979e08E")
	token := common.HexToAddress("0x1006dd1B8C3D0eF53489beD27577C75299F71473")

	claim, err := GetProofForEarner(
		distro,
		0,
		accounts,
		tokens,
		earner,
		[]common.Address{token},
	)
	assert.Nil(t, err)

	claimStrings := FormatProofForSolidity(accounts.Root(), claim)

	assert.NotNil(t, claimStrings)
}
