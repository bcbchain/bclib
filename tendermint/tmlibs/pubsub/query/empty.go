package query

import "github.com/bcbchain/bclib/tendermint/tmlibs/pubsub"

// Empty query matches any set of tags.
type Empty struct {
}

// Matches always returns true.
func (Empty) Matches(tags pubsub.TagMap) bool {
	return true
}

func (Empty) String() string {
	return "empty"
}
