package okcoin

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/symbol"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	okcoinAPIURL                = "https://www.okcoin.com/api/v1/"
	okcoinAPIURLChina           = "https://www.okcoin.com/api/v1/"
	okcoinAPIURLBase            = "https://www.okcoin.com/api/"
	okcoinAPIVersion            = "1"
	okcoinWebsocketURL          = "wss://real.okcoin.com:10440/websocket/okcoinapi"
	okcoinWebsocketURLChina     = "wss://real.okcoin.cn:10440/websocket/okcoinapi"
	okcoinInstruments           = "instruments"
	okcoinTicker                = "ticker.do"
	okcoinDepth                 = "depth.do"
	okcoinTrades                = "trades.do"
	okcoinKline                 = "kline.do"
	okcoinUserInfo              = "userinfo.do"
	okcoinTrade                 = "trade.do"
	okcoinTradeHistory          = "trade_history.do"
	okcoinTradeBatch            = "batch_trade.do"
	okcoinOrderCancel           = "cancel_order.do"
	okcoinOrderInfo             = "order_info.do"
	okcoinOrdersInfo            = "orders_info.do"
	okcoinOrderHistory          = "order_history.do"
	okcoinWithdraw              = "withdraw.do"
	okcoinWithdrawCancel        = "cancel_withdraw.do"
	okcoinWithdrawInfo          = "withdraw_info.do"
	okcoinOrderFee              = "order_fee.do"
	okcoinLendDepth             = "lend_depth.do"
	okcoinBorrowsInfo           = "borrows_info.do"
	okcoinBorrowMoney           = "borrow_money.do"
	okcoinBorrowCancel          = "cancel_borrow.do"
	okcoinBorrowOrderInfo       = "borrow_order_info.do"
	okcoinRepayment             = "repayment.do"
	okcoinUnrepaymentsInfo      = "unrepayments_info.do"
	okcoinAccountRecords        = "account_records.do"
	okcoinFuturesTicker         = "future_ticker.do"
	okcoinFuturesDepth          = "future_depth.do"
	okcoinFuturesTrades         = "future_trades.do"
	okcoinFuturesIndex          = "future_index.do"
	okcoinExchangeRate          = "exchange_rate.do"
	okcoinFuturesEstimatedPrice = "future_estimated_price.do"
	okcoinFuturesKline          = "future_kline.do"
	okcoinFuturesHoldAmount     = "future_hold_amount.do"
	okcoinFuturesUserInfo       = "future_userinfo.do"
	okcoinFuturesPosition       = "future_position.do"
	okcoinFuturesTrade          = "future_trade.do"
	okcoinFuturesTradeHistory   = "future_trades_history.do"
	okcoinFuturesTradeBatch     = "future_batch_trade.do"
	okcoinFuturesCancel         = "future_cancel.do"
	okcoinFuturesOrderInfo      = "future_order_info.do"
	okcoinFuturesOrdersInfo     = "future_orders_info.do"
	okcoinFuturesUserInfo4Fix   = "future_userinfo_4fix.do"
	okcoinFuturesposition4Fix   = "future_position_4fix.do"
	okcoinFuturesExplosive      = "future_explosive.do"
	okcoinFuturesDevolve        = "future_devolve.do"

	okcoinAuthRate   = 0
	okcoinUnauthRate = 0
)

// OKCoin is the overarching type across this package
type OKCoin struct {
	exchange.Base
	RESTErrors      map[string]string
	WebsocketErrors map[string]string
	FuturesValues   []string
	WebsocketConn   *websocket.Conn
}

// setCurrencyPairFormats sets currency pair formatting for this package
func (o *OKCoin) setCurrencyPairFormats() {
	o.RequestCurrencyPairFormat.Delimiter = "_"
	o.RequestCurrencyPairFormat.Uppercase = false
	o.ConfigCurrencyPairFormat.Delimiter = ""
	o.ConfigCurrencyPairFormat.Uppercase = true
}

