package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SeedGUID GUID
type SeedGUID2CIDMap map[SeedGUID]CID

var (
	StewardConcept           ConceptGUID
	AssetConcept             ConceptGUID
	CoinConcept              ConceptGUID
	SmartContractConcept     ConceptGUID
	ContractEvaluatorConcept ConceptGUID
	InvestmentConcept        ConceptGUID
	TransactionConcept       ConceptGUID
	ReturnConcept            ConceptGUID
)

type ConceptSeed_i interface {
	GetSeedID() SeedGUID
	GetCID() CID
	SetCID(cid CID)

	Update(ctx context.Context) error

	Seed() *ConceptSeed
}

// ConceptSeed base structure for all seeds of concepts
type ConceptSeed struct {
	CID         CID `json:"-"`
	SeedID      SeedGUID
	ConceptID   ConceptGUID // Identifies the type of concept this seed represents
	Name        string
	Description string
	Timestamp   time.Time
}

type CoinValue_i interface {
	Value() float64
}

type CoherenceScore_i interface {
	Score(ctx context.Context, other CoherenceScore_i) float64
}

type CoherenceVector_i interface {
	Values(ctx context.Context) []float64
}

type Copy_i interface {
	Copy(ctx context.Context) (ConceptSeed_i, error)
}

type Merge_i interface {
	Merge(ctx context.Context, other ConceptSeed_i) (ConceptSeed_i, error)
}

type Move_i interface {
	Move(ctx context.Context, from ConceptSeed_i, to ConceptSeed_i) error
}

type Transform_i interface {
	Transform(ctx context.Context, conceptID ConceptGUID) (ConceptSeed_i, error)
}

type Parent_i interface {
	Parent() (ConceptSeed_i, error)
}

type Children_i interface {
	Children() ([]ConceptSeed_i, error)
}

type Related_i interface {
	Related() ([]ConceptSeed_i, error)
}

type RelatedByConcept_i interface {
	RelatedByConcept(conceptID ConceptGUID) ([]ConceptSeed_i, error)
}

type RelatedByName_i interface {
	RelatedByName(name string) ([]ConceptSeed_i, error)
}

type Stream_i interface {
	Read(ctx context.Context) (io.ReadCloser, error)
}

type Render_i interface {
	Render(ctx context.Context) (io.ReadCloser, error)
}

type RenderAs_i interface {
	RenderAs(ctx context.Context, contentType string) (io.ReadCloser, error)
}

// StewardSeed represents an entity that can steward assets and make investments
type StewardSeed struct {
	*ConceptSeed
	StewardAssets []SeedGUID // List of asset IDs the steward cares for
	Investments   []SeedGUID // Investments made by the steward
}

// Asset represents a valuable item or resource within the network
type AssetSeed struct {
	*ConceptSeed
	Steward SeedGUID // ID of the steward
}

// Coin represents units of currency used within the network for transactions
type CoinSeed struct {
	*ConceptSeed
	Value float64 // Monetary value of the coin
}

// SmartContract represents the contractual conditions attached to transactions
type SmartContractSeed struct {
	*ConceptSeed
	ContractEvaluator SeedGUID // ID of the evaluator responsible for this contract
	Conditions        string   // Detailed conditions as a string or structured data
}

// ContractEvaluator defines an entity responsible for evaluating smart contracts
type ContractEvaluatorSeed struct {
	*ConceptSeed
	EvaluationCriteria string // Criteria used to evaluate contracts
}

// Investment is a type of transaction with associated smart contracts
type InvestmentSeed struct {
	*ConceptSeed
	Steward       SeedGUID // Steward of the investment
	Asset         SeedGUID // Asset involved in the investment
	SmartContract SeedGUID // Associated smart contract
}

// Transaction represents an exchange or transfer of assets, coins, or services
type TransactionSeed struct {
	*ConceptSeed
	FromSteward SeedGUID // ID of the steward sending the asset or coins
	ToSteward   SeedGUID // ID of the steward receiving the asset or coins
	Asset       SeedGUID // Asset being transacted, if applicable
	Coin        SeedGUID // Coin being transacted, if applicable
}

// Return represents the benefits or gains from investments
type ReturnSeed struct {
	*ConceptSeed
	Investment SeedGUID // Investment that generated this return
	Amount     float64  // Quantitative value of the return
}

func (ci ConceptSeed) GetSeedID() SeedGUID {
	return ci.SeedID
}

func (ci ConceptSeed) GetCID() CID {
	return ci.CID
}

