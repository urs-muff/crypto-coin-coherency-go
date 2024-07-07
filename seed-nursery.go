package main

import "fmt"

type SeedNursery struct {
}

// CreateSeed creates a new Seed based on the provided concept type
func (sf *SeedNursery) CreateSeed(concept string, name string, description string, data map[string]any) (Seed_i, error) {
	baseSeed := NewCoreSeed(ConceptGUID(concept), name, description)

	switch concept {
	case "Steward":
		return sf.createStewardSeed(baseSeed, data)
	case "Asset":
		return sf.createAssetSeed(baseSeed, data)
	case "Coin":
		return sf.createCoinSeed(baseSeed, data)
	case "SmartContract":
		return sf.createSmartContractSeed(baseSeed, data)
	case "ContractEvaluator":
		return sf.createContractEvaluatorSeed(baseSeed, data)
	case "Investment":
		return sf.createInvestmentSeed(baseSeed, data)
	case "Transaction":
		return sf.createTransactionSeed(baseSeed, data)
	case "Return":
		return sf.createReturnSeed(baseSeed, data)
	default:
		return nil, fmt.Errorf("concept not handled: %s", concept)
	}
}

func (sf *SeedNursery) createStewardSeed(base *CoreSeed, data map[string]any) (*StewardSeed, error) {
	seed := &StewardSeed{CoreSeed: base}
	if assets, ok := data["stewardAssets"].([]string); ok {
		for _, asset := range assets {
			seed.StewardAssets = append(seed.StewardAssets, SeedGUID(asset))
		}
	}
	if investments, ok := data["investments"].([]string); ok {
		for _, investment := range investments {
			seed.Investments = append(seed.Investments, SeedGUID(investment))
		}
	}
	return seed, nil
}

func (sf *SeedNursery) createAssetSeed(base *CoreSeed, data map[string]any) (*AssetSeed, error) {
	seed := &AssetSeed{CoreSeed: base}
	if steward, ok := data["steward"].(string); ok {
		seed.Steward = SeedGUID(steward)
	}
	return seed, nil
}

func (sf *SeedNursery) createCoinSeed(base *CoreSeed, data map[string]any) (*CoinSeed, error) {
	seed := &CoinSeed{CoreSeed: base}
	if value, ok := data["value"].(float64); ok {
		seed.Value = value
	}
	return seed, nil
}

func (sf *SeedNursery) createSmartContractSeed(base *CoreSeed, data map[string]any) (*SmartContractSeed, error) {
	seed := &SmartContractSeed{CoreSeed: base}
	if evaluator, ok := data["contractEvaluator"].(string); ok {
		seed.ContractEvaluator = SeedGUID(evaluator)
	}
	if conditions, ok := data["conditions"].(string); ok {
		seed.Conditions = conditions
	}
	return seed, nil
}

func (sf *SeedNursery) createContractEvaluatorSeed(base *CoreSeed, data map[string]any) (*ContractEvaluatorSeed, error) {
	seed := &ContractEvaluatorSeed{CoreSeed: base}
	if criteria, ok := data["evaluationCriteria"].(string); ok {
		seed.EvaluationCriteria = criteria
	}
	return seed, nil
}

func (sf *SeedNursery) createInvestmentSeed(base *CoreSeed, data map[string]any) (*InvestmentSeed, error) {
	seed := &InvestmentSeed{CoreSeed: base}
	if steward, ok := data["steward"].(string); ok {
		seed.Steward = SeedGUID(steward)
	}
	if asset, ok := data["asset"].(string); ok {
		seed.Asset = SeedGUID(asset)
	}
	if contract, ok := data["smartContract"].(string); ok {
		seed.SmartContract = SeedGUID(contract)
	}
	return seed, nil
}

func (sf *SeedNursery) createTransactionSeed(base *CoreSeed, data map[string]any) (*TransactionSeed, error) {
	seed := &TransactionSeed{CoreSeed: base}
	if fromSteward, ok := data["fromSteward"].(string); ok {
		seed.FromSteward = SeedGUID(fromSteward)
	}
	if toSteward, ok := data["toSteward"].(string); ok {
		seed.ToSteward = SeedGUID(toSteward)
	}
	if asset, ok := data["asset"].(string); ok {
		seed.Asset = SeedGUID(asset)
	}
	if coin, ok := data["coin"].(string); ok {
		seed.Coin = SeedGUID(coin)
	}
	return seed, nil
}

func (sf *SeedNursery) createReturnSeed(base *CoreSeed, data map[string]any) (*ReturnSeed, error) {
	seed := &ReturnSeed{CoreSeed: base}
	if investment, ok := data["investment"].(string); ok {
		seed.Investment = SeedGUID(investment)
	}
	if amount, ok := data["amount"].(float64); ok {
		seed.Amount = amount
	}
	return seed, nil
}
