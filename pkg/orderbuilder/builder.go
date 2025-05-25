package orderbuilder

import (
	"fmt"
	"math/big"

	"github.com/polymarket/go-order-utils/pkg/builder"
	"github.com/polymarket/go-order-utils/pkg/model"
	"github.com/pooofdevelopment/go-clob-client/pkg/config"
	"github.com/pooofdevelopment/go-clob-client/pkg/signer"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
	"github.com/pooofdevelopment/go-clob-client/pkg/utilities"
)

// RoundingConfig maps tick sizes to rounding configurations
// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:30-35
var RoundingConfig = map[types.TickSize]types.RoundConfig{
	types.TickSize01:    {Price: 1, Size: 2, Amount: 3},
	types.TickSize001:   {Price: 2, Size: 2, Amount: 4},
	types.TickSize0001:  {Price: 3, Size: 2, Amount: 5},
	types.TickSize00001: {Price: 4, Size: 2, Amount: 6},
}

// OrderBuilder handles order creation and signing
// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:38-49
type OrderBuilder struct {
	signer  *signer.Signer
	sigType model.SignatureType
	funder  string
}

// NewOrderBuilder creates a new order builder
// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:39-49
func NewOrderBuilder(s *signer.Signer, sigType *model.SignatureType, funder *string) *OrderBuilder {
	// Default signature type to EOA
	// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:43
	st := model.EOA
	if sigType != nil {
		st = *sigType
	}

	// Default funder to signer address
	// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:48
	f := s.Address()
	if funder != nil {
		f = *funder
	}

	return &OrderBuilder{
		signer:  s,
		sigType: st,
		funder:  f,
	}
}

// GetOrderAmounts calculates maker and taker amounts for a regular order
// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:50-83
func (ob *OrderBuilder) GetOrderAmounts(side string, size float64, price float64, roundConfig types.RoundConfig) (model.Side, *big.Int, *big.Int, error) {
	// Round price
	// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:53
	rawPrice := utilities.RoundNormal(price, roundConfig.Price)

	if side == types.BUY {
		// BUY order logic
		// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:55-67
		rawTakerAmt := utilities.RoundDown(size, roundConfig.Size)

		rawMakerAmt := rawTakerAmt * rawPrice
		if utilities.DecimalPlaces(rawMakerAmt) > roundConfig.Amount {
			rawMakerAmt = utilities.RoundUp(rawMakerAmt, roundConfig.Amount+4)
			if utilities.DecimalPlaces(rawMakerAmt) > roundConfig.Amount {
				rawMakerAmt = utilities.RoundDown(rawMakerAmt, roundConfig.Amount)
			}
		}

		makerAmount := utilities.ToTokenDecimals(rawMakerAmt)
		takerAmount := utilities.ToTokenDecimals(rawTakerAmt)

		return model.BUY, makerAmount, takerAmount, nil
	} else if side == types.SELL {
		// SELL order logic
		// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:68-81
		rawMakerAmt := utilities.RoundDown(size, roundConfig.Size)

		rawTakerAmt := rawMakerAmt * rawPrice
		if utilities.DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
			rawTakerAmt = utilities.RoundUp(rawTakerAmt, roundConfig.Amount+4)
			if utilities.DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
				rawTakerAmt = utilities.RoundDown(rawTakerAmt, roundConfig.Amount)
			}
		}

		makerAmount := utilities.ToTokenDecimals(rawMakerAmt)
		takerAmount := utilities.ToTokenDecimals(rawTakerAmt)

		return model.SELL, makerAmount, takerAmount, nil
	}

	return 0, nil, nil, fmt.Errorf("order_args.side must be '%s' or '%s'", types.BUY, types.SELL)
}

// GetMarketOrderAmounts calculates maker and taker amounts for a market order
// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:84-116
func (ob *OrderBuilder) GetMarketOrderAmounts(side string, amount float64, price float64, roundConfig types.RoundConfig) (model.Side, *big.Int, *big.Int, error) {
	// Round price
	// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:87
	rawPrice := utilities.RoundNormal(price, roundConfig.Price)

	if side == types.BUY {
		// BUY market order logic
		// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:89-100
		rawMakerAmt := utilities.RoundDown(amount, roundConfig.Size)
		rawTakerAmt := rawMakerAmt / rawPrice
		if utilities.DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
			rawTakerAmt = utilities.RoundUp(rawTakerAmt, roundConfig.Amount+4)
			if utilities.DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
				rawTakerAmt = utilities.RoundDown(rawTakerAmt, roundConfig.Amount)
			}
		}

		makerAmount := utilities.ToTokenDecimals(rawMakerAmt)
		takerAmount := utilities.ToTokenDecimals(rawTakerAmt)

		return model.BUY, makerAmount, takerAmount, nil
	} else if side == types.SELL {
		// SELL market order logic
		// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:102-114
		rawMakerAmt := utilities.RoundDown(amount, roundConfig.Size)

		rawTakerAmt := rawMakerAmt * rawPrice
		if utilities.DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
			rawTakerAmt = utilities.RoundUp(rawTakerAmt, roundConfig.Amount+4)
			if utilities.DecimalPlaces(rawTakerAmt) > roundConfig.Amount {
				rawTakerAmt = utilities.RoundDown(rawTakerAmt, roundConfig.Amount)
			}
		}

		makerAmount := utilities.ToTokenDecimals(rawMakerAmt)
		takerAmount := utilities.ToTokenDecimals(rawTakerAmt)

		return model.SELL, makerAmount, takerAmount, nil
	}

	return 0, nil, nil, fmt.Errorf("order_args.side must be '%s' or '%s'", types.BUY, types.SELL)
}