func (ci *ConceptSeed) SetCID(cid CID) {
	ci.CID = cid
}

func (ci *ConceptSeed) Seed() *ConceptSeed {
	return ci
}

func (ci *ConceptSeed) DefaultUpdate(ctx context.Context, json json.RawMessage) error {
	// DEBUG: oldCID := ci.CID
	if ci.CID != "" {
		network.Remove(ctx, ci.CID)
		delete(seedMap, ci.SeedID)
		ci.CID = ""
	}
	cid, err := network.Add(ctx, strings.NewReader(string(json)))
	// DEBUG: fmt.Printf("Seed Update: JSON=%s => %s\n", string(json), cid)
	if err != nil {
		return err
	}
	ci.CID = cid
	// DEBUG:  if oldCID != "" && ci.CID != oldCID {
	// DEBUG:    fmt.Printf("Seed [%s] CID %s => %s\n", ci, oldCID, ci.CID)
	// DEBUG:  }
	return nil
}

func (ci ConceptSeed) URI() string {
	return "ipfs://" + string(ci.CID)
}

func (id SeedGUID) AsSeed() ConceptSeed_i {
	seed, ok := seedMap[id]
	if ok {
		return seed
	}
	return nil
}

func (id SeedGUID) AsStewardSeed() *StewardSeed {
	seed := id.AsSeed()
	steward, ok := seed.(*StewardSeed)
	if ok {
		return steward
	}
	return nil
}

func addOrUpdateSeed(ctx context.Context, seed ConceptSeed_i, pID PeerID) error {
	if seed.GetCID() != "" {
		peerMap[pID].RemoveSeedCID(seed.GetCID())
	}

	if err := seed.Update(ctx); err != nil {
		log.Printf("Failed to update seed: %v", err)
		return err
	}
	seedMap[seed.GetSeedID()] = seed
	seedID2CID[seed.GetSeedID()] = seed.GetCID()
	log.Printf("Added/Updated seed: %s\n", seed)

	if err := saveSeeds(ctx); err != nil {
		log.Printf("Failed to save seed list: %v", err)
		return err
	}

	peerMap[pID].AddSeedCID(seed.GetCID())
	return nil
}

func NewConceptSeed(conceptID ConceptGUID, name string, desc string) *ConceptSeed {
	return &ConceptSeed{
		SeedID:      SeedGUID(uuid.New().String()),
		ConceptID:   conceptID,
		Name:        name,
		Description: desc,
		Timestamp:   time.Now(),
	}
}

func (ci *ConceptSeed) AsString() string {
	return fmt.Sprintf("Name=%s, Desc=%s", ci.Name, ci.Description)
}

func (ci *ConceptSeed) DefaultString() string {
	return fmt.Sprintf("CID=%s, ID=%s, Concept=%s, [%s]", ci.CID, ci.SeedID, ci.ConceptID.AsConcept().Name, ci.AsString())
}

func (ci *ConceptSeed) String() string { return ci.DefaultString() }

func NewStewardSeed(name string, desc string) *StewardSeed {
	return &StewardSeed{
		ConceptSeed:   NewConceptSeed(StewardConcept, name, desc),
		StewardAssets: []SeedGUID{},
		Investments:   []SeedGUID{},
	}
}

func (i *StewardSeed) String() string {
	return fmt.Sprintf("%s [OwnedAssets=%v, Investments=%v]", i.DefaultString(), i.StewardAssets, i.Investments)
}

