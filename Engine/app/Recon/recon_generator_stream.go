// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package recon

import (
	"errors"

	mutationgen "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/app/Recon/Mutation"
	dgagen "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/app/Recon/dga"
)

type ScoredDomainSink func(GeneratedDomain) error

func StreamScoredDomains(path string, sink ScoredDomainSink) error {
	if sink == nil {
		return errors.New("scored domain sink is required")
	}

	mutationCandidates, mutationErr := mutationgen.GenerateDetailedFromCSV(path)
	dgaCandidates, dgaErr := dgagen.GenerateDetailedFromCSV(path)
	if mutationErr != nil && dgaErr != nil {
		return mutationErr
	}

	candidates := collectCandidates(mutationCandidates, dgaCandidates)
	valid := validateCandidates(candidates)
	return streamCandidates(valid, sink)
}

func streamCandidates(candidates map[string]candidateMeta, sink ScoredDomainSink) error {
	for domain, meta := range candidates {
		candidate := scoreCandidate(domain, meta)
		if !passesCandidateFilters(candidate) {
			continue
		}
		if err := sink(candidate); err != nil {
			return err
		}
	}
	return nil
}

func passesCandidateFilters(candidate GeneratedDomain) bool {
	if shannonEntropy(domainLabel(candidate.Domain)) < lowEntropyDropThreshold {
		return false
	}
	return IsHumanLikeDomain(candidate.Domain)
}
