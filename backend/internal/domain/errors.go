package domain

import "errors"

var (
	ErrAuctionEnded = errors.New("auction has ended")
	ErrBidTooLow    = errors.New("bid amount must be higher than current price")
)
