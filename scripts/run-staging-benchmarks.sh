#!/bin/bash
# Run comprehensive guidellm benchmarks on staging models
# Bead: aas-gu71k
# Created: 2026-01-08

set -e

# Configuration
API_URL="https://api.staging.otherjamesbrown.com"
API_KEY="aas-master-staging-aHkHICegxiOoyKjfiMKUHJ5TJXi7nGQz"
RESULTS_DIR="$HOME/guidellm-runner/results/staging/$(date +%Y-%m-%d_%H-%M-%S)"
mkdir -p "$RESULTS_DIR"

# Benchmark parameters
MAX_REQUESTS=10        # Number of requests (reduced for quota)
MAX_DURATION=30        # Max duration in seconds
RATE=0.5               # Requests per second
PROMPT_TOKENS=100
OUTPUT_TOKENS=50

# Export API key for guidellm
export OPENAI_API_KEY="$API_KEY"

echo "Starting comprehensive staging benchmarks"
echo "Results directory: $RESULTS_DIR"
echo ""

# Function to run a benchmark
run_benchmark() {
    local model=$1
    local display_name=$2
    local notes=$3

    echo "========================================="
    echo "Benchmarking: $display_name"
    echo "Model: $model"
    echo "Notes: $notes"
    echo "========================================="

    local output_file="$RESULTS_DIR/${model/\//_}.json"
    local report_file="$RESULTS_DIR/${model/\//_}.html"

    # Run guidellm benchmark
    guidellm benchmark run \
        --target "$API_URL" \
        --model "$model" \
        --request-type chat_completions \
        --data-type emulated \
        --data "prompt_tokens=$PROMPT_TOKENS,output_tokens=$OUTPUT_TOKENS" \
        --load-gen-mode synchronous \
        --rate "$RATE" \
        --max-requests "$MAX_REQUESTS" \
        --max-duration "$MAX_DURATION" \
        --output-path "$output_file" \
        --report-html "$report_file" \
        2>&1 | tee "$RESULTS_DIR/${model/\//_}.log"

    if [ $? -eq 0 ]; then
        echo "✓ Benchmark completed: $model"
    else
        echo "✗ Benchmark failed: $model"
    fi
    echo ""
}

# vLLM Models
run_benchmark "llama-3-1-8b-instruct-vllm-ada" "Llama 3.1 8B (vLLM on Ada)" "vLLM on RTX 4000 Ada GPU"
run_benchmark "unsloth-gpt-oss-20b" "GPT-OSS 20B (vLLM)" "unsloth/gpt-oss-20b via vLLM"
run_benchmark "qwen2-vl-7b-instruct" "Qwen2-VL 7B (vLLM)" "Vision-language model via vLLM"

# TRT-LLM Model on Ada
run_benchmark "llama-3-1-8b-instruct-trtllm-ada" "Llama 3.1 8B (TRT-LLM on Ada)" "TensorRT-LLM on RTX 4000 Ada GPU"

echo "========================================="
echo "All benchmarks completed!"
echo "Results saved to: $RESULTS_DIR"
echo "========================================="

# Generate summary
echo ""
echo "Generating summary..."
python3 - <<EOF
import json
import glob
import os

results_dir = "$RESULTS_DIR"
summary = []

for json_file in sorted(glob.glob(os.path.join(results_dir, "*.json"))):
    try:
        with open(json_file, 'r') as f:
            data = json.load(f)

        model_name = os.path.basename(json_file).replace('.json', '').replace('_', '/')

        # Extract metrics (structure depends on guidellm output format)
        metrics = {
            'model': model_name,
            'file': os.path.basename(json_file)
        }

        summary.append(metrics)
    except Exception as e:
        print(f"Error processing {json_file}: {e}")

print("\nBenchmark Summary:")
print(json.dumps(summary, indent=2))

# Save summary
with open(os.path.join(results_dir, "summary.json"), 'w') as f:
    json.dump(summary, f, indent=2)
EOF

echo ""
echo "Summary saved to: $RESULTS_DIR/summary.json"
