package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	kitemodels "github.com/zerodha/gokiteconnect/v4/models"
	kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"
)

type TickChan chan kitemodels.Tick

var (
	ticker *kiteticker.Ticker

	instrumentsIDs = []uint32{3861249, 40193, 60417, 1510401, 4267265, 81153, 136442372, 4268801,
		134657, 2714625, 140033, 177665, 5215745, 2800641, 225537, 232961, 315393, 1850625, 341249,
		119553, 345089, 348929, 356865, 340481, 1270529, 424961, 1346049, 408065, 3001089, 492033,
		2939649, 519937, 2815745, 2977281, 4598529, 633601, 3834113, 738561, 5582849, 794369, 779521,
		857857, 2953217, 878593, 884737, 895745, 3465729, 897537, 2889473, 2952193, 969473}

	instLastPriceDataChan map[uint32]TickChan
)

// Triggered when any error is raised
func onError(err error) {
	fmt.Println("Error: ", err)
}

// Triggered when websocket connection is closed
func onClose(code int, reason string) {
	fmt.Println("Close: ", code, reason)
}

// Triggered when connection is established and ready to send and accept data
func onConnect() {
	fmt.Println("Connected")
	err := ticker.Subscribe(instrumentsIDs)
	if err != nil {
		fmt.Println("err: ", err)
	}

	// Set subscription mode for given list of tokens
	// Default mode is Quote
	err = ticker.SetMode(kiteticker.ModeFull, instrumentsIDs)
	// err = ticker.SetMode(kiteticker.ModeFull, []uint32{738561})
	if err != nil {
		fmt.Println("err: ", err)
	}
}

// Triggered when tick is recevived
// Send to Instrument's Tick Channel
func onTick(tick kitemodels.Tick) {
	instLastPriceDataChan[tick.InstrumentToken] <- tick
}

// Triggered when reconnection is attempted which is enabled by default
func onReconnect(attempt int, delay time.Duration) {
	fmt.Printf("Reconnect attempt %d in %fs\n", attempt, delay.Seconds())
}

// Triggered when maximum number of reconnect attempt is made and the program is terminated
func onNoReconnect(attempt int) {
	fmt.Printf("Maximum no of reconnect attempt reached: %d", attempt)
}

// Triggered when order update is received
func onOrderUpdate(order kiteconnect.Order) {
	fmt.Printf("Order: %s", order.OrderID)
}

func cleanUp() {
	ticker.Unsubscribe(instrumentsIDs)
	os.Exit(1)
}

func main() {
	apiKey := "API Key"
	accessToken := "API Token"

	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signalChannel
		switch sig {
		case os.Interrupt:
			cleanUp()
		case syscall.SIGTERM:
			cleanUp()
		}
	}()

	// Create Per Instrument Tick Channel of 1000 Size
	instLastPriceDataChan = make(map[uint32]TickChan, 50)
	for _, instID := range instrumentsIDs {
		instLastPriceDataChan[instID] = make(TickChan, 1000)
		go processTick(instID, instLastPriceDataChan[instID])
	}

	// Create new Kite ticker instance
	ticker = kiteticker.New(apiKey, accessToken)

	// Assign callbacks
	ticker.OnError(onError)
	ticker.OnClose(onClose)
	ticker.OnConnect(onConnect)
	ticker.OnReconnect(onReconnect)
	ticker.OnNoReconnect(onNoReconnect)
	ticker.OnTick(onTick)
	ticker.OnOrderUpdate(onOrderUpdate)

	// Start the connection
	ticker.Serve()
}
