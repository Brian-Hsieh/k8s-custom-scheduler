package main

import (
	"fmt"
	"math/rand"
)

type Particle struct {
	Position    []float64 // node assignments per pod
	Velocity    []float64
	BestPos     []float64
	BestFitness float64
}

type Swarm struct {
	Particles         []Particle
	GlobalBest        []float64
	GlobalBestFitness float64
}

func fitness(pos []float64, latency RegionLatency, load NodeLoad) float64 {
	var total float64

	// latency fitness
	latency_weight := 1.0
	latency_fit := 0.0
	for _, p := range pos {
		if p <= 1 {
			latency_fit += latency.tokyo
		} else if p <= 3 {
			latency_fit += latency.sydney
		} else {
			latency_fit += latency.singapore
		}
	}

	// scale this to [1,6]
	max_latency := max(latency.tokyo, latency.sydney, latency.singapore) * float64(len(pos))
	min_latency := min(latency.tokyo, latency.sydney, latency.singapore) * float64(len(pos))
	latency_fit_scaled := (latency_fit-min_latency)/(max_latency-min_latency)*5 + 1

	total += latency_fit_scaled * latency_weight

	// load fitness
	load_weight := 1.0
	load_0 := load.w0
	load_1 := load.w1
	load_2 := load.w2
	load_3 := load.w3
	load_4 := load.w4
	load_5 := load.w5

	// assume workload of pod to be 10%
	for _, p := range pos {
		if p <= 1 {
			load_0 += 10
		} else if p <= 2 {
			load_1 += 10
		} else if p <= 3 {
			load_2 += 10
		} else if p <= 4 {
			load_3 += 10
		} else if p <= 5 {
			load_4 += 10
		} else {
			load_5 += 10
		}
	}
	ideal_load := (load_0 + load_1 + load_2 + load_3 + load_4 + load_5) / 6.0
	load_bal := max(load_0, load_1, load_2, load_3, load_4, load_5) / ideal_load // range from 1 to 6
	total += load_bal * load_weight
	return total
}

func runPSO(latency RegionLatency, load NodeLoad, numPods int, numIterations int, swarmSize int) []float64 {
	fmt.Println("running PSO...")
	fmt.Printf("latency values: %v\n", latency)
	fmt.Printf("load values: %v\n", load)

	// Initialize swarm
	swarm := Swarm{
		Particles:         make([]Particle, swarmSize),
		GlobalBestFitness: 1e9, // intial large value <- we want it small
	}

	// Initialize pos and vel for each Particle
	for i := 0; i < swarmSize; i++ {
		pos := make([]float64, numPods)
		vel := make([]float64, numPods)
		for j := 0; j < numPods; j++ {
			pos[j] = rand.Float64() * 6 // 6 VMs
			vel[j] = (rand.Float64()*2 - 1) * 6
		}
		fit := fitness(pos, latency, load)
		swarm.Particles[i] = Particle{
			Position:    pos,
			Velocity:    vel,
			BestPos:     append([]float64(nil), pos...), // copy of pos
			BestFitness: fit,
		}
		if fit < swarm.GlobalBestFitness {
			swarm.GlobalBestFitness = fit
			swarm.GlobalBest = append([]float64(nil), pos...)
		}
	}

	// PSO main loop
	for iter := 0; iter < numIterations; iter++ {
		for i, p := range swarm.Particles {
			for j := 0; j < numPods; j++ {

				omega := 1.0
				c1 := 0.5
				c2 := 0.5

				// velocity update
				p.Velocity[j] = omega*p.Velocity[j] +
					c1*rand.Float64()*(p.BestPos[j]-p.Position[j]) +
					c2*rand.Float64()*(swarm.GlobalBest[j]-p.Position[j])

				if p.Velocity[j] > 6 {
					p.Velocity[j] = 6
				}
				if p.Velocity[j] < -6 {
					p.Velocity[j] = -6
				}

				// position update
				p.Position[j] += p.Velocity[j]

				if p.Position[j] > 6 {
					p.Position[j] = 6
				}
				if p.Position[j] < 0 {
					p.Position[j] = 0
				}
			}

			fit := fitness(p.Position, latency, load)
			if fit < p.BestFitness {
				p.BestFitness = fit
				p.BestPos = append([]float64(nil), p.Position...)
			}
			if fit < swarm.GlobalBestFitness {
				swarm.GlobalBestFitness = fit
				swarm.GlobalBest = append([]float64(nil), p.Position...)
			}
			swarm.Particles[i] = p
		}
	}
	return swarm.GlobalBest
}
