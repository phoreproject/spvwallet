package exchangerates

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/op/go-logging"
	"golang.org/x/net/proxy"
	"strings"
)

const SatoshiPerBTC = 100000000

var log = logging.MustGetLogger("exchangeRates")

type ExchangeRateProvider struct {
	fetchUrl string
	cache    map[string]float64
	client   *http.Client
	decoder  ExchangeRateDecoder
}

type ExchangeRateDecoder interface {
	decode(dat interface{}, cache map[string]float64) (err error)
}

// empty structs to tag the different ExchangeRateDecoder implementations
type CMCDecoder struct{}
type CoinGeckoDecoder struct {}
type BitcoinAverageDecoder struct{}
type BitPayDecoder struct{}
type BlockchainInfoDecoder struct{}
type BitcoinChartsDecoder struct{}

type BitcoinPriceFetcher struct {
	sync.Mutex
	cache     map[string]float64
	providers []*ExchangeRateProvider
}

func NewBitcoinPriceFetcher(dialer proxy.Dialer) *BitcoinPriceFetcher {
	b := BitcoinPriceFetcher{
		cache: make(map[string]float64),
	}
	dial := net.Dial
	if dialer != nil {
		dial = dialer.Dial
	}
	tbTransport := &http.Transport{Dial: dial}
	client := &http.Client{Transport: tbTransport, Timeout: time.Minute}

	b.providers = []*ExchangeRateProvider{
		{"https://api.coingecko.com/api/v3/coins/phore?tickers=false&community_data=false&developer_data=false&sparkline=false",b.cache, client, CoinGeckoDecoder{}},
		{"https://api.coinmarketcap.com/v2/ticker/2158/?convert=BTC", b.cache, client, CMCDecoder{}},
	}
	//b.providers = []*ExchangeRateProvider{
	//	{"https://ticker.openbazaar.org/api", b.cache, client, BitcoinAverageDecoder{}},
	//	{"https://bitpay.com/api/rates", b.cache, client, BitPayDecoder{}},
	//	{"https://blockchain.info/ticker", b.cache, client, BlockchainInfoDecoder{}},
	//	{"https://api.bitcoincharts.com/v1/weighted_prices.json", b.cache, client, BitcoinChartsDecoder{}},
	//}
	return &b
}

func (b *BitcoinPriceFetcher) GetExchangeRate(currencyCode string) (float64, error) {
	currencyCode = NormalizeCurrencyCode(currencyCode)

	b.Lock()
	defer b.Unlock()
	price, ok := b.cache[currencyCode]
	if !ok {
		return 0, errors.New("Currency not tracked")
	}
	return price, nil
}

func (b *BitcoinPriceFetcher) GetLatestRate(currencyCode string) (float64, error) {
	currencyCode = NormalizeCurrencyCode(currencyCode)

	b.fetchCurrentRates()
	b.Lock()
	defer b.Unlock()
	price, ok := b.cache[currencyCode]
	if !ok {
		return 0, errors.New("Currency not tracked")
	}
	return price, nil
}

func (b *BitcoinPriceFetcher) GetAllRates(cacheOK bool) (map[string]float64, error) {
	if !cacheOK {
		err := b.fetchCurrentRates()
		if err != nil {
			return nil, err
		}
	}
	b.Lock()
	defer b.Unlock()
	copy := make(map[string]float64, len(b.cache))
	for k, v := range b.cache {
		copy[k] = v
	}
	return copy, nil
}

func (b *BitcoinPriceFetcher) UnitsPerCoin() int {
	return SatoshiPerBTC
}

func (b *BitcoinPriceFetcher) fetchCurrentRates() error {
	b.Lock()
	defer b.Unlock()
	for _, provider := range b.providers {
		err := provider.fetch()
		if err == nil {
			return nil
		}
	}
	log.Error("Failed to fetch bitcoin exchange rates")
	return errors.New("All exchange rate API queries failed")
}

func (provider *ExchangeRateProvider) fetch() (err error) {
	if len(provider.fetchUrl) == 0 {
		err = errors.New("Provider has no fetchUrl")
		return err
	}
	resp, err := provider.client.Get(provider.fetchUrl)
	if err != nil {
		log.Error("Failed to fetch from "+provider.fetchUrl, err)
		return err
	}
	decoder := json.NewDecoder(resp.Body)
	var dataMap interface{}
	err = decoder.Decode(&dataMap)
	if err != nil {
		log.Error("Failed to decode JSON from "+provider.fetchUrl, err)
		return err
	}
	return provider.decoder.decode(dataMap, provider.cache)
}

func (b *BitcoinPriceFetcher) Run() {
	b.fetchCurrentRates()
	ticker := time.NewTicker(time.Minute * 15)
	for range ticker.C {
		b.fetchCurrentRates()
	}
}