// SetDefaults sets current default values for this package
func (o *OKCoin) SetDefaults() {
	o.SetErrorDefaults()
	o.SetWebsocketErrorDefaults()
	o.Enabled = false
	o.Verbose = false
	o.RESTPollingDelay = 10
	o.AssetTypes = []string{ticker.Spot}
	o.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.WithdrawFiatViaWebsiteOnly
	o.SupportsAutoPairUpdating = false
	o.SupportsRESTTickerBatching = false
	o.WebsocketInit()
	o.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketOrderbookSupported |
		exchange.WebsocketKlineSupported
}

// Setup sets exchange configuration parameters
func (o *OKCoin) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		o.SetEnabled(false)
	} else {
		if exch.Name == "OKCOIN International" {
			o.AssetTypes = append(o.AssetTypes, o.FuturesValues...)
			o.APIUrlDefault = okcoinAPIURL
			o.APIUrl = o.APIUrlDefault
			o.Name = "OKCOIN International"
			o.WebsocketURL = okcoinWebsocketURL
			o.setCurrencyPairFormats()
			o.Requester = request.New(o.Name,
				request.NewRateLimit(time.Second, okcoinAuthRate),
				request.NewRateLimit(time.Second, okcoinUnauthRate),
				common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
			o.ConfigCurrencyPairFormat.Delimiter = "_"
			o.ConfigCurrencyPairFormat.Uppercase = true
			o.RequestCurrencyPairFormat.Uppercase = false
			o.RequestCurrencyPairFormat.Delimiter = "_"
			o.SupportsAutoPairUpdating = true
		} else {
			o.APIUrlDefault = okcoinAPIURLChina
			o.APIUrl = o.APIUrlDefault
			o.Name = "OKCOIN China"
			o.WebsocketURL = okcoinWebsocketURLChina
			o.setCurrencyPairFormats()
			o.Requester = request.New(o.Name,
				request.NewRateLimit(time.Second, okcoinAuthRate),
				request.NewRateLimit(time.Second, okcoinUnauthRate),
				common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
			o.ConfigCurrencyPairFormat.Delimiter = ""
			o.ConfigCurrencyPairFormat.Uppercase = true
			o.RequestCurrencyPairFormat.Uppercase = false
			o.RequestCurrencyPairFormat.Delimiter = ""
		}

		o.Enabled = true
		o.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		o.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		o.SetHTTPClientTimeout(exch.HTTPTimeout)
		o.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		o.RESTPollingDelay = exch.RESTPollingDelay
		o.Verbose = exch.Verbose
		o.Websocket.SetEnabled(exch.Websocket)
		o.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		o.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		o.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := o.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = o.WebsocketSetup(o.WsConnect,
			exch.Name,
			exch.Websocket,
			okcoinWebsocketURL,
			o.WebsocketURL)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetSpotInstruments returns a list of tradable spot instruments and their properties
func (o *OKCoin) GetSpotInstruments() ([]SpotInstrument, error) {
	var resp []SpotInstrument

	path := fmt.Sprintf("%sspot/v3/%s", okcoinAPIURLBase, okcoinInstruments)
	err := o.SendHTTPRequest(path, &resp)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetTicker returns the current ticker
func (o *OKCoin) GetTicker(symbol string) (Ticker, error) {
	resp := TickerResponse{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	path := common.EncodeURLValues(o.APIUrl+okcoinTicker, vals)

	return resp.Ticker, o.SendHTTPRequest(path, &resp)
}

// GetOrderBook returns the current order book by size
func (o *OKCoin) GetOrderBook(symbol string, size int64, merge bool) (Orderbook, error) {
	resp := Orderbook{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	if size != 0 {
		vals.Set("size", strconv.FormatInt(size, 10))
	}
	if merge {
		vals.Set("merge", "1")
	}

	path := common.EncodeURLValues(o.APIUrl+okcoinDepth, vals)
	return resp, o.SendHTTPRequest(path, &resp)
}

// GetTrades returns historic trades since a timestamp
func (o *OKCoin) GetTrades(symbol string, since int64) ([]Trades, error) {
	result := []Trades{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	if since != 0 {
		vals.Set("since", strconv.FormatInt(since, 10))
	}

	path := common.EncodeURLValues(o.APIUrl+okcoinTrades, vals)
	return result, o.SendHTTPRequest(path, &result)
}

// GetKline returns kline data
func (o *OKCoin) GetKline(symbol, klineType string, size, since int64) ([]interface{}, error) {
	resp := []interface{}{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("type", klineType)

	if size != 0 {
		vals.Set("size", strconv.FormatInt(size, 10))
	}

	if since != 0 {
		vals.Set("since", strconv.FormatInt(since, 10))
	}

	path := common.EncodeURLValues(o.APIUrl+okcoinKline, vals)
	return resp, o.SendHTTPRequest(path, &resp)
}

// GetFuturesTicker returns a current ticker for the futures market
func (o *OKCoin) GetFuturesTicker(symbol, contractType string) (FuturesTicker, error) {
	resp := FuturesTickerResponse{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)
	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesTicker, vals)

	return resp.Ticker, o.SendHTTPRequest(path, &resp)
}

// GetFuturesDepth returns current depth for the futures market
func (o *OKCoin) GetFuturesDepth(symbol, contractType string, size int64, merge bool) (Orderbook, error) {
	result := Orderbook{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)

	if size != 0 {
		vals.Set("size", strconv.FormatInt(size, 10))
	}
	if merge {
		vals.Set("merge", "1")
	}

	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesDepth, vals)
	return result, o.SendHTTPRequest(path, &result)
}

// GetFuturesTrades returns historic trades for the futures market
func (o *OKCoin) GetFuturesTrades(symbol, contractType string) ([]FuturesTrades, error) {
	result := []FuturesTrades{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)

	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesTrades, vals)
	return result, o.SendHTTPRequest(path, &result)
}

// GetFuturesIndex returns an index for the futures market
func (o *OKCoin) GetFuturesIndex(symbol string) (float64, error) {
	type Response struct {
		Index float64 `json:"future_index"`
	}

	result := Response{}
	vals := url.Values{}
	vals.Set("symbol", symbol)

	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesIndex, vals)
	return result.Index, o.SendHTTPRequest(path, &result)
}

// GetFuturesExchangeRate returns the exchange rate for the futures market
func (o *OKCoin) GetFuturesExchangeRate() (float64, error) {
	type Response struct {
		Rate float64 `json:"rate"`
	}

	result := Response{}
	return result.Rate, o.SendHTTPRequest(o.APIUrl+okcoinExchangeRate, &result)
}

// GetFuturesEstimatedPrice returns a current estimated futures price for a
// currency
func (o *OKCoin) GetFuturesEstimatedPrice(symbol string) (float64, error) {
	type Response struct {
		Price float64 `json:"forecast_price"`
	}

	result := Response{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesEstimatedPrice, vals)

	return result.Price, o.SendHTTPRequest(path, &result)
}

// GetFuturesKline returns kline data for a specific currency on the futures
// market
func (o *OKCoin) GetFuturesKline(symbol, klineType, contractType string, size, since int64) ([]interface{}, error) {
	resp := []interface{}{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("type", klineType)
	vals.Set("contract_type", contractType)

	if size != 0 {
		vals.Set("size", strconv.FormatInt(size, 10))
	}
	if since != 0 {
		vals.Set("since", strconv.FormatInt(since, 10))
	}

	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesKline, vals)
	return resp, o.SendHTTPRequest(path, &resp)
}

// GetFuturesHoldAmount returns the hold amount for a futures trade
func (o *OKCoin) GetFuturesHoldAmount(symbol, contractType string) ([]FuturesHoldAmount, error) {
	resp := []FuturesHoldAmount{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)

	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesHoldAmount, vals)
	return resp, o.SendHTTPRequest(path, &resp)
}

// GetFuturesExplosive returns the explosive for a futures contract
func (o *OKCoin) GetFuturesExplosive(symbol, contractType string, status, currentPage, pageLength int64) ([]FuturesExplosive, error) {
	type Response struct {
		Data []FuturesExplosive `json:"data"`
	}
	resp := Response{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)
	vals.Set("status", strconv.FormatInt(status, 10))
	vals.Set("current_page", strconv.FormatInt(currentPage, 10))
	vals.Set("page_length", strconv.FormatInt(pageLength, 10))

	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesExplosive, vals)

	return resp.Data, o.SendHTTPRequest(path, &resp)
}

// GetUserInfo returns user information associated with the calling APIkeys
func (o *OKCoin) GetUserInfo() (UserInfo, error) {
	result := UserInfo{}

	return result,
		o.SendAuthenticatedHTTPRequest(okcoinUserInfo, url.Values{}, &result)
}

// Trade initiates a new trade
func (o *OKCoin) Trade(amount, price float64, symbol, orderType string) (int64, error) {
	type Response struct {
		Result  bool  `json:"result"`
		OrderID int64 `json:"order_id"`
	}
	v := url.Values{}
	v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	v.Set("symbol", symbol)
	v.Set("type", orderType)

	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinTrade, v, &result)

	if err != nil {
		return 0, err
	}

	if !result.Result {
		return 0, errors.New("unable to place order")
	}

	return result.OrderID, nil
}

// GetTradeHistory returns client trade history
func (o *OKCoin) GetTradeHistory(symbol string, TradeID int64) ([]Trades, error) {
	result := []Trades{}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("since", strconv.FormatInt(TradeID, 10))

	err := o.SendAuthenticatedHTTPRequest(okcoinTradeHistory, v, &result)

	if err != nil {
		return nil, err
	}

	return result, nil
}

// BatchTrade initiates a trade by batch order
func (o *OKCoin) BatchTrade(orderData string, symbol, orderType string) (BatchTrade, error) {
	v := url.Values{}
	v.Set("orders_data", orderData)
	v.Set("symbol", symbol)
	v.Set("type", orderType)

	result := BatchTrade{}
	return result, o.SendAuthenticatedHTTPRequest(okcoinTradeBatch, v, &result)
}

// CancelExistingOrder cancels a specific order or list of orders by orderID
func (o *OKCoin) CancelExistingOrder(orderID []int64, symbol string) (CancelOrderResponse, error) {
	v := url.Values{}
	orders := []string{}
	result := CancelOrderResponse{}

	orderStr := strconv.FormatInt(orderID[0], 10)

	if len(orderID) > 1 {
		for x := range orderID {
			orders = append(orders, strconv.FormatInt(orderID[x], 10))
		}
		orderStr = common.JoinStrings(orders, ",")
	}

	v.Set("order_id", orderStr)
	v.Set("symbol", symbol)

	return result, o.SendAuthenticatedHTTPRequest(okcoinOrderCancel, v, &result)
}

// GetOrderInformation returns order information by orderID
func (o *OKCoin) GetOrderInformation(orderID int64, symbol string) ([]OrderInfo, error) {
	type Response struct {
		Result bool        `json:"result"`
		Orders []OrderInfo `json:"orders"`
	}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinOrderInfo, v, &result)

	if err != nil {
		return nil, err
	}

	if !result.Result {
		return nil, errors.New("unable to retrieve order info")
	}

	return result.Orders, nil
}

