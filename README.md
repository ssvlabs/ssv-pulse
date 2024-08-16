# SSV benchmark

![main](https://github.com/ssvlabs/ssv-benchmark/actions/workflows/go.yml/badge.svg?branch=main)

# Metrics Evaluation System

This Go application provides a framework for evaluating the health and severity of various metrics over time. The system is designed to be flexible, allowing different metrics to have their own set of conditions that determine their health status and severity levels.

## Overview

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

## How It Works

### Metric Evaluation

When evaluating a metric, the system:

1. Iterates over each data point within the metric.
2. For each data point, evaluates it against all associated health conditions.
3. Determines the overall health of the metric based on the conditions met.
4. Assigns the highest severity level from the triggered conditions.

## Architecture
[Architecture](https://github.com/ssvlabs/ssv-benchmark/blob/main/docs/architecture.png)\