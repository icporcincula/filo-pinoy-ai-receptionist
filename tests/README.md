# Filo Phase 2 Intelligence Implementation - Test Suite

This directory contains comprehensive tests for the Phase 2 intelligence implementation of Filo, the Pinoy AI Receptionist.

## 🧪 Test Coverage

The test suite covers all major components of the intelligence system:

### 1. **RAG System Tests** (`rag_test.go`)
- **Qdrant Client**: Tests for vector database operations
- **RAG System**: End-to-end retrieval and context generation
- **Document Ingestion**: Tests for document processing and storage
- **Embedding Simulation**: Tests for placeholder embedding functionality
- **Performance Benchmarks**: Latency and throughput testing

**Key Test Cases:**
- Document search and retrieval accuracy
- RAG context generation with business knowledge
- Error handling for empty results
- Embedding normalization and consistency

### 2. **Intent Classification Tests** (`intent_test.go`)
- **Keyword-based Classification**: Tests for all intent types
- **Case Insensitive Matching**: Ensures robust text processing
- **Confidence Scoring**: Tests for classification confidence
- **Workflow Integration**: Intent-to-workflow mapping
- **Performance Testing**: High-volume intent processing

**Supported Intents:**
- `book` - Appointment booking requests
- `inquire` - Information requests
- `transfer` - Transfer to human agent
- `greeting` - Welcome interactions
- `goodbye` - Session termination

### 3. **Workflow Integration Tests** (`workflow_test.go`)
- **n8n Integration**: Tests for workflow automation platform
- **Webhook Handling**: Tests for external system integration
- **Workflow Registry**: Intent-to-workflow mapping
- **Error Handling**: Graceful failure management
- **Performance Testing**: Workflow execution benchmarks

**Workflow Types:**
- `appointment_booking` - Schedules appointments
- `transfer_to_agent` - Connects to human representatives
- `knowledge_search` - Retrieves business information

### 4. **Persona System Tests** (`persona_test.go`)
- **Business Context Management**: Tests for dynamic business information
- **Response Personalization**: Tests for tailored responses
- **Template System**: Tests for response template processing
- **System Prompt Generation**: Dynamic prompt creation
- **Performance Testing**: Response generation benchmarks

**Features Tested:**
- Multi-language support
- Business hour integration
- Service catalog management
- FAQ system integration

### 5. **Calendar Integration Tests** (`calendar_test.go`)
- **Google Calendar API**: Tests for calendar operations
- **Availability Checking**: Tests for time slot validation
- **Appointment Booking**: Tests for scheduling functionality
- **Conflict Detection**: Tests for overlapping appointments
- **Error Handling**: Calendar API failure scenarios

**Calendar Features:**
- Business hour validation
- Time slot availability
- Appointment conflict detection
- Multi-attendee support

### 6. **Integration Tests** (`integration_test.go`)
- **End-to-End Workflows**: Complete conversation flows
- **Component Interaction**: Cross-system communication
- **Error Propagation**: Error handling across components
- **Performance Testing**: System-wide performance validation

**Integration Scenarios:**
- Greeting → Inquiry → RAG → Response
- Booking → Calendar → Workflow → Confirmation
- Transfer → Workflow → Agent Connection

## 🚀 Running Tests

### Prerequisites
- Go 1.24.0 or later
- Access to test dependencies (see `go.mod`)

### Quick Start
```bash
# Run all tests
./tests/run_tests.sh

# Run specific test categories
go test ./tests -run TestRAGSystem
go test ./tests -run TestIntentClassifier
go test ./tests -run TestWorkflowManager
go test ./tests -run TestCalendarManager
go test ./tests -run TestPersonaManager

# Run with coverage
go test -coverprofile=coverage.out ./tests
go tool cover -html=coverage.out

# Run performance benchmarks
go test -bench=. -benchmem ./tests
```

### Test Script Features
The `run_tests.sh` script provides:
- **Automated Test Execution**: Runs all test categories
- **Dependency Management**: Installs required testing tools
- **Performance Analysis**: Benchmark execution and analysis
- **Coverage Reporting**: HTML coverage report generation
- **Error Handling**: Comprehensive error reporting
- **Test Results**: Organized test result storage

## 📊 Test Results