// GetOrderInfoBatch returns order info on a batch of orders
func (o *OKCoin) GetOrderInfoBatch(orderID []int64, symbol string) ([]OrderInfo, error) {
	type Response struct {
		Result bool        `json:"result"`
		Orders []OrderInfo `json:"orders"`
	}

	orders := []string{}
	for x := range orderID {
		orders = append(orders, strconv.FormatInt(orderID[x], 10))
	}

	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("order_id", common.JoinStrings(orders, ","))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinOrderInfo, v, &result)

	if err != nil {
		return nil, err
	}

	if !result.Result {
		return nil, errors.New("unable to retrieve order info")
	}

	return result.Orders, nil
}

// GetOrderHistory returns a history of orders
func (o *OKCoin) GetOrderHistory(pageLength, currentPage int64, status, symbol string) (OrderHistory, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("status", status)
	v.Set("current_page", strconv.FormatInt(currentPage, 10))
	v.Set("page_length", strconv.FormatInt(pageLength, 10))

	result := OrderHistory{}
	return result, o.SendAuthenticatedHTTPRequest(okcoinOrderHistory, v, &result)
}

// Withdrawal withdraws a cryptocurrency to a supplied address
func (o *OKCoin) Withdrawal(symbol string, fee float64, tradePWD, address string, amount float64) (int, error) {
	v := url.Values{}
	v.Set("symbol", symbol)

	if fee != 0 {
		v.Set("chargefee", strconv.FormatFloat(fee, 'f', -1, 64))
	}
	v.Set("trade_pwd", tradePWD)
	v.Set("withdraw_address", address)
	v.Set("withdraw_amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("target", "address")
	result := WithdrawalResponse{}

	err := o.SendAuthenticatedHTTPRequest(okcoinWithdraw, v, &result)
	if err != nil {
		return 0, err
	}

	if !result.Result {
		return 0, errors.New("unable to process withdrawal request")
	}

	return result.WithdrawID, nil
}

// CancelWithdrawal cancels a withdrawal
func (o *OKCoin) CancelWithdrawal(symbol string, withdrawalID int64) (int, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("withdrawal_id", strconv.FormatInt(withdrawalID, 10))
	result := WithdrawalResponse{}

	err := o.SendAuthenticatedHTTPRequest(okcoinWithdrawCancel, v, &result)

	if err != nil {
		return 0, err
	}

	if !result.Result {
		return 0, errors.New("unable to process withdrawal cancel request")
	}

	return result.WithdrawID, nil
}

// GetWithdrawalInfo returns withdrawal information
func (o *OKCoin) GetWithdrawalInfo(symbol string, withdrawalID int64) ([]WithdrawInfo, error) {
	type Response struct {
		Result   bool
		Withdraw []WithdrawInfo `json:"withdraw"`
	}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("withdrawal_id", strconv.FormatInt(withdrawalID, 10))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinWithdrawInfo, v, &result)

	if err != nil {
		return nil, err
	}

	if !result.Result {
		return nil, errors.New("unable to process withdrawal cancel request")
	}

	return result.Withdraw, nil
}