// Decoders
func (b BitcoinAverageDecoder) decode(dat interface{}, cache map[string]float64) (err error) {
	data, ok := dat.(map[string]interface{})
	if !ok {
		return errors.New(reflect.TypeOf(b).Name() + ".decode: Type assertion failed")
	}
	for k, v := range data {
		if k != "timestamp" {
			val, ok := v.(map[string]interface{})
			if !ok {
				return errors.New(reflect.TypeOf(b).Name() + ".decode: Type assertion failed")
			}
			price, ok := val["last"].(float64)
			if !ok {
				return errors.New(reflect.TypeOf(b).Name() + ".decode: Type assertion failed, missing 'last' (float) field")
			}
			cache[k] = price
		}
	}
	return nil
}

func (b BitPayDecoder) decode(dat interface{}, cache map[string]float64) (err error) {
	data, ok := dat.([]interface{})
	if !ok {
		return errors.New(reflect.TypeOf(b).Name() + ".decode: Type assertion failed, not JSON array")
	}

	for _, obj := range data {
		code := obj.(map[string]interface{})
		k, ok := code["code"].(string)
		if !ok {
			return errors.New(reflect.TypeOf(b).Name() + ".decode: Type assertion failed, missing 'code' (string) field")
		}
		price, ok := code["rate"].(float64)
		if !ok {
			return errors.New(reflect.TypeOf(b).Name() + ".decode: Type assertion failed, missing 'rate' (float) field")
		}
		cache[k] = price
	}
	return nil
}

func (b BlockchainInfoDecoder) decode(dat interface{}, cache map[string]float64) (err error) {
	data, ok := dat.(map[string]interface{})
	if !ok {
		return errors.New(reflect.TypeOf(b).Name() + ".decode: Type assertion failed, not JSON object")
	}
	for k, v := range data {
		val, ok := v.(map[string]interface{})
		if !ok {
			return errors.New(reflect.TypeOf(b).Name() + ".decode: Type assertion failed")
		}
		price, ok := val["last"].(float64)
		if !ok {
			return errors.New(reflect.TypeOf(b).Name() + ".decode: Type assertion failed, missing 'last' (float) field")
		}
		cache[k] = price
	}
	return nil
}

func (b BitcoinChartsDecoder) decode(dat interface{}, cache map[string]float64) (err error) {
	data, ok := dat.(map[string]interface{})
	if !ok {
		return errors.New(reflect.TypeOf(b).Name() + ".decode: Type assertion failed, not JSON object")
	}
	for k, v := range data {
		if k != "timestamp" {
			val, ok := v.(map[string]interface{})
			if !ok {
				return errors.New("Type assertion failed")
			}
			p, ok := val["24h"]
			if !ok {
				continue
			}
			pr, ok := p.(string)
			if !ok {
				return errors.New("Type assertion failed")
			}
			price, err := strconv.ParseFloat(pr, 64)
			if err != nil {
				return err
			}
			cache[k] = price
		}
	}
	return nil
}

func (b CMCDecoder) decode(dat interface{}, cache map[string]float64) (err error) {
	currencyInfo, ok := dat.(map[string]interface{})
	if !ok {
		return errors.New("coinmarketcap returned malformed information")
	}

	metadata, found := currencyInfo["metadata"].(map[string]interface{})
	if !found {
		return errors.New("coinmarketcap did not return metadata")
	}

	error, found := metadata["error"].(interface{})
	if found && error != nil {
		return errors.New("coinmarketcap returned error: " + error.(string))
	}

	data, found := currencyInfo["data"].(map[string]interface{})
	if !found {
		return errors.New("coinmarketcap did not return data")
	}

	priceQuotes, found := data["quotes"].(map[string]interface{})
	if !found {
		return errors.New("coinmarketcap did not return quotes")
	}
	for currency, price := range priceQuotes {
		priceAmount, found := price.(map[string]interface{})["price"].(float64)
		if !found {
			return errors.New("coinmarketcap did not return pricedata for " + currency)
		}
		cache[currency] = priceAmount
	}

	return nil
}

func (b CoinGeckoDecoder) decode(dat interface{}, cache map[string]float64) (err error) {
	currencyInfo, ok := dat.(map[string]interface{})
	if !ok {
		return errors.New("coin gecko returned malformed information")
	}

	marketData, found := currencyInfo["market_data"].(map[string]interface{})
	if !found {
		return errors.New("coin gecko did not return market data")
	}

	currentPrice, found := marketData["current_price"].(map[string]interface{})
	if !found {
		return errors.New("coin gecko did not return current price in market data")
	}

	for currency, price := range currentPrice {
		if !found {
			return errors.New("coin gecko did not return pricedata for " + currency)
		}
		cache[strings.ToUpper(currency)] = price.(float64)
	}
	return nil
}

// NormalizeCurrencyCode standardizes the format for the given currency code
func NormalizeCurrencyCode(currencyCode string) string {
	return strings.ToUpper(currencyCode)
}
