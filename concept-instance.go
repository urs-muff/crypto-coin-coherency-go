package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type InstanceGUID GUID
type InstanceGUID2CIDMap map[InstanceGUID]CID

var (
	OwnerConcept             ConceptGUID
	AssetConcept             ConceptGUID
	CoinConcept              ConceptGUID
	SmartContractConcept     ConceptGUID
	ContractEvaluatorConcept ConceptGUID
	InvestmentConcept        ConceptGUID
	TransactionConcept       ConceptGUID
	ReturnConcept            ConceptGUID
)

type ConceptInstance_i interface {
	GetInstanceID() InstanceGUID
	GetCID() CID
	SetCID(cid CID)

	Update(ctx context.Context) error
}

// ConceptInstance base structure for all instances of concepts
type ConceptInstance struct {
	CID         CID `json:"-"`
	InstanceID  InstanceGUID
	ConceptID   ConceptGUID // Identifies the type of concept this instance represents
	Name        string
	Description string
	Timestamp   time.Time
}

// OwnerInstance represents an entity that can own assets and make investments
type OwnerInstance struct {
	*ConceptInstance
	OwnedAssets []InstanceGUID // List of asset IDs the owner possesses
	Investments []InstanceGUID // Investments made by the owner
}

// Asset represents a valuable item or resource within the network
type AssetInstance struct {
	*ConceptInstance
	Owner InstanceGUID // ID of the owner
}

// Coin represents units of currency used within the network for transactions
type CoinInstance struct {
	*ConceptInstance
	Value float64 // Monetary value of the coin
}

// SmartContract represents the contractual conditions attached to transactions
type SmartContractInstance struct {
	*ConceptInstance
	ContractEvaluator InstanceGUID // ID of the evaluator responsible for this contract
	Conditions        string       // Detailed conditions as a string or structured data
}

// ContractEvaluator defines an entity responsible for evaluating smart contracts
type ContractEvaluatorInstance struct {
	*ConceptInstance
	EvaluationCriteria string // Criteria used to evaluate contracts
}

// Investment is a type of transaction with associated smart contracts
type InvestmentInstance struct {
	*ConceptInstance
	Owner         InstanceGUID // Owner of the investment
	Asset         InstanceGUID // Asset involved in the investment
	SmartContract InstanceGUID // Associated smart contract
}

// Transaction represents an exchange or transfer of assets, coins, or services
type TransactionInstance struct {
	*ConceptInstance
	FromOwner InstanceGUID // ID of the owner sending the asset or coins
	ToOwner   InstanceGUID // ID of the owner receiving the asset or coins
	Asset     InstanceGUID // Asset being transacted, if applicable
	Coin      InstanceGUID // Coin being transacted, if applicable
}

// Return represents the benefits or gains from investments
type ReturnInstance struct {
	*ConceptInstance
	Investment InstanceGUID // Investment that generated this return
	Amount     float64      // Quantitative value of the return
}

func (ci ConceptInstance) GetInstanceID() InstanceGUID {
	return ci.InstanceID
}

func (ci ConceptInstance) GetCID() CID {
	return ci.CID
}

func (ci *ConceptInstance) SetCID(cid CID) {
	ci.CID = cid
}

func (ci *ConceptInstance) DefaultUpdate(ctx context.Context, json json.RawMessage) error {
	// DEBUG: oldCID := ci.CID
	if ci.CID != "" {
		network.Remove(ctx, ci.CID)
		delete(instanceMap, ci.InstanceID)
		ci.CID = ""
	}
	cid, err := network.Add(ctx, strings.NewReader(string(json)))
	// DEBUG: fmt.Printf("Instance Update: JSON=%s => %s\n", string(json), cid)
	if err != nil {
		return err
	}
	ci.CID = cid
	// DEBUG:  if oldCID != "" && ci.CID != oldCID {
	// DEBUG:    fmt.Printf("Istance [%s] CID %s => %s\n", ci, oldCID, ci.CID)
	// DEBUG:  }
	return nil
}

