# Benchmark

## Description

The `benchmark` feature allows for evaluating the health and severity of various SSV client-related metrics over time by running the application as a daemon. The addresses of the SSV client, Consensus Client(s), and Execution Client need to be supplied. During runtime, the benchmark will communicate with these clients through API endpoints to gather various metrics that help troubleshoot underperforming SSV clients. Additionally, it will provide infrastructure-related metrics, such as CPU and memory usage, from the environment it is running on. The metrics will be provided in an aggregated manner when the application shuts down (the `--duration` flag sets the execution time. The process can also be interrupted manually).

The system is designed to be flexible, allowing different metrics to have their own set of conditions that determine their health status and severity levels.

## How to use

## Configuration
CLI `flags` or `config.yaml` file.

## Docker
```bash
docker run ghcr.io/ssvlabs/ssv-pulse:latest benchmark --consensus-addr=REPLACE_WITH_ADDR --execution-addr=REPLACE_WITH_ADDR --ssv-addr=REPLACE_WITH_ADDR
```

All available CLI flags can be viewed by using the --help flag.

```bash
docker run ghcr.io/ssvlabs/ssv-pulse:latest benchmark --help
```

## Metrics Overview

### Available Metrics

- SSV Client
    - Peers
	- Connections
- Infrastructure
    - CPU
	- Memory
- Execution Client
    - Latency
	- Peers
- Consensus Client
	- Attestations
	- Client Version
	- Latency
	- Peers

### Metric

A **Metric** represents a measurable entity, such as CPU usage, memory usage, or network latency. Each metric has the following components:

- **Name**: A descriptive name for the metric.
- **Values**: A collection of values representing the metric's values over time.
- **HealthConditions**: A collection of conditions that are used to evaluate the health and severity of the metric.

### HealthCondition

A **HealthCondition** defines the criteria under which a metric is evaluated. Each condition contains:

- **Threshold**: A numerical value that serves as the benchmark for the condition.
- **Operator**: The operator that determines how the threshold is applied (`>`, `<`, `>=`, `<=`, `==`).
- **Severity**: The severity level assigned if the condition is met (`None`, `Low`, `Medium`, `High`).

### Severity Levels

Severity levels indicate the importance or urgency of a condition. The system currently supports four severity levels:

- **none**: Represents no issue (usually combined with **Healthy** health status)
- **Low**: Represents a minor issue.
- **Medium**: Represents a moderate issue that requires attention.
- **High**: Represents a critical issue that needs immediate action.

### Health Status

The health status of a metric is determined by evaluating all of its data points against its health conditions:

- **Healthy**: The metric meets all conditions with a severity of `None` or no conditions are triggered.
- **Unhealthy**: At least one condition is triggered with a severity greater than `None`.

### Metric Evaluation

When evaluating a metric, the system:

1. Iterates over each data point within the metric.
2. For each data point, evaluates it against all associated health conditions.
3. Determines the overall health of the metric based on the conditions met.
4. Assigns the highest severity level from the triggered conditions.

## [Architecture](https://github.com/ssvlabs/ssv-pulse/blob/main/docs/architecture-benchmark.png)