package usecase

import (
	"context"

	"control_page/internal/adaptor"
)

var _ adaptor.KlineUseCase = (*KlineUseCase)(nil)

// Popular trading pairs on Binance
var defaultSymbols = []string{
	"BTCUSDT",
	"ETHUSDT",
	"BNBUSDT",
	"SOLUSDT",
	"XRPUSDT",
	"DOGEUSDT",
	"ADAUSDT",
	"AVAXUSDT",
	"DOTUSDT",
	"LINKUSDT",
}

// Available kline intervals
var defaultIntervals = []string{
	"1m",
	"3m",
	"5m",
	"15m",
	"30m",
	"1h",
	"2h",
	"4h",
	"6h",
	"8h",
	"12h",
	"1d",
	"3d",
	"1w",
	"1M",
}

type KlineUseCase struct{}

func NewKlineUseCase() *KlineUseCase {
	return &KlineUseCase{}
}

func (uc *KlineUseCase) GetAvailableSymbols(ctx context.Context) ([]string, error) {
	symbols := make([]string, len(defaultSymbols))
	copy(symbols, defaultSymbols)
	return symbols, nil
}

func (uc *KlineUseCase) GetAvailableIntervals(ctx context.Context) ([]string, error) {
	intervals := make([]string, len(defaultIntervals))
	copy(intervals, defaultIntervals)
	return intervals, nil
}