// GetOrderFeeInfo returns order fee information
func (o *OKCoin) GetOrderFeeInfo(symbol string, orderID int64) (OrderFeeInfo, error) {
	type Response struct {
		Data   OrderFeeInfo `json:"data"`
		Result bool         `json:"result"`
	}

	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinOrderFee, v, &result)

	if err != nil {
		return result.Data, err
	}

	if !result.Result {
		return result.Data, errors.New("unable to get order fee info")
	}

	return result.Data, nil
}

// GetLendDepth returns the depth of lends
func (o *OKCoin) GetLendDepth(symbol string) ([]LendDepth, error) {
	type Response struct {
		LendDepth []LendDepth `json:"lend_depth"`
	}

	v := url.Values{}
	v.Set("symbol", symbol)
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinLendDepth, v, &result)

	if err != nil {
		return nil, err
	}

	return result.LendDepth, nil
}

// GetBorrowInfo returns borrow information
func (o *OKCoin) GetBorrowInfo(symbol string) (BorrowInfo, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	result := BorrowInfo{}

	err := o.SendAuthenticatedHTTPRequest(okcoinBorrowsInfo, v, &result)

	if err != nil {
		return result, nil
	}

	return result, nil
}

// Borrow initiates a borrow request
func (o *OKCoin) Borrow(symbol, days string, amount, rate float64) (int, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("days", days)
	v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	result := BorrowResponse{}

	err := o.SendAuthenticatedHTTPRequest(okcoinBorrowMoney, v, &result)

	if err != nil {
		return 0, err
	}

	if !result.Result {
		return 0, errors.New("unable to borrow")
	}

	return result.BorrowID, nil
}