func (i *StewardSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewAssetSeed(name string, desc string, steward SeedGUID) *AssetSeed {
	return &AssetSeed{
		ConceptSeed: NewConceptSeed(AssetConcept, name, desc),
		Steward:     steward,
	}
}

func (ci *AssetSeed) String() string {
	return fmt.Sprintf("%s, Steward=[%s]", ci.DefaultString(), ci.Steward.AsStewardSeed().AsString())
}

func (i *AssetSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewCoinSeed(value float64) *CoinSeed {
	return &CoinSeed{
		ConceptSeed: NewConceptSeed(CoinConcept, "", ""),
		Value:       value,
	}
}

func (i *CoinSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewSmartContractSeed(name string, desc string, contractEvaluator SeedGUID, conditions string) *SmartContractSeed {
	return &SmartContractSeed{
		ConceptSeed:       NewConceptSeed(SmartContractConcept, name, desc),
		ContractEvaluator: contractEvaluator,
		Conditions:        conditions,
	}
}

func (i *SmartContractSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewContractEvaluatorSeed(name string, desc string, evaluationCriteria string) *ContractEvaluatorSeed {
	return &ContractEvaluatorSeed{
		ConceptSeed:        NewConceptSeed(ContractEvaluatorConcept, name, desc),
		EvaluationCriteria: evaluationCriteria,
	}
}

func (i *ContractEvaluatorSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewInvestmentSeed(name string, desc string, steward SeedGUID, asset SeedGUID, smartContract SeedGUID) *InvestmentSeed {
	return &InvestmentSeed{
		ConceptSeed:   NewConceptSeed(InvestmentConcept, name, desc),
		Steward:       steward,
		Asset:         asset,
		SmartContract: smartContract,
	}
}

func (i *InvestmentSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewTransactionSeed(name string, desc string, fromSteward SeedGUID, toSteward SeedGUID, asset SeedGUID, coin SeedGUID) *TransactionSeed {
	return &TransactionSeed{
		ConceptSeed: NewConceptSeed(TransactionConcept, name, desc),
		FromSteward: fromSteward,
		ToSteward:   toSteward,
		Asset:       asset,
		Coin:        coin,
	}
}

func (i *TransactionSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewReturnSeed(name string, desc string, investment SeedGUID, amount float64) *ReturnSeed {
	return &ReturnSeed{
		ConceptSeed: NewConceptSeed(ReturnConcept, name, desc),
		Investment:  investment,
		Amount:      amount,
	}
}

func (i *ReturnSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

type UnmarshalSeedFunc func(data json.RawMessage) (ConceptSeed_i, error)

var unmarshalSeedFuncs map[ConceptGUID]UnmarshalSeedFunc

func initSeedUnmarshal() {
	unmarshalSeedFuncs = map[ConceptGUID]UnmarshalSeedFunc{
		StewardConcept: func(data json.RawMessage) (ConceptSeed_i, error) {
			return genericUnmarshalSeed[*StewardSeed](data)
		},
		AssetConcept: func(data json.RawMessage) (ConceptSeed_i, error) {
			return genericUnmarshalSeed[*AssetSeed](data)
		},
		CoinConcept: func(data json.RawMessage) (ConceptSeed_i, error) {
			return genericUnmarshalSeed[*CoinSeed](data)
		},
		SmartContractConcept: func(data json.RawMessage) (ConceptSeed_i, error) {
			return genericUnmarshalSeed[*SmartContractSeed](data)
		},
		ContractEvaluatorConcept: func(data json.RawMessage) (ConceptSeed_i, error) {
			return genericUnmarshalSeed[*ContractEvaluatorSeed](data)
		},
		InvestmentConcept: func(data json.RawMessage) (ConceptSeed_i, error) {
			return genericUnmarshalSeed[*InvestmentSeed](data)
		},
		TransactionConcept: func(data json.RawMessage) (ConceptSeed_i, error) {
			return genericUnmarshalSeed[*TransactionSeed](data)
		},
		ReturnConcept: func(data json.RawMessage) (ConceptSeed_i, error) {
			return genericUnmarshalSeed[*ReturnSeed](data)
		},
	}
}

func genericUnmarshalSeed[T ConceptSeed_i](data json.RawMessage) (ConceptSeed_i, error) {
	var seed T
	if err := json.Unmarshal(data, &seed); err != nil {
		return nil, err
	}
	return seed, nil
}

type ConceptSeedMap map[SeedGUID]ConceptSeed_i

func UnmarshalJSON2ConceptSeed(raw json.RawMessage) (ConceptSeed_i, error) {
	var ci ConceptSeed
	json.Unmarshal(raw, &ci)
	unmarshal, exists := unmarshalSeedFuncs[ci.ConceptID]
	if !exists {
		return nil, fmt.Errorf("unmarshal function not found for ConceptID: %s", ci.ConceptID)
	}

	seed, err := unmarshal(raw)
	if err != nil {
		return nil, err
	}
	return seed, nil
}

func (cim *ConceptSeedMap) UnmarshalJSON(data []byte) error {
	var rawSeeds map[SeedGUID]json.RawMessage
	if err := json.Unmarshal(data, &rawSeeds); err != nil {
		return err
	}

	*cim = make(ConceptSeedMap)
	for id, raw := range rawSeeds {
		seed, err := UnmarshalJSON2ConceptSeed(raw)
		if err != nil {
			return err
		}
		(*cim)[id] = seed
		cid, ok := seedID2CID[id]
		if ok {
			seed.SetCID(cid)
		}
	}
	return nil
}
