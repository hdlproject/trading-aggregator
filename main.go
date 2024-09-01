package main

import (
	"context"
	"net"

	"trading-aggregator/binance"
	"trading-aggregator/webhook"
)

func main() {
	binance.NewClient(binance.Config{
		URL:       "https://api.binance.com/api",
		APIKey:    "XYUUr93tv7v6VEMNVVAKFH0n3AmqczOeDPUQowSTR95oMZeqQYI0npkFlzP3aTre",
		APISecret: "2z9LTZOF76LXrmakM5Qec2ATCwmDUOQAz2WX8BjxoOK65lHOZqax1RxPEZ75Lgdn",
	})

	listener, err := net.Listen("tcp", "localhost:8888")
	if err != nil {
		panic(err)
	}

	webhookServer := webhook.NewWebhook(listener)
	webhookServer.Serve(context.Background())
}