// CancelBorrow cancels a borrow request
func (o *OKCoin) CancelBorrow(symbol string, borrowID int64) (bool, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("borrow_id", strconv.FormatInt(borrowID, 10))
	result := BorrowResponse{}

	err := o.SendAuthenticatedHTTPRequest(okcoinBorrowCancel, v, &result)

	if err != nil {
		return false, err
	}

	if !result.Result {
		return false, errors.New("unable to cancel borrow")
	}

	return true, nil
}

// GetBorrowOrderInfo returns information about a borrow order
func (o *OKCoin) GetBorrowOrderInfo(borrowID int64) (BorrowInfo, error) {
	type Response struct {
		Result      bool       `json:"result"`
		BorrowOrder BorrowInfo `json:"borrow_order"`
	}

	v := url.Values{}
	v.Set("borrow_id", strconv.FormatInt(borrowID, 10))
	result := Response{}
	err := o.SendAuthenticatedHTTPRequest(okcoinBorrowOrderInfo, v, &result)

	if err != nil {
		return result.BorrowOrder, err
	}

	if !result.Result {
		return result.BorrowOrder, errors.New("unable to get borrow info")
	}

	return result.BorrowOrder, nil
}

// GetRepaymentInfo returns information on a repayment
func (o *OKCoin) GetRepaymentInfo(borrowID int64) (bool, error) {
	v := url.Values{}
	v.Set("borrow_id", strconv.FormatInt(borrowID, 10))
	result := BorrowResponse{}

	err := o.SendAuthenticatedHTTPRequest(okcoinRepayment, v, &result)

	if err != nil {
		return false, err
	}

	if !result.Result {
		return false, errors.New("unable to get repayment info")
	}

	return true, nil
}

