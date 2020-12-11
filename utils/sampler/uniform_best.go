// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package sampler

import (
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/liraxapp/avalanchego/utils/timer"
)

var (
	errNoValidUniformSamplers = errors.New("no valid uniform samplers found")
)

func init() { rand.Seed(time.Now().UnixNano()) }

// uniformBest implements the Uniform interface.
//
// Sampling is performed by using another implementation of the Uniform
// interface.
//
// Initialization attempts to find the best sampling algorithm given the dataset
// by performing a benchmark of the provided implementations.
type uniformBest struct {
	Uniform
	samplers            []Uniform
	maxSampleSize       int
	benchmarkIterations int
	clock               timer.Clock
}

// NewBestUniform returns a new sampler
func NewBestUniform(expectedSampleSize int) Uniform {
	return &uniformBest{
		samplers: []Uniform{
			&uniformReplacer{},
			&uniformResample{},
		},
		maxSampleSize:       expectedSampleSize,
		benchmarkIterations: 100,
	}
}

func (s *uniformBest) Initialize(length uint64) error {
	s.Uniform = nil
	bestDuration := time.Duration(math.MaxInt64)

	sampleSize := s.maxSampleSize
	if length < uint64(sampleSize) {
		sampleSize = int(length)
	}

samplerLoop:
	for _, sampler := range s.samplers {
		if err := sampler.Initialize(length); err != nil {
			continue
		}

		start := s.clock.Time()
		for i := 0; i < s.benchmarkIterations; i++ {
			if _, err := sampler.Sample(sampleSize); err != nil {
				continue samplerLoop
			}
		}
		end := s.clock.Time()
		duration := end.Sub(start)
		if duration < bestDuration {
			bestDuration = duration
			s.Uniform = sampler
		}
	}

	if s.Uniform == nil {
		return errNoValidUniformSamplers
	}
	return nil
}
