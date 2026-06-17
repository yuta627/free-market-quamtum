package domain

import "errors"

var (
	ErrAuctionEnded    = errors.New("auction has ended")
	ErrBidTooLow       = errors.New("bid amount must be higher than current price")
	ErrAuctionNotEnded = errors.New("auction has not ended yet")
	ErrAlreadyFinalized = errors.New("auction is already finalized")
	ErrNoBids          = errors.New("no bids on this auction")
)
