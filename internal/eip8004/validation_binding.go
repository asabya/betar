// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package eip8004

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// ValidationRegistryMetaData contains all meta data concerning the ValidationRegistry contract.
var ValidationRegistryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"}],\"name\":\"AddressEmptyCode\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"ERC1967InvalidImplementation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ERC1967NonPayable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"FailedCall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidInitialization\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotInitializing\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnableInvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"OwnableUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UUPSUnauthorizedCallContext\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"slot\",\"type\":\"bytes32\"}],\"name\":\"UUPSUnsupportedProxiableUUID\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"version\",\"type\":\"uint64\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"Upgraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validatorAddress\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"agentId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"requestURI\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"requestHash\",\"type\":\"bytes32\"}],\"name\":\"ValidationRequest\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"validatorAddress\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"agentId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"requestHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"response\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"responseURI\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"responseHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"tag\",\"type\":\"string\"}],\"name\":\"ValidationResponse\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"UPGRADE_INTERFACE_VERSION\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"agentId\",\"type\":\"uint256\"}],\"name\":\"getAgentValidations\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getIdentityRegistry\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"agentId\",\"type\":\"uint256\"},{\"internalType\":\"address[]\",\"name\":\"validatorAddresses\",\"type\":\"address[]\"},{\"internalType\":\"string\",\"name\":\"tag\",\"type\":\"string\"}],\"name\":\"getSummary\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"count\",\"type\":\"uint64\"},{\"internalType\":\"uint8\",\"name\":\"avgResponse\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"requestHash\",\"type\":\"bytes32\"}],\"name\":\"getValidationStatus\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"validatorAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"agentId\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"response\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"responseHash\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"tag\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"lastUpdate\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validatorAddress\",\"type\":\"address\"}],\"name\":\"getValidatorRequests\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getVersion\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"identityRegistry_\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proxiableUUID\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newImplementation\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"upgradeToAndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"validatorAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"agentId\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"requestURI\",\"type\":\"string\"},{\"internalType\":\"bytes32\",\"name\":\"requestHash\",\"type\":\"bytes32\"}],\"name\":\"validationRequest\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"requestHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"response\",\"type\":\"uint8\"},{\"internalType\":\"string\",\"name\":\"responseURI\",\"type\":\"string\"},{\"internalType\":\"bytes32\",\"name\":\"responseHash\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"tag\",\"type\":\"string\"}],\"name\":\"validationResponse\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ValidationRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use ValidationRegistryMetaData.ABI instead.
var ValidationRegistryABI = ValidationRegistryMetaData.ABI

// ValidationRegistry is an auto generated Go binding around an Ethereum contract.
type ValidationRegistry struct {
	ValidationRegistryCaller     // Read-only binding to the contract
	ValidationRegistryTransactor // Write-only binding to the contract
	ValidationRegistryFilterer   // Log filterer for contract events
}

// ValidationRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type ValidationRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidationRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ValidationRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidationRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ValidationRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidationRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ValidationRegistrySession struct {
	Contract     *ValidationRegistry // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ValidationRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ValidationRegistryCallerSession struct {
	Contract *ValidationRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// ValidationRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ValidationRegistryTransactorSession struct {
	Contract     *ValidationRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// ValidationRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type ValidationRegistryRaw struct {
	Contract *ValidationRegistry // Generic contract binding to access the raw methods on
}

// ValidationRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ValidationRegistryCallerRaw struct {
	Contract *ValidationRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// ValidationRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ValidationRegistryTransactorRaw struct {
	Contract *ValidationRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewValidationRegistry creates a new instance of ValidationRegistry, bound to a specific deployed contract.
func NewValidationRegistry(address common.Address, backend bind.ContractBackend) (*ValidationRegistry, error) {
	contract, err := bindValidationRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ValidationRegistry{ValidationRegistryCaller: ValidationRegistryCaller{contract: contract}, ValidationRegistryTransactor: ValidationRegistryTransactor{contract: contract}, ValidationRegistryFilterer: ValidationRegistryFilterer{contract: contract}}, nil
}