// GetUnrepaymentsInfo returns information on an unrepayment
func (o *OKCoin) GetUnrepaymentsInfo(symbol string, currentPage, pageLength int) ([]BorrowOrder, error) {
	type Response struct {
		Unrepayments []BorrowOrder `json:"unrepayments"`
		Result       bool          `json:"result"`
	}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("current_page", strconv.Itoa(currentPage))
	v.Set("page_length", strconv.Itoa(pageLength))
	result := Response{}
	err := o.SendAuthenticatedHTTPRequest(okcoinUnrepaymentsInfo, v, &result)

	if err != nil {
		return nil, err
	}

	if !result.Result {
		return nil, errors.New("unable to get unrepayments info")
	}

	return result.Unrepayments, nil
}

// GetAccountRecords returns account records
func (o *OKCoin) GetAccountRecords(symbol string, recType, currentPage, pageLength int) ([]AccountRecords, error) {
	type Response struct {
		Records []AccountRecords `json:"records"`
		Symbol  string           `json:"symbol"`
	}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("type", strconv.Itoa(recType))
	v.Set("current_page", strconv.Itoa(currentPage))
	v.Set("page_length", strconv.Itoa(pageLength))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinAccountRecords, v, &result)

	if err != nil {
		return nil, err
	}

	return result.Records, nil
}

// GetFuturesUserInfo returns information on a users futures
func (o *OKCoin) GetFuturesUserInfo() {
	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesUserInfo, url.Values{}, nil)

	if err != nil {
		log.Error(err)
	}
}

// GetFuturesPosition returns position on a futures contract
func (o *OKCoin) GetFuturesPosition(symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesPosition, v, nil)

	if err != nil {
		log.Error(err)
	}
}

