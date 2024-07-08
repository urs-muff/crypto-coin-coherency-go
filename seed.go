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
	ConceptInvestmentConcept ConceptGUID
	SeedInvestmentConcept    ConceptGUID
	TransactionConcept       ConceptGUID
	ReturnConcept            ConceptGUID
	ProposalConcept          ConceptGUID
	ProposalActionConcept    ConceptGUID
	HarmonyGuidelineConcept  ConceptGUID
)

type Seed_i interface {
	GetSeedID() SeedGUID
	GetCID() CID
	SetCID(cid CID)

	Update(ctx context.Context) error

	GetCoreSeed() *CoreSeed
}

// CoreSeed base structure for all seeds of concepts
type CoreSeed struct {
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
	Copy(ctx context.Context) (Seed_i, error)
}

type Merge_i interface {
	Merge(ctx context.Context, other Seed_i) (Seed_i, error)
}

type Move_i interface {
	Move(ctx context.Context, from Seed_i, to Seed_i) error
}

type Transform_i interface {
	Transform(ctx context.Context, conceptID ConceptGUID) (Seed_i, error)
}

type Parent_i interface {
	Parent() (Seed_i, error)
}

type Children_i interface {
	Children() ([]Seed_i, error)
}

type Related_i interface {
	Related() ([]Seed_i, error)
}

type RelatedByConcept_i interface {
	RelatedByConcept(conceptID ConceptGUID) ([]Seed_i, error)
}

type RelatedByName_i interface {
	RelatedByName(name string) ([]Seed_i, error)
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
	*CoreSeed
	EnergyBalance float64
}

// Asset represents a valuable item or resource within the network
type AssetSeed struct {
	*CoreSeed
	StewardID   SeedGUID
	ContentType string
	Content     string
}

// Coin represents units of currency used within the network for transactions
type CoinSeed struct {
	*CoreSeed
	Value float64
}

// SmartContract represents the contractual conditions attached to transactions
type SmartContractSeed struct {
	*CoreSeed
	ContractEvaluator SeedGUID // ID of the evaluator responsible for this contract
	Conditions        string   // Detailed conditions as a string or structured data
}

// ContractEvaluator defines an entity responsible for evaluating smart contracts
type ContractEvaluatorSeed struct {
	*CoreSeed
	EvaluationCriteria string // Criteria used to evaluate contracts
}

// Investment is a type of transaction with associated smart contracts
type ConceptInvestmentSeed struct {
	*CoreSeed
	InvestorID SeedGUID
	TargetID   ConceptGUID
	Amount     float64
}

type SeedInvestmentSeed struct {
	*CoreSeed
	InvestorID SeedGUID
	TargetID   SeedGUID
	Amount     float64
}

// Transaction represents an exchange or transfer of assets, coins, or services
type TransactionSeed struct {
	*CoreSeed
	FromSteward SeedGUID // ID of the steward sending the asset or coins
	ToSteward   SeedGUID // ID of the steward receiving the asset or coins
	Asset       SeedGUID // Asset being transacted, if applicable
	Coin        SeedGUID // Coin being transacted, if applicable
}

// Return represents the benefits or gains from investments
type ReturnSeed struct {
	*CoreSeed
	Investment SeedGUID // Investment that generated this return
	Amount     float64  // Quantitative value of the return
}

type ProposalAction struct {
	*CoreSeed
	TargetID   ConceptGUID
	ActionType string // e.g., 'UPDATE', 'CREATE', 'DELETE'
	ActionData map[string]any
}

type Proposal struct {
	*CoreSeed
	StewardID    SeedGUID
	ActionSeedID SeedGUID
	VotesFor     int
	VotesAgainst int
	Status       string
}

func (ci CoreSeed) GetSeedID() SeedGUID {
	return ci.SeedID
}

func (ci CoreSeed) GetCID() CID {
	return ci.CID
}

