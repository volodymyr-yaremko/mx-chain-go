package builtInFunctions

import (
	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/builtInFunctions"
	"github.com/mitchellh/mapstructure"
)

var log = logger.GetOrCreate("process/smartContract/builtInFunctions")

// ArgsCreateBuiltInFunctionContainer -
type ArgsCreateBuiltInFunctionContainer struct {
	GasSchedule          core.GasScheduleNotifier
	MapDNSAddresses      map[string]struct{}
	EnableUserNameChange bool
	Marshalizer          marshal.Marshalizer
	Accounts             state.AccountsAdapter
	ShardCoordinator     sharding.Coordinator
}

type builtInFuncFactory struct {
	mapDNSAddresses      map[string]struct{}
	enableUserNameChange bool
	marshalizer          marshal.Marshalizer
	accounts             state.AccountsAdapter
	builtInFunctions     vmcommon.BuiltInFunctionContainer
	gasConfig            *vmcommon.GasCost
	shardCoordinator     sharding.Coordinator
}

// NewBuiltInFunctionsFactory creates a factory which will instantiate the built in functions contracts
func NewBuiltInFunctionsFactory(args ArgsCreateBuiltInFunctionContainer) (*builtInFuncFactory, error) {
	if check.IfNil(args.GasSchedule) {
		return nil, process.ErrNilGasSchedule
	}
	if check.IfNil(args.Marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(args.Accounts) {
		return nil, process.ErrNilAccountsAdapter
	}
	if args.MapDNSAddresses == nil {
		return nil, process.ErrNilDnsAddresses
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}

	b := &builtInFuncFactory{
		mapDNSAddresses:      args.MapDNSAddresses,
		enableUserNameChange: args.EnableUserNameChange,
		marshalizer:          args.Marshalizer,
		accounts:             args.Accounts,
		shardCoordinator:     args.ShardCoordinator,
	}

	var err error
	b.gasConfig, err = createGasConfig(args.GasSchedule.LatestGasSchedule())
	if err != nil {
		return nil, err
	}
	b.builtInFunctions = NewBuiltInFunctionContainer()

	args.GasSchedule.RegisterNotifyHandler(b)

	return b, nil
}

// GasScheduleChange is called when gas schedule is changed, thus all contracts must be updated
func (b *builtInFuncFactory) GasScheduleChange(gasSchedule map[string]map[string]uint64) {
	newGasConfig, err := createGasConfig(gasSchedule)
	if err != nil {
		log.Error("builtInFuncFactory GasScheduleChange createNewGasConfig", "err", err)
		return
	}

	b.gasConfig = newGasConfig
	for key := range b.builtInFunctions.Keys() {
		builtInFunc, errGet := b.builtInFunctions.Get(key)
		if errGet != nil {
			log.Error("builtInFuncFactory GasScheduleChange get", "err", errGet)
			return
		}

		builtInFunc.SetNewGasConfig(b.gasConfig)
	}
}

// CreateBuiltInFunctionContainer will create the list of built-in functions
func (b *builtInFuncFactory) CreateBuiltInFunctionContainer() (vmcommon.BuiltInFunctionContainer, error) {

	b.builtInFunctions = NewBuiltInFunctionContainer()
	var newFunc vmcommon.BuiltinFunction
	newFunc = builtInFunctions.NewClaimDeveloperRewardsFunc(b.gasConfig.BuiltInCost.ClaimDeveloperRewards)
	err := b.builtInFunctions.Add(core.BuiltInFunctionClaimDeveloperRewards, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc = builtInFunctions.NewChangeOwnerAddressFunc(b.gasConfig.BuiltInCost.ChangeOwnerAddress)
	err = b.builtInFunctions.Add(core.BuiltInFunctionChangeOwnerAddress, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewSaveUserNameFunc(b.gasConfig.BuiltInCost.SaveUserName, b.mapDNSAddresses, b.enableUserNameChange)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetUserName, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewSaveKeyValueStorageFunc(b.gasConfig.BaseOperationCost, b.gasConfig.BuiltInCost.SaveKeyValue)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSaveKeyValue, newFunc)
	if err != nil {
		return nil, err
	}

	pauseFunc, err := builtInFunctions.NewESDTPauseFunc(b.accounts, true)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTPause, pauseFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTTransferFunc(b.gasConfig.BuiltInCost.ESDTTransfer, b.marshalizer, pauseFunc, b.shardCoordinator)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTBurnFunc(b.gasConfig.BuiltInCost.ESDTBurn, b.marshalizer, pauseFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTBurn, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTFreezeWipeFunc(b.marshalizer, true, false)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTFreeze, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTFreezeWipeFunc(b.marshalizer, false, false)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTUnFreeze, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTFreezeWipeFunc(b.marshalizer, false, true)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTWipe, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTPauseFunc(b.accounts, false)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTUnPause, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTRolesFunc(b.marshalizer, false)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionUnSetESDTRole, newFunc)
	if err != nil {
		return nil, err
	}

	setRoleFunc, err := builtInFunctions.NewESDTRolesFunc(b.marshalizer, true)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetESDTRole, setRoleFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTLocalBurnFunc(b.gasConfig.BuiltInCost.ESDTLocalBurn, b.marshalizer, pauseFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTLocalBurn, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTLocalMintFunc(b.gasConfig.BuiltInCost.ESDTLocalMint, b.marshalizer, pauseFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTLocalMint, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTNFTAddQuantityFunc(b.gasConfig.BuiltInCost.ESDTNFTAddQuantity, b.marshalizer, pauseFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTAddQuantity, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTNFTBurnFunc(b.gasConfig.BuiltInCost.ESDTNFTBurn, b.marshalizer, pauseFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTBurn, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTNFTCreateFunc(b.gasConfig.BuiltInCost.ESDTNFTCreate, b.gasConfig.BaseOperationCost, b.marshalizer, pauseFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTCreate, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTNFTTransferFunc(b.gasConfig.BuiltInCost.ESDTNFTTransfer, b.marshalizer, pauseFunc, b.accounts, b.shardCoordinator, b.gasConfig.BaseOperationCost)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = builtInFunctions.NewESDTNFTCreateRoleTransfer(b.marshalizer, b.accounts, b.shardCoordinator)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTCreateRoleTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	return b.builtInFunctions, nil
}

func createGasConfig(gasMap map[string]map[string]uint64) (*vmcommon.GasCost, error) {
	baseOps := &vmcommon.BaseOperationCost{}
	err := mapstructure.Decode(gasMap[core.BaseOperationCost], baseOps)
	if err != nil {
		return nil, err
	}

	err = check.ForZeroUintFields(*baseOps)
	if err != nil {
		return nil, err
	}

	builtInOps := &vmcommon.BuiltInCost{}
	err = mapstructure.Decode(gasMap[core.BuiltInCost], builtInOps)
	if err != nil {
		return nil, err
	}

	err = check.ForZeroUintFields(*builtInOps)
	if err != nil {
		return nil, err
	}

	gasCost := vmcommon.GasCost{
		BaseOperationCost: *baseOps,
		BuiltInCost:       *builtInOps,
	}

	return &gasCost, nil
}

// SetPayableHandler sets the payable interface to the needed functions
func SetPayableHandler(container vmcommon.BuiltInFunctionContainer, payableHandler process.PayableHandler) error {
	builtInFunc, err := container.Get(core.BuiltInFunctionESDTTransfer)
	if err != nil {
		log.Warn("SetIsPayable", "error", err.Error())
		return err
	}

	esdtTransferFunc, ok := builtInFunc.(vmcommon.AcceptPayableHandler)
	if !ok {
		log.Warn("SetIsPayable", "error", process.ErrWrongTypeAssertion)
		return process.ErrWrongTypeAssertion
	}

	err = esdtTransferFunc.SetPayableHandler(payableHandler)
	if err != nil {
		return err
	}

	builtInFunc, err = container.Get(core.BuiltInFunctionESDTNFTTransfer)
	if err != nil {
		log.Warn("SetIsPayable", "error", err.Error())
		return err
	}

	esdtNFTTransferFunc, ok := builtInFunc.(AcceptPayableHandler)
	if !ok {
		log.Warn("SetIsPayable", "error", process.ErrWrongTypeAssertion)
		return process.ErrWrongTypeAssertion
	}

	err = esdtNFTTransferFunc.SetPayableHandler(payableHandler)
	if err != nil {
		return err
	}

	return nil
}

// IsInterfaceNil returns true if underlying object is nil
func (b *builtInFuncFactory) IsInterfaceNil() bool {
	return b == nil
}
