# Marksman
Marksman is the master server that aggregates the data received from different [`recond`](https://github.com/codeignition/recon/tree/master/cmd/recond) agents and exposes a public HTTP API.

#### Installation

If you have Go installed and workspace setup,

```sh
go get github.com/codeignition/marksman
```

#### Usage
```
Usage of marksman:
  -addr address
    	serve HTTP on address (default ":8080")
  -nats string
    	nats URL (default "nats://localhost:4222")
```

#### Terminology

[`recond`](https://github.com/codeignition/recon/tree/master/cmd/recond) is the daemon (agent) that runs on your target machine (server).

#### Disclaimer

So far the project is tested only on Linux, specifically Ubuntu 14.04.

#### Contributing

Check out the code and jump to the Issues section to join a conversation or to start one. Also, please read the  [CONTRIBUTING](https://github.com/codeignition/marksman/blob/master/CONTRIBUTING.md) document.

#### License
BSD 3-clause "New" or "Revised" license