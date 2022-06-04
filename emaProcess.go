package main

import (
	"fmt"
	"time"

	kitemodels "github.com/zerodha/gokiteconnect/v4/models"
)

/*
TickData Format
instrumentToken timeStamp lastTradeTime averageTradedPrice lastPrice netChange high low open close
738561	2022-05-30 09:17:00	2022-05-30 09:16:59	2606.32	2603.3	28.2	2615	2594.2	2615	2575.1
738561	2022-05-30 09:17:00	2022-05-30 09:16:59	2606.32	2603.5	28.4	2615	2594.2	2615	2575.1
738561	2022-05-30 09:17:01	2022-05-30 09:17:00	2606.31	2603.45	28.35	2615	2594.2	2615	2575.1
738561	2022-05-30 09:17:01	2022-05-30 09:17:00	2606.31	2603.3	28.2	2615	2594.2	2615	2575.1
738561	2022-05-30 09:17:02	2022-05-30 09:17:00	2606.31	2602.45	27.35	2615	2594.2	2615	2575.1
738561	2022-05-30 09:17:03	2022-05-30 09:17:03	2606.28	2602.5	27.4	2615	2594.2	2615	2575.1

Sometimes Timestamp is duplicate, because of continous live data keeps coming from Exchange
*/

func calSMA(priceData, resultData []float64, period float64) {
	var total float64 = 0
	for _, value := range priceData {
		total += value
	}
	for i := 0; i < int(period); i++ {
		resultData[i] = total / period
	}
}

func processTick(instID uint32, tickChan TickChan) {
	fmt.Println("Started processTick: ", instID)
	timeout := time.NewTimer(time.Minute * 2)
	var currentCandle kitemodels.OHLC
	var candleHigh, candleLow, tickPrice float64
	var tickSecondVal, previousEndMinVal, currentEndMinVal, minuteCount int
	var newMinStart bool = true

	candleData := make([]kitemodels.OHLC, 4000)
	lastPriceData := make(map[kitemodels.Time]float64, 30000)

	/* 	ema5PriceData := make([]float64, 600)
	   	ema10PriceData := make([]float64, 600)
	   	ema20PriceData := make([]float64, 600)
	   	ema50PriceData := make([]float64, 600)
	   	ema100PriceData := make([]float64, 600)
	*/

	highLowFunc := func(tickPrice float64) {
		if tickPrice > candleHigh {
			candleHigh = tickPrice
		} else if tickPrice < candleLow {
			candleLow = tickPrice
		}
	}

	for {
		select {
		case tick := <-tickChan:
			tickPrice = tick.LastPrice
			lastPriceData[tick.Timestamp] = tick.LastPrice

			/*
			 * Create OHLC
			 * If Tick Second Value is 0 (Start of Minute Candle) means OPEN
			 * Else If Tick Second Value is 59 (End of Minute Candle) means CLOSE
			 * Else Make HIGH & LOW from Tick Values
			 * Start from 0 Seconds.. In Case Application Starts between 1-59 Seconds
			 */
			tickSecondVal = tick.Timestamp.Second()
			if tickSecondVal == 0 {
				timeout.Reset(time.Minute * 2)

				if newMinStart {
					currentCandle.Open = tickPrice
					candleHigh = tickPrice
					candleLow = tickPrice
					newMinStart = false
				}
			} else if tickSecondVal == 59 && currentCandle.Open != 0 {
				currentCandle.Close = tickPrice
				currentCandle.High = candleHigh
				currentCandle.Low = candleLow

				/*
				 * Sometimes Tick Entry is like
				 * 2022-05-30 09:16:59	2022-05-30 09:16:59 2603.3
				 * 2022-05-30 09:16:59	2022-05-30 09:16:59 2603.5
				 */
				currentEndMinVal = tick.Timestamp.Minute()
				candleData[minuteCount] = currentCandle
				if previousEndMinVal != currentEndMinVal {
					minuteCount++
				}
				previousEndMinVal = tick.Timestamp.Minute()
				newMinStart = true
				fmt.Println(tick.Timestamp.Format("2006-01-02 15:04:05"), tick.InstrumentToken, currentCandle)
			}
			highLowFunc(tickPrice)

		case <-timeout.C:
			fmt.Println("No Tick Data.. Returning from routine..  Instrument Token: ", instID)
			return
		}
	}
}
