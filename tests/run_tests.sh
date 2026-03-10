#!/bin/bash

# Test runner script for Filo Phase 2 Intelligence Implementation
# This script runs all tests and generates reports

set -e

echo "🧪 Filo Phase 2 Intelligence Implementation - Test Suite"
echo "======================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_TIMEOUT=300 # 5 minutes timeout for all tests
COVERAGE_PROFILE="coverage.out"
TEST_RESULTS_DIR="test_results"

# Create test results directory
mkdir -p $TEST_RESULTS_DIR

echo -e "\n${BLUE}📋 Test Configuration${NC}"
echo "Test Timeout: ${TEST_TIMEOUT}s"
echo "Coverage Profile: ${COVERAGE_PROFILE}"
echo "Results Directory: ${TEST_RESULTS_DIR}"

# Function to print test section headers
print_section() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Function to run tests with error handling
run_test() {
    local test_name=$1
    local test_command=$2
    local description=$3
    
    echo -e "\n${YELLOW}🧪 Running: $description${NC}"
    echo "Command: $test_command"
    
    if timeout $TEST_TIMEOUT $test_command > "$TEST_RESULTS_DIR/${test_name}.log" 2>&1; then
        echo -e "${GREEN}✅ $test_name: PASSED${NC}"
        return 0
    else
        echo -e "${RED}❌ $test_name: FAILED${NC}"
        echo "Check log: $TEST_RESULTS_DIR/${test_name}.log"
        return 1
    fi
}

