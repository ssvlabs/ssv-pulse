# Analyzer

## Description

The `analyzer` feature allows for evaluating the health of an SSV client by analyzing log files. It can simultaneously analyze multiple log files, aggregate all the data, and support logs from one or multiple SSV client instances.

## How to use

## Configuration
CLI `flags` or `config.yaml` file.

## Docker
Example of a Docker command. It assumes that the log files are stored on the local machine in the `/path/to/local/dir` directory.

```bash
docker run -v /path/to/local/dir:/path/in/container ghcr.io/ssvlabs/ssv-pulse:latest analyzer --log-files-directory=/path/in/container
```

All available CLI flags can be viewed by using the --help flag.

```bash
docker run ghcr.io/ssvlabs/ssv-pulse:latest analyzer --help
```

## Metrics Overview

At a high level, metrics can be divided into two categories: metrics that belong to the _log file owner_ (the Operator whose logs are being analyzed), such as _peer ID_, _number of peers_, _peers’ client versions_, the _clusters_ the operator is part of, _consensus client response time_, and metrics related to other SSV clients the log file owner communicates with through P2P networking. The latter is fairly limited and primarily provides the log file owner’s _perspective_.

## [Architecture](https://github.com/ssvlabs/ssv-pulse/blob/main/docs/architecture-analyzer.png)