// NewValidationRegistryCaller creates a new read-only instance of ValidationRegistry, bound to a specific deployed contract.
func NewValidationRegistryCaller(address common.Address, caller bind.ContractCaller) (*ValidationRegistryCaller, error) {
	contract, err := bindValidationRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ValidationRegistryCaller{contract: contract}, nil
}

// NewValidationRegistryTransactor creates a new write-only instance of ValidationRegistry, bound to a specific deployed contract.
func NewValidationRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*ValidationRegistryTransactor, error) {
	contract, err := bindValidationRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ValidationRegistryTransactor{contract: contract}, nil
}

// NewValidationRegistryFilterer creates a new log filterer instance of ValidationRegistry, bound to a specific deployed contract.
func NewValidationRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*ValidationRegistryFilterer, error) {
	contract, err := bindValidationRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ValidationRegistryFilterer{contract: contract}, nil
}

// bindValidationRegistry binds a generic wrapper to an already deployed contract.
func bindValidationRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ValidationRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValidationRegistry *ValidationRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ValidationRegistry.Contract.ValidationRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValidationRegistry *ValidationRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.ValidationRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValidationRegistry *ValidationRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.ValidationRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValidationRegistry *ValidationRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ValidationRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValidationRegistry *ValidationRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValidationRegistry *ValidationRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.contract.Transact(opts, method, params...)
}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_ValidationRegistry *ValidationRegistryCaller) UPGRADEINTERFACEVERSION(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ValidationRegistry.contract.Call(opts, &out, "UPGRADE_INTERFACE_VERSION")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_ValidationRegistry *ValidationRegistrySession) UPGRADEINTERFACEVERSION() (string, error) {
	return _ValidationRegistry.Contract.UPGRADEINTERFACEVERSION(&_ValidationRegistry.CallOpts)
}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_ValidationRegistry *ValidationRegistryCallerSession) UPGRADEINTERFACEVERSION() (string, error) {
	return _ValidationRegistry.Contract.UPGRADEINTERFACEVERSION(&_ValidationRegistry.CallOpts)
}