// FuturesTrade initiates a new futures trade
func (o *OKCoin) FuturesTrade(amount, price float64, matchPrice, leverage int64, symbol, contractType, orderType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("type", orderType)
	v.Set("match_price", strconv.FormatInt(matchPrice, 10))
	v.Set("lever_rate", strconv.FormatInt(leverage, 10))

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesTrade, v, nil)

	if err != nil {
		log.Error(err)
	}
}

// FuturesBatchTrade initiates a batch of futures contract trades
func (o *OKCoin) FuturesBatchTrade(orderData, symbol, contractType string, leverage int64, orderType string) {
	v := url.Values{} //to-do batch trade support for orders_data)
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("orders_data", orderData)
	v.Set("lever_rate", strconv.FormatInt(leverage, 10))

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesTradeBatch, v, nil)

	if err != nil {
		log.Error(err)
	}
}

// CancelFuturesOrder cancels a futures contract order
func (o *OKCoin) CancelFuturesOrder(orderID int64, symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("order_id", strconv.FormatInt(orderID, 10))

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesCancel, v, nil)

	if err != nil {
		log.Error(err)
	}
}

// GetFuturesOrderInfo returns information on a specific futures contract order
func (o *OKCoin) GetFuturesOrderInfo(orderID, status, currentPage, pageLength int64, symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("status", strconv.FormatInt(status, 10))
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	v.Set("current_page", strconv.FormatInt(currentPage, 10))
	v.Set("page_length", strconv.FormatInt(pageLength, 10))

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesOrderInfo, v, nil)

	if err != nil {
		log.Error(err)
	}
}

// GetFutureOrdersInfo returns information on a range of futures orders
func (o *OKCoin) GetFutureOrdersInfo(orderID int64, contractType, symbol string) {
	v := url.Values{}
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	v.Set("contract_type", contractType)
	v.Set("symbol", symbol)

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesOrdersInfo, v, nil)

	if err != nil {
		log.Error(err)
	}
}

// GetFuturesUserInfo4Fix returns futures user info fix rate
func (o *OKCoin) GetFuturesUserInfo4Fix() {
	v := url.Values{}

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesUserInfo4Fix, v, nil)

	if err != nil {
		log.Error(err)
	}
}

// GetFuturesUserPosition4Fix returns futures user info on a fixed position
func (o *OKCoin) GetFuturesUserPosition4Fix(symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("type", strconv.FormatInt(1, 10))

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesUserInfo4Fix, v, nil)

	if err != nil {
		log.Error(err)
	}
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (o *OKCoin) SendHTTPRequest(path string, result interface{}) error {
	return o.SendPayload("GET", path, nil, nil, result, false, o.Verbose)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (o *OKCoin) SendAuthenticatedHTTPRequest(method string, v url.Values, result interface{}) (err error) {
	if !o.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, o.Name)
	}

	v.Set("api_key", o.APIKey)
	hasher := common.GetMD5([]byte(v.Encode() + "&secret_key=" + o.APISecret))
	v.Set("sign", strings.ToUpper(common.HexEncodeToString(hasher)))

	encoded := v.Encode()
	path := o.APIUrl + method

	if o.Verbose {
		log.Debugf("Sending POST request to %s with params %s\n", path, encoded)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	return o.SendPayload("POST", path, headers, strings.NewReader(encoded), result, true, o.Verbose)
}

