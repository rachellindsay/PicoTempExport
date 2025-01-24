package main

import (
	"bufio"
	"encoding/json"
	"io"
	"machine"
	"net/netip"
	"time"

	"log/slog"

	"github.com/soypat/cyw43439"
	"github.com/soypat/cyw43439/examples/common"

	"github.com/soypat/seqs/httpx"
	"github.com/soypat/seqs/stacks"
)

// constants that will be used to initialize the TCP stack
const (
	connTimeout = 3 * time.Second
	maxconns		= 3
	tcpbufsize	=2030
	hostname		= "picotemp"
	listenPort 	= 80
)

// new struct type that will convert temperature from temp sensor to JSON
type temp struct {
	TempC float64 `json:"tempC"`
	TempF float64 `json:"tempF"`
}

// define package scoped variable of type pointer to slog.Logger that will serve as the default structured logger for the entire application
var logger *slog.Logger

// initialize default logger with Go init function, redirecting the output to the PicoW serial interface using machine.Serial type as first parameter - this will allow monitoring using 'tinygo monitor' command
func init() {
	logger = slog.New(
		slog.NewTextHandler(machine.Serial, &slog.HandlerOptions{
			// changed to LevelDebug from LevelInfo in order to get more information as it is not connecting
			Level: slog.LevelInfo,
		}))
}

// function to change PicoW LED state to show that device is active and listening for connections
func changeLEDState(dev *cyw43439.Device, state bool) {
	if err := dev.GPIOSet(0, state); err != nil {
		logger.Error("failed to change LED state:",
			slog.String("err", err.Error()))
	}
}

//  use cyW43439 driver and the example code in examples/common from cyw43439 repo to define function that sets up the PicoW wifi
func setupDevice() (*stacks.PortStack, *cyw43439.Device) {
	logger.Info("Rae setupDevice here")
	_, stack, dev, err := common.SetupWithDHCP(common.SetupConfig{
		Hostname: hostname,
		Logger:		logger,
		TCPPorts: 1,
	})
	logger.Info("Rae setupDevice after SetupWithDHCP")
	
	if err != nil {
		panic("setup DHCP:" + err.Error())
	}
	// Turn LED on
	changeLEDState(dev, true)

	return stack, dev
}

// use the seqs/stack package to define function that listens on port 80 (as defined in const section near top) for incoming TCP connections 
func newListener(stack *stacks.PortStack) *stacks.TCPListener {
	// start TCP server
	logger.Info("Rae newListener here")
	listenAddr := netip.AddrPortFrom(stack.Addr(), listenPort)
	listener, err := stacks.NewTCPListener(
		stack, stacks.TCPListenerConfig{
			MaxConnections:	maxconns,
			ConnTxBufSize:	tcpbufsize,
			ConnRxBufSize:	tcpbufsize,
		})
	if err != nil {
		panic("listener create:" + err.Error())
	}
	err = listener.StartListening(listenPort)
	if err != nil {
		panic("listener start:" + err.Error())
	}
	logger.Info("listening",
		slog.String("addr", "http://"+listenAddr.String()),
	)
	return listener
}

// function to blink the LED in order to provide visual feedback when the PicoW server receives a connection request. it runs concurrently (using a Go channel) with other functions to avoid blocking the program
func blinkLED(dev *cyw43439.Device, blink chan uint) {
	logger.Info("Rae, blinkLED entered")
	for {
		select {
		case n := <-blink:
			logger.Info("Rae blinkLED n from channel:", n)
			lastLEDState := true
			if n == 0 {
				n = 5
			}
			for i := uint(0); i < n; i++ {
				lastLEDState = !lastLEDState
				changeLEDState(dev, lastLEDState)
				time.Sleep(500 * time.Millisecond)
			}
			// ensure LED is on at the end
			changeLEDState(dev, true)
		}
	}
} 

// Function to obtain the temp from the sensor and output values in both Celcius and Fahrenheit using the previously defined temp custom type
func getTemperature() *temp {
	curTemp := machine.ReadTemperature()

	return & temp{
		TempC: float64(curTemp) / 1000,
		TempF: ((float64(curTemp) / 1000) * 9 / 5) + 32,
	}	
}

// Define HTTPHandler function to handle http request for temperature. This function uses the seqs/httpx library to define HTTP headers for the response. In the function body, get the temp from the sensor (getTemperature) and convert to JSON using the standard library encoding/json package. if there is an error, return 500. If successul, return the JSON containing the temperature.
func HTTPHandler(respWriter io.Writer, resp *httpx.ResponseHeader) {
	resp.SetConnectionClose()
	logger.Info("Got temperature request...")
	t := getTemperature()

	body, err := json.Marshal(t)
	if err != nil {
		logger.Error(
			"temperature json:",
			slog.String("err", err.Error()),
		)
		resp.SetStatusCode(500)
	} else {
		resp.SetContentType("application/json")
		resp.SetContentLength(len(body))
	}
	respWriter.Write(resp.Header())
	respWriter.Write(body)
}

// Function that handles HTTP connections and responds with temperature JSON. Define some buffer that can be reused for all connections to avoid memory allocations (because it's a small device). This function also takes a channel as an input. This channel notifies the blinkLED goroutine to blink the LED, showing that it is processing a request.
func handleConnection(listener *stacks.TCPListener, blink chan uint) {
	// Reuse the same buffers for each connection to avoid heap allocations
	var resp httpx.ResponseHeader
	logger.Info("Rae handleConnection start")
	buf := bufio.NewReaderSize(nil, 1024)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error(
				"listener accept:",
				slog.String("err", err.Error()),
			)
			time.Sleep(time.Second)
			continue
		}
		
		logger.Info(
			"new connection",
			slog.String("remote",
				conn.RemoteAddr().String()),
		)
		err = conn.SetDeadline(time.Now().Add(connTimeout))
		if err != nil {
			conn.Close()
			logger.Error(
				"conn set deadline:",
				slog.String("err", err.Error()),
			)
			continue
		}
		buf.Reset(conn)
		resp.Reset()
		HTTPHandler(conn, &resp)
		conn.Close()

		blink <- 5

	}
}

// main function that serves the program. It sets up the wifi connection, creates the TCP listener, defines the blink channel to connect the blinkLED and handleConnections goroutines, and then starts both goroutines to process requests. It uses an infinite loop with a blocking select statement to prevent the program from terminating while the goroutines run in the background. It also sends a message every minute to the log to show that the program is running.
func main() {
	// time.Sleep seems to be needed in order for it to make the connection and it also gives time to get the monitor started.
	time.Sleep(time.Second * 15)
	logger.Info("Rae top of main")
	stack, dev := setupDevice()
	listener := newListener(stack)

	blink := make(chan uint, 3)
	go blinkLED(dev, blink)
	go handleConnection(listener, blink)

	for {
		select {
		case <-time.After(1 * time.Minute):
			logger.Info("waiting for connections...")
		}	
	}
}




