// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"time"

	"github.com/liraxapp/avalanchego/utils/units"
)

var (
	liraxGenesisConfigJSON = `{
		"networkID": 6,
		"allocations": [
			{
				"ethAddr": "0x0aBa5358d88939c955e18827CBFb31fD778Ec70b",
				"avaxAddr": "X-lirax1kujhum0qak9uzgrquu3kmvy79sr7vlv7gppc5k",
				"initialAmount": 9900000000000000000,
				"unlockSchedule": 
				[
					{
						"amount": 2000000000000000
					}
				]
			}
		],
		"startTime": 1605800550,
		"initialStakeDuration": 31536000,
		"initialStakeDurationOffset": 54000,
		"initialStakedFunds": [
			"X-lirax1kujhum0qak9uzgrquu3kmvy79sr7vlv7gppc5k"
		],
		"initialStakers": [			
			{
				"nodeID": "NodeID-NsQwu1AsEXHTcF7v1GnLf99vkfHjMMEfS",
				"rewardAddress": "X-lirax1kujhum0qak9uzgrquu3kmvy79sr7vlv7gppc5k",
				"delegationFee": 1000000
			}
		],
		"cChainGenesis": "{\"config\":{\"chainId\":43114,\"homesteadBlock\":0,\"daoForkBlock\":0,\"daoForkSupport\":true,\"eip150Block\":0,\"eip150Hash\":\"0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0\",\"eip155Block\":0,\"eip158Block\":0,\"byzantiumBlock\":0,\"constantinopleBlock\":0,\"petersburgBlock\":0,\"istanbulBlock\":0,\"muirGlacierBlock\":0},\"nonce\":\"0x0\",\"timestamp\":\"0x0\",\"extraData\":\"0x00\",\"gasLimit\":\"0x5f5e100\",\"difficulty\":\"0x0\",\"mixHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\",\"coinbase\":\"0x0000000000000000000000000000000000000000\",\"alloc\":{\"0100000000000000000000000000000000000000\":{\"code\":\"0x7300000000000000000000000000000000000000003014608060405260043610603d5760003560e01c80631e010439146042578063b6510bb314606e575b600080fd5b605c60048036036020811015605657600080fd5b503560b1565b60408051918252519081900360200190f35b818015607957600080fd5b5060af60048036036080811015608e57600080fd5b506001600160a01b03813516906020810135906040810135906060013560b6565b005b30cd90565b836001600160a01b031681836108fc8690811502906040516000604051808303818888878c8acf9550505050505015801560f4573d6000803e3d6000fd5b505050505056fea26469706673582212201eebce970fe3f5cb96bf8ac6ba5f5c133fc2908ae3dcd51082cfee8f583429d064736f6c634300060a0033\",\"balance\":\"0x0\"}},\"number\":\"0x0\",\"gasUsed\":\"0x0\",\"parentHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\"}",
		"message": "sorry 4 fork"
	}`

	LiraxParams = Params{
		TxFee:              10 * units.MilliAvax,
		CreationTxFee:      10 * units.MilliAvax,
		UptimeRequirement:  .1, // 10%
		MinValidatorStake:  1 * units.MegaAvax,
		MaxValidatorStake:  2 * units.MegaAvax,
		MinDelegatorStake:  100000 * units.Avax,
		MinDelegationFee:   990000, // 99%
		MinStakeDuration:   1 * time.Hour,
		MaxStakeDuration:   365 * 24 * time.Hour,
		StakeMintingPeriod: 365 * 24 * time.Hour,
		ApricotPhase0Time:  time.Date(2020, 12, 5, 5, 00, 0, 0, time.UTC),
	}
)