// SetErrorDefaults sets default error map
func (o *OKCoin) SetErrorDefaults() {
	o.RESTErrors = map[string]string{
		"10000": "Required field, can not be null",
		"10001": "Request frequency too high",
		"10002": "System error",
		"10003": "Not in reqest list, please try again later",
		"10004": "IP not allowed to access the resource",
		"10005": "'secretKey' does not exist",
		"10006": "'partner' does not exist",
		"10007": "Signature does not match",
		"10008": "Illegal parameter",
		"10009": "Order does not exist",
		"10010": "Insufficient funds",
		"10011": "Amount too low",
		"10012": "Only btc_usd/btc_cny ltc_usd,ltc_cny supported",
		"10013": "Only support https request",
		"10014": "Order price must be between 0 and 1,000,000",
		"10015": "Order price differs from current market price too much",
		"10016": "Insufficient coins balance",
		"10017": "API authorization error",
		"10018": "Borrow amount less than lower limit [usd/cny:100,btc:0.1,ltc:1]",
		"10019": "Loan agreement not checked",
		"10020": `Rate cannot exceed 1%`,
		"10021": `Rate cannot less than 0.01%`,
		"10023": "Fail to get latest ticker",
		"10024": "Balance not sufficient",
		"10025": "Quota is full, cannot borrow temporarily",
		"10026": "Loan (including reserved loan) and margin cannot be withdrawn",
		"10027": "Cannot withdraw within 24 hrs of authentication information modification",
		"10028": "Withdrawal amount exceeds daily limit",
		"10029": "Account has unpaid loan, please cancel/pay off the loan before withdraw",
		"10031": "Deposits can only be withdrawn after 6 confirmations",
		"10032": "Please enabled phone/google authenticator",
		"10033": "Fee higher than maximum network transaction fee",
		"10034": "Fee lower than minimum network transaction fee",
		"10035": "Insufficient BTC/LTC",
		"10036": "Withdrawal amount too low",
		"10037": "Trade password not set",
		"10040": "Withdrawal cancellation fails",
		"10041": "Withdrawal address not approved",
		"10042": "Admin password error",
		"10043": "Account equity error, withdrawal failure",
		"10044": "fail to cancel borrowing order",
		"10047": "This function is disabled for sub-account",
		"10100": "User account frozen",
		"10216": "Non-available API",
		"20001": "User does not exist",
		"20002": "Account frozen",
		"20003": "Account frozen due to liquidation",
		"20004": "Futures account frozen",
		"20005": "User futures account does not exist",
		"20006": "Required field missing",
		"20007": "Illegal parameter",
		"20008": "Futures account balance is too low",
		"20009": "Future contract status error",
		"20010": "Risk rate ratio does not exist",
		"20011": `Risk rate higher than 90% before opening position`,
		"20012": `Risk rate higher than 90% after opening position`,
		"20013": "Temporally no counter party price",
		"20014": "System error",
		"20015": "Order does not exist",
		"20016": "Close amount bigger than your open positions",
		"20017": "Not authorized/illegal operation",
		"20018": `Order price differ more than 5% from the price in the last minute`,
		"20019": "IP restricted from accessing the resource",
		"20020": "secretKey does not exist",
		"20021": "Index information does not exist",
		"20022": "Wrong API interface (Cross margin mode shall call cross margin API, fixed margin mode shall call fixed margin API)",
		"20023": "Account in fixed-margin mode",
		"20024": "Signature does not match",
		"20025": "Leverage rate error",
		"20026": "API Permission Error",
		"20027": "No transaction record",
		"20028": "No such contract",
	}
}

// GetFee returns an estimate of fee based on type of transaction
func (o *OKCoin) GetFee(feeBuilder exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
	case exchange.InternationalBankWithdrawalFee:
		fee = calculateInternationalBankWithdrawalFee(feeBuilder.CurrencyItem, feeBuilder.PurchasePrice, feeBuilder.Amount)
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.FirstCurrency)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

func calculateTradingFee(purchasePrice, amount float64, isMaker bool) (fee float64) {
	// TODO volume based fees
	if isMaker {
		fee = 0.0005
	} else {
		fee = 0.0015
	}
	return fee * amount * purchasePrice
}

func calculateInternationalBankWithdrawalFee(currency string, purchasePrice, amount float64) (fee float64) {
	if currency == symbol.USD {
		if purchasePrice*amount*0.001 < 15 {
			fee = 15
		} else {
			fee = purchasePrice * amount * 0.001
		}
	}
	return fee
}

func getWithdrawalFee(currency string) float64 {
	return WithdrawalFees[currency]
}