func (id InstanceGUID) AsInstance() ConceptInstance_i {
	inst, ok := instanceMap[id]
	if ok {
		return inst
	}
	return nil
}

func (id InstanceGUID) AsOwnerInstance() *OwnerInstance {
	inst := id.AsInstance()
	owner, ok := inst.(*OwnerInstance)
	if ok {
		return owner
	}
	return nil
}

func addOrUpdateInstance(ctx context.Context, instance ConceptInstance_i, pID PeerID) error {
	if instance.GetCID() != "" {
		peerMap[pID].RemoveInstanceCID(instance.GetCID())
	}

	if err := instance.Update(ctx); err != nil {
		log.Printf("Failed to update instance: %v", err)
		return err
	}
	instanceMap[instance.GetInstanceID()] = instance
	instanceID2CID[instance.GetInstanceID()] = instance.GetCID()
	log.Printf("Added/Updated instance: %s\n", instance)

	if err := saveInstances(ctx); err != nil {
		log.Printf("Failed to save instance list: %v", err)
		return err
	}

	peerMap[pID].AddInstanceCID(instance.GetCID())
	return nil
}

func NewConceptInstance(conceptID ConceptGUID, name string, desc string) *ConceptInstance {
	return &ConceptInstance{
		InstanceID:  InstanceGUID(uuid.New().String()),
		ConceptID:   conceptID,
		Name:        name,
		Description: desc,
		Timestamp:   time.Now(),
	}
}

func (ci *ConceptInstance) AsString() string {
	return fmt.Sprintf("Name=%s, Desc=%s", ci.Name, ci.Description)
}

func (ci *ConceptInstance) DefaultString() string {
	return fmt.Sprintf("CID=%s, ID=%s, Concept=%s, [%s]", ci.CID, ci.InstanceID, ci.ConceptID.AsConcept().Name, ci.AsString())
}

func (ci *ConceptInstance) String() string { return ci.DefaultString() }

func NewOwnerInstance(name string, desc string) *OwnerInstance {
	return &OwnerInstance{
		ConceptInstance: NewConceptInstance(OwnerConcept, name, desc),
		OwnedAssets:     []InstanceGUID{},
		Investments:     []InstanceGUID{},
	}
}

func (i *OwnerInstance) String() string {
	return fmt.Sprintf("%s [OwnedAssets=%v, Investments=%v]", i.DefaultString(), i.OwnedAssets, i.Investments)
}

func (i *OwnerInstance) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewAssetInstance(name string, desc string, owner InstanceGUID) *AssetInstance {
	return &AssetInstance{
		ConceptInstance: NewConceptInstance(AssetConcept, name, desc),
		Owner:           owner,
	}
}

func (ci *AssetInstance) String() string {
	return fmt.Sprintf("%s, Owner=[%s]", ci.DefaultString(), ci.Owner.AsOwnerInstance().AsString())
}

func (i *AssetInstance) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewCoinInstance(value float64) *CoinInstance {
	return &CoinInstance{
		ConceptInstance: NewConceptInstance(CoinConcept, "", ""),
		Value:           value,
	}
}

func (i *CoinInstance) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewSmartContractInstance(name string, desc string, contractEvaluator InstanceGUID, conditions string) *SmartContractInstance {
	return &SmartContractInstance{
		ConceptInstance:   NewConceptInstance(SmartContractConcept, name, desc),
		ContractEvaluator: contractEvaluator,
		Conditions:        conditions,
	}
}

func (i *SmartContractInstance) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewContractEvaluatorInstance(name string, desc string, evaluationCriteria string) *ContractEvaluatorInstance {
	return &ContractEvaluatorInstance{
		ConceptInstance:    NewConceptInstance(ContractEvaluatorConcept, name, desc),
		EvaluationCriteria: evaluationCriteria,
	}
}

func (i *ContractEvaluatorInstance) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewInvestmentInstance(name string, desc string, owner InstanceGUID, asset InstanceGUID, smartContract InstanceGUID) *InvestmentInstance {
	return &InvestmentInstance{
		ConceptInstance: NewConceptInstance(InvestmentConcept, name, desc),
		Owner:           owner,
		Asset:           asset,
		SmartContract:   smartContract,
	}
}