func (ci *CoreSeed) SetCID(cid CID) {
	ci.CID = cid
}

func (ci *CoreSeed) GetCoreSeed() *CoreSeed {
	return ci
}

func (ci *CoreSeed) DefaultUpdate(ctx context.Context, json json.RawMessage) error {
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

func (ci CoreSeed) URI() string {
	return "ipfs://" + string(ci.CID)
}

func (id SeedGUID) AsSeed() Seed_i {
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

func addOrUpdateSeed(ctx context.Context, seed Seed_i, pID PeerID) error {
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

func NewCoreSeed(conceptID ConceptGUID, name string, desc string) *CoreSeed {
	return &CoreSeed{
		SeedID:      SeedGUID(uuid.New().String()),
		ConceptID:   conceptID,
		Name:        name,
		Description: desc,
		Timestamp:   time.Now(),
	}
}

func (ci *CoreSeed) AsString() string {
	return fmt.Sprintf("Name=%s, Desc=%s", ci.Name, ci.Description)
}

func (ci *CoreSeed) DefaultString() string {
	return fmt.Sprintf("CID=%s, ID=%s, Concept=%s, [%s]", ci.CID, ci.SeedID, ci.ConceptID.AsConcept().Name, ci.AsString())
}

func (ci *CoreSeed) String() string { return ci.DefaultString() }

func (i *CoreSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewStewardSeed(name string, desc string) *StewardSeed {
	return &StewardSeed{
		CoreSeed: NewCoreSeed(StewardConcept, name, desc),
	}
}

func (i *StewardSeed) String() string {
	return fmt.Sprintf("%s [EnergyBalance=%f]", i.DefaultString(), i.EnergyBalance)
}

func (i *StewardSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewAssetSeed(name string, desc string, steward SeedGUID) *AssetSeed {
	return &AssetSeed{
		CoreSeed:  NewCoreSeed(AssetConcept, name, desc),
		StewardID: steward,
	}
}

func (ci *AssetSeed) String() string {
	return fmt.Sprintf("%s, Steward=[%s]", ci.DefaultString(), ci.StewardID.AsStewardSeed().AsString())
}

func (i *AssetSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewCoinSeed(value float64) *CoinSeed {
	return &CoinSeed{
		CoreSeed: NewCoreSeed(CoinConcept, "", ""),
		Value:    value,
	}
}

func (i *CoinSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewSmartContractSeed(name string, desc string, contractEvaluator SeedGUID, conditions string) *SmartContractSeed {
	return &SmartContractSeed{
		CoreSeed:          NewCoreSeed(SmartContractConcept, name, desc),
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
		CoreSeed:           NewCoreSeed(ContractEvaluatorConcept, name, desc),
		EvaluationCriteria: evaluationCriteria,
	}
}

func (i *ContractEvaluatorSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func (i *ConceptInvestmentSeed) String() string {
	return fmt.Sprintf("%s, Investor=[%s], Target=[%s], Amount=%f",
		i.DefaultString(),
		i.InvestorID.AsStewardSeed().AsString(),
		i.TargetID.AsConcept(),
		i.Amount,
	)
}

func (i *ConceptInvestmentSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func (i *SeedInvestmentSeed) String() string {
	return fmt.Sprintf("%s, Investor=[%s], Target=[%s], Amount=%f",
		i.DefaultString(),
		i.InvestorID.AsStewardSeed().AsString(),
		i.TargetID.AsSeed(),
		i.Amount,
	)
}

func (i *SeedInvestmentSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func NewTransactionSeed(name string, desc string, fromSteward SeedGUID, toSteward SeedGUID, asset SeedGUID, coin SeedGUID) *TransactionSeed {
	return &TransactionSeed{
		CoreSeed:    NewCoreSeed(TransactionConcept, name, desc),
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
		CoreSeed:   NewCoreSeed(ReturnConcept, name, desc),
		Investment: investment,
		Amount:     amount,
	}
}

func (i *ReturnSeed) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func (i *ProposalAction) String() string {
	return fmt.Sprintf("%s, Target=[%s], Action=[%s], Data=[%s]",
		i.DefaultString(),
		i.TargetID.AsConcept().String(),
		i.ActionType,
		i.ActionData,
	)
}

func (i *ProposalAction) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

func (i *Proposal) String() string {
	return fmt.Sprintf("%s, Steward=[%s], Action=[%s], For=[%d], Against=[%d], Status=[%s]",
		i.DefaultString(),
		i.StewardID.AsStewardSeed().String(),
		i.ActionSeedID.AsSeed(),
		i.VotesFor,
		i.VotesAgainst,
		i.Status,
	)
}

func (i *Proposal) Update(ctx context.Context) error {
	json, _ := json.Marshal(i)
	return i.DefaultUpdate(ctx, json)
}

type UnmarshalSeedFunc func(data json.RawMessage) (Seed_i, error)

var unmarshalSeedFuncs map[ConceptGUID]UnmarshalSeedFunc

func initSeedUnmarshal() {
	unmarshalSeedFuncs = map[ConceptGUID]UnmarshalSeedFunc{
		StewardConcept: func(data json.RawMessage) (Seed_i, error) {
			return genericUnmarshalSeed[*StewardSeed](data)
		},
		AssetConcept: func(data json.RawMessage) (Seed_i, error) {
			return genericUnmarshalSeed[*AssetSeed](data)
		},
		CoinConcept: func(data json.RawMessage) (Seed_i, error) {
			return genericUnmarshalSeed[*CoinSeed](data)
		},
		SmartContractConcept: func(data json.RawMessage) (Seed_i, error) {
			return genericUnmarshalSeed[*SmartContractSeed](data)
		},
		ContractEvaluatorConcept: func(data json.RawMessage) (Seed_i, error) {
			return genericUnmarshalSeed[*ContractEvaluatorSeed](data)
		},
		ConceptInvestmentConcept: func(data json.RawMessage) (Seed_i, error) {
			return genericUnmarshalSeed[*ConceptInvestmentSeed](data)
		},
		SeedInvestmentConcept: func(data json.RawMessage) (Seed_i, error) {
			return genericUnmarshalSeed[*SeedInvestmentSeed](data)
		},
		TransactionConcept: func(data json.RawMessage) (Seed_i, error) {
			return genericUnmarshalSeed[*TransactionSeed](data)
		},
		ReturnConcept: func(data json.RawMessage) (Seed_i, error) {
			return genericUnmarshalSeed[*ReturnSeed](data)
		},
		ProposalActionConcept: func(data json.RawMessage) (Seed_i, error) {
			return genericUnmarshalSeed[*ProposalAction](data)
		},
		ProposalConcept: func(data json.RawMessage) (Seed_i, error) {
			return genericUnmarshalSeed[*Proposal](data)
		},
		HarmonyGuidelineConcept: func(data json.RawMessage) (Seed_i, error) {
			return genericUnmarshalSeed[*CoinSeed](data)
		},
	}
}

func genericUnmarshalSeed[T Seed_i](data json.RawMessage) (Seed_i, error) {
	var seed T
	if err := json.Unmarshal(data, &seed); err != nil {
		return nil, err
	}
	return seed, nil
}

type SeedMap map[SeedGUID]Seed_i

func UnmarshalJSON2Seed(raw json.RawMessage) (Seed_i, error) {
	var ci CoreSeed
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

func (cim *SeedMap) UnmarshalJSON(data []byte) error {
	var rawSeeds map[SeedGUID]json.RawMessage
	if err := json.Unmarshal(data, &rawSeeds); err != nil {
		return err
	}

	*cim = make(SeedMap)
	for id, raw := range rawSeeds {
		seed, err := UnmarshalJSON2Seed(raw)
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