// GetAgentValidations is a free data retrieval call binding the contract method 0x8d5d0c2d.
//
// Solidity: function getAgentValidations(uint256 agentId) view returns(bytes32[])
func (_ValidationRegistry *ValidationRegistryCaller) GetAgentValidations(opts *bind.CallOpts, agentId *big.Int) ([][32]byte, error) {
	var out []interface{}
	err := _ValidationRegistry.contract.Call(opts, &out, "getAgentValidations", agentId)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetAgentValidations is a free data retrieval call binding the contract method 0x8d5d0c2d.
//
// Solidity: function getAgentValidations(uint256 agentId) view returns(bytes32[])
func (_ValidationRegistry *ValidationRegistrySession) GetAgentValidations(agentId *big.Int) ([][32]byte, error) {
	return _ValidationRegistry.Contract.GetAgentValidations(&_ValidationRegistry.CallOpts, agentId)
}

// GetAgentValidations is a free data retrieval call binding the contract method 0x8d5d0c2d.
//
// Solidity: function getAgentValidations(uint256 agentId) view returns(bytes32[])
func (_ValidationRegistry *ValidationRegistryCallerSession) GetAgentValidations(agentId *big.Int) ([][32]byte, error) {
	return _ValidationRegistry.Contract.GetAgentValidations(&_ValidationRegistry.CallOpts, agentId)
}

// GetIdentityRegistry is a free data retrieval call binding the contract method 0xbc4d861b.
//
// Solidity: function getIdentityRegistry() view returns(address)
func (_ValidationRegistry *ValidationRegistryCaller) GetIdentityRegistry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ValidationRegistry.contract.Call(opts, &out, "getIdentityRegistry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetIdentityRegistry is a free data retrieval call binding the contract method 0xbc4d861b.
//
// Solidity: function getIdentityRegistry() view returns(address)
func (_ValidationRegistry *ValidationRegistrySession) GetIdentityRegistry() (common.Address, error) {
	return _ValidationRegistry.Contract.GetIdentityRegistry(&_ValidationRegistry.CallOpts)
}

// GetIdentityRegistry is a free data retrieval call binding the contract method 0xbc4d861b.
//
// Solidity: function getIdentityRegistry() view returns(address)
func (_ValidationRegistry *ValidationRegistryCallerSession) GetIdentityRegistry() (common.Address, error) {
	return _ValidationRegistry.Contract.GetIdentityRegistry(&_ValidationRegistry.CallOpts)
}

// GetSummary is a free data retrieval call binding the contract method 0x1b7cabd6.
//
// Solidity: function getSummary(uint256 agentId, address[] validatorAddresses, string tag) view returns(uint64 count, uint8 avgResponse)
func (_ValidationRegistry *ValidationRegistryCaller) GetSummary(opts *bind.CallOpts, agentId *big.Int, validatorAddresses []common.Address, tag string) (struct {
	Count       uint64
	AvgResponse uint8
}, error) {
	var out []interface{}
	err := _ValidationRegistry.contract.Call(opts, &out, "getSummary", agentId, validatorAddresses, tag)

	outstruct := new(struct {
		Count       uint64
		AvgResponse uint8
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Count = *abi.ConvertType(out[0], new(uint64)).(*uint64)
	outstruct.AvgResponse = *abi.ConvertType(out[1], new(uint8)).(*uint8)

	return *outstruct, err

}

// GetSummary is a free data retrieval call binding the contract method 0x1b7cabd6.
//
// Solidity: function getSummary(uint256 agentId, address[] validatorAddresses, string tag) view returns(uint64 count, uint8 avgResponse)
func (_ValidationRegistry *ValidationRegistrySession) GetSummary(agentId *big.Int, validatorAddresses []common.Address, tag string) (struct {
	Count       uint64
	AvgResponse uint8
}, error) {
	return _ValidationRegistry.Contract.GetSummary(&_ValidationRegistry.CallOpts, agentId, validatorAddresses, tag)
}

// GetSummary is a free data retrieval call binding the contract method 0x1b7cabd6.
//
// Solidity: function getSummary(uint256 agentId, address[] validatorAddresses, string tag) view returns(uint64 count, uint8 avgResponse)
func (_ValidationRegistry *ValidationRegistryCallerSession) GetSummary(agentId *big.Int, validatorAddresses []common.Address, tag string) (struct {
	Count       uint64
	AvgResponse uint8
}, error) {
	return _ValidationRegistry.Contract.GetSummary(&_ValidationRegistry.CallOpts, agentId, validatorAddresses, tag)
}

// GetValidationStatus is a free data retrieval call binding the contract method 0xff2febfc.
//
// Solidity: function getValidationStatus(bytes32 requestHash) view returns(address validatorAddress, uint256 agentId, uint8 response, bytes32 responseHash, string tag, uint256 lastUpdate)
func (_ValidationRegistry *ValidationRegistryCaller) GetValidationStatus(opts *bind.CallOpts, requestHash [32]byte) (struct {
	ValidatorAddress common.Address
	AgentId          *big.Int
	Response         uint8
	ResponseHash     [32]byte
	Tag              string
	LastUpdate       *big.Int
}, error) {
	var out []interface{}
	err := _ValidationRegistry.contract.Call(opts, &out, "getValidationStatus", requestHash)

	outstruct := new(struct {
		ValidatorAddress common.Address
		AgentId          *big.Int
		Response         uint8
		ResponseHash     [32]byte
		Tag              string
		LastUpdate       *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ValidatorAddress = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.AgentId = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Response = *abi.ConvertType(out[2], new(uint8)).(*uint8)
	outstruct.ResponseHash = *abi.ConvertType(out[3], new([32]byte)).(*[32]byte)
	outstruct.Tag = *abi.ConvertType(out[4], new(string)).(*string)
	outstruct.LastUpdate = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetValidationStatus is a free data retrieval call binding the contract method 0xff2febfc.
//
// Solidity: function getValidationStatus(bytes32 requestHash) view returns(address validatorAddress, uint256 agentId, uint8 response, bytes32 responseHash, string tag, uint256 lastUpdate)
func (_ValidationRegistry *ValidationRegistrySession) GetValidationStatus(requestHash [32]byte) (struct {
	ValidatorAddress common.Address
	AgentId          *big.Int
	Response         uint8
	ResponseHash     [32]byte
	Tag              string
	LastUpdate       *big.Int
}, error) {
	return _ValidationRegistry.Contract.GetValidationStatus(&_ValidationRegistry.CallOpts, requestHash)
}

// GetValidationStatus is a free data retrieval call binding the contract method 0xff2febfc.
//
// Solidity: function getValidationStatus(bytes32 requestHash) view returns(address validatorAddress, uint256 agentId, uint8 response, bytes32 responseHash, string tag, uint256 lastUpdate)
func (_ValidationRegistry *ValidationRegistryCallerSession) GetValidationStatus(requestHash [32]byte) (struct {
	ValidatorAddress common.Address
	AgentId          *big.Int
	Response         uint8
	ResponseHash     [32]byte
	Tag              string
	LastUpdate       *big.Int
}, error) {
	return _ValidationRegistry.Contract.GetValidationStatus(&_ValidationRegistry.CallOpts, requestHash)
}

// GetValidatorRequests is a free data retrieval call binding the contract method 0x4bf3158c.
//
// Solidity: function getValidatorRequests(address validatorAddress) view returns(bytes32[])
func (_ValidationRegistry *ValidationRegistryCaller) GetValidatorRequests(opts *bind.CallOpts, validatorAddress common.Address) ([][32]byte, error) {
	var out []interface{}
	err := _ValidationRegistry.contract.Call(opts, &out, "getValidatorRequests", validatorAddress)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetValidatorRequests is a free data retrieval call binding the contract method 0x4bf3158c.
//
// Solidity: function getValidatorRequests(address validatorAddress) view returns(bytes32[])
func (_ValidationRegistry *ValidationRegistrySession) GetValidatorRequests(validatorAddress common.Address) ([][32]byte, error) {
	return _ValidationRegistry.Contract.GetValidatorRequests(&_ValidationRegistry.CallOpts, validatorAddress)
}

// GetValidatorRequests is a free data retrieval call binding the contract method 0x4bf3158c.
//
// Solidity: function getValidatorRequests(address validatorAddress) view returns(bytes32[])
func (_ValidationRegistry *ValidationRegistryCallerSession) GetValidatorRequests(validatorAddress common.Address) ([][32]byte, error) {
	return _ValidationRegistry.Contract.GetValidatorRequests(&_ValidationRegistry.CallOpts, validatorAddress)
}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(string)
func (_ValidationRegistry *ValidationRegistryCaller) GetVersion(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ValidationRegistry.contract.Call(opts, &out, "getVersion")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(string)
func (_ValidationRegistry *ValidationRegistrySession) GetVersion() (string, error) {
	return _ValidationRegistry.Contract.GetVersion(&_ValidationRegistry.CallOpts)
}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(string)
func (_ValidationRegistry *ValidationRegistryCallerSession) GetVersion() (string, error) {
	return _ValidationRegistry.Contract.GetVersion(&_ValidationRegistry.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ValidationRegistry *ValidationRegistryCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ValidationRegistry.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ValidationRegistry *ValidationRegistrySession) Owner() (common.Address, error) {
	return _ValidationRegistry.Contract.Owner(&_ValidationRegistry.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ValidationRegistry *ValidationRegistryCallerSession) Owner() (common.Address, error) {
	return _ValidationRegistry.Contract.Owner(&_ValidationRegistry.CallOpts)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_ValidationRegistry *ValidationRegistryCaller) ProxiableUUID(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ValidationRegistry.contract.Call(opts, &out, "proxiableUUID")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_ValidationRegistry *ValidationRegistrySession) ProxiableUUID() ([32]byte, error) {
	return _ValidationRegistry.Contract.ProxiableUUID(&_ValidationRegistry.CallOpts)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_ValidationRegistry *ValidationRegistryCallerSession) ProxiableUUID() ([32]byte, error) {
	return _ValidationRegistry.Contract.ProxiableUUID(&_ValidationRegistry.CallOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address identityRegistry_) returns()
func (_ValidationRegistry *ValidationRegistryTransactor) Initialize(opts *bind.TransactOpts, identityRegistry_ common.Address) (*types.Transaction, error) {
	return _ValidationRegistry.contract.Transact(opts, "initialize", identityRegistry_)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address identityRegistry_) returns()
func (_ValidationRegistry *ValidationRegistrySession) Initialize(identityRegistry_ common.Address) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.Initialize(&_ValidationRegistry.TransactOpts, identityRegistry_)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address identityRegistry_) returns()
func (_ValidationRegistry *ValidationRegistryTransactorSession) Initialize(identityRegistry_ common.Address) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.Initialize(&_ValidationRegistry.TransactOpts, identityRegistry_)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ValidationRegistry *ValidationRegistryTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValidationRegistry.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ValidationRegistry *ValidationRegistrySession) RenounceOwnership() (*types.Transaction, error) {
	return _ValidationRegistry.Contract.RenounceOwnership(&_ValidationRegistry.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ValidationRegistry *ValidationRegistryTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _ValidationRegistry.Contract.RenounceOwnership(&_ValidationRegistry.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ValidationRegistry *ValidationRegistryTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _ValidationRegistry.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ValidationRegistry *ValidationRegistrySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.TransferOwnership(&_ValidationRegistry.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ValidationRegistry *ValidationRegistryTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.TransferOwnership(&_ValidationRegistry.TransactOpts, newOwner)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_ValidationRegistry *ValidationRegistryTransactor) UpgradeToAndCall(opts *bind.TransactOpts, newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _ValidationRegistry.contract.Transact(opts, "upgradeToAndCall", newImplementation, data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_ValidationRegistry *ValidationRegistrySession) UpgradeToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.UpgradeToAndCall(&_ValidationRegistry.TransactOpts, newImplementation, data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_ValidationRegistry *ValidationRegistryTransactorSession) UpgradeToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.UpgradeToAndCall(&_ValidationRegistry.TransactOpts, newImplementation, data)
}

// ValidationRequest is a paid mutator transaction binding the contract method 0xaaf400c4.
//
// Solidity: function validationRequest(address validatorAddress, uint256 agentId, string requestURI, bytes32 requestHash) returns()
func (_ValidationRegistry *ValidationRegistryTransactor) ValidationRequest(opts *bind.TransactOpts, validatorAddress common.Address, agentId *big.Int, requestURI string, requestHash [32]byte) (*types.Transaction, error) {
	return _ValidationRegistry.contract.Transact(opts, "validationRequest", validatorAddress, agentId, requestURI, requestHash)
}

// ValidationRequest is a paid mutator transaction binding the contract method 0xaaf400c4.
//
// Solidity: function validationRequest(address validatorAddress, uint256 agentId, string requestURI, bytes32 requestHash) returns()
func (_ValidationRegistry *ValidationRegistrySession) ValidationRequest(validatorAddress common.Address, agentId *big.Int, requestURI string, requestHash [32]byte) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.ValidationRequest(&_ValidationRegistry.TransactOpts, validatorAddress, agentId, requestURI, requestHash)
}

// ValidationRequest is a paid mutator transaction binding the contract method 0xaaf400c4.
//
// Solidity: function validationRequest(address validatorAddress, uint256 agentId, string requestURI, bytes32 requestHash) returns()
func (_ValidationRegistry *ValidationRegistryTransactorSession) ValidationRequest(validatorAddress common.Address, agentId *big.Int, requestURI string, requestHash [32]byte) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.ValidationRequest(&_ValidationRegistry.TransactOpts, validatorAddress, agentId, requestURI, requestHash)
}

// ValidationResponse is a paid mutator transaction binding the contract method 0x3d659a96.
//
// Solidity: function validationResponse(bytes32 requestHash, uint8 response, string responseURI, bytes32 responseHash, string tag) returns()
func (_ValidationRegistry *ValidationRegistryTransactor) ValidationResponse(opts *bind.TransactOpts, requestHash [32]byte, response uint8, responseURI string, responseHash [32]byte, tag string) (*types.Transaction, error) {
	return _ValidationRegistry.contract.Transact(opts, "validationResponse", requestHash, response, responseURI, responseHash, tag)
}

// ValidationResponse is a paid mutator transaction binding the contract method 0x3d659a96.
//
// Solidity: function validationResponse(bytes32 requestHash, uint8 response, string responseURI, bytes32 responseHash, string tag) returns()
func (_ValidationRegistry *ValidationRegistrySession) ValidationResponse(requestHash [32]byte, response uint8, responseURI string, responseHash [32]byte, tag string) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.ValidationResponse(&_ValidationRegistry.TransactOpts, requestHash, response, responseURI, responseHash, tag)
}

// ValidationResponse is a paid mutator transaction binding the contract method 0x3d659a96.
//
// Solidity: function validationResponse(bytes32 requestHash, uint8 response, string responseURI, bytes32 responseHash, string tag) returns()
func (_ValidationRegistry *ValidationRegistryTransactorSession) ValidationResponse(requestHash [32]byte, response uint8, responseURI string, responseHash [32]byte, tag string) (*types.Transaction, error) {
	return _ValidationRegistry.Contract.ValidationResponse(&_ValidationRegistry.TransactOpts, requestHash, response, responseURI, responseHash, tag)
}

// ValidationRegistryInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the ValidationRegistry contract.
type ValidationRegistryInitializedIterator struct {
	Event *ValidationRegistryInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ValidationRegistryInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidationRegistryInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ValidationRegistryInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ValidationRegistryInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidationRegistryInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidationRegistryInitialized represents a Initialized event raised by the ValidationRegistry contract.
type ValidationRegistryInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_ValidationRegistry *ValidationRegistryFilterer) FilterInitialized(opts *bind.FilterOpts) (*ValidationRegistryInitializedIterator, error) {

	logs, sub, err := _ValidationRegistry.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &ValidationRegistryInitializedIterator{contract: _ValidationRegistry.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_ValidationRegistry *ValidationRegistryFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *ValidationRegistryInitialized) (event.Subscription, error) {

	logs, sub, err := _ValidationRegistry.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidationRegistryInitialized)
				if err := _ValidationRegistry.contract.UnpackLog(event, "Initialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseInitialized is a log parse operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_ValidationRegistry *ValidationRegistryFilterer) ParseInitialized(log types.Log) (*ValidationRegistryInitialized, error) {
	event := new(ValidationRegistryInitialized)
	if err := _ValidationRegistry.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ValidationRegistryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the ValidationRegistry contract.
type ValidationRegistryOwnershipTransferredIterator struct {
	Event *ValidationRegistryOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ValidationRegistryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidationRegistryOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ValidationRegistryOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ValidationRegistryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidationRegistryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidationRegistryOwnershipTransferred represents a OwnershipTransferred event raised by the ValidationRegistry contract.
type ValidationRegistryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ValidationRegistry *ValidationRegistryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ValidationRegistryOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ValidationRegistry.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ValidationRegistryOwnershipTransferredIterator{contract: _ValidationRegistry.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ValidationRegistry *ValidationRegistryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ValidationRegistryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ValidationRegistry.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidationRegistryOwnershipTransferred)
				if err := _ValidationRegistry.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ValidationRegistry *ValidationRegistryFilterer) ParseOwnershipTransferred(log types.Log) (*ValidationRegistryOwnershipTransferred, error) {
	event := new(ValidationRegistryOwnershipTransferred)
	if err := _ValidationRegistry.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ValidationRegistryUpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the ValidationRegistry contract.
type ValidationRegistryUpgradedIterator struct {
	Event *ValidationRegistryUpgraded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ValidationRegistryUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidationRegistryUpgraded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ValidationRegistryUpgraded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ValidationRegistryUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidationRegistryUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidationRegistryUpgraded represents a Upgraded event raised by the ValidationRegistry contract.
type ValidationRegistryUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_ValidationRegistry *ValidationRegistryFilterer) FilterUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*ValidationRegistryUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _ValidationRegistry.contract.FilterLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &ValidationRegistryUpgradedIterator{contract: _ValidationRegistry.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_ValidationRegistry *ValidationRegistryFilterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *ValidationRegistryUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _ValidationRegistry.contract.WatchLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidationRegistryUpgraded)
				if err := _ValidationRegistry.contract.UnpackLog(event, "Upgraded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUpgraded is a log parse operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_ValidationRegistry *ValidationRegistryFilterer) ParseUpgraded(log types.Log) (*ValidationRegistryUpgraded, error) {
	event := new(ValidationRegistryUpgraded)
	if err := _ValidationRegistry.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ValidationRegistryValidationRequestIterator is returned from FilterValidationRequest and is used to iterate over the raw logs and unpacked data for ValidationRequest events raised by the ValidationRegistry contract.
type ValidationRegistryValidationRequestIterator struct {
	Event *ValidationRegistryValidationRequest // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ValidationRegistryValidationRequestIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidationRegistryValidationRequest)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ValidationRegistryValidationRequest)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ValidationRegistryValidationRequestIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidationRegistryValidationRequestIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidationRegistryValidationRequest represents a ValidationRequest event raised by the ValidationRegistry contract.
type ValidationRegistryValidationRequest struct {
	ValidatorAddress common.Address
	AgentId          *big.Int
	RequestURI       string
	RequestHash      [32]byte
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterValidationRequest is a free log retrieval operation binding the contract event 0x530436c3634a98e1e626b0898be2f1e9980cc1bd2a78c07a0aba52d0a48a5059.
//
// Solidity: event ValidationRequest(address indexed validatorAddress, uint256 indexed agentId, string requestURI, bytes32 indexed requestHash)
func (_ValidationRegistry *ValidationRegistryFilterer) FilterValidationRequest(opts *bind.FilterOpts, validatorAddress []common.Address, agentId []*big.Int, requestHash [][32]byte) (*ValidationRegistryValidationRequestIterator, error) {

	var validatorAddressRule []interface{}
	for _, validatorAddressItem := range validatorAddress {
		validatorAddressRule = append(validatorAddressRule, validatorAddressItem)
	}
	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	var requestHashRule []interface{}
	for _, requestHashItem := range requestHash {
		requestHashRule = append(requestHashRule, requestHashItem)
	}

	logs, sub, err := _ValidationRegistry.contract.FilterLogs(opts, "ValidationRequest", validatorAddressRule, agentIdRule, requestHashRule)
	if err != nil {
		return nil, err
	}
	return &ValidationRegistryValidationRequestIterator{contract: _ValidationRegistry.contract, event: "ValidationRequest", logs: logs, sub: sub}, nil
}

// WatchValidationRequest is a free log subscription operation binding the contract event 0x530436c3634a98e1e626b0898be2f1e9980cc1bd2a78c07a0aba52d0a48a5059.
//
// Solidity: event ValidationRequest(address indexed validatorAddress, uint256 indexed agentId, string requestURI, bytes32 indexed requestHash)
func (_ValidationRegistry *ValidationRegistryFilterer) WatchValidationRequest(opts *bind.WatchOpts, sink chan<- *ValidationRegistryValidationRequest, validatorAddress []common.Address, agentId []*big.Int, requestHash [][32]byte) (event.Subscription, error) {

	var validatorAddressRule []interface{}
	for _, validatorAddressItem := range validatorAddress {
		validatorAddressRule = append(validatorAddressRule, validatorAddressItem)
	}
	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}

	var requestHashRule []interface{}
	for _, requestHashItem := range requestHash {
		requestHashRule = append(requestHashRule, requestHashItem)
	}

	logs, sub, err := _ValidationRegistry.contract.WatchLogs(opts, "ValidationRequest", validatorAddressRule, agentIdRule, requestHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidationRegistryValidationRequest)
				if err := _ValidationRegistry.contract.UnpackLog(event, "ValidationRequest", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseValidationRequest is a log parse operation binding the contract event 0x530436c3634a98e1e626b0898be2f1e9980cc1bd2a78c07a0aba52d0a48a5059.
//
// Solidity: event ValidationRequest(address indexed validatorAddress, uint256 indexed agentId, string requestURI, bytes32 indexed requestHash)
func (_ValidationRegistry *ValidationRegistryFilterer) ParseValidationRequest(log types.Log) (*ValidationRegistryValidationRequest, error) {
	event := new(ValidationRegistryValidationRequest)
	if err := _ValidationRegistry.contract.UnpackLog(event, "ValidationRequest", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ValidationRegistryValidationResponseIterator is returned from FilterValidationResponse and is used to iterate over the raw logs and unpacked data for ValidationResponse events raised by the ValidationRegistry contract.
type ValidationRegistryValidationResponseIterator struct {
	Event *ValidationRegistryValidationResponse // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ValidationRegistryValidationResponseIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidationRegistryValidationResponse)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ValidationRegistryValidationResponse)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ValidationRegistryValidationResponseIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidationRegistryValidationResponseIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidationRegistryValidationResponse represents a ValidationResponse event raised by the ValidationRegistry contract.
type ValidationRegistryValidationResponse struct {
	ValidatorAddress common.Address
	AgentId          *big.Int
	RequestHash      [32]byte
	Response         uint8
	ResponseURI      string
	ResponseHash     [32]byte
	Tag              string
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterValidationResponse is a free log retrieval operation binding the contract event 0xafddf629e874ccc3963b6a888c477bd464a6c8525024fc88759ea3b2326349ae.
//
// Solidity: event ValidationResponse(address indexed validatorAddress, uint256 indexed agentId, bytes32 indexed requestHash, uint8 response, string responseURI, bytes32 responseHash, string tag)
func (_ValidationRegistry *ValidationRegistryFilterer) FilterValidationResponse(opts *bind.FilterOpts, validatorAddress []common.Address, agentId []*big.Int, requestHash [][32]byte) (*ValidationRegistryValidationResponseIterator, error) {

	var validatorAddressRule []interface{}
	for _, validatorAddressItem := range validatorAddress {
		validatorAddressRule = append(validatorAddressRule, validatorAddressItem)
	}
	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var requestHashRule []interface{}
	for _, requestHashItem := range requestHash {
		requestHashRule = append(requestHashRule, requestHashItem)
	}

	logs, sub, err := _ValidationRegistry.contract.FilterLogs(opts, "ValidationResponse", validatorAddressRule, agentIdRule, requestHashRule)
	if err != nil {
		return nil, err
	}
	return &ValidationRegistryValidationResponseIterator{contract: _ValidationRegistry.contract, event: "ValidationResponse", logs: logs, sub: sub}, nil
}

// WatchValidationResponse is a free log subscription operation binding the contract event 0xafddf629e874ccc3963b6a888c477bd464a6c8525024fc88759ea3b2326349ae.
//
// Solidity: event ValidationResponse(address indexed validatorAddress, uint256 indexed agentId, bytes32 indexed requestHash, uint8 response, string responseURI, bytes32 responseHash, string tag)
func (_ValidationRegistry *ValidationRegistryFilterer) WatchValidationResponse(opts *bind.WatchOpts, sink chan<- *ValidationRegistryValidationResponse, validatorAddress []common.Address, agentId []*big.Int, requestHash [][32]byte) (event.Subscription, error) {

	var validatorAddressRule []interface{}
	for _, validatorAddressItem := range validatorAddress {
		validatorAddressRule = append(validatorAddressRule, validatorAddressItem)
	}
	var agentIdRule []interface{}
	for _, agentIdItem := range agentId {
		agentIdRule = append(agentIdRule, agentIdItem)
	}
	var requestHashRule []interface{}
	for _, requestHashItem := range requestHash {
		requestHashRule = append(requestHashRule, requestHashItem)
	}

	logs, sub, err := _ValidationRegistry.contract.WatchLogs(opts, "ValidationResponse", validatorAddressRule, agentIdRule, requestHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidationRegistryValidationResponse)
				if err := _ValidationRegistry.contract.UnpackLog(event, "ValidationResponse", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseValidationResponse is a log parse operation binding the contract event 0xafddf629e874ccc3963b6a888c477bd464a6c8525024fc88759ea3b2326349ae.
//
// Solidity: event ValidationResponse(address indexed validatorAddress, uint256 indexed agentId, bytes32 indexed requestHash, uint8 response, string responseURI, bytes32 responseHash, string tag)
func (_ValidationRegistry *ValidationRegistryFilterer) ParseValidationResponse(log types.Log) (*ValidationRegistryValidationResponse, error) {
	event := new(ValidationRegistryValidationResponse)
	if err := _ValidationRegistry.contract.UnpackLog(event, "ValidationResponse", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
