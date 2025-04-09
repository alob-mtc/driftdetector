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

# Run in verbose mode
./driftdetector --instance-ids i-xxxxxxxxxxxxxxxxx --config-path ./configs/sample.tf --verbose
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

### Test Organization

The test structure is organized as follows:

- **Unit Tests**: Located alongside the code they test

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

   cmd:
   - `cmd/driftdetector`: Contains the main CLI entry point and command-line argument parsing
   
   internal:
   - `/providers`: Contains provider implementations (this makes it easy to add in new providers in the future):
     - `/providers/aws`: Handles AWS API interactions to fetch instance details
   - `/terraform`: Parses HCL configuration files
   - `/driftcheck`: Implements drift detection logic
   - `/reporting`: Implements drift reporting logic
   - `/orchestrator`: Coordinates the workflow between components
   - `/models`: Defines shared data structures (domain models)
   - `/pkg/logging`: Handles logging
   - Each package includes dedicated mocks directory for testing (e.g., `internal/providers/aws/mocks`)

2. **Testability**: The code uses dependency injection and interfaces to facilitate testing. Mock implementations are used extensively in unit tests.

3. **Concurrency**: EC2 instance checks are performed concurrently using goroutines, with an optional concurrency limit to avoid overwhelming AWS API limits.

4. **Flexibility**: Users can specify which attributes to check and control the output format to suit their needs.

## Key Design Decisions

1. **AWS SDK Choice**: I chose AWS SDK for Go v2 for improved performance and better context-aware API design.
   - **Bulk AWS API Operations**: Leveraged AWS DescribeInstances' support for bulk queries. Bulk-fetching the instance as opposed to getting them 1 by 1 which is very good for performance.
   - Also introduced batching to minimize the chance of throttling that can be called by abusing the bulk-fetch feature of the AWS API.

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

5. **Error Handling**: I used wrapped errors (`fmt.Errorf` with `%w`) to provide context while preserving the original error information. 

6. **Concurrency Model**: I implemented Go's `errgroup` package to manage concurrent instance checks.

## Challenges Faced

1. **Drift Detection Algorithm** Complexity: The drift detection algorithm needed to handle various data types and structures, including nested maps and lists. I had to ensure that the comparison logic was robust enough to handle these complexities while remaining efficient. 
Also coming up with a design that is easy to extend with very minimal/no change to the core logic was a challenge of its own I had to figure out.

> The bulk of the task is a lot of research and experimentation. which was fun

## Sample Data

- **Sample Terraform Configuration**: A basic example can be found in `configs/sample.tf`.
- **Sample AWS EC2 Response**: A mock JSON response structure can be found in `testdata/aws_ec2_response.json`.

## Future Improvements

- **Terraform State File Support**: Add support for `.tfstate` files using direct JSON parsing or the `terraform-exec` library.

- **Enhanced Attribute Coverage**: Add support for more resource properties.

- **Additional Output Formats**: Implement YAML and colorized console output options.

- **Multi-Resource Support**: Extend drift detection to other AWS resources beyond EC2 instances (e.g., Security Groups, IAM Roles, S3 Buckets).

- **Tag-Based Filtering**: Add ability to select instances for checking based on tags rather than instance IDs.

- **Custom Error Types**: I added a couple customer error type mostly for AWS since there is more error variant there and I saw the benefit of implementing this here it improved my bugging time. (I could also extend this to a lot more of the module)

- **Attribute Enum System**: Implement an enumeration system for attribute names. (I could also extend this to a lot more of the module)
  - This would help with type safety and reduce the risk of typos in attribute names.
  - It would also make it easier to add new attributes support in the future.
- **Logging Improvements**: Extend debug logging support to other module like AWS. 
