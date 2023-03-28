![GitHub Workflow Status](https://img.shields.io/github/workflow/status/stianfro/tlscheck/CI?style=flat-square)

# tlscheck - Kubernetes TLS Certificate Expiration Checker

This program lists the TLS certificates used in Kubernetes secrets of type kubernetes.io/tls and shows their remaining lifetime (in days). It can be useful for monitoring certificate expiration and taking timely action to renew them.

## Features
- Retrieves all the namespaces in the cluster
- Lists all the secrets of type kubernetes.io/tls within each namespace
- Decodes and parses the certificates
- Calculates the remaining lifetime of each certificate in days
- Displays the results in a tabular format

## Prerequisites
- Go 1.16 or higher
- Access to a Kubernetes cluster (kubeconfig or in-cluster configuration)
- Kubernetes client-go library

## Installation
1. Clone the repository:
```
git clone https://github.com/stianfro/tlscheck.git
cd tlscheck
```
2. Build the binary:
```
go build -o tlscheck
```
3. Move the binary to a directory in your $PATH:
```bash
sudo mv tlscheck /usr/local/bin/
```

## Usage
Run the program:

```
./tlscheck
```

The output will be displayed in the following format:

```
SECRET_NAME                    NAMESPACE                       ISSUER                          REMAINING_LIFETIME (days)
-----------------------------------------------------------------------
example-tls-secret             default                         CN=example.com                  89
another-tls-secret             kube-system                     CN=another-example.com          150
...
```

## Contributing
Feel free to open issues or submit pull requests if you find any bugs or have suggestions for improvements. Your contributions are always welcome!

## License
This project is licensed under the MIT License.