// CreateOrder creates and signs an order
// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:118-155
func (ob *OrderBuilder) CreateOrder(orderArgs *types.OrderArgs, options *types.CreateOrderOptions) (*model.SignedOrder, error) {
	// Get order amounts
	// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:124-129
	side, makerAmount, takerAmount, err := ob.GetOrderAmounts(
		orderArgs.Side,
		orderArgs.Size,
		orderArgs.Price,
		RoundingConfig[options.TickSize],
	)
	if err != nil {
		return nil, err
	}

	// Create order data
	// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:131-143
	orderData := &model.OrderData{
		Maker:         ob.funder,
		Taker:         orderArgs.Taker,
		TokenId:       orderArgs.TokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          side,
		FeeRateBps:    fmt.Sprintf("%d", orderArgs.FeeRateBps),
		Nonce:         fmt.Sprintf("%d", orderArgs.Nonce),
		Signer:        ob.signer.Address(),
		Expiration:    fmt.Sprintf("%d", orderArgs.Expiration),
		SignatureType: ob.sigType,
	}

	// Get contract config to validate chain ID
	// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:145-147
	_, err = config.GetContractConfig(ob.signer.GetChainID(), options.NegRisk)
	if err != nil {
		return nil, err
	}

	// Build signed order using go-order-utils
	// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:149-154
	chainID := big.NewInt(int64(ob.signer.GetChainID()))
	orderBuilder := builder.NewExchangeOrderBuilderImpl(chainID, nil)

	// Convert contract address to VerifyingContract
	// Based on: go-order-utils-main/pkg/model/module.go:5-8
	var verifyingContract model.VerifyingContract
	if options.NegRisk {
		verifyingContract = model.NegRiskCTFExchange
	} else {
		verifyingContract = model.CTFExchange
	}

	signedOrder, err := orderBuilder.BuildSignedOrder(ob.signer.GetPrivateKey(), orderData, verifyingContract)
	if err != nil {
		return nil, err
	}

	return signedOrder, nil
}

// CreateMarketOrder creates and signs a market order
// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:157-194
func (ob *OrderBuilder) CreateMarketOrder(orderArgs *types.MarketOrderArgs, options *types.CreateOrderOptions) (*model.SignedOrder, error) {
	// Get market order amounts
	// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:162-168
	side, makerAmount, takerAmount, err := ob.GetMarketOrderAmounts(
		orderArgs.Side,
		orderArgs.Amount,
		orderArgs.Price,
		RoundingConfig[options.TickSize],
	)
	if err != nil {
		return nil, err
	}

	// Create order data
	// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:170-182
	orderData := &model.OrderData{
		Maker:         ob.funder,
		Taker:         orderArgs.Taker,
		TokenId:       orderArgs.TokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          side,
		FeeRateBps:    fmt.Sprintf("%d", orderArgs.FeeRateBps),
		Nonce:         fmt.Sprintf("%d", orderArgs.Nonce),
		Signer:        ob.signer.Address(),
		Expiration:    "0", // Market orders have no expiration
		SignatureType: ob.sigType,
	}

	// Get contract config to validate chain ID
	// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:184-186
	_, err = config.GetContractConfig(ob.signer.GetChainID(), options.NegRisk)
	if err != nil {
		return nil, err
	}

	// Build signed order using go-order-utils
	// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:188-193
	chainID := big.NewInt(int64(ob.signer.GetChainID()))
	orderBuilder := builder.NewExchangeOrderBuilderImpl(chainID, nil)

	// Convert contract address to VerifyingContract
	// Based on: go-order-utils-main/pkg/model/module.go:5-8
	var verifyingContract model.VerifyingContract
	if options.NegRisk {
		verifyingContract = model.NegRiskCTFExchange
	} else {
		verifyingContract = model.CTFExchange
	}

	signedOrder, err := orderBuilder.BuildSignedOrder(ob.signer.GetPrivateKey(), orderData, verifyingContract)
	if err != nil {
		return nil, err
	}

	return signedOrder, nil
}

// CalculateBuyMarketPrice calculates the matching price for a buy order
// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:196-203
func (ob *OrderBuilder) CalculateBuyMarketPrice(positions []types.OrderSummary, amountToMatch float64) (float64, error) {
	sum := 0.0
	for _, p := range positions {
		size, _ := utilities.ParseFloat(p.Size)
		price, _ := utilities.ParseFloat(p.Price)
		sum += size * price
		if sum >= amountToMatch {
			return price, nil
		}
	}
	return 0, fmt.Errorf("no match")
}

// CalculateSellMarketPrice calculates the matching price for a sell order
// Based on: py-clob-client-main/py_clob_client/order_builder/builder.py:205-214
func (ob *OrderBuilder) CalculateSellMarketPrice(positions []types.OrderSummary, amountToMatch float64) (float64, error) {
	sum := 0.0
	// Iterate in reverse order
	for i := len(positions) - 1; i >= 0; i-- {
		p := positions[i]
		size, _ := utilities.ParseFloat(p.Size)
		price, _ := utilities.ParseFloat(p.Price)
		sum += size
		if sum >= amountToMatch {
			return price, nil
		}
	}
	return 0, fmt.Errorf("no match")
}
