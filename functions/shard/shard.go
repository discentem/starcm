package shard

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/logging"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"
	"go.starlark.net/starlark"
)

func getSeededFlexibleShard(identifier string, shardSize int, seed string) (int, error) {
	if shardSize < 10 {
		return 0, fmt.Errorf("shard_size must be at least 10")
	}

	// Create an MD5 hash of the fqdn concatenated with the string seed.
	data := identifier + seed
	hash := md5.Sum([]byte(data))

	// Convert the first 7 characters of the hash to an integer.
	hexString := hex.EncodeToString(hash[:])

	if len(hexString) < 7 {
		return 0, fmt.Errorf("hexadecimal string too short: %s", hexString)
	}

	intValue := 0
	_, err := fmt.Sscanf(hexString[0:7], "%x", &intValue)
	if err != nil {
		return 0, err
	}

	// Return the shard number.
	return intValue % shardSize, nil
}

type shardAction struct{}

func (a *shardAction) Run(ctx context.Context, workingDirectory string, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	id, err := starlarkhelpers.FindValueinKwargs(kwargs, "identifier")
	if err != nil {
		logging.Log("shard", deck.V(3), "error", "failed to find identifier in kwargs")
		return nil, err
	}

	shardSizeIdx, err := starlarkhelpers.FindIndexOfValueInKwargs(kwargs, "shard_size")
	if err != nil {
		logging.Log("shard", deck.V(3), "error", "failed to find index of shard_size in kwargs")
		return nil, err
	}
	shardSize := kwargs[shardSizeIdx][1].(starlark.Int)

	seed, err := starlarkhelpers.FindValueinKwargs(kwargs, "seed")
	if err != nil {
		logging.Log("shard", deck.V(3), "error", "failed to find seed in kwargs")
		return nil, err
	}
	i, ok := shardSize.Int64()
	if !ok {
		logging.Log("shard", deck.V(3), "error", "failed to convert shard_size to int64")
		return nil, fmt.Errorf("failed to convert shard_size to int64")
	}

	shard, err := getSeededFlexibleShard(*id, int(i), *seed)
	if err != nil {
		logging.Log("shard", deck.V(3), "error", "failed to calculate shard")
		return nil, fmt.Errorf("failed to calculate shard: %w", err)
	}
	logging.Log(moduleName, deck.V(3), "shard", fmt.Sprintf("%d", shard))

	return &base.Result{
		Name: &moduleName,
		Output: func() *string {
			s := fmt.Sprintf("%d", shard)
			return &s
		}(),
		Success: true,
		Changed: false,
		Error:   err,
	}, err
}

func New(ctx context.Context) *base.Module {
	var (
		identifier string
		shardSize  starlark.Int
		seed       string
	)

	return base.NewModule(
		ctx,
		"shard",
		[]base.ArgPair{
			{Key: "identifier", Type: &identifier},
			{Key: "shard_size", Type: &shardSize},
			{Key: "seed", Type: &seed},
		},
		&shardAction{},
	)
}