# Function to check if Go is available
check_go() {
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ Go is not installed or not in PATH${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ Go version: $(go version)${NC}"
}

# Function to install dependencies
install_dependencies() {
    print_section "Installing Dependencies"
    
    # Install test dependencies
    run_test "deps" "go mod tidy" "Installing Go module dependencies"
    
    # Install additional testing tools
    echo -e "\n${YELLOW}📦 Installing additional testing tools...${NC}"
    go install github.com/axw/gocov/gocov@latest
    go install github.com/axw/gocov/gocov-xml@latest
    go install github.com/jstemmer/go-junit-report@latest
    
    echo -e "${GREEN}✅ Dependencies installed successfully${NC}"
}

# Function to run unit tests
run_unit_tests() {
    print_section "Unit Tests"
    
    # Run all unit tests
    run_test "unit_tests" "go test -v -race ./..." "Running all unit tests with race detection"
    
    # Run specific component tests
    run_test "rag_tests" "go test -v ./tests -run TestQdrantClient -run TestRAGSystem -run TestSimulateEmbedding" "RAG System Tests"
    run_test "intent_tests" "go test -v ./tests -run TestIntentClassifier -run TestIntentHandler -run TestIntentWorkflow" "Intent Classification Tests"
    run_test "workflow_tests" "go test -v ./tests -run TestWorkflowManager -run TestWorkflowRegistry -run TestWorkflowExecutor" "Workflow Integration Tests"
    run_test "persona_tests" "go test -v ./tests -run TestPersonaManager -run TestContextManager -run TestPersonalizationEngine" "Persona System Tests"
    run_test "calendar_tests" "go test -v ./tests -run TestCalendarManager -run TestCalendarExecutor -run TestCalendarMonitor" "Calendar Integration Tests"
}

# Function to run integration tests
run_integration_tests() {
    print_section "Integration Tests"
    
    run_test "integration_tests" "go test -v ./tests -run TestEndToEndConversationFlow -run TestRAGWorkflowIntegration -run TestIntentWorkflowIntegration -run TestCalendarIntegration -run TestPersonaSystemIntegration" "End-to-End Integration Tests"
}

# Function to run performance tests
run_performance_tests() {
    print_section "Performance Tests"
    
    run_test "performance_tests" "go test -v -bench=. -benchmem ./tests" "Performance Benchmarks"
    run_test "stress_tests" "go test -v -run TestSystemPerformance -timeout 60s" "System Stress Tests"
}

# Function to run coverage analysis
run_coverage() {
    print_section "Coverage Analysis"
    
    # Run tests with coverage
    run_test "coverage" "go test -coverprofile=$COVERAGE_PROFILE ./..." "Generating coverage report"
    
    # Generate HTML coverage report
    if [ -f "$COVERAGE_PROFILE" ]; then
        run_test "coverage_html" "go tool cover -html=$COVERAGE_PROFILE -o $TEST_RESULTS_DIR/coverage.html" "Generating HTML coverage report"
        echo -e "${GREEN}📊 Coverage report generated: $TEST_RESULTS_DIR/coverage.html${NC}"
    fi
    
    # Show coverage summary
    if command -v go &> /dev/null; then
        echo -e "\n${YELLOW}📈 Coverage Summary:${NC}"
        go tool cover -func=$COVERAGE_PROFILE | tail -1
    fi
}

# Function to run specific benchmark tests
run_benchmarks() {
    print_section "Benchmark Tests"
    
    run_test "rag_benchmarks" "go test -bench=BenchmarkRAGSystem -benchmem ./tests" "RAG System Benchmarks"
    run_test "intent_benchmarks" "go test -bench=BenchmarkIntentClassifier -benchmem ./tests" "Intent Classification Benchmarks"
    run_test "workflow_benchmarks" "go test -bench=BenchmarkWorkflowManager -benchmem ./tests" "Workflow System Benchmarks"
    run_test "persona_benchmarks" "go test -bench=BenchmarkPersonaManager -benchmem ./tests" "Persona System Benchmarks"
    run_test "calendar_benchmarks" "go test -bench=BenchmarkCalendarManager -benchmem ./tests" "Calendar System Benchmarks"
}

# Function to validate test environment
validate_environment() {
    print_section "Environment Validation"
    
    # Check if required files exist
    local required_files=(
        "tests/go.mod"
        "tests/rag_test.go"
        "tests/intent_test.go"
        "tests/workflow_test.go"
        "tests/persona_test.go"
        "tests/calendar_test.go"
        "tests/integration_test.go"
    )
    
    for file in "${required_files[@]}"; do
        if [ -f "$file" ]; then
            echo -e "${GREEN}✅ $file exists${NC}"
        else
            echo -e "${RED}❌ $file missing${NC}"
            exit 1
        fi
    done
    
    # Check if main Go modules exist
    if [ -f "go.mod" ]; then
        echo -e "${GREEN}✅ Main go.mod exists${NC}"
    else
        echo -e "${RED}❌ Main go.mod missing${NC}"
        exit 1
    fi
}

# Function to generate test report
generate_report() {
    print_section "Test Report Generation"
    
    # Count test results
    local passed=0
    local failed=0
    
    for log_file in $TEST_RESULTS_DIR/*.log; do
        if [ -f "$log_file" ]; then
            if grep -q "✅" "$log_file"; then
                ((passed++))
            elif grep -q "❌" "$log_file"; then
                ((failed++))
            fi
        fi
    done
    
    echo -e "\n${BLUE}📊 Test Summary:${NC}"
    echo "Total Tests: $((passed + failed))"
    echo -e "Passed: ${GREEN}$passed${NC}"
    echo -e "Failed: ${RED}$failed${NC}"
    
    if [ $failed -eq 0 ]; then
        echo -e "\n${GREEN}🎉 All tests passed!${NC}"
    else
        echo -e "\n${RED}⚠️  Some tests failed. Check the logs in $TEST_RESULTS_DIR/${NC}"
    fi
    
    # Generate summary report
    cat > $TEST_RESULTS_DIR/test_summary.md << EOF
# Filo Phase 2 Intelligence Implementation - Test Report

## Test Summary
- **Total Tests**: $((passed + failed))
- **Passed**: $passed
- **Failed**: $failed
- **Success Rate**: $(( passed * 100 / (passed + failed) ))%

## Test Categories
- Unit Tests: Core component functionality
- Integration Tests: End-to-end system workflows
- Performance Tests: System performance and scalability
- Coverage Analysis: Code coverage metrics

## Test Environment
- Go Version: $(go version)
- Test Timeout: ${TEST_TIMEOUT}s
- Coverage Profile: ${COVERAGE_PROFILE}

## Generated Reports
- Coverage Report: coverage.html
- Individual Test Logs: *.log files

## Next Steps
1. Review failed tests in the logs
2. Address any performance bottlenecks identified
3. Improve test coverage for uncovered code
4. Validate integration with real external services
EOF
    
    echo -e "\n${GREEN}📄 Test summary generated: $TEST_RESULTS_DIR/test_summary.md${NC}"
}

# Function to cleanup
cleanup() {
    print_section "Cleanup"
    
    # Remove temporary files
    rm -f $COVERAGE_PROFILE
    
    echo -e "${GREEN}✅ Cleanup completed${NC}"
}

# Main execution
main() {
    echo -e "\n${BLUE}🚀 Starting Filo Phase 2 Intelligence Test Suite${NC}"
    
    # Validate environment
    validate_environment
    
    # Check Go installation
    check_go
    
    # Install dependencies
    install_dependencies
    
    # Run test suites
    run_unit_tests
    run_integration_tests
    run_performance_tests
    run_benchmarks
    
    # Generate coverage report
    run_coverage
    
    # Generate final report
    generate_report
    
    # Cleanup
    cleanup
    
    echo -e "\n${GREEN}🎉 Test suite completed successfully!${NC}"
    echo -e "📁 Check results in: ${BLUE}$TEST_RESULTS_DIR${NC}"
}

# Handle script interruption
trap 'echo -e "\n${RED}⚠️  Tests interrupted${NC}"; cleanup; exit 1' INT TERM

# Run main function
main "$@"