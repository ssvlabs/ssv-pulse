package configs

import (
	"strings"
	"testing"
)

func TestBenchmark_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Benchmark
		want    bool
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config with consensus metrics",
			cfg: Benchmark{
				Consensus: Consensus{
					Addresses: []string{"http://localhost:8545"},
					Metrics: ConsensusMetrics{
						Peers: Metric{Enabled: true},
					},
				},
				Network: "mainnet",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Valid config with consensus metrics",
			cfg: Benchmark{
				Consensus: Consensus{
					Addresses: []string{"http://localhost:8545"},
					Metrics: ConsensusMetrics{
						Peers: Metric{Enabled: true},
					},
				},
				Network: "mainnet",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Valid config with execution metrics",
			cfg: Benchmark{
				Execution: Execution{
					Addresses: []string{"http://localhost:8545"},
					Metrics: ExecutionMetrics{
						Peers: Metric{Enabled: true},
					},
				},
				Network: "mainnet",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Valid config with SSV metrics",
			cfg: Benchmark{
				SSV: SSV{
					Address: "http://localhost:8545",
					Metrics: SSVMetrics{
						Peers: Metric{Enabled: true},
					},
				},
				Network: "mainnet",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Invalid network name",
			cfg: Benchmark{
				Network: "invalid",
			},
			want:    false,
			wantErr: true,
			errMsg:  "network name was not valid",
		},
		{
			name: "Multiple consensus addresses with separator",
			cfg: Benchmark{
				Consensus: Consensus{
					Addresses: []string{"http://localhost:8545;http://localhost:8546"},
					Metrics: ConsensusMetrics{
						Peers: Metric{Enabled: true},
					},
				},
				Network: "mainnet",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Single consensus address",
			cfg: Benchmark{
				Consensus: Consensus{
					Addresses: []string{"http://localhost:8545"},
					Metrics: ConsensusMetrics{
						Peers: Metric{Enabled: true},
					},
				},
				Network: "mainnet",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Multiple separate consensus addresses",
			cfg: Benchmark{
				Consensus: Consensus{
					Addresses: []string{"http://localhost:8545", "http://localhost:8546", "http://localhost:8547"},
					Metrics: ConsensusMetrics{
						Peers: Metric{Enabled: true},
					},
				},
				Network: "mainnet",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Semicolon separated consensus addresses",
			cfg: Benchmark{
				Consensus: Consensus{
					Addresses: []string{"http://localhost:8545;http://localhost:8546;http://localhost:8547"},
					Metrics: ConsensusMetrics{
						Peers: Metric{Enabled: true},
					},
				},
				Network: "mainnet",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Single execution address",
			cfg: Benchmark{
				Execution: Execution{
					Addresses: []string{"http://localhost:8545"},
					Metrics: ExecutionMetrics{
						Peers: Metric{Enabled: true},
					},
				},
				Network: "mainnet",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Multiple separate execution addresses",
			cfg: Benchmark{
				Execution: Execution{
					Addresses: []string{"http://localhost:8545", "http://localhost:8546", "http://localhost:8547"},
					Metrics: ExecutionMetrics{
						Peers: Metric{Enabled: true},
					},
				},
				Network: "mainnet",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Semicolon separated execution addresses",
			cfg: Benchmark{
				Execution: Execution{
					Addresses: []string{"http://localhost:8545;http://localhost:8546;http://localhost:8547"},
					Metrics: ExecutionMetrics{
						Peers: Metric{Enabled: true},
					},
				},
				Network: "mainnet",
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Benchmark.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message = %v, want to contain %v", err, tt.errMsg)
				}
			}
			if got != tt.want {
				t.Errorf("Benchmark.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
