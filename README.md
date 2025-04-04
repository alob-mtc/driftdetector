# Drift Detector

## Project Overview

This Go application detects configuration drift between AWS EC2 instances and their corresponding Terraform configurations. It compares the actual state of an instance in AWS against its definition in a Terraform state file or HCL configuration, identifying discrepancies in specified attributes.

## Setup and Installation

- Ensure Go 1.19+ is installed: [https://go.dev/doc/install](https://go.dev/doc/install)
- Configure AWS credentials (e.g., via environment variables `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`, or shared credentials file `~/.aws/credentials`): [https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-gosdk.html](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-gosdk.html),
`aws configure` command can also be used to set up credentials.
- Clone the repository: `git clone <repository-url>` (Replace `<repository-url>` with the actual URL once available)
- Navigate to the project directory: `cd driftdetector`
- Build the application: `go build ./cmd/driftdetector`

## Running the Application

The application can be run in various ways depending on your requirements:

```bash
# Basic usage: Check drift for a specific instance using HCL config
./driftdetector --instance-ids i-xxxxxxxxxxxxxxxxx --config-path ./configs/sample.tf

# Check drift for multiple instances
./driftdetector --instance-ids i-xxxxxxxxxxxxxxxxx,i-yyyyyyyyyyyyyyyyy --config-path ./configs/sample.tf

# Check drift with controlled concurrency (limit to 2 instances at a time)
./driftdetector --instance-ids i-xxxxxxxxx,i-yyyyyyyyy,i-zzzzzzzzz --config-path ./configs/sample.tf --concurrency 2

# Output results in JSON format
./driftdetector --instance-ids i-xxxxxxxxxxxxxxxxx --config-path ./configs/sample.tf --output json

# Specify attributes to check
./driftdetector --instance-ids i-xxxxxxxxxxxxxxxxx --config-path ./configs/sample.tf --attributes instance_type,tags,security_groups
```

### Command-line Arguments

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `--instance-ids` | Comma-separated list of EC2 instance IDs to check | None | Yes |
| `--config-path` | Path to Terraform configuration file | None | Yes |
| `--attributes` | Comma-separated list of attributes to check | All supported attributes | No |
| `--concurrency` | Maximum number of instances to check in parallel | No limit | No |
| `--output` | Output format: `table` or `json` | `table` | No |
| `--help` | Show help message | | No |

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Mocking Dependencies

I use [mockery](https://github.com/vektra/mockery) to generate mocks for testing. To install:

```bash
go install github.com/vektra/mockery/v2@latest
```

To generate mocks:
> mockery must be installed

```bash
# Generate mocks
go generate ./...
```

## Design Approach

The drift detector follows these key principles:

1. **Modularity**: The codebase is organized into dedicated packages with clear responsibilities:
   - `aws`: Handles AWS API interactions to fetch instance details
   - `terraform`: Parses HCL configuration files
   - `driftcheck`: Implements drift detection logic
   - `orchestrator`: Coordinates the workflow between components
   - `models`: Defines shared data structures

2. **Testability**: The code uses dependency injection and interfaces to facilitate testing. Mock implementations are used extensively in unit tests.

3. **Concurrency**: EC2 instance checks are performed concurrently using goroutines, with an optional concurrency limit to avoid overwhelming AWS API limits.

4. **Flexibility**: Users can specify which attributes to check and control the output format to suit their needs.

## Key Design Decisions

1. **AWS SDK Choice**: I chose AWS SDK for Go v2 for improved performance and better context-aware API design.

2. **Terraform Parsing**: I utilized HashiCorp's HCL library for parsing Terraform configuration files. This gives direct access to the configuration structure without having to run Terraform commands.

3. **Drift Detection Algorithm**: I implemented a flexible comparison logic that:
   - Normalizes attribute names to handle different naming conventions
   - Compares complex data structures like maps and lists
   - Ignores certain attributes that are known to be different by design (e.g., instance ID)
   - Handles type conversion for proper comparison

4. **Testing Strategy**: I used a combination of:
   - Unit tests with mocks for external dependencies
   - Table-driven tests for comprehensive coverage
   - The `testify/assert` package for readable test assertions

5. **Error Handling**: I used wrapped errors (`fmt.Errorf` with `%w`) to provide context while preserving the original error information. With regards to errors there's more that can be done here but i've decided to keep it simple. For example deifining custom error type is one idea that can be explored here, but i consiously choose to keep it simple.

6. **Concurrency Model**: I implemented Go's `errgroup` package to manage concurrent instance checks with proper error propagation and context cancellation.

## Challenges Faced

1. **HCL Parsing Complexity**: Terraform's HCL format has a complex structure that can be challenging to parse correctly. I had to carefully handle different resource types, attribute formats, and nested blocks.

2. **AWS API Limitations**: AWS API has rate limits that can be hit when checking many instances. My solution was to implement concurrency controls to limit the number of simultaneous API calls. 
> However i discovered the the AWs describe support bulk-actions, but i've choosen not to unitlize this to keep things simple although i know actually implementing it will not bring too much complexities, but i choose this simple solution (Depending on the final usecase for this project and if we see that there is need to check a large number of instances, then i think going for the bulk-action can have a compeling benefit)

## Sample Data

- **Sample Terraform Configuration**: A basic example can be found in `configs/sample.tf`.
- **Sample AWS EC2 Response**: A mock JSON response structure can be found in `testdata/aws_ec2_response.json`.

## Future Improvements

- Support for Terraform state files (`.tfstate`) using direct JSON parsing or the `terraform-exec` library.
- More comprehensive attribute comparison (add more resource properties like volumes, etc.).
- Enhanced output formats (e.g., YAML, colorized output).
- Improved error handling with retry logic for AWS API calls.
- Support for comparing other AWS resources beyond EC2 instances (e.g., Security Groups, IAM Roles, S3 Buckets).
- Instance filtering by tags or other criteria for bulk checks.
- Defining custom error type is one idea that can be explored here, this will give a lot more robustness on the error handling a reporting side.
- Leveraging AWs describe support for bulk-actions, use this to bulk-fetch the instance as oppose to getting this 1 by 1 like i do not (there's not significant draw back to my current approach) how ever if the uses case is to expore a large set of ec2 instances, then bulk-actions is definatly the way to go here.
- Leverage the use of enums for keeping the attribute much more consistent 
