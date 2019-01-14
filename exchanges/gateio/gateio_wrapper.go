package gateio

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Start starts the GateIO go routine
func (g *Gateio) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		g.Run()
		wg.Done()
	}()
}

// Run implements the GateIO wrapper
func (g *Gateio) Run() {
	if g.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", g.GetName(), common.IsEnabled(g.Websocket.IsEnabled()), g.WebsocketURL)
		log.Debugf("%s polling delay: %ds.\n", g.GetName(), g.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", g.GetName(), len(g.EnabledPairs), g.EnabledPairs)
	}

	symbols, err := g.GetSymbols()
	if err != nil {
		log.Errorf("%s Unable to fetch symbols.\n", g.GetName())
	} else {
		err = g.UpdateCurrencies(symbols, false, false)
		if err != nil {
			log.Errorf("%s Failed to update available currencies.\n", g.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *Gateio) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	result, err := g.GetTickers()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range g.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(g.Name, x).String()
		var tp ticker.Price
		tp.Pair = x
		tp.High = result[currency].High
		tp.Last = result[currency].Last
		tp.Last = result[currency].Last
		tp.Low = result[currency].Low
		tp.Volume = result[currency].Volume
		ticker.ProcessTicker(g.Name, x, tp, assetType)
	}

	return ticker.GetTicker(g.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (g *Gateio) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (g *Gateio) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(g.GetName(), currency, assetType)
	if err != nil {
		return g.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gateio) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	currency := exchange.FormatExchangeCurrency(g.Name, p).String()

	orderbookNew, err := g.GetOrderbook(currency)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Amount, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Amount, Price: data.Price})
	}

	orderbook.ProcessOrderbook(g.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(g.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// ZB exchange
func (g *Gateio) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo

	balance, err := g.GetBalances()
	if err != nil {
		return info, err
	}

	if len(balance.Available) == 0 && len(balance.Locked) == 0 {
		return info, nil
	}

	var balances []exchange.AccountCurrencyInfo

	for _, data := range balance.Locked {
		for key, amountStr := range data {
			lockedF, err := strconv.ParseFloat(amountStr, 64)
			if err != nil {
				return info, err
			}

			balances = append(balances, exchange.AccountCurrencyInfo{
				CurrencyName: key,
				Hold:         lockedF,
			})
		}
	}

	for _, data := range balance.Available {
		for key, amountStr := range data {
			availAmount, err := strconv.ParseFloat(amountStr, 64)
			if err != nil {
				return info, err
			}

			var updated bool
			for i := range balances {
				if balances[i].CurrencyName == key {
					balances[i].TotalValue = balances[i].Hold + availAmount
					updated = true
					break
				}
			}

			if !updated {
				balances = append(balances, exchange.AccountCurrencyInfo{
					CurrencyName: key,
					TotalValue:   availAmount,
				})
			}
		}
	}

	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: balances,
	})

	info.Exchange = g.GetName()

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (g *Gateio) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (g *Gateio) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (g *Gateio) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var orderTypeFormat SpotNewOrderRequestParamsType

	if side == exchange.Buy {
		orderTypeFormat = SpotNewOrderRequestParamsTypeBuy
	} else {
		orderTypeFormat = SpotNewOrderRequestParamsTypeSell
	}

	var spotNewOrderRequestParams = SpotNewOrderRequestParams{
		Amount: amount,
		Price:  price,
		Symbol: p.Pair().String(),
		Type:   orderTypeFormat,
	}

	response, err := g.SpotNewOrder(spotNewOrderRequestParams)

	if response.OrderNumber > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (g *Gateio) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (g *Gateio) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}
	_, err = g.CancelExistingOrder(orderIDInt, exchange.FormatExchangeCurrency(g.Name, order.CurrencyPair).String())

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (g *Gateio) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := g.GetOpenOrders("")
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var uniqueSymbols map[string]string
	for _, openOrder := range openOrders.Orders {
		uniqueSymbols[openOrder.CurrencyPair] = openOrder.CurrencyPair
	}

	for _, uniqueSymbol := range uniqueSymbols {
		err = g.CancelAllExistingOrders(-1, uniqueSymbol)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (g *Gateio) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (g *Gateio) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	addr, err := g.GetCryptoDepositAddress(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	// Waits for new generated address if not created yet, its variable per
	// currency
	if addr == gateioGenerateAddress {
		time.Sleep(10 * time.Second)
		addr, err = g.GetCryptoDepositAddress(cryptocurrency.String())
		if addr == gateioGenerateAddress {
			return "", errors.New("address not generated in time")
		}
		return addr, nil
	}

	return addr, err
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (g *Gateio) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return g.WithdrawCrypto(withdrawRequest.Currency.String(), withdrawRequest.Address, withdrawRequest.Amount)
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gateio) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gateio) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (g *Gateio) GetWebsocket() (*exchange.Websocket, error) {
	return g.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (g *Gateio) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return g.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (g *Gateio) GetWithdrawCapabilities() uint32 {
	return g.GetWithdrawPermissions()
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (g *Gateio) GetOrderHistory(open, closed, cancelled bool, startDate, endDate string) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}