### Expected Performance Metrics
- **RAG Response Time**: < 500ms for context retrieval
- **Intent Classification**: < 100ms per classification
- **Workflow Execution**: < 200ms per workflow trigger
- **Calendar Operations**: < 300ms for availability checks
- **System Throughput**: 100+ requests/second

### Coverage Targets
- **Overall Coverage**: > 85%
- **RAG System**: > 90%
- **Intent Classification**: > 95%
- **Workflow Integration**: > 85%
- **Persona System**: > 90%
- **Calendar Integration**: > 80%

## 🔧 Test Configuration

### Environment Variables
```bash
# Test-specific configurations
TEST_TIMEOUT=300                    # Test timeout in seconds
COVERAGE_PROFILE="coverage.out"     # Coverage output file
TEST_RESULTS_DIR="test_results"     # Results storage directory
```

### Mock Servers
The test suite uses mock servers for external dependencies:
- **Qdrant Mock**: Vector database operations
- **n8n Mock**: Workflow automation platform
- **Calendar Mock**: Google Calendar API
- **Turn Detection Mock**: Speech turn detection

## 🐛 Troubleshooting

### Common Issues

1. **Missing Dependencies**
   ```bash
   go mod tidy
   go get github.com/stretchr/testify
   ```

2. **Test Timeouts**
   - Increase `TEST_TIMEOUT` in `run_tests.sh`
   - Check network connectivity to mock servers

3. **Coverage Issues**
   - Ensure all test files are in the `tests/` directory
   - Check that test functions follow Go naming conventions

4. **Performance Test Failures**
   - Run on a machine with sufficient resources
   - Check for background processes consuming CPU/memory

### Debug Mode
```bash
# Run tests with verbose output
go test -v ./tests

# Run specific test with debug output
go test -v -run TestRAGSystem_Search ./tests
```

## 📈 Performance Testing

### Benchmark Categories
- **RAG Benchmarks**: Vector search and context retrieval
- **Intent Benchmarks**: Classification speed and accuracy
- **Workflow Benchmarks**: Workflow execution performance
- **Persona Benchmarks**: Response generation speed
- **Calendar Benchmarks**: Calendar operation performance

### Performance Validation
The test suite validates:
- **Response Time SLAs**: Ensures all components meet performance targets
- **Memory Usage**: Validates memory efficiency and leak detection
- **Concurrency**: Tests concurrent request handling
- **Stress Testing**: High-volume request validation

## 🔍 Code Quality

### Test Quality Standards
- **Isolation**: Each test is independent and repeatable
- **Mocking**: External dependencies are properly mocked
- **Coverage**: Comprehensive coverage of all code paths
- **Performance**: Performance benchmarks for all critical paths
- **Documentation**: Clear test descriptions and expected outcomes

### Best Practices
- **Test Naming**: Descriptive test names following Go conventions
- **Setup/Teardown**: Proper test initialization and cleanup
- **Error Handling**: Comprehensive error case testing
- **Edge Cases**: Boundary condition and edge case testing

## 🚀 Continuous Integration

### CI/CD Integration
The test suite is designed for CI/CD integration:
```yaml
# Example GitHub Actions workflow
- name: Run Tests
  run: ./tests/run_tests.sh

- name: Upload Coverage
  uses: actions/upload-artifact@v3
  with:
    name: coverage-report
    path: tests/test_results/
```

### Automated Testing
- **Unit Tests**: Fast feedback on code changes
- **Integration Tests**: Validation of component interaction
- **Performance Tests**: Regression detection for performance issues
- **Coverage Reports**: Code coverage tracking and reporting

## 📞 Support

For issues with the test suite:
1. Check the troubleshooting section above
2. Review test logs in `test_results/` directory
3. Verify test environment setup
4. Ensure all dependencies are properly installed

## 🔄 Future Enhancements

Planned improvements to the test suite:
- **Real Service Integration**: Tests against actual external services
- **Load Testing**: Advanced load testing scenarios
- **Security Testing**: Security vulnerability testing
- **Accessibility Testing**: Accessibility compliance validation
- **Cross-Platform Testing**: Multi-platform compatibility testing

---

**Note**: This test suite ensures the Phase 2 intelligence implementation meets quality, performance, and reliability standards for production deployment.