func (i *InvestmentInstance) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewTransactionInstance(name string, desc string, fromOwner InstanceGUID, toOwner InstanceGUID, asset InstanceGUID, coin InstanceGUID) *TransactionInstance {
	return &TransactionInstance{
		ConceptInstance: NewConceptInstance(TransactionConcept, name, desc),
		FromOwner:       fromOwner,
		ToOwner:         toOwner,
		Asset:           asset,
		Coin:            coin,
	}
}

func (i *TransactionInstance) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewReturnInstance(name string, desc string, investment InstanceGUID, amount float64) *ReturnInstance {
	return &ReturnInstance{
		ConceptInstance: NewConceptInstance(ReturnConcept, name, desc),
		Investment:      investment,
		Amount:          amount,
	}
}

func (i *ReturnInstance) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

type UnmarshalInstanceFunc func(data json.RawMessage) (ConceptInstance_i, error)

var unmarshalInstanceFuncs map[ConceptGUID]UnmarshalInstanceFunc

func initInstanceUnmarshal() {
	unmarshalInstanceFuncs = map[ConceptGUID]UnmarshalInstanceFunc{
		OwnerConcept: func(data json.RawMessage) (ConceptInstance_i, error) {
			return genericUnmarshalInstance[*OwnerInstance](data)
		},
		AssetConcept: func(data json.RawMessage) (ConceptInstance_i, error) {
			return genericUnmarshalInstance[*AssetInstance](data)
		},
		CoinConcept: func(data json.RawMessage) (ConceptInstance_i, error) {
			return genericUnmarshalInstance[*CoinInstance](data)
		},
		SmartContractConcept: func(data json.RawMessage) (ConceptInstance_i, error) {
			return genericUnmarshalInstance[*SmartContractInstance](data)
		},
		ContractEvaluatorConcept: func(data json.RawMessage) (ConceptInstance_i, error) {
			return genericUnmarshalInstance[*ContractEvaluatorInstance](data)
		},
		InvestmentConcept: func(data json.RawMessage) (ConceptInstance_i, error) {
			return genericUnmarshalInstance[*InvestmentInstance](data)
		},
		TransactionConcept: func(data json.RawMessage) (ConceptInstance_i, error) {
			return genericUnmarshalInstance[*TransactionInstance](data)
		},
		ReturnConcept: func(data json.RawMessage) (ConceptInstance_i, error) {
			return genericUnmarshalInstance[*ReturnInstance](data)
		},
	}
}

func genericUnmarshalInstance[T ConceptInstance_i](data json.RawMessage) (ConceptInstance_i, error) {
	var instance T
	if err := json.Unmarshal(data, &instance); err != nil {
		return nil, err
	}
	return instance, nil
}

type ConceptInstanceMap map[InstanceGUID]ConceptInstance_i

func UnmarshalJSON2ConceptInstance(raw json.RawMessage) (ConceptInstance_i, error) {
	var ci ConceptInstance
	json.Unmarshal(raw, &ci)
	unmarshal, exists := unmarshalInstanceFuncs[ci.ConceptID]
	if !exists {
		return nil, fmt.Errorf("unmarshal function not found for ConceptID: %s", ci.ConceptID)
	}

	instance, err := unmarshal(raw)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (cim *ConceptInstanceMap) UnmarshalJSON(data []byte) error {
	var rawInstances map[InstanceGUID]json.RawMessage
	if err := json.Unmarshal(data, &rawInstances); err != nil {
		return err
	}

	*cim = make(ConceptInstanceMap)
	for id, raw := range rawInstances {
		instance, err := UnmarshalJSON2ConceptInstance(raw)
		if err != nil {
			return err
		}
		(*cim)[id] = instance
		cid, ok := instanceID2CID[id]
		if ok {
			instance.SetCID(cid)
		}
	}
	return nil
